// 亚丁ATS系统客户端
// 实现与亚丁ATS（Automated Trading System）系统的连接和通信
//
// 主要功能：
// 1. 用户登录认证（混合加密）
// 2. WebSocket连接建立
// 3. STOMP协议通信
// 4. 实时债券行情数据接收
//
// 技术特点：
// - RSA+AES混合加密通信
// - WebSocket+STOMP双协议栈
// - 实时消息推送
// - 优雅的连接管理
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"time"
	config "wealth-bond-quote-service/internal/conf"
	"wealth-bond-quote-service/internal/dataSource"
	logger "wealth-bond-quote-service/pkg/log"
	"wealth-bond-quote-service/service"

	_ "modernc.org/sqlite"
)

var (
	rawCap     int           // 原始 JSON 缓冲
	parsedCap  int           // 解析后缓冲
	workerNum  int           // 解析/写库协程数
	batchSize  int           // 单次批写条数
	flushDelay time.Duration // 刷新延迟
)

const (
	APP_NAME      = "wealth-bond-quote-aden"
	APP_NACOS_KEY = "wealth-bond-quote-aden@@wealth@@wealth"
)

func init() {
	var initCfgOK = true
	er := config.NewNacosClientInsFromEnv(APP_NAME)
	if er == nil {
		err := config.GetViperCfgFromNacos(APP_NACOS_KEY, "", "yaml") // add config here
		if err != nil {
			fmt.Printf("getCfgFromNacos error:%s\n", err.Error())
			initCfgOK = false
		} else {
			for localKey, v := range config.NacosKeys {
				err := config.GetViperCfgFromNacos(v, localKey, "yaml")
				if err != nil {
					fmt.Printf("getCfgFromNacos key:%s error:%s\n", v, err.Error())
					initCfgOK = false
					break
				}
			}
			if initCfgOK {
				// 日志配置
				logConfig := config.GetLogConfig()
				logger.SetLogLevel(logConfig.Level)
				logger.SetLogFileName(logConfig.Path)
				fmt.Printf("日志配置 - 路径: %s, 级别: %s\n", logConfig.Path, logConfig.Level)

				cfg := config.GetDataProcessConfig()
				rawCap = cfg.RawBufferSize
				parsedCap = cfg.ParsedBufferSize
				workerNum = cfg.WorkerNum
				batchSize = cfg.BatchSize
				flushDelay = time.Duration(cfg.FlushDelayMs) * time.Millisecond

				fmt.Println("init from Nacos OK!")
				return
			}
		}
	} else {
		fmt.Printf("nacos init error:%s\n", er.Error())
	}
	config.InitFromLocalFile("config", "yaml")
	fmt.Println("init from local YAML file!")
}

