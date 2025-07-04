package model

import (
	"time"
)

// BondQuoteDetail 债券行情明细表
type BondQuoteDetail struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                            // 主键ID
	MessageID        string     `gorm:"column:message_id;not null;index" json:"messageId"`                       // 消息ID
	MessageType      string     `gorm:"column:message_type;not null" json:"messageType"`                         // 消息类型
	Timestamp        int64      `gorm:"column:timestamp;not null" json:"timestamp"`                              // 时间戳
	SecurityCode     string     `gorm:"column:security_code;not null;index" json:"securityCode"`                 // 债券代码
	BrokerID         string     `gorm:"column:broker_id;not null" json:"brokerId"`                               // 券商ID
	Side             string     `gorm:"column:side;not null" json:"side"`                                        // 方向(BID/ASK)
	Price            float64    `gorm:"column:price;not null;type:decimal(18,6)" json:"price"`                   // 报价
	Yield            *float64   `gorm:"column:yield;type:decimal(18,6)" json:"yield"`                            // 收益率
	OrderQty         float64    `gorm:"column:order_qty;not null;type:decimal(18,2)" json:"orderQty"`            // 数量
	MinTransQuantity *float64   `gorm:"column:min_trans_quantity;type:decimal(18,2)" json:"minTransQuantity"`    // 最小交易量
	QuoteOrderNo     string     `gorm:"column:quote_order_no;not null" json:"quoteOrderNo"`                      // 报价单号
	QuoteTime        time.Time  `gorm:"column:quote_time;not null;index" json:"quoteTime"`                       // 报价时间
	SettleType       *string    `gorm:"column:settle_type" json:"settleType"`                                    // 结算类型
	SettleDate       *time.Time `gorm:"column:settle_date;type:date" json:"settleDate"`                          // 结算日期
	IsValid          *string    `gorm:"column:is_valid;type:char(1)" json:"isValid"`                             // 是否有效(Y/N)
	IsTbd            *string    `gorm:"column:is_tbd;type:char(1)" json:"isTbd"`                                 // 是否待定(Y/N)
	CreateTime       time.Time  `gorm:"column:create_time;not null;default:CURRENT_TIMESTAMP" json:"createTime"` // 创建时间
}

// TableName 设置表名
func (BondQuoteDetail) TableName() string {
	return "t_bond_quote_detail"
}

// BondLatestQuote 债券最新行情表
type BondLatestQuote struct {
	SecurityCode   string     `gorm:"column:security_code;primaryKey" json:"securityCode"`                                                 // 债券代码
	LastUpdateTime time.Time  `gorm:"column:last_update_time;not null" json:"lastUpdateTime"`                                              // 最后更新时间
	BidPrice       *float64   `gorm:"column:bid_price;type:decimal(18,6)" json:"bidPrice"`                                                 // 最优买入价
	BidYield       *float64   `gorm:"column:bid_yield;type:decimal(18,6)" json:"bidYield"`                                                 // 买入收益率
	BidQty         *float64   `gorm:"column:bid_qty;type:decimal(18,2)" json:"bidQty"`                                                     // 买入数量
	BidBrokerID    *string    `gorm:"column:bid_broker_id" json:"bidBrokerId"`                                                             // 买入券商ID
	BidQuoteTime   *time.Time `gorm:"column:bid_quote_time" json:"bidQuoteTime"`                                                           // 买入报价时间
	AskPrice       *float64   `gorm:"column:ask_price;type:decimal(18,6)" json:"askPrice"`                                                 // 最优卖出价
	AskYield       *float64   `gorm:"column:ask_yield;type:decimal(18,6)" json:"askYield"`                                                 // 卖出收益率
	AskQty         *float64   `gorm:"column:ask_qty;type:decimal(18,2)" json:"askQty"`                                                     // 卖出数量
	AskBrokerID    *string    `gorm:"column:ask_broker_id" json:"askBrokerId"`                                                             // 卖出券商ID
	AskQuoteTime   *time.Time `gorm:"column:ask_quote_time" json:"askQuoteTime"`                                                           // 卖出报价时间
	Spread         *float64   `gorm:"column:spread;type:decimal(18,6)" json:"spread"`                                                      // 买卖价差(计算列)
	UpdateTime     time.Time  `gorm:"column:update_time;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updateTime"` // 更新时间
}

// TableName 设置表名
func (BondLatestQuote) TableName() string {
	return "t_bond_latest_quote"
}
