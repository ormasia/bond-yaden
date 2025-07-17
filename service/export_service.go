package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"os"
	"path/filepath"
	"time"
	logger "wealth-bond-quote-service/pkg/log"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	config "wealth-bond-quote-service/internal/conf"
	"wealth-bond-quote-service/model"
	"wealth-bond-quote-service/pkg/dtalk"
	"wealth-bond-quote-service/pkg/oss"
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
func (s *ExportLatestQuotesService) StartHourlyExport(exportDir string, interval int) {
	// 确保导出目录存在
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		logger.Fatal("创建导出目录失败: %v", err)
	}

	// // 计算下一个整点时间
	// now := time.Now()
	// nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	// initialDelay := nextHour.Sub(now)

	// log.Printf("债券最新行情导出服务已启动，将在 %s 后开始首次导出", initialDelay)

	intervalDuration := time.Duration(interval) * time.Minute
	logger.Info("债券最新行情导出服务已启动，每 %d 分钟执行一次导出", interval)
	// 启动定时任务
	go func(intervalDuration time.Duration) {
		// 等待到下一个整点
		// time.Sleep(initialDelay)

		// 每小时执行一次导出
		ticker := time.NewTicker(intervalDuration)
		defer ticker.Stop()

		for {
			exportTime := time.Now()
			filename := filepath.Join(exportDir, fmt.Sprintf("bond_latest_quotes_%s.xlsx", exportTime.Format("20060102_150405")))

			if err := s.ExportToExcel(filename); err != nil {
				logger.Error("导出Excel失败: %v", err)
			} else {
				logger.Info("成功导出Excel文件: %s", filename)
			}

			<-ticker.C // 等待下一个小时
		}
	}(intervalDuration)
}

// ExportToExcel 导出最新行情到Excel文件
func (s *ExportLatestQuotesService) ExportToExcel(filename string) error {
	tableName := GetTodayLatestTableName()
	// 查询所有最新行情数据
	var latestQuotes []model.BondLatestQuote
	if err := s.db.Table(tableName).Find(&latestQuotes).Error; err != nil {
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
		"买方价格", "买方收益率", "买方数量", "买方报价时间",
		"卖方价格", "卖方收益率", "卖方数量", "卖方报价时间",
		"消息ID", "消息类型", "发送时间", "时间戳", "更新时间",
		"买方券商ID", "卖方券商ID",
	}

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
			logger.Error("解析JSON失败 (ISIN=%s): %v", quote.ISIN, err)
			continue
		}

		// 解析内部报价数据
		var payload QuotePriceData
		if err := json.Unmarshal([]byte(msg.Data.QuotePriceData), &payload); err != nil {
			logger.Error("解析报价数据失败 (ISIN=%s): %v", quote.ISIN, err)
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
			// 基本信息
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), quote.ISIN)
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

			// 消息元数据（放在最后）
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

	// 保存文件
	if err := f.SaveAs(filename); err != nil {
		return fmt.Errorf("保存Excel文件失败: %w", err)
	}

	// 这里需要使用 OSS 上传文件，拿到 ossid 和 url，拼接成可以直接下载的url
	ossConfig := config.GetExportConfig()
	ossInfo := oss.OssInfo{
		Url:     ossConfig.URL,
		Timeout: ossConfig.Timeout,
	}
	fileNameOnly := filepath.Base(filename)
	requestheaders := map[string]string{
		"x-request-id":     "345678876",
		"x-origin-service": "wealth-bond-quote-service",
		"x-uin":            "123456",
	}
	_, url, err := oss.UploadFile(filename, fileNameOnly, "", requestheaders, &ossInfo)
	if err != nil {
		return fmt.Errorf("上传文件到OSS失败: %w", err)
	}

	// 然后调用钉钉发送消息的服务，不用拼接url直接是可以下载的连接，把这个url用钉钉发送一下就行
	// 发送钉钉消息
	fmt.Print("发送钉钉消息: ", url)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	message := fmt.Sprintf("%s下载链接: \n%s", time.Now().Format("2006-01-02 15:04:05"), url)
	if err := dtalk.DTalkSendTextMsg(ctx, message); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("发送钉钉消息超时: %w", err)
		}
		return fmt.Errorf("发送钉钉消息失败: %w", err)
	}

	// 删除本地文件
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("删除本地文件失败: %w", err)
	}
	logger.Info("成功删除本地文件: %s", filename)
	return nil
}
