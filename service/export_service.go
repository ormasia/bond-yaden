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
		time.Sleep(initialDelay)

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

	// 设置表头
	headers := []string{
		"债券代码", "消息ID", "消息类型", "发送时间", "时间戳", "更新时间",
		"买方券商ID", "买方价格", "买方收益率", "买方数量", "买方报价时间",
		"卖方券商ID", "卖方价格", "卖方收益率", "卖方数量", "卖方报价时间",
		"价差",
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 填充数据
	for i, quote := range latestQuotes {
		row := i + 2 // 从第2行开始（第1行是表头）

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

		// 基本信息
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), quote.ISIN)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), quote.MessageID)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), quote.MessageType)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), time.UnixMilli(quote.SendTime).Format("2006-01-02 15:04:05.000"))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), time.UnixMilli(quote.Timestamp).Format("2006-01-02 15:04:05.000"))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), quote.LastUpdateTime.Format("2006-01-02 15:04:05.000"))

		// 找出最优买入价和卖出价
		var bestBid, bestAsk *QuotePrice

		// 找最优买入价（价格最高的BID）
		for i := range payload.BidPrices {
			if bestBid == nil || payload.BidPrices[i].Price > bestBid.Price {
				bestBid = &payload.BidPrices[i]
			}
		}

		// 找最优卖出价（价格最低的ASK）
		for i := range payload.AskPrices {
			if bestAsk == nil || payload.AskPrices[i].Price < bestAsk.Price {
				bestAsk = &payload.AskPrices[i]
			}
		}

		// 填充买方数据
		if bestBid != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), bestBid.BrokerID)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), bestBid.Price)
			f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), bestBid.Yield)
			f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), bestBid.OrderQty)
			f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), time.UnixMilli(bestBid.QuoteTime).Format("2006-01-02 15:04:05.000"))
		}

		// 填充卖方数据
		if bestAsk != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), bestAsk.BrokerID)
			f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), bestAsk.Price)
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), bestAsk.Yield)
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), bestAsk.OrderQty)
			f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), time.UnixMilli(bestAsk.QuoteTime).Format("2006-01-02 15:04:05.000"))
		}

		// 计算价差
		if bestBid != nil && bestAsk != nil {
			spread := bestAsk.Price - bestBid.Price
			f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), spread)
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