func generateTestMessage(index int, securityIDs, brokerIDs []string) []byte {
	now := time.Now()
	timestamp := now.UnixMilli()

	// 随机选择债券代码
	securityID := securityIDs[rand.Intn(len(securityIDs))]

	// 生成随机BID价格 (1-3个)
	bidCount := 1 + rand.Intn(3)
	bidPrices := make([]map[string]any, bidCount)
	basePrice := 95.0 + rand.Float64()*10.0 // 95-105之间

	for j := 0; j < bidCount; j++ {
		price := basePrice + (rand.Float64()-0.5)*2.0
		yield := 4.0 + (rand.Float64()-0.5)*2.0 // 3-5%之间

		bidPrices[j] = map[string]any{
			"brokerId":         brokerIDs[rand.Intn(len(brokerIDs))],
			"isTbd":            "N",
			"isValid":          "Y",
			"minTransQuantity": float64(1000000 + rand.Intn(9000000)),
			"orderQty":         float64(5000000 + rand.Intn(20000000)),
			"price":            price,
			"quoteOrderNo":     fmt.Sprintf("BID%d%08d", index, j),
			"quoteTime":        timestamp + int64(j*100),
			"securityId":       securityID,
			"settleType":       "T2",
			"side":             "BID",
			"yield":            yield,
		}
	}

	// 生成随机ASK价格 (0-2个，有时候没有ASK)
	askCount := rand.Intn(3)
	askPrices := make([]map[string]any, askCount)

	for j := 0; j < askCount; j++ {
		price := basePrice + 0.5 + (rand.Float64()-0.5)*2.0 // ASK通常比BID高一点
		yield := 3.8 + (rand.Float64()-0.5)*2.0

		askPrices[j] = map[string]any{
			"brokerId":         brokerIDs[rand.Intn(len(brokerIDs))],
			"isTbd":            "N",
			"isValid":          "Y",
			"minTransQuantity": float64(1000000 + rand.Intn(9000000)),
			"orderQty":         float64(5000000 + rand.Intn(20000000)),
			"price":            price,
			"quoteOrderNo":     fmt.Sprintf("ASK%d%08d", index, j),
			"quoteTime":        timestamp + int64(j*100),
			"securityId":       securityID,
			"settleType":       "T2",
			"side":             "ASK",
			"yield":            yield,
		}
	}

	// 构建内层数据
	quotePriceData := map[string]any{
		"askPrices":  askPrices,
		"bidPrices":  bidPrices,
		"securityId": securityID,
	}

	quotePriceJSON, _ := json.Marshal(quotePriceData)

	// 构建外层消息
	message := map[string]any{
		"data": map[string]any{
			"data":         string(quotePriceJSON),
			"messageId":    fmt.Sprintf("TEST%d%010d", index, timestamp%10000000000),
			"messageType":  "BOND_ORDER_BOOK_MSG",
			"organization": "AF",
			"receiverId":   securityID,
			"timestamp":    timestamp,
		},
		"sendTime":      timestamp + int64(rand.Intn(100)),
		"wsMessageType": "ATS_QUOTE",
	}

	jsonData, _ := json.Marshal(message)
	return jsonData
}

// 如果你想直接在代码中使用，可以调用这个函数
func GenerateAndSendToChannel(rawChan chan []byte, count int) {
	securityIDs := []string{
		"HK0000098928", "HK0000098929", "HK0000098930", "HK0000098931", "HK0000098932",
		"CN0000001001", "CN0000001002", "CN0000001003", "CN0000001004", "CN0000001005",
		"CN0000001006", "CN0000001007", "CN0000001008", "CN0000001009", "CN0000001010",
	}

	brokerIDs := []string{
		"1941007160979324928", "1941007160979324929", "1941007160979324930",
		"1941007160979324931", "1941007160979324932", "1941007160979324933",
	}

	fmt.Printf("开始向通道发送 %d 条测试消息...\n", count)
	for i := 0; i < count%10; i++ {
		for i := 0; i < count; i++ {
			message := generateTestMessage(i, securityIDs, brokerIDs)

			select {
			case rawChan <- message:
				if i%100 == 0 {
					fmt.Printf("已发送 %d/%d 条消息\n", i+1, count)
				}
			default:
				fmt.Printf("通道已满，消息 #%d 发送失败\n", i+1)
			}

			// 控制发送速度
			time.Sleep(5 * time.Millisecond)
		}
		fmt.Printf("已发送 第%d批次\n", i+1)
		time.Sleep(5 * time.Second)
	}
	fmt.Printf("完成发送 %d 条消息\n", count)
}

