package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"test/model"
)

var (
	msgPool = sync.Pool{
		New: func() interface{} {
			return &BondQuoteMessage{}
		},
	}
	qpdPool = sync.Pool{
		New: func() interface{} {
			return &QuotePriceData{}
		},
	}
)

// BondQuoteService 债券行情服务
type BondQuoteService struct {
	db         *gorm.DB
	wg         *sync.WaitGroup
	RawChan    chan []byte
	ParsedChan chan *ParsedQuote
	DeadChan   chan []byte
}

// NewBondQuoteService 创建债券行情服务
func NewBondQuoteService(db *gorm.DB, wg *sync.WaitGroup, RawChan chan []byte, ParsedChan chan *ParsedQuote, DeadChan chan []byte) *BondQuoteService {
	return &BondQuoteService{
		db:         db,
		wg:         wg,
		RawChan:    RawChan,
		ParsedChan: ParsedChan,
		DeadChan:   DeadChan,
	}
}

// 响应消息结构体
type BondQuoteMessage struct {
	Data          BondQuoteData `json:"data"`
	SendTime      int64         `json:"sendTime"`
	WsMessageType string        `json:"wsMessageType"`
}

type BondQuoteData struct {
	QuotePriceData string `json:"data"` // 内部JSON字符串
	MessageID      string `json:"messageId"`
	MessageType    string `json:"messageType"`
	Organization   string `json:"organization"`
	ReceiverID     string `json:"receiverId"`
	Timestamp      int64  `json:"timestamp"`
}

// 报价数据结构体 - 用于解析内部JSON字符串
type QuotePriceData struct {
	AskPrices  []QuotePrice `json:"askPrices"`
	BidPrices  []QuotePrice `json:"bidPrices"`
	SecurityID string       `json:"securityId"`
}

// 报价结构体
type QuotePrice struct {
	BrokerID         string  `json:"brokerId"`
	IsTbd            string  `json:"isTbd"`
	IsValid          string  `json:"isValid"`
	MinTransQuantity float64 `json:"minTransQuantity"`
	OrderQty         float64 `json:"orderQty"`
	Price            float64 `json:"price"`
	QuoteOrderNo     string  `json:"quoteOrderNo"`
	QuoteTime        int64   `json:"quoteTime"`
	SecurityID       string  `json:"securityId"`
	SettleType       string  `json:"settleType"`
	Side             string  `json:"side"`
	Yield            float64 `json:"yield"`
}

// ParsedQuote 解析结果：外层元信息 + 内层行情数据
type ParsedQuote struct {
	Meta    BondQuoteMessage // WsMessageType、MessageId...
	Payload QuotePriceData   // askPrices / bidPrices / securityId
}

// ParseBondQuote 把 STOMP body 原始 JSON 解析成领域对象
func ParseBondQuote(raw []byte) (*ParsedQuote, error) {
	msg := msgPool.Get().(*BondQuoteMessage)
	defer msgPool.Put(msg)
	if err := json.Unmarshal(raw, msg); err != nil {
		return nil, fmt.Errorf("unmarshal BondQuoteMessage: %w", err)
	}

	payload := qpdPool.Get().(*QuotePriceData)
	defer qpdPool.Put(payload)
	if err := json.Unmarshal([]byte(msg.Data.QuotePriceData), payload); err != nil {
		return nil, fmt.Errorf("unmarshal QuotePriceData: %w", err)
	}

	// isin必须存在
	if payload.SecurityID == "" {
		return nil, errors.New("securityId is empty")
	}

	return &ParsedQuote{
		Meta:    *msg,
		Payload: *payload,
	}, nil
}

// StartParseWorkers — 解析层
func (bqs *BondQuoteService) StartParseWorkers(workerNum int) {
	for i := 0; i < workerNum; i++ {
		bqs.wg.Add(1)
		go func() {
			defer bqs.wg.Done()
			for raw := range bqs.RawChan {
				pq, err := ParseBondQuote(raw)
				switch {
				// case err == service.ErrNotQuote:
				// 	continue // 过滤非行情
				case err != nil:
					bqs.DeadChan <- raw
					continue
				}
				bqs.ParsedChan <- pq
			}
		}()
	}
}

