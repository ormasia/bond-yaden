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

// BondQuoteService 债券行情服务
type BondQuoteService struct {
	db *gorm.DB
}

// NewBondQuoteService 创建债券行情服务
func NewBondQuoteService(db *gorm.DB) *BondQuoteService {
	return &BondQuoteService{db: db}
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
	QuoteTime        int64   `json:"quoteTime"` // 注意：这是毫秒时间戳
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
	var msg BondQuoteMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal BondQuoteMessage: %w", err)
	}

	// if msg.WsMessageType != "ATS_QUOTE" {
	// 	return nil, ErrNotQuote // 上层可选择直接丢弃
	// }

	var payload QuotePriceData
	if err := json.Unmarshal([]byte(msg.Data.QuotePriceData), &payload); err != nil {
		return nil, fmt.Errorf("unmarshal QuotePriceData: %w", err)
	}

	// isin必须存在
	if payload.SecurityID == "" {
		return nil, errors.New("securityId is empty")
	}

	return &ParsedQuote{
		Meta:    msg,
		Payload: payload,
	}, nil
}

// channels & constants – 调整容量/批量大小即可
const (
	rawCap     = 20000 // 原始 JSON 缓冲
	parsedCap  = 4000  // 解析后缓冲
	workerNum  = 8     // 解析/写库协程数
	batchSize  = 300   // 单次批写条数
	flushDelay = 100 * time.Millisecond
)

var (
	RawChan    = make(chan []byte, rawCap)
	ParsedChan = make(chan *ParsedQuote, parsedCap)
	DeadChan   = make(chan []byte, 1000) // 解析失败
)

// StartParseWorkers — 解析层
func StartParseWorkers(pool *sync.WaitGroup) {
	for i := 0; i < workerNum; i++ {
		pool.Add(1)
		go func() {
			defer pool.Done()
			for raw := range RawChan {
				pq, err := ParseBondQuote(raw)
				switch {
				// case err == service.ErrNotQuote:
				// 	continue // 过滤非行情
				case err != nil:
					DeadChan <- raw
					continue
				}
				ParsedChan <- pq
			}
		}()
	}
}

