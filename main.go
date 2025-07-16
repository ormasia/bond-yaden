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
	"fmt"
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

var (
	wg         sync.WaitGroup
	RawChan    = make(chan []byte, rawCap)
	ParsedChan = make(chan *service.ParsedQuote, parsedCap)
	DeadChan   = make(chan []byte, 1000) // 解析失败
)

// 1. 用户登录获取访问令牌
// 2. 建立WebSocket连接
// 3. 建立STOMP协议连接
// 4. 订阅债券行情消息
// 5. 持续监听消息推送
func main() {

	// 获取亚丁ATS配置
	adenConfig := config.GetAdenATSConfig()

	logger.Debug("使用配置 - ATS地址: %s, 用户: %s", adenConfig.BaseURL, adenConfig.Username)

	db := dataSource.GetDBConn("bond")

	exportConfig := config.GetExportConfig()
	// 每小时导出最新行情数据
	service.NewExportLatestQuotesService(db).StartHourlyExport(exportConfig.Path, exportConfig.Interval)

	// 每周创建表
	service.NewCreateTableService(db).StartWeeklyTableCreation()

	fmt.Println("开始亚丁ATS系统测试...")

	// 创建STOMP客户端实例
	client := &service.StompClient{}

	// 第一步：用户登录获取访问令牌
	// 使用加密通信协议，获取后续API调用所需的token
	fmt.Println("第一步：登录获取Token...")
	if err := client.Login(adenConfig.Username, adenConfig.Password, adenConfig.SmsCode, adenConfig.PublicKey, adenConfig.BaseURL, adenConfig.ClientId); err != nil {
		// log.Fatal("登录失败:", err)
		logger.Fatal("登录失败: %v", err.Error())
	}
	// fmt.Printf("登录成功，获取到Token: %s\n", client.token[:20]+"...")
	logger.Info("登录成功，获取到Token: %s", client.Token[:20]+"...")

	// 第二步：建立WebSocket连接
	// 使用获取的token建立安全的WebSocket连接
	fmt.Println("第二步：建立WebSocket连接...")
	if err := client.ConnectWebSocket(adenConfig.WssURL); err != nil {
		// log.Fatal("WebSocket连接失败:", err)
		logger.Fatal("WebSocket连接失败: %v", err.Error())
	}
	defer client.Conn.Close() // 确保程序退出时关闭连接

	// 第三步：建立STOMP协议连接
	// 在WebSocket基础上建立STOMP消息协议连接
	fmt.Println("第三步：建立STOMP连接...")
	if err := client.ConnectStomp(); err != nil {
		// log.Fatal("STOMP连接失败:", err)
		logger.Fatal("STOMP连接失败: %v", err.Error())
	}
	defer client.StompConn.Disconnect() // 确保程序退出时断开STOMP连接

	// 第四步：订阅债券行情消息
	// 订阅指定的消息队列，开始接收实时行情数据
	fmt.Println("第四步：订阅行情消息...")
	// log.Fatal("订阅失败:", err)
	if err := client.Subscribe(RawChan); err != nil {
		logger.Fatal("订阅失败: %v", err.Error())
	}

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

	// 阻塞等待中断信号
	<-interrupt
	fmt.Println("正在断开连接...")

	// 优雅关闭：关闭 channels，等待 workers 完成
	close(RawChan)
	close(ParsedChan)
	close(DeadChan)

	// 现在等待所有 worker 完成清理工作
	wg.Wait()
	fmt.Println("所有后台任务已停止")
}
