package service

import (
	"fmt"
	"log"
	"test/model"
	"time"

	"gorm.io/gorm"
)

// 每周创建七张表（t_bond_quote_detail_%s，t_bond_latest_quote_%s）
type createTableService struct {
	db *gorm.DB
}

func NewCreateTableService(db *gorm.DB) *createTableService {
	return &createTableService{db: db}
}

// 为指定日期所在周创建表
func (s *createTableService) CreateTables(date time.Time) error {
	// 获取从指定日期到下一个周一的所有日期
	weekDates := getWeekDates(date)

	for _, d := range weekDates {
		dateStr := d.Format("20060102")
		detailTable := fmt.Sprintf("t_bond_quote_detail_%s", dateStr)
		latestTable := fmt.Sprintf("t_bond_latest_quote_%s", dateStr)

		// 创建明细表
		if err := s.db.Exec("CREATE TABLE IF NOT EXISTS " + detailTable + " LIKE t_bond_quote_detail").Error; err != nil {
			return fmt.Errorf("创建明细表失败 %s: %w", detailTable, err)
		}

		// 创建最新行情表
		if err := s.db.Exec("CREATE TABLE IF NOT EXISTS " + latestTable + " LIKE t_bond_latest_quote").Error; err != nil {
			return fmt.Errorf("创建最新行情表失败 %s: %w", latestTable, err)
		}

		log.Printf("成功创建表: %s, %s", detailTable, latestTable)
	}

	return nil
}

// 启动每周创建表的定时任务
func (s *createTableService) StartWeeklyTableCreation() {
	log.Println("启动每周建表服务...")

	// 立即为本周创建表
	if err := s.CreateTables(time.Now()); err != nil {
		log.Printf("为本周创建表失败: %v", err)
	}

	// 启动定时任务
	go func() {
		for {
			// 计算到下周一的时间
			now := time.Now()
			daysUntilMonday := int(time.Monday - now.Weekday())
			if daysUntilMonday <= 0 {
				daysUntilMonday += 7
			}
			nextMonday := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 0, 0, 0, 0, now.Location())

			// 等待到下周一
			waitDuration := nextMonday.Sub(now)
			log.Printf("下次建表将在 %s 后执行 (下周一: %s)", waitDuration, nextMonday.Format("2006-01-02"))
			time.Sleep(waitDuration)

			// 为下一周创建表
			if err := s.CreateTables(nextMonday); err != nil {
				log.Printf("为下周创建表失败: %v", err)
			} else {
				log.Printf("成功为下周创建表 (周开始日期: %s)", nextMonday.Format("2006-01-02"))
			}
		}
	}()
}

// 获取从指定日期到下一个周一的所有日期
// 例如：如果今天是周三，则返回[周三, 周四, 周五, 周六, 周日, 周一]
// 用于批量创建本周剩余天数的表
func getWeekDates(date time.Time) []time.Time {
	// 计算距离下周一还有多少天
	daysUntilMonday := int(time.Monday - date.Weekday())
	if daysUntilMonday <= 0 {
		daysUntilMonday += 7 // 如果今天是周一，则返回本周一到下周一
	}

	// 生成从当前日期到下周一前的所有日期
	dates := make([]time.Time, daysUntilMonday)
	for i := 0; i < daysUntilMonday; i++ {
		dates[i] = date.AddDate(0, 0, i)
	}

	return dates
}

// 检查指定日期的表是否存在，不存在则创建
func (s *createTableService) EnsureDailyTablesExist(date time.Time) error {
	dateStr := date.Format("20060102")
	detailTable := fmt.Sprintf("t_bond_quote_detail_%s", dateStr)
	latestTable := fmt.Sprintf("t_bond_latest_quote_%s", dateStr)

	// 检查并创建明细表
	if err := s.db.Table(detailTable).AutoMigrate(&model.BondQuoteDetail{}); err != nil {
		return fmt.Errorf("创建明细表失败 %s: %w", detailTable, err)
	}

	// 检查并创建最新行情表
	if err := s.db.Table(latestTable).AutoMigrate(&model.BondLatestQuote{}); err != nil {
		return fmt.Errorf("创建最新行情表失败 %s: %w", latestTable, err)
	}

	log.Printf("成功创建表: %s, %s", detailTable, latestTable)
	return nil
}
