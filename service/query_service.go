package service

import (
	"encoding/json"
	"fmt"
	"time"
	"wealth-bond-quote-service/model"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

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
func (s *BondQueryService) ExportDailyEndData(param DateRangeParam) (string, error) {
	// 解析日期
	startDate, err := time.Parse("20060102", param.StartDate)
	if err != nil {
		return "", fmt.Errorf("开始日期格式错误: %w", err)
	}

	endDate, err := time.Parse("20060102", param.EndDate)
	if err != nil {
		return "", fmt.Errorf("结束日期格式错误: %w", err)
	}

	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "债券日终行情"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{
		"日期", "债券代码",
		"买方价格", "买方收益率", "买方数量", "买方报价时间",
		"卖方价格", "卖方收益率", "卖方数量", "卖方报价时间",
		"消息ID", "消息类型", "发送时间", "时间戳", "更新时间",
		"买方券商ID", "卖方券商ID",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 设置行索引
	rowIndex := 2

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
			return "", fmt.Errorf("查询%s数据失败: %w", dateStr, err)
		}

		// 填充数据
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

			// 获取买方和卖方报价
			var bidPrices []QuotePrice
			var askPrices []QuotePrice

			if len(payload.BidPrices) > 0 {
				bidPrices = payload.BidPrices
			}

			if len(payload.AskPrices) > 0 {
				askPrices = payload.AskPrices
			}

			// 确定需要多少行
			maxRows := len(bidPrices)
			if len(askPrices) > maxRows {
				maxRows = len(askPrices)
			}

			// 如果没有任何报价，至少创建一行基本信息
			if maxRows == 0 {
				// 日期和债券代码
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), dateStr)
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), quote.ISIN)

				// 消息元数据
				f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), quote.MessageID)
				f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), quote.MessageType)
				f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"))
				f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowIndex), time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"))
				f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowIndex), quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"))
				rowIndex++
				continue
			}

			// 填充每一行数据
			for i := 0; i < maxRows; i++ {
				// 日期和债券代码
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), dateStr)
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), quote.ISIN)

				// 买方数据
				if i < len(bidPrices) {
					bid := bidPrices[i]
					f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIndex), bid.Price)
					f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIndex), bid.Yield)
					f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIndex), bid.OrderQty)
					f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), time.UnixMilli(bid.QuoteTime).Format("2006-01-02 15:04:05.000"))
					f.SetCellValue(sheetName, fmt.Sprintf("P%d", rowIndex), bid.BrokerID)
				}

				// 卖方数据
				if i < len(askPrices) {
					ask := askPrices[i]
					f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIndex), ask.Price)
					f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIndex), ask.Yield)
					f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIndex), ask.OrderQty)
					f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), time.UnixMilli(ask.QuoteTime).Format("2006-01-02 15:04:05.000"))
					f.SetCellValue(sheetName, fmt.Sprintf("Q%d", rowIndex), ask.BrokerID)
				}

				// 消息元数据
				f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), quote.MessageID)
				f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), quote.MessageType)
				f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"))
				f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowIndex), time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"))
				f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowIndex), quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"))

				rowIndex++
			}
		}
	}

	// 设置列宽
	for i := 0; i < len(headers); i++ {
		colName := fmt.Sprint('A' + i)
		f.SetColWidth(sheetName, colName, colName, 18)
	}

	// 保存文件
	filename := fmt.Sprintf("bond_daily_end_data_%s_to_%s.xlsx", param.StartDate, param.EndDate)
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败: %w", err)
	}

	return filename, nil
}

