package model

import (
	"fmt"
	"time"
)

// BondQuoteDetail 债券行情明细表
type BondQuoteDetail struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                            // 主键ID
	MessageID        string     `gorm:"column:message_id;not null;index" json:"messageId"`                       // 消息ID
	MessageType      string     `gorm:"column:message_type;not null" json:"messageType"`                         // 消息类型
	Timestamp        int64      `gorm:"column:timestamp;not null" json:"timestamp"`                              // 时间戳
	ISIN             string     `gorm:"column:isin;not null;index" json:"isin"`                                  // 债券代码
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

// // TableName 设置表名
// func (BondQuoteDetail) TableName() string {
// 	return "t_bond_quote_detail"
// }

// TableName 设置表名
func (BondQuoteDetail) TableName() string {
	datestr := time.Now().Format("20060102")
	return fmt.Sprintf("t_bond_quote_detail_%s", datestr)
}

// BondLatestQuote 债券最新行情表
type BondLatestQuote struct {
	ISIN           string    `gorm:"column:isin;primaryKey" json:"isin"`                                               // 债券代码
	RawJSON        string    `gorm:"column:raw_json;type:text" json:"rawJson"`                                         // 存储完整消息JSON
	MessageID      string    `gorm:"column:message_id;index" json:"messageId"`                                         // 消息ID，便于查询
	MessageType    string    `gorm:"column:message_type" json:"messageType"`                                           // 消息类型
	SendTime       int64     `gorm:"column:send_time;index" json:"sendTime"`                                           // 消息发送时间
	Timestamp      int64     `gorm:"column:timestamp;index" json:"timestamp"`                                          // 业务时间戳
	LastUpdateTime time.Time `gorm:"column:last_update_time;not null;default:CURRENT_TIMESTAMP" json:"lastUpdateTime"` // 最后更新时间
}

// // TableName 设置表名
// func (BondLatestQuote) TableName() string {
// 	return "t_bond_latest_quote"
// }

// TableName 设置表名
func (BondLatestQuote) TableName() string {
	datestr := time.Now().Format("20060102")
	return fmt.Sprintf("t_bond_latest_quote_%s", datestr)
}
