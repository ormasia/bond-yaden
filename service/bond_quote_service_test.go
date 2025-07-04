package service_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"test/service"
)

func TestBondQuoteService_ProcessMessage(t *testing.T) {
	// 1. 设置模拟数据库
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建模拟数据库失败: %v", err)
	}
	defer db.Close()

	// 2. 设置GORM连接
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("打开GORM连接失败: %v", err)
	}

	// 3. 创建服务实例
	bondQuoteService := service.NewBondQuoteService(gormDB)

	// 4. 准备测试数据
	testJSON := `{"data":{"data":"{\"askPrices\":[{\"brokerId\":\"1941007146139877377\",\"isTbd\":\"N\",\"isValid\":\"Y\",\"minTransQuantity\":1000000,\"orderQty\":14000000,\"price\":91.356894,\"quoteOrderNo\":\"D1KNER1XUNB003EKWSKG\",\"quoteTime\":1751607138931,\"securityId\":\"HK0000108958\",\"settleType\":\"T2\",\"side\":\"ASK\",\"yield\":9.112088}],\"bidPrices\":[{\"brokerId\":\"1941007146139877376\",\"isTbd\":\"N\",\"isValid\":\"Y\",\"minTransQuantity\":1000000,\"orderQty\":14000000,\"price\":90.356894,\"quoteOrderNo\":\"D1KNER1XUNB003EKWSKG\",\"quoteTime\":1751607138790,\"securityId\":\"HK0000108958\",\"settleType\":\"T2\",\"side\":\"BID\",\"yield\":9.764276}],\"securityId\":\"HK0000108958\"}","messageId":"D1KNER1XUNB003EKWSKG","messageType":"BOND_ORDER_BOOK_MSG","organization":"AF","receiverId":"HK0000108958","timestamp":1751607140490},"sendTime":1751607140494,"wsMessageType":"ATS_QUOTE"}`

	// 5. 设置数据库期望
	// 开始事务
	mock.ExpectBegin()

	// 插入ASK报价明细
	mock.ExpectExec("INSERT INTO `t_bond_quote_detail`").
		WithArgs(
			sqlmock.AnyArg(),       // ID (自动生成)
			"D1KNER1XUNB003EKWSKG", // MessageID
			"BOND_ORDER_BOOK_MSG",  // MessageType
			int64(1751607140490),   // Timestamp
			"HK0000108958",         // SecurityCode
			"1941007146139877377",  // BrokerID
			"ASK",                  // Side
			float64(91.356894),     // Price
			float64(9.112088),      // Yield
			float64(14000000),      // OrderQty
			float64(1000000),       // MinTransQuantity
			"D1KNER1XUNB003EKWSKG", // QuoteOrderNo
			sqlmock.AnyArg(),       // QuoteTime (转换后的时间)
			"T2",                   // SettleType
			nil,                    // SettleDate
			"Y",                    // IsValid
			"N",                    // IsTbd
			sqlmock.AnyArg(),       // CreateTime
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 插入BID报价明细
	mock.ExpectExec("INSERT INTO `t_bond_quote_detail`").
		WithArgs(
			sqlmock.AnyArg(),       // ID (自动生成)
			"D1KNER1XUNB003EKWSKG", // MessageID
			"BOND_ORDER_BOOK_MSG",  // MessageType
			int64(1751607140490),   // Timestamp
			"HK0000108958",         // SecurityCode
			"1941007146139877376",  // BrokerID
			"BID",                  // Side
			float64(90.356894),     // Price
			float64(9.764276),      // Yield
			float64(14000000),      // OrderQty
			float64(1000000),       // MinTransQuantity
			"D1KNER1XUNB003EKWSKG", // QuoteOrderNo
			sqlmock.AnyArg(),       // QuoteTime (转换后的时间)
			"T2",                   // SettleType
			nil,                    // SettleDate
			"Y",                    // IsValid
			"N",                    // IsTbd
			sqlmock.AnyArg(),       // CreateTime
		).
		WillReturnResult(sqlmock.NewResult(2, 1))

	// 更新最新行情
	mock.ExpectExec("INSERT INTO `t_bond_latest_quote`").
		WithArgs(
			"HK0000108958",        // SecurityCode
			sqlmock.AnyArg(),      // LastUpdateTime
			float64(90.356894),    // BidPrice
			float64(9.764276),     // BidYield
			float64(14000000),     // BidQty
			"1941007146139877376", // BidBrokerID
			sqlmock.AnyArg(),      // BidQuoteTime
			float64(91.356894),    // AskPrice
			float64(9.112088),     // AskYield
			float64(14000000),     // AskQty
			"1941007146139877377", // AskBrokerID
			sqlmock.AnyArg(),      // AskQuoteTime
			float64(1.0),          // Spread (计算值: 91.356894 - 90.356894)
			sqlmock.AnyArg(),      // UpdateTime
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 提交事务
	mock.ExpectCommit()

	// 6. 执行测试
	err = bondQuoteService.ProcessMessage([]byte(testJSON))

	// 7. 验证结果
	assert.NoError(t, err, "处理消息应该成功")

	// 验证所有期望都已满足
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("有未满足的期望: %s", err)
	}
}