// StartDBWorkers — 写库层
func StartDBWorkers(db *gorm.DB, pool *sync.WaitGroup) {
	// 提前关掉自动事务和预编译
	db = db.Session(&gorm.Session{SkipDefaultTransaction: true, PrepareStmt: true})

	for i := 0; i < workerNum; i++ {
		pool.Add(1)
		go func() {
			defer pool.Done()
			ticker := time.NewTicker(flushDelay)
			batch := make([]*ParsedQuote, 0, batchSize)

			flush := func() {
				if len(batch) == 0 {
					return
				}
				if err := InsertBatch(db, batch); err != nil {
					log.Printf("批量写库失败: %v", err)
				}
				batch = batch[:0]
			}

			for {
				select {
				case pq, ok := <-ParsedChan:
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

// InsertBatch 把解析后的批次写入 DB
func InsertBatch(db *gorm.DB, batch []*ParsedQuote) error {
	// 1. 聚合
	var details []model.BondQuoteDetail
	latestMap := make(map[string]*model.BondLatestQuote)

	for _, pq := range batch {
		meta := pq.Meta  // 外层
		pd := pq.Payload // 内层

		// ASK / BID 明细
		addDetail := func(q QuotePrice) {
			qTime := time.UnixMilli(q.QuoteTime)
			yield := q.Yield
			minQty := q.MinTransQuantity

			details = append(details, model.BondQuoteDetail{
				MessageID:        meta.Data.MessageID,
				MessageType:      meta.Data.MessageType,
				Timestamp:        meta.Data.Timestamp,
				ISIN:             pd.SecurityID,
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

		for _, ask := range pd.AskPrices {
			addDetail(ask)
		}
		for _, bid := range pd.BidPrices {
			addDetail(bid)
		}

		// 最新价挑选（在同一批内取最优）
		updateBest := func(q QuotePrice) {
			lq, ok := latestMap[pd.SecurityID]
			if !ok {
				lq = &model.BondLatestQuote{ISIN: pd.SecurityID}
				latestMap[pd.SecurityID] = lq
			}
			switch q.Side {
			case "BID":
				if lq.BidPrice == nil || q.Price > *lq.BidPrice {
					price, qty, yld := q.Price, q.OrderQty, q.Yield
					brokerID := q.BrokerID
					qTime := time.UnixMilli(q.QuoteTime)
					lq.BidPrice, lq.BidQty, lq.BidYield = &price, &qty, &yld
					lq.BidBrokerID, lq.BidQuoteTime = &brokerID, &qTime
				}
			case "ASK":
				if lq.AskPrice == nil || q.Price < *lq.AskPrice {
					price, qty, yld := q.Price, q.OrderQty, q.Yield
					brokerID := q.BrokerID
					qTime := time.UnixMilli(q.QuoteTime)
					lq.AskPrice, lq.AskQty, lq.AskYield = &price, &qty, &yld
					lq.AskBrokerID, lq.AskQuoteTime = &brokerID, &qTime
				}
			}
			lq.LastUpdateTime = time.Now()
		}
		for _, ask := range pd.AskPrices {
			updateBest(ask)
		}
		for _, bid := range pd.BidPrices {
			updateBest(bid)
		}
	}

	// 2. 执行事务
	return db.Transaction(func(tx *gorm.DB) error {
		// 明细批量写
		if len(details) > 0 {
			if err := tx.CreateInBatches(details, 1000).Error; err != nil {
				return err
			}
		}

		// 最新价 UPSERT
		if len(latestMap) > 0 {
			var latestSlice []model.BondLatestQuote
			for _, v := range latestMap {
				latestSlice = append(latestSlice, *v)
			}

			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "isin"}}, // 唯一键
				UpdateAll: true,
			}).Create(&latestSlice).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ProcessMessage 处理推送消息
func (s *BondQuoteService) ProcessMessage(messageBody []byte) error {
	// 1. 解析外层JSON消息
	var message BondQuoteMessage
	if err := json.Unmarshal(messageBody, &message); err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 只处理债券行情消息
	if message.WsMessageType != "ATS_QUOTE" {
		return nil
	}

	// 2. 解析内部JSON字符串
	var priceData QuotePriceData
	if err := json.Unmarshal([]byte(message.Data.QuotePriceData), &priceData); err != nil {
		return fmt.Errorf("解析报价数据失败: %w", err)
	}

	// 开启事务
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 3. 处理卖出报价(ASK)
		for _, ask := range priceData.AskPrices {
			if err := s.processQuoteDetail(tx, message, ask, priceData.SecurityID); err != nil {
				return err
			}
		}

		// 4. 处理买入报价(BID)
		for _, bid := range priceData.BidPrices {
			if err := s.processQuoteDetail(tx, message, bid, priceData.SecurityID); err != nil {
				return err
			}
		}

		// 5. 更新最新行情
		if err := s.updateLatestQuote(tx, message, priceData); err != nil {
			return err
		}

		return nil
	})
}

// 处理行情明细
func (s *BondQuoteService) processQuoteDetail(tx *gorm.DB, message BondQuoteMessage, quote QuotePrice, securityID string) error {
	// 转换报价时间（毫秒时间戳）
	quoteTime := time.UnixMilli(quote.QuoteTime)

	// 创建行情明细记录
	yield := quote.Yield
	minTransQty := quote.MinTransQuantity
	detail := model.BondQuoteDetail{
		MessageID:        message.Data.MessageID,
		MessageType:      message.Data.MessageType,
		Timestamp:        message.Data.Timestamp,
		ISIN:             securityID,
		BrokerID:         quote.BrokerID,
		Side:             quote.Side,
		Price:            quote.Price,
		Yield:            &yield,
		OrderQty:         quote.OrderQty,
		MinTransQuantity: &minTransQty,
		QuoteOrderNo:     quote.QuoteOrderNo,
		QuoteTime:        quoteTime,
		SettleType:       &quote.SettleType,
		IsValid:          &quote.IsValid,
		IsTbd:            &quote.IsTbd,
	}

	// 保存到数据库
	return tx.Create(&detail).Error
}

// 更新最新行情
func (s *BondQuoteService) updateLatestQuote(tx *gorm.DB, message BondQuoteMessage, priceData QuotePriceData) error {
	securityID := priceData.SecurityID

	// 查找最新买入和卖出报价
	var bestBid, bestAsk *QuotePrice

	// 找出最优买入价(价格最高的买入报价)
	if len(priceData.BidPrices) > 0 {
		bestBid = &priceData.BidPrices[0]
		for i := range priceData.BidPrices {
			if priceData.BidPrices[i].Price > bestBid.Price {
				bestBid = &priceData.BidPrices[i]
			}
		}
	}

	// 找出最优卖出价(价格最低的卖出报价)
	if len(priceData.AskPrices) > 0 {
		bestAsk = &priceData.AskPrices[0]
		for i := range priceData.AskPrices {
			if priceData.AskPrices[i].Price < bestAsk.Price {
				bestAsk = &priceData.AskPrices[i]
			}
		}
	}

	// 准备更新数据
	latestQuote := model.BondLatestQuote{
		ISIN:           securityID,
		LastUpdateTime: time.Now(),
	}

	// 设置买入报价信息
	if bestBid != nil {
		bidQuoteTime := time.UnixMilli(bestBid.QuoteTime)
		bidPrice := bestBid.Price
		bidYield := bestBid.Yield
		bidQty := bestBid.OrderQty
		bidBrokerID := bestBid.BrokerID

		latestQuote.BidPrice = &bidPrice
		latestQuote.BidYield = &bidYield
		latestQuote.BidQty = &bidQty
		latestQuote.BidBrokerID = &bidBrokerID
		latestQuote.BidQuoteTime = &bidQuoteTime
	}

	// 设置卖出报价信息
	if bestAsk != nil {
		askQuoteTime := time.UnixMilli(bestAsk.QuoteTime)
		askPrice := bestAsk.Price
		askYield := bestAsk.Yield
		askQty := bestAsk.OrderQty
		askBrokerID := bestAsk.BrokerID

		latestQuote.AskPrice = &askPrice
		latestQuote.AskYield = &askYield
		latestQuote.AskQty = &askQty
		latestQuote.AskBrokerID = &askBrokerID
		latestQuote.AskQuoteTime = &askQuoteTime
	}

	// 使用Upsert操作(有则更新，无则插入)
	return tx.Save(&latestQuote).Error
}