// ExportTimeRangeData 导出时间段数据（从明细表）
func (s *BondQueryService) ExportTimeRangeData(param TimeRangeParam) (string, error) {
	// 解析日期和时间
	date, err := time.Parse("20060102", param.Date)
	if err != nil {
		return "", fmt.Errorf("日期格式错误: %w", err)
	}

	dateStr := date.Format("20060102")
	tableName := fmt.Sprintf("t_bond_quote_detail_%s", dateStr)

	// 检查表是否存在
	if !s.tableExists(tableName) {
		return "", fmt.Errorf("表%s不存在", tableName)
	}

	// 构建时间范围
	startDateTime, err := time.Parse("20060102 15:04:05", fmt.Sprintf("%s %s", dateStr, param.StartTime))
	if err != nil {
		return "", fmt.Errorf("开始时间格式错误: %w", err)
	}

	endDateTime, err := time.Parse("20060102 15:04:05", fmt.Sprintf("%s %s", dateStr, param.EndTime))
	if err != nil {
		return "", fmt.Errorf("结束时间格式错误: %w", err)
	}

	// 先查询分组信息，按 quote_time 和 message_id 分组
	type MessageGroup struct {
		QuoteTime   time.Time `json:"quote_time"`
		MessageID   string    `json:"message_id"`
		MessageType string    `json:"message_type"`
		Timestamp   int64     `json:"timestamp"`
		ISIN        string    `json:"isin"`
	}

	var messageGroups []MessageGroup
	if err := s.db.Table(tableName).
		Select("quote_time, message_id, message_type, timestamp, isin").
		Where("quote_time BETWEEN ? AND ?", startDateTime, endDateTime).
		Group("message_id, isin").  // 按消息ID和债券代码分组，因为同一消息ID的其他字段应该相同
		Order("quote_time, message_id").
		Find(&messageGroups).Error; err != nil {
		return "", fmt.Errorf("查询分组数据失败: %w", err)
	}

	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "债券时间段行情"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{
		"债券代码",
		"买方价格", "买方收益率", "买方数量", "买方报价时间",
		"卖方价格", "卖方收益率", "卖方数量", "卖方报价时间",
		"消息ID", "消息类型", "发送时间", "时间戳",
		"买方券商ID", "卖方券商ID",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 设置行索引
	rowIndex := 2

	// 遍历每个消息组，获取完整的买卖报价信息
	for _, group := range messageGroups {
		// 查询该消息组的所有明细记录
		var details []model.BondQuoteDetail
		if err := s.db.Table(tableName).
			Where("message_id = ? AND isin = ?", group.MessageID, group.ISIN).
			Find(&details).Error; err != nil {
			return "", fmt.Errorf("查询明细数据失败: %w", err)
		}

		// 按买卖方向分组
		var bidPrices []model.BondQuoteDetail
		var askPrices []model.BondQuoteDetail

		for _, detail := range details {
			if detail.Side == "BID" {
				bidPrices = append(bidPrices, detail)
			} else if detail.Side == "ASK" {
				askPrices = append(askPrices, detail)
			}
		}

		// 确定需要多少行
		maxRows := len(bidPrices)
		if len(askPrices) > maxRows {
			maxRows = len(askPrices)
		}

		// 如果没有任何报价，至少创建一行基本信息
		if maxRows == 0 {
			// 债券代码
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), group.ISIN)

			// 消息元数据
			f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), group.MessageID)
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), group.MessageType)
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), group.QuoteTime.Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), time.UnixMilli(group.Timestamp).Format("2006-01-02 15:04:05.000"))
			rowIndex++
			continue
		}

		// 填充每一行数据
		for i := 0; i < maxRows; i++ {
			// 债券代码
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), group.ISIN)

			// 买方数据
			if i < len(bidPrices) {
				bid := bidPrices[i]
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), bid.Price)
				if bid.Yield != nil {
					f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIndex), *bid.Yield)
				}
				f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIndex), bid.OrderQty)
				f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIndex), bid.QuoteTime.Format("2006-01-02 15:04:05.000"))
				f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowIndex), bid.BrokerID)
			}

			// 卖方数据
			if i < len(askPrices) {
				ask := askPrices[i]
				f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), ask.Price)
				if ask.Yield != nil {
					f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIndex), *ask.Yield)
				}
				f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIndex), ask.OrderQty)
				f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIndex), ask.QuoteTime.Format("2006-01-02 15:04:05.000"))
				f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowIndex), ask.BrokerID)
			}

			// 消息元数据
			f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), group.MessageID)
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), group.MessageType)
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), group.QuoteTime.Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), time.UnixMilli(group.Timestamp).Format("2006-01-02 15:04:05.000"))

			rowIndex++
		}
	}

	// 设置列宽
	for i := 0; i < len(headers); i++ {
		colName := fmt.Sprint('A' + i)
		f.SetColWidth(sheetName, colName, colName, 18)
	}

	// 保存文件
	filename := fmt.Sprintf("bond_time_range_data_%s_%s_to_%s.xlsx",
		param.Date, param.StartTime, param.EndTime)
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败: %w", err)
	}

	return filename, nil
}

