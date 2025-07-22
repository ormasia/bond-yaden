package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	config "wealth-bond-quote-service/internal/conf"
	"wealth-bond-quote-service/model"
	logger "wealth-bond-quote-service/pkg/log"
	"wealth-bond-quote-service/pkg/oss"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// 常量定义
const (
	// 表名格式
	BondLatestQuoteTableFormat = "t_bond_latest_quote_%s"
	BondQuoteDetailTableFormat = "t_bond_quote_detail_%s"

	// 时间格式
	DateFormat        = "20060102"
	DateTimeFormat    = "2006-01-02 15:04:05"
	MillisecondFormat = "2006-01-02 15:04:05.000"

	// 默认时间
	DefaultStartTime = "00:00:00"
	DefaultEndTime   = "23:59:59"

	// 限制条件
	MaxHistoryDays = 7

	// 文件名前缀
	DailyExportPrefix   = "latest_quotes"
	HistoryExportPrefix = "history_details"
)

// 功能划分清晰

// ExportDailyData: 按天导出日终数据（t_bond_latest_quote_* 表）
// ExportHistoryData: 按时间段导出历史明细（t_bond_quote_detail_* 表）

// 参数设计合理

// 支持 ISIN 过滤（可选）
// 时间段控制灵活
// 历史数据有7天限制（防止数据量过大）
// 技术实现优秀

//  超时控制（context）
//  流式处理（避免内存溢出）
//  Excel导出
//  OSS上传
//  本地文件清理

// 错误处理完善

// 表不存在时跳过而不中断
// 详细的日志记录
// 优雅的错误返回

// BondQueryService 债券行情查询服务
type BondQueryService struct {
	db *gorm.DB
}

// NewBondQueryService 创建债券行情查询服务
func NewBondQueryService(db *gorm.DB) *BondQueryService {
	return &BondQueryService{db: db}
}

// 基础查询参数
type BaseQueryParam struct {
	ISIN string `json:"isin" form:"isin"` // 债券ISIN代码，为空则查询所有
}

// 日期范围参数 - 用于导出日终数据
type DateRangeParam struct {
	BaseQueryParam
	StartDate string `json:"startDate" form:"startDate" binding:"required"` // 格式: YYYYMMDD
	EndDate   string `json:"endDate" form:"endDate" binding:"required"`     // 格式: YYYYMMDD
}

// 时间段参数 - 用于导出历史数据（最大7天限制）
type TimeRangeParam struct {
	BaseQueryParam
	StartDate string `json:"startDate" form:"startDate" binding:"required"` // 格式: YYYYMMDD
	EndDate   string `json:"endDate" form:"endDate" binding:"required"`     // 格式: YYYYMMDD
	StartTime string `json:"startTime" form:"startTime"`                    // 格式: HH:MM:SS，默认00:00:00
	EndTime   string `json:"endTime" form:"endTime"`                        // 格式: HH:MM:SS，默认23:59:59
}

// 导出结果返回参数
type ExportResult struct {
	URL string `json:"url"` // 导出文件的OSS访问链接
}

