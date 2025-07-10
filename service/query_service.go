package service

import (
	"encoding/json"
	"fmt"
	"test/model"
	"time"

	"gorm.io/gorm"
)

// BondQueryService 债券行情查询服务
type BondQueryService struct {
	db *gorm.DB
}

// NewBondQueryService 创建债券行情查询服务
func NewBondQueryService(db *gorm.DB) *BondQueryService {
	return &BondQueryService{db: db}
}

// 日期范围参数
type DateRangeParam struct {
	StartDate string `json:"startDate" form:"startDate"` // 格式: YYYYMMDD
	EndDate   string `json:"endDate" form:"endDate"`     // 格式: YYYYMMDD
}

// 时间段参数
type TimeRangeParam struct {
	Date      string `json:"date" form:"date"`           // 格式: YYYYMMDD
	StartTime string `json:"startTime" form:"startTime"` // 格式: HH:MM:SS
	EndTime   string `json:"endTime" form:"endTime"`     // 格式: HH:MM:SS
}

// ExportDailyEndData 导出日终数据（从最新行情表）
func (s *BondQueryService) ExportDailyEndData(param DateRangeParam) ([]map[string]any, error) {
	// 解析日期
	startDate, err := time.Parse("20060102", param.StartDate)
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误: %w", err)
	}

	endDate, err := time.Parse("20060102", param.EndDate)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误: %w", err)
	}

	// 存储结果
	var result []map[string]any

	// 遍历日期范围
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("20060102")
		tableName := fmt.Sprintf("t_bond_latest_quote_%s", dateStr)

		// 检查表是否存在
		if !s.tableExists(tableName) {
			continue
		}

		// 查询当天最新行情数据
		var latestQuotes []model.BondLatestQuote
		if err := s.db.Table(tableName).Find(&latestQuotes).Error; err != nil {
			return nil, fmt.Errorf("查询%s数据失败: %w", dateStr, err)
		}

		// 处理每条数据
		for _, quote := range latestQuotes {
			// 解析JSON数据
			var msg BondQuoteMessage
			if err := json.Unmarshal([]byte(quote.RawJSON), &msg); err != nil {
				continue
			}

			// 解析内部报价数据
			var payload QuotePriceData
			if err := json.Unmarshal([]byte(msg.Data.QuotePriceData), &payload); err != nil {
				continue
			}

			// 构造返回数据
			data := map[string]any{
				"date":           dateStr,
				"bondCode":       quote.ISIN,
				"messageID":      quote.MessageID,
				"messageType":    quote.MessageType,
				"sendTime":       time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"),
				"timestamp":      time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"),
				"lastUpdateTime": quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"),
				"bidPrices":      payload.BidPrices,
				"askPrices":      payload.AskPrices,
			}

			result = append(result, data)
		}
	}

	return result, nil
}

// ExportTimeRangeData 导出时间段数据（从明细表）
func (s *BondQueryService) ExportTimeRangeData(param TimeRangeParam) ([]map[string]any, error) {
	// 解析日期和时间
	date, err := time.Parse("20060102", param.Date)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误: %w", err)
	}

	dateStr := date.Format("20060102")
	tableName := fmt.Sprintf("t_bond_quote_detail_%s", dateStr)

	// 检查表是否存在
	if !s.tableExists(tableName) {
		return nil, fmt.Errorf("表%s不存在", tableName)
	}

	// 构建时间范围
	startDateTime, err := time.Parse("20060102 15:04:05", fmt.Sprintf("%s %s", dateStr, param.StartTime))
	if err != nil {
		return nil, fmt.Errorf("开始时间格式错误: %w", err)
	}

	endDateTime, err := time.Parse("20060102 15:04:05", fmt.Sprintf("%s %s", dateStr, param.EndTime))
	if err != nil {
		return nil, fmt.Errorf("结束时间格式错误: %w", err)
	}

	// 查询指定时间段的明细数据
	var details []model.BondQuoteDetail
	if err := s.db.Table(tableName).
		Where("quote_time BETWEEN ? AND ?", startDateTime, endDateTime).
		Order("quote_time").
		Find(&details).Error; err != nil {
		return nil, fmt.Errorf("查询时间段数据失败: %w", err)
	}

	// 构造返回数据
	var result []map[string]any
	for _, detail := range details {
		data := map[string]any{
			"bondCode":    detail.ISIN,
			"side":        detail.Side,
			"price":       detail.Price,
			"yield":       detail.Yield,
			"orderQty":    detail.OrderQty,
			"quoteTime":   detail.QuoteTime.Format("2006-01-02 15:04:05.000"),
			"brokerID":    detail.BrokerID,
			"messageID":   detail.MessageID,
			"messageType": detail.MessageType,
			"timestamp":   time.UnixMilli(detail.Timestamp).Format("2006-01-02 15:04:05.000"),
		}
		result = append(result, data)
	}

	return result, nil
}

// 检查表是否存在
func (s *BondQueryService) tableExists(tableName string) bool {
	var count int64
	s.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", tableName).Count(&count)
	return count > 0
}