// ExportCurrentLatestQuotes 导出当前最新行情到Excel
func (s *BondQueryService) ExportCurrentLatestQuotes() (string, error) {
	// 查询当前最新行情数据
	var latestQuotes []model.BondLatestQuote
	if err := s.db.Find(&latestQuotes).Error; err != nil {
		return "", fmt.Errorf("查询最新行情数据失败: %w", err)
	}

	// 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "债券最新行情"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return "", fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{
		"债券代码",
		"买方价格", "买方收益率", "买方数量", "买方报价时间",
		"卖方价格", "卖方收益率", "卖方数量", "卖方报价时间",
		"消息ID", "消息类型", "发送时间", "时间戳", "更新时间",
		"买方券商ID", "卖方券商ID",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 设置行索引
	rowIndex := 2

	// 填充数据
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

		// 获取买方和卖方报价
		var bidPrices []QuotePrice
		var askPrices []QuotePrice

		if len(payload.BidPrices) > 0 {
			bidPrices = payload.BidPrices
		}

		if len(payload.AskPrices) > 0 {
			askPrices = payload.AskPrices
		}

		// 确定需要多少行
		maxRows := len(bidPrices)
		if len(askPrices) > maxRows {
			maxRows = len(askPrices)
		}

		// 如果没有任何报价，至少创建一行基本信息
		if maxRows == 0 {
			// 债券代码
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), quote.ISIN)

			// 消息元数据
			f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), quote.MessageID)
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), quote.MessageType)
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowIndex), quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"))
			rowIndex++
			continue
		}

		// 填充每一行数据
		for i := 0; i < maxRows; i++ {
			// 债券代码
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), quote.ISIN)

			// 买方数据
			if i < len(bidPrices) {
				bid := bidPrices[i]
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), bid.Price)
				f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIndex), bid.Yield)
				f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIndex), bid.OrderQty)
				f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIndex), time.UnixMilli(bid.QuoteTime).Format("2006-01-02 15:04:05.000"))
				f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowIndex), bid.BrokerID)
			}

			// 卖方数据
			if i < len(askPrices) {
				ask := askPrices[i]
				f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), ask.Price)
				f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIndex), ask.Yield)
				f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIndex), ask.OrderQty)
				f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIndex), time.UnixMilli(ask.QuoteTime).Format("2006-01-02 15:04:05.000"))
				f.SetCellValue(sheetName, fmt.Sprintf("P%d", rowIndex), ask.BrokerID)
			}

			// 消息元数据
			f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), quote.MessageID)
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), quote.MessageType)
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowIndex), quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"))

			rowIndex++
		}
	}

	// 设置列宽
	for i := 0; i < len(headers); i++ {
		colName := fmt.Sprint('A' + i)
		f.SetColWidth(sheetName, colName, colName, 18)
	}

	// 生成文件名
	filename := fmt.Sprintf("bond_latest_quotes_%s.xlsx", time.Now().Format("20060102_150405"))

	// 保存文件
	if err := f.SaveAs(filename); err != nil {
		return "", fmt.Errorf("保存Excel文件失败: %w", err)
	}

	return filename, nil
}

func (s *BondQueryService) tableExists(tableName string) bool {
	return s.db.Migrator().HasTable(tableName)
}