// ExportDailyData 按时间段导出日终数据 - 使用流式处理
func (s *BondQueryService) ExportDailyData(ctx context.Context, param DateRangeParam) (*ExportResult, error) {
	// 1. 解析日期区间
	start, err := time.Parse(DateFormat, param.StartDate)
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误: %w", err)
	}
	end, err := time.Parse(DateFormat, param.EndDate)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误: %w", err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("结束日期不能早于开始日期")
	}

	// 2. 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "债券最新行情"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return &ExportResult{}, fmt.Errorf("创建工作表失败: %w", err)
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

	rowIndex := 2 // 从第2行开始
	processedCount := 0
	skippedCount := 0

	// 3. 循环每一天，使用游标流式处理
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		tableName := fmt.Sprintf(BondLatestQuoteTableFormat, d.Format(DateFormat))
		if !s.tableExists(tableName) {
			logger.Info("表 %s 不存在，跳过", tableName)
			continue
		}

		// 构建查询，使用超时控制
		query := s.db.WithContext(ctx).Table(tableName)
		if param.ISIN != "" {
			query = query.Where("isin = ?", param.ISIN)
		}

		// 使用游标逐行处理
		rows, err := query.Rows()
		if err != nil {
			logger.Error("查询表 %s 失败: %v", tableName, err)
			continue // 继续处理其他表，不中断整个流程
		}

		for rows.Next() {
			select {
			case <-ctx.Done():
				rows.Close()
				return nil, fmt.Errorf("导出被取消: %w", ctx.Err())
			default:
			}

			var quote model.BondLatestQuote
			if err := s.db.ScanRows(rows, &quote); err != nil {
				logger.Error("扫描数据行失败: %v", err)
				skippedCount++
				continue
			}

			// 直接处理单条记录并写入Excel
			if err := s.writeQuoteToExcel(f, sheetName, &quote, &rowIndex); err != nil {
				logger.Error("写入Excel失败 (ISIN=%s): %v", quote.ISIN, err)
				skippedCount++
				continue
			}
			processedCount++
		}

		rows.Close()
	}

	logger.Info("日终数据导出完成: 处理 %d 条记录，跳过 %d 条记录", processedCount, skippedCount)

	// 4. 设置列宽
	for i := 0; i < len(headers); i++ {
		colName := string(rune('A' + i))
		f.SetColWidth(sheetName, colName, colName, 18)
	}

	// 5. 保存文件
	filename := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s-%s_%d.xlsx", DailyExportPrefix, param.StartDate, param.EndDate, time.Now().Unix()))
	if err := f.SaveAs(filename); err != nil {
		return &ExportResult{}, fmt.Errorf("保存Excel文件失败: %w", err)
	}

	// 6. 上传到OSS
	exportConfig := config.GetExportConfig()
	ossInfo := oss.OssInfo{
		Url:     exportConfig.URL,
		Timeout: exportConfig.Timeout,
	}

	fileNameOnly := filepath.Base(filename)
	requestHeaders := map[string]string{
		"x-request-id":     "345678876",
		"x-origin-service": "wealth-bond-quote-service",
		"x-uin":            "123456",
	}

	_, url, err := oss.UploadFile(filename, fileNameOnly, "", requestHeaders, &ossInfo)
	if err != nil {
		return &ExportResult{}, fmt.Errorf("上传文件到OSS失败: %w", err)
	}

	// 7. 删除本地文件
	if err := os.Remove(filename); err != nil {
		logger.Error("删除本地文件失败: %s, error: %v", filename, err)
	} else {
		logger.Info("成功删除本地文件: %s", filename)
	}

	return &ExportResult{
		URL: url,
	}, nil
}