func main() {

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	var wg sync.WaitGroup
	RawChan := make(chan []byte, rawCap)
	ParsedChan := make(chan *service.ParsedQuote, parsedCap)
	DeadChan := make(chan []byte, 1000) // 解析失败
	db := dataSource.GetDBConn("bond")

	go GenerateAndSendToChannel(RawChan, 25)

	exportConfig := config.GetExportConfig()
	// 每小时导出最新行情数据
	service.NewExportLatestQuotesService(db).StartHourlyExport(exportConfig.Path, exportConfig.Interval)

	// 每周创建表
	service.NewCreateTableService(db).StartWeeklyTableCreation()

	// 建立STOMP连接并订阅消息
	errChan := make(chan error, 1) // 监听出现错误通道
	ctx, cancel := context.WithCancel(context.Background())
	go startMessageListener(ctx, errChan, RawChan, &wg)

	// 第五步：启动后台处理工作协程
	bqs := service.NewBondQuoteService(db, &wg, RawChan, ParsedChan, DeadChan)
	go bqs.StartParseWorkers(workerNum)
	go bqs.StartDBWorkers(workerNum, batchSize, flushDelay)

	// 设置中断信号处理
	// 监听Ctrl+C信号，优雅退出程序
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Println("连接成功，等待消息推送...")
	fmt.Println("按 Ctrl+C 退出")

	for {
		select {
		case err := <-errChan:
			logger.Error("消息监听错误: %v", err)
			go func() {
				time.Sleep(5 * time.Second)                      // 等待一段时间后重启监听
				startMessageListener(ctx, errChan, RawChan, &wg) // 重启监听
			}()
		case <-interrupt:
			fmt.Println("收到中断信号，正在退出...")
			fmt.Println("正在断开连接...")
			cancel() // 取消上下文，通知所有协程退出
			// 优雅关闭：关闭 channels，等待 workers 完成
			close(RawChan)
			close(ParsedChan)
			close(DeadChan)
			close(errChan)

			// 现在等待所有 worker 完成清理工作
			wg.Wait()
			fmt.Println("所有后台任务已停止")
			return
		}
	}
}

// 1. 用户登录获取访问令牌
// 2. 建立WebSocket连接
// 3. 建立STOMP协议连接
// 4. 订阅债券行情消息
// 5. 持续监听消息推送
func startMessageListener(ctx context.Context, errChan chan error, rawChan chan []byte, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done() // 确保函数退出时调用 Done
	// 获取亚丁ATS配置
	adenConfig := config.GetAdenATSConfig()

	logger.Debug("使用配置 - ATS地址: %s, 用户: %s", adenConfig.BaseURL, adenConfig.Username)

	fmt.Println("开始亚丁ATS系统测试...")

	// 创建STOMP客户端实例
	client := &service.StompClient{}

	// 第一步：用户登录获取访问令牌
	// 使用加密通信协议，获取后续API调用所需的token
	fmt.Println("第一步：登录获取Token...")
	if err := client.Login(adenConfig.Username, adenConfig.Password, adenConfig.SmsCode, adenConfig.PublicKey, adenConfig.BaseURL, adenConfig.ClientId); err != nil {
		logger.Fatal("登录失败: %v", err.Error())
		return
	}
	logger.Info("登录成功，获取到Token: %s", client.Token[:20]+"...")

	// 第二步：建立WebSocket连接
	// 使用获取的token建立安全的WebSocket连接
	fmt.Println("第二步：建立WebSocket连接...")
	if err := client.ConnectWebSocket(adenConfig.WssURL); err != nil {
		logger.Fatal("WebSocket连接失败: %v", err.Error())
		return
	}
	defer client.Conn.Close() // 确保程序退出时关闭连接

	// 第三步：建立STOMP协议连接
	// 在WebSocket基础上建立STOMP消息协议连接
	fmt.Println("第三步：建立STOMP连接...")
	if err := client.ConnectStomp(); err != nil {
		logger.Fatal("STOMP连接失败: %v", err.Error())
		return
	}
	defer client.StompConn.Disconnect() // 确保程序退出时断开STOMP连接

	// 第四步：订阅债券行情消息
	// 订阅指定的消息队列，开始接收实时行情数据
	fmt.Println("第四步：订阅行情消息...")
	if err := client.Subscribe(ctx, rawChan, errChan /*, &Mwg*/); err != nil {
		logger.Fatal("订阅失败: %v", err.Error())
		return
	}
}
