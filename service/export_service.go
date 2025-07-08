package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"test/model"
)

// ExportLatestQuotesService 债券最新行情导出服务
type ExportLatestQuotesService struct {
	db *gorm.DB
}

// NewExportLatestQuotesService 创建债券最新行情导出服务
func NewExportLatestQuotesService(db *gorm.DB) *ExportLatestQuotesService {
	return &ExportLatestQuotesService{db: db}
}

// StartHourlyExport 启动每小时导出任务
func (s *ExportLatestQuotesService) StartHourlyExport(exportDir string) {
	// 确保导出目录存在
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Fatalf("创建导出目录失败: %v", err)
	}

	// 计算下一个整点时间
	now := time.Now()
	nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	initialDelay := nextHour.Sub(now)

	log.Printf("债券最新行情导出服务已启动，将在 %s 后开始首次导出", initialDelay)

	// 启动定时任务
	go func() {
		// 等待到下一个整点
		// time.Sleep(initialDelay)

		// 每小时执行一次导出
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			exportTime := time.Now()
			filename := filepath.Join(exportDir, fmt.Sprintf("bond_latest_quotes_%s.xlsx", exportTime.Format("20060102_150405")))

			if err := s.ExportToExcel(filename); err != nil {
				log.Printf("导出Excel失败: %v", err)
			} else {
				log.Printf("成功导出Excel文件: %s", filename)
			}

			<-ticker.C // 等待下一个小时
		}
	}()
}

// ExportToExcel 导出最新行情到Excel文件
func (s *ExportLatestQuotesService) ExportToExcel(filename string) error {
	// 查询所有最新行情数据
	var latestQuotes []model.BondLatestQuote
	if err := s.db.Find(&latestQuotes).Error; err != nil {
		return fmt.Errorf("查询最新行情数据失败: %w", err)
	}

	// 创建新的Excel文件
	f := excelize.NewFile()
	defer f.Close()

	// 设置工作表名称
	sheetName := "债券最新行情"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	// 设置表头 - 按照要求的顺序排列
	headers := []string{
		"债券代码",
		"价格", "收益率", "数量", "最小交易量", "报价时间",
		"结算类型", "是否有效", "是否待定", "报价单号", "方向",
		"消息ID", "消息类型", "发送时间", "时间戳", "更新时间",
		"券商ID",
	}

	// headers := []string{
	// 	"债券代码",
	// 	"价格", "收益率", "数量", "最小交易量", "报价时间",
	// 	"结算类型", "是否有效", "是否待定", "报价单号", "方向",
	// 	"消息ID", "消息类型", "发送时间", "时间戳", "更新时间",
	// 	"券商ID",
	// }

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 填充数据
	rowIndex := 2 // 从第2行开始（第1行是表头）

	for _, quote := range latestQuotes {
		// 解析JSON数据
		var msg BondQuoteMessage
		if err := json.Unmarshal([]byte(quote.RawJSON), &msg); err != nil {
			log.Printf("解析JSON失败 (ISIN=%s): %v", quote.ISIN, err)
			continue
		}

		// 解析内部报价数据
		var payload QuotePriceData
		if err := json.Unmarshal([]byte(msg.Data.QuotePriceData), &payload); err != nil {
			log.Printf("解析报价数据失败 (ISIN=%s): %v", quote.ISIN, err)
			continue
		}

		// 如果没有买入或卖出报价，至少创建一行基本信息
		if len(payload.BidPrices) == 0 && len(payload.AskPrices) == 0 {
			// 基本信息
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), quote.ISIN)
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), quote.MessageID)
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), quote.MessageType)
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowIndex), time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowIndex), time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("P%d", rowIndex), quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"))
			rowIndex++
			continue
		}

		// 处理所有买入报价
		for _, bid := range payload.BidPrices {
			// 债券代码
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), quote.ISIN)

			// 报价详情
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), bid.Price)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIndex), bid.Yield)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIndex), bid.OrderQty)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIndex), bid.MinTransQuantity)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), time.UnixMilli(bid.QuoteTime).Format("2006-01-02 15:04:05.000"))

			// 其他字段
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIndex), bid.SettleType)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIndex), bid.IsValid)
			f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIndex), bid.IsTbd)
			f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), bid.QuoteOrderNo)
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), bid.Side)

			// 消息元数据（放在最后）
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), quote.MessageID)
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), quote.MessageType)
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowIndex), time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowIndex), time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("P%d", rowIndex), quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"))

			// 券商ID（放在最后）
			f.SetCellValue(sheetName, fmt.Sprintf("Q%d", rowIndex), bid.BrokerID)

			rowIndex++
		}

		// 处理所有卖出报价
		for _, ask := range payload.AskPrices {
			// 债券代码
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), quote.ISIN)

			// 报价详情
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), ask.Price)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIndex), ask.Yield)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIndex), ask.OrderQty)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIndex), ask.MinTransQuantity)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), time.UnixMilli(ask.QuoteTime).Format("2006-01-02 15:04:05.000"))

			// 其他字段
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIndex), ask.SettleType)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIndex), ask.IsValid)
			f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIndex), ask.IsTbd)
			f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), ask.QuoteOrderNo)
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), ask.Side)

			// 消息元数据（放在最后）
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), quote.MessageID)
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", rowIndex), quote.MessageType)
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", rowIndex), time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", rowIndex), time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"))
			f.SetCellValue(sheetName, fmt.Sprintf("P%d", rowIndex), quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"))

			// 券商ID（放在最后）
			f.SetCellValue(sheetName, fmt.Sprintf("Q%d", rowIndex), ask.BrokerID)

			rowIndex++
		}
	}

	// 设置列宽
	for i := 0; i < len(headers); i++ {
		colName := fmt.Sprint('A' + i)
		f.SetColWidth(sheetName, colName, colName, 18)
	}

	// 保存文件
	if err := f.SaveAs(filename); err != nil {
		return fmt.Errorf("保存Excel文件失败: %w", err)
	}

	return nil
}