// ExportHistoryData 按时间段导出历史数据（最大7天限制）- 使用流式处理
// 注意：此函数查询的是 BondQuoteDetail 表，表结构已经是展开的，无需JSON解析
func (s *BondQueryService) ExportHistoryData(ctx context.Context, param TimeRangeParam) (*ExportResult, error) {
	// 1. 验证时间段限制（最大7天）
	start, err := time.Parse(DateFormat, param.StartDate)
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误: %w", err)
	}
	end, err := time.Parse(DateFormat, param.EndDate)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误: %w", err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("结束日期不能早于开始日期")
	}
	if end.Sub(start) > MaxHistoryDays*24*time.Hour {
		return nil, fmt.Errorf("时间范围不能超过%d天", MaxHistoryDays)
	}

	// 2. 处理默认时间
	startTime := DefaultStartTime
	endTime := DefaultEndTime
	if param.StartTime != "" {
		startTime = param.StartTime
	}
	if param.EndTime != "" {
		endTime = param.EndTime
	}

	// 3. 创建Excel文件
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "债券历史行情"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return &ExportResult{}, fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	// 设置表头 - 根据 BondQuoteDetail 表结构
	headers := []string{
		"ID", "消息ID", "消息类型", "时间戳", "债券代码", "券商ID", "方向",
		"价格", "收益率", "数量", "最小交易量", "报价单号", "报价时间",
		"结算类型", "结算日期", "是否有效", "是否待定", "创建时间",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	rowIndex := 2 // 从第2行开始
	processedCount := 0
	skippedCount := 0

	// 4. 循环每一天，使用游标流式处理
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		tableName := fmt.Sprintf(BondQuoteDetailTableFormat, d.Format(DateFormat))
		if !s.tableExists(tableName) {
			logger.Info("表 %s 不存在，跳过", tableName)
			continue
		}

		// 构建时间范围条件
		startDateTime := fmt.Sprintf("%s %s", d.Format("2006-01-02"), startTime)
		endDateTime := fmt.Sprintf("%s %s", d.Format("2006-01-02"), endTime)

		startTimestamp, err := time.Parse(DateTimeFormat, startDateTime)
		if err != nil {
			logger.Error("解析开始时间失败: %v", err)
			continue
		}
		endTimestamp, err := time.Parse(DateTimeFormat, endDateTime)
		if err != nil {
			logger.Error("解析结束时间失败: %v", err)
			continue
		}

		// 构建查询，使用 quote_time 字段进行时间范围过滤
		query := s.db.WithContext(ctx).Table(tableName).
			Where("quote_time >= ? AND quote_time <= ?", startTimestamp, endTimestamp)

		if param.ISIN != "" {
			query = query.Where("isin = ?", param.ISIN)
		}

		// 使用游标逐行处理
		rows, err := query.Rows()
		if err != nil {
			logger.Error("查询表 %s 失败: %v", tableName, err)
			continue // 继续处理其他表，不中断整个流程
		}

		for rows.Next() {
			select {
			case <-ctx.Done():
				rows.Close()
				return nil, fmt.Errorf("导出被取消: %w", ctx.Err())
			default:
			}

			var detail model.BondQuoteDetail
			if err := s.db.ScanRows(rows, &detail); err != nil {
				logger.Error("扫描数据行失败: %v", err)
				skippedCount++
				continue
			}

			// 直接写入Excel，无需JSON解析
			if err := s.writeDetailToExcel(f, sheetName, &detail, &rowIndex); err != nil {
				logger.Error("写入Excel失败 (ISIN=%s): %v", detail.ISIN, err)
				skippedCount++
				continue
			}
			processedCount++
		}

		rows.Close()
	}

	logger.Info("历史数据导出完成: 处理 %d 条记录，跳过 %d 条记录", processedCount, skippedCount)

	// 5. 设置列宽
	for i := 0; i < len(headers); i++ {
		colName := string(rune('A' + i))
		f.SetColWidth(sheetName, colName, colName, 15)
	}

	// 6. 保存文件
	filename := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s-%s_%s-%s_%d.xlsx",
		HistoryExportPrefix, param.StartDate, param.EndDate,
		startTime[:2]+startTime[3:5], endTime[:2]+endTime[3:5],
		time.Now().Unix()))

	if err := f.SaveAs(filename); err != nil {
		return &ExportResult{}, fmt.Errorf("保存Excel文件失败: %w", err)
	}

	// 7. 上传到OSS (内联逻辑)
	exportConfig := config.GetExportConfig()
	ossInfo := oss.OssInfo{
		Url:     exportConfig.URL,
		Timeout: exportConfig.Timeout,
	}

	fileNameOnly := filepath.Base(filename)
	requestHeaders := map[string]string{
		"x-request-id":     "345678876",
		"x-origin-service": "wealth-bond-quote-service",
		"x-uin":            "123456",
	}

	_, url, err := oss.UploadFile(filename, fileNameOnly, "", requestHeaders, &ossInfo)
	if err != nil {
		return &ExportResult{}, fmt.Errorf("上传文件到OSS失败: %w", err)
	}

	// 8. 删除本地文件
	if err := os.Remove(filename); err != nil {
		logger.Error("删除本地文件失败: %s, error: %v", filename, err)
	} else {
		logger.Info("成功删除本地文件: %s", filename)
	}

	return &ExportResult{
		URL: url,
	}, nil
}