func TestBondQuoteService_ProcessInvalidMessage(t *testing.T) {
	// 1. 设置模拟数据库
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建模拟数据库失败: %v", err)
	}
	defer db.Close()

	// 2. 设置GORM连接
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("打开GORM连接失败: %v", err)
	}

	// 3. 创建服务实例
	bondQuoteService := service.NewBondQuoteService(gormDB)

	// 4. 测试无效的JSON
	invalidJSON := `{"invalid": "json"`
	err = bondQuoteService.ProcessMessage([]byte(invalidJSON))
	assert.Error(t, err, "处理无效JSON应该返回错误")

	// 5. 测试非债券行情消息
	nonBondQuoteJSON := `{"data":{"data":"{}","messageId":"TEST","messageType":"OTHER_TYPE","timestamp":1234567890},"sendTime":1234567890,"wsMessageType":"OTHER_TYPE"}`
	err = bondQuoteService.ProcessMessage([]byte(nonBondQuoteJSON))
	assert.NoError(t, err, "处理非债券行情消息应该成功但不执行操作")

	// 6. 测试内部JSON解析错误
	invalidInnerJSON := `{"data":{"data":"{invalid json}","messageId":"TEST","messageType":"BOND_ORDER_BOOK_MSG","timestamp":1234567890},"sendTime":1234567890,"wsMessageType":"ATS_QUOTE"}`
	err = bondQuoteService.ProcessMessage([]byte(invalidInnerJSON))
	assert.Error(t, err, "处理内部无效JSON应该返回错误")
}

// 测试边界情况：没有报价数据
func TestBondQuoteService_ProcessEmptyQuotes(t *testing.T) {
	// 1. 设置模拟数据库
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建模拟数据库失败: %v", err)
	}
	defer db.Close()

	// 2. 设置GORM连接
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("打开GORM连接失败: %v", err)
	}

	// 3. 创建服务实例
	bondQuoteService := service.NewBondQuoteService(gormDB)

	// 4. 准备测试数据 - 没有报价
	emptyQuotesJSON := `{"data":{"data":"{\"askPrices\":[],\"bidPrices\":[],\"securityId\":\"HK0000108958\"}","messageId":"D1KNER1XUNB003EKWSKG","messageType":"BOND_ORDER_BOOK_MSG","organization":"AF","receiverId":"HK0000108958","timestamp":1751607140490},"sendTime":1751607140494,"wsMessageType":"ATS_QUOTE"}`

	// 5. 设置数据库期望
	mock.ExpectBegin()

	// 更新最新行情 - 但没有报价数据
	mock.ExpectExec("INSERT INTO `t_bond_latest_quote`").
		WithArgs(
			"HK0000108958",   // SecurityCode
			sqlmock.AnyArg(), // LastUpdateTime
			nil,              // BidPrice
			nil,              // BidYield
			nil,              // BidQty
			nil,              // BidBrokerID
			nil,              // BidQuoteTime
			nil,              // AskPrice
			nil,              // AskYield
			nil,              // AskQty
			nil,              // AskBrokerID
			nil,              // AskQuoteTime
			nil,              // Spread
			sqlmock.AnyArg(), // UpdateTime
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	// 6. 执行测试
	err = bondQuoteService.ProcessMessage([]byte(emptyQuotesJSON))

	// 7. 验证结果
	assert.NoError(t, err, "处理空报价消息应该成功")

	// 验证所有期望都已满足
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("有未满足的期望: %s", err)
	}
}

// 测试时间解析
func TestBondQuoteService_TimeConversion(t *testing.T) {
	// 测试毫秒时间戳转换
	timestamp := int64(1751607138931)
	expectedTime := time.UnixMilli(timestamp)

	// 验证转换结果
	assert.Equal(t, 2025, expectedTime.Year(), "年份应该是2025")
	assert.Equal(t, time.Month(7), expectedTime.Month(), "月份应该是7")
	assert.Equal(t, 4, expectedTime.Day(), "日期应该是4")
}
