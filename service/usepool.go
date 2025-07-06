package service

import (
	"encoding/json"
	"errors"
	"sync"
)

// ErrSkip 返回给上层表示“这条消息类型不处理”
var ErrSkip = errors.New("skip message")

// ParseService 负责把原始 JSON 转为业务结构体
type ParseService struct {
	msgPool sync.Pool // 复用外层对象
	qpdPool sync.Pool // 复用内层对象
}

func NewParseService() *ParseService {
	return &ParseService{
		msgPool: sync.Pool{New: func() any { return new(BondQuoteMessage) }},
		qpdPool: sync.Pool{New: func() any { return new(QuotePriceData) }},
	}
}

// Parse 把 STOMP 带来的原始 JSON 解析成结构体
// 返回：外层消息、内层报价数据、错误
func (p *ParseService) Parse(raw []byte) (*BondQuoteMessage,
	*QuotePriceData, error) {

	// ---------- 外层 ----------
	msg := p.msgPool.Get().(*BondQuoteMessage)
	*msg = BondQuoteMessage{} // reset
	if err := json.Unmarshal(raw, msg); err != nil {
		p.msgPool.Put(msg)
		return nil, nil, err
	}

	if msg.WsMessageType != "ATS_QUOTE" {
		p.msgPool.Put(msg)
		return nil, nil, ErrSkip
	}

	// ---------- 内层 ----------
	qpd := p.qpdPool.Get().(*QuotePriceData)
	*qpd = QuotePriceData{} // reset
	if err := json.Unmarshal([]byte(msg.Data.QuotePriceData), qpd); err != nil {
		p.msgPool.Put(msg)
		p.qpdPool.Put(qpd)
		return nil, nil, err
	}

	return msg, qpd, nil
}

// Put 回收对象（调用方在用完后记得归还）
func (p *ParseService) Put(msg *BondQuoteMessage, qpd *QuotePriceData) {
	if msg != nil {
		p.msgPool.Put(msg)
	}
	if qpd != nil {
		p.qpdPool.Put(qpd)
	}
}