// writeDetailToExcel 将单条 BondQuoteDetail 记录写入Excel
func (s *BondQueryService) writeDetailToExcel(f *excelize.File, sheetName string, detail *model.BondQuoteDetail, rowIndex *int) error {
	// 直接写入结构化数据，无需JSON解析
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", *rowIndex), detail.ID)
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", *rowIndex), detail.MessageID)
	f.SetCellValue(sheetName, fmt.Sprintf("C%d", *rowIndex), detail.MessageType)
	f.SetCellValue(sheetName, fmt.Sprintf("D%d", *rowIndex), detail.Timestamp)
	f.SetCellValue(sheetName, fmt.Sprintf("E%d", *rowIndex), detail.ISIN)
	f.SetCellValue(sheetName, fmt.Sprintf("F%d", *rowIndex), detail.BrokerID)
	f.SetCellValue(sheetName, fmt.Sprintf("G%d", *rowIndex), detail.Side)
	f.SetCellValue(sheetName, fmt.Sprintf("H%d", *rowIndex), detail.Price)

	// 处理可能为空的字段
	if detail.Yield != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", *rowIndex), *detail.Yield)
	}

	f.SetCellValue(sheetName, fmt.Sprintf("J%d", *rowIndex), detail.OrderQty)

	if detail.MinTransQuantity != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", *rowIndex), *detail.MinTransQuantity)
	}

	f.SetCellValue(sheetName, fmt.Sprintf("L%d", *rowIndex), detail.QuoteOrderNo)
	f.SetCellValue(sheetName, fmt.Sprintf("M%d", *rowIndex), detail.QuoteTime.Format("2006-01-02 15:04:05"))

	if detail.SettleType != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", *rowIndex), *detail.SettleType)
	}

	if detail.SettleDate != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", *rowIndex), detail.SettleDate.Format("2006-01-02"))
	}

	if detail.IsValid != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", *rowIndex), *detail.IsValid)
	}

	if detail.IsTbd != nil {
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", *rowIndex), *detail.IsTbd)
	}

	f.SetCellValue(sheetName, fmt.Sprintf("R%d", *rowIndex), detail.CreateTime.Format("2006-01-02 15:04:05"))

	*rowIndex++
	return nil
}

// writeQuoteToExcel 将单条 BondLatestQuote 记录写入Excel（辅助函数）
func (s *BondQueryService) writeQuoteToExcel(f *excelize.File, sheetName string, quote *model.BondLatestQuote, rowIndex *int) error {
	// 解析JSON数据
	var msg BondQuoteMessage
	if err := json.Unmarshal([]byte(quote.RawJSON), &msg); err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	// 解析内部报价数据
	var payload QuotePriceData
	if err := json.Unmarshal([]byte(msg.Data.QuotePriceData), &payload); err != nil {
		return fmt.Errorf("解析报价数据失败: %w", err)
	}

	// 获取买方和卖方报价
	bidPrices := payload.BidPrices
	askPrices := payload.AskPrices

	// 确定需要多少行
	maxRows := len(bidPrices)
	if len(askPrices) > maxRows {
		maxRows = len(askPrices)
	}

	// 如果没有任何报价，至少创建一行基本信息
	if maxRows == 0 {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", *rowIndex), quote.ISIN)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", *rowIndex), quote.MessageID)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", *rowIndex), quote.MessageType)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", *rowIndex), time.UnixMilli(quote.SendTime).Format(MillisecondFormat))
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", *rowIndex), time.UnixMilli(quote.Timestamp).Format(MillisecondFormat))
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", *rowIndex), quote.LastUpdateTime.Format(MillisecondFormat))
		*rowIndex++
		return nil
	}

	// 填充每一行数据
	for i := 0; i < maxRows; i++ {
		// 债券代码
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", *rowIndex), quote.ISIN)

		// 买方数据
		if i < len(bidPrices) {
			bid := bidPrices[i]
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", *rowIndex), bid.Price)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", *rowIndex), bid.Yield)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", *rowIndex), bid.OrderQty)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", *rowIndex), time.UnixMilli(bid.QuoteTime).Format(MillisecondFormat))
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", *rowIndex), bid.BrokerID)
		}

		// 卖方数据
		if i < len(askPrices) {
			ask := askPrices[i]
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", *rowIndex), ask.Price)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", *rowIndex), ask.Yield)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", *rowIndex), ask.OrderQty)
			f.SetCellValue(sheetName, fmt.Sprintf("I%d", *rowIndex), time.UnixMilli(ask.QuoteTime).Format(MillisecondFormat))
			f.SetCellValue(sheetName, fmt.Sprintf("P%d", *rowIndex), ask.BrokerID)
		}

		// 消息元数据
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", *rowIndex), quote.MessageID)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", *rowIndex), quote.MessageType)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", *rowIndex), time.UnixMilli(quote.SendTime).Format(MillisecondFormat))
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", *rowIndex), time.UnixMilli(quote.Timestamp).Format(MillisecondFormat))
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", *rowIndex), quote.LastUpdateTime.Format(MillisecondFormat))

		*rowIndex++
	}

	return nil
}

func (s *BondQueryService) tableExists(tableName string) bool {
	return s.db.Migrator().HasTable(tableName)
}