// StartDBWorkers — 写库层
func (bqs *BondQuoteService) StartDBWorkers(workerNum int, batchSize int, flushDelay time.Duration) {
	// 提前关掉自动事务和预编译
	bqs.db = bqs.db.Session(&gorm.Session{SkipDefaultTransaction: true, PrepareStmt: true})

	for i := 0; i < workerNum; i++ {
		bqs.wg.Add(1)
		go func() {
			defer bqs.wg.Done()
			ticker := time.NewTicker(flushDelay)
			batch := make([]*ParsedQuote, 0, batchSize)

			flush := func() {
				if len(batch) == 0 {
					return
				}
				if err := InsertBatch(bqs.db, batch); err != nil {
					log.Printf("批量写库失败: %v", err)
				}
				batch = batch[:0]
			}

			for {
				select {
				case pq, ok := <-bqs.ParsedChan:
					if !ok { // channel 关闭，写最后一批
						flush()
						return
					}
					batch = append(batch, pq)
					if len(batch) >= batchSize {
						flush()
					}
				case <-ticker.C:
					flush()
				}
			}
		}()
	}
}

// GetTodayTableName 获取当天表名
func GetTodayDetailTableName() string {
	// return fmt.Sprintf("t_bond_quote_detail_%s", time.Now().Format("20060102"))
	return "t_bond_quote_detail"
}

func GetTodayLatestTableName() string {
	// return fmt.Sprintf("t_bond_latest_quote_%s", time.Now().Format("20060102"))
	return "t_bond_latest_quote"

}

// InsertBatch 把解析后的批次写入 DB
func InsertBatch(db *gorm.DB, batch []*ParsedQuote) error {
	// 获取当天表名
	detailName := GetTodayDetailTableName()
	lastestName := GetTodayLatestTableName()

	// // 确保表存在
	// if err := EnsureTableExists(db, tableName); err != nil {
	// 	return err
	// }

	// 1. 聚合
	var details []model.BondQuoteDetail
	latestMap := make(map[string]*model.BondLatestQuote)

	for _, pq := range batch {
		meta := pq.Meta       // 外层
		payload := pq.Payload // 内层

		// ASK / BID 明细
		addDetail := func(q QuotePrice) {
			qTime := time.UnixMilli(q.QuoteTime)
			yield := q.Yield
			minQty := q.MinTransQuantity

			details = append(details, model.BondQuoteDetail{
				MessageID:        meta.Data.MessageID,
				MessageType:      meta.Data.MessageType,
				Timestamp:        meta.Data.Timestamp,
				ISIN:             payload.SecurityID,
				BrokerID:         q.BrokerID,
				Side:             q.Side,
				Price:            q.Price,
				Yield:            &yield,
				OrderQty:         q.OrderQty,
				MinTransQuantity: &minQty,
				QuoteOrderNo:     q.QuoteOrderNo,
				QuoteTime:        qTime,
				SettleType:       &q.SettleType,
				IsValid:          &q.IsValid,
				IsTbd:            &q.IsTbd,
				CreateTime:       time.Now(),
			})
		}

		for _, ask := range payload.AskPrices {
			addDetail(ask)
		}
		for _, bid := range payload.BidPrices {
			addDetail(bid)
		}

		// 最新价处理（基于消息发送时间比较）
		sendTime := time.UnixMilli(meta.SendTime)

		// 检查是否需要更新（基于SendTime）
		lq, ok := latestMap[payload.SecurityID]
		if !ok {
			lq = &model.BondLatestQuote{ISIN: payload.SecurityID}
			latestMap[payload.SecurityID] = lq
		}

		// 如果消息更新，则更新记录
		shouldUpdate := lq.LastUpdateTime.IsZero() || sendTime.After(lq.LastUpdateTime)
		if shouldUpdate {
			// 将整个消息存储为JSON
			rawJSON, err := json.Marshal(meta)
			if err != nil {
				return fmt.Errorf("marshal message to JSON: %w", err)
			}

			lq.RawJSON = string(rawJSON)
			lq.MessageID = meta.Data.MessageID
			lq.MessageType = meta.Data.MessageType
			lq.SendTime = meta.SendTime
			lq.Timestamp = meta.Data.Timestamp
			lq.LastUpdateTime = sendTime
		}
	}

	// 2. 执行事务
	return db.Transaction(func(tx *gorm.DB) error {
		// 明细批量写 - 使用指定的表名
		if len(details) > 0 {
			// 使用指定表名插入数据
			if err := tx.Table(detailName).CreateInBatches(details, 1000).Error; err != nil {
				return err
			}
		}

		// 最新价 UPSERT
		if len(latestMap) > 0 {
			var latestSlice []model.BondLatestQuote
			for _, v := range latestMap {
				latestSlice = append(latestSlice, *v)
			}

			if err := tx.Table(lastestName).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "isin"}}, // 唯一键
				UpdateAll: true,
			}).Create(&latestSlice).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
