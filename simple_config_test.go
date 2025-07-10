package main

import (
	"fmt"
	"log"
	config "test/conf"
	"testing"
)

func TestMain(t *testing.T) {
	fmt.Println("=== 简化配置测试 ===")

	// 1. 初始化配置
	config.InitFromLocalFile("config", "yaml")
	fmt.Println("✓ 配置初始化完成")

	// 2. 验证配置
	err := config.ValidateConfig()
	if err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}
	fmt.Println("✓ 配置验证通过")

	// 3. 打印配置摘要
	config.PrintConfigSummary()

	// 4. 测试各项配置
	testConfigurations(t)

	fmt.Println("=== 配置测试完成 ===")
}

// TestConfigDetails 测试配置详细信息
func TestConfigDetails(t *testing.T) {
	// 初始化配置
	config.InitFromLocalFile("config", "yaml")

	// 测试各项配置
	testConfigurations(t)
}

func testConfigurations(t *testing.T) {
	fmt.Println("\n=== 详细配置信息 ===")

	// 测试亚丁ATS配置
	fmt.Println("\n1. 亚丁ATS系统配置:")
	adenATS := config.GetAdenATSConfig()

	// 验证配置值
	if adenATS.BaseURL == "" {
		t.Error("亚丁ATS基础URL不能为空")
	}
	if adenATS.Username == "" {
		t.Error("亚丁ATS用户名不能为空")
	}
	if adenATS.Timeout <= 0 {
		t.Error("亚丁ATS连接超时时间必须大于0")
	}

	fmt.Printf("   - 基础URL: %s\n", adenATS.BaseURL)
	fmt.Printf("   - WebSocket URL: %s\n", adenATS.WssURL)
	fmt.Printf("   - 用户名: %s\n", adenATS.Username)
	fmt.Printf("   - 客户端ID: %s\n", adenATS.ClientId)
	fmt.Printf("   - 连接超时: %d秒\n", adenATS.Timeout)
	fmt.Printf("   - 心跳间隔: %d毫秒\n", adenATS.Heartbeat)
	fmt.Printf("   - 重连间隔: %d毫秒\n", adenATS.ReconnectInterval)
	fmt.Printf("   - 最大重连次数: %d\n", adenATS.MaxReconnectAttempts)

	// 测试数据处理配置
	fmt.Println("\n2. 数据处理配置:")
	dataProcess := config.GetDataProcessConfig()

	// 验证配置值
	if dataProcess.WorkerNum <= 0 {
		t.Error("数据处理工作线程数必须大于0")
	}
	if dataProcess.BatchSize <= 0 {
		t.Error("批处理大小必须大于0")
	}
	if dataProcess.RawBufferSize <= 0 {
		t.Error("原始数据缓冲区大小必须大于0")
	}

	fmt.Printf("   - 原始数据缓冲区: %d\n", dataProcess.RawBufferSize)
	fmt.Printf("   - 解析后数据缓冲区: %d\n", dataProcess.ParsedBufferSize)
	fmt.Printf("   - 死信队列缓冲区: %d\n", dataProcess.DeadBufferSize)
	fmt.Printf("   - 数据处理工作线程: %d\n", dataProcess.WorkerNum)
	fmt.Printf("   - 数据解析工作线程: %d\n", dataProcess.ParserWorkerNum)
	fmt.Printf("   - 数据库写入工作线程: %d\n", dataProcess.DbWorkerNum)
	fmt.Printf("   - 批处理大小: %d\n", dataProcess.BatchSize)
	fmt.Printf("   - 刷新延迟: %d毫秒\n", dataProcess.FlushDelayMs)
	fmt.Printf("   - 数据保留天数: %d\n", dataProcess.DataRetentionDays)
	fmt.Printf("   - 清理任务间隔: %d小时\n", dataProcess.CleanupIntervalHours)

	// 测试文件导出配置
	fmt.Println("\n3. 文件导出配置:")
	exportConfig := config.GetExportConfig()

	// 验证配置值
	if exportConfig.Path == "" {
		t.Error("文件导出路径不能为空")
	}
	if exportConfig.RetentionDays <= 0 {
		t.Error("文件保留天数必须大于0")
	}

	fmt.Printf("   - 导出路径: %s\n", exportConfig.Path)
	fmt.Printf("   - URL前缀: %s\n", exportConfig.URLPrefix)
	fmt.Printf("   - 文件保留天数: %d\n", exportConfig.RetentionDays)

	t.Log("所有配置测试通过")
}
