package service

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

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
	Data struct {
		Data         string `json:"data"` // 注意：这是JSON字符串，需要二次解析
		MessageID    string `json:"messageId"`
		MessageType  string `json:"messageType"`
		Organization string `json:"organization"`
		ReceiverID   string `json:"receiverId"`
		Timestamp    int64  `json:"timestamp"`
	} `json:"data"`
	SendTime      int64  `json:"sendTime"`
	WsMessageType string `json:"wsMessageType"`
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
	if err := json.Unmarshal([]byte(message.Data.Data), &priceData); err != nil {
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
