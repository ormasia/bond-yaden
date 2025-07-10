package config

import (
	"fmt"
	"log"
)

// InitConfig 初始化配置，支持从nacos读取，降级到本地yaml
func InitConfig(appName string) error {
	// 1. 首先尝试从本地文件初始化基础配置
	InitFromLocalFile("config", "yaml")

	// 2. 尝试初始化nacos客户端
	err := NewNacosClientInsFromEnv(appName)
	if err != nil {
		log.Printf("初始化nacos客户端失败，将使用本地配置: %v", err)
		return nil // 不返回错误，继续使用本地配置
	}

	// 3. 尝试从nacos读取各项配置
	initConfigFromNacos()

	return nil
}

// initConfigFromNacos 从nacos读取配置，如果失败则使用本地yaml配置
func initConfigFromNacos() {
	// 亚丁ATS系统配置
	initAdenATSConfigFromNacos()

	// 数据处理配置
	initDataProcessConfigFromNacos()
}

// initAdenATSConfigFromNacos 从nacos读取亚丁ATS系统配置
func initAdenATSConfigFromNacos() {
	configKey := "adenATS"
	if nacosKey, exists := NacosKeys[configKey]; exists {
		err := GetViperCfgFromNacos(nacosKey, configKey, "yaml")
		if err != nil {
			log.Printf("从nacos读取%s配置失败，使用本地配置: %v", configKey, err)
		} else {
			log.Printf("成功从nacos读取%s配置", configKey)
		}
	}
}

// initDataProcessConfigFromNacos 从nacos读取数据处理配置
func initDataProcessConfigFromNacos() {
	configKey := "dataProcess"
	if nacosKey, exists := NacosKeys[configKey]; exists {
		err := GetViperCfgFromNacos(nacosKey, configKey, "yaml")
		if err != nil {
			log.Printf("从nacos读取%s配置失败，使用本地配置: %v", configKey, err)
		} else {
			log.Printf("成功从nacos读取%s配置", configKey)
		}
	}
}

// PrintConfigSummary 打印配置摘要信息
func PrintConfigSummary() {
	fmt.Println("=== 配置摘要 ===")

	// 亚丁ATS配置摘要
	adenATS := GetAdenATSConfig()
	fmt.Printf("亚丁ATS系统: %s (用户: %s)\n", adenATS.BaseURL, adenATS.Username)

	// 数据处理配置摘要
	dataProcess := GetDataProcessConfig()
	fmt.Printf("数据处理: 工作线程=%d, 批处理大小=%d\n", dataProcess.WorkerNum, dataProcess.BatchSize)

	// 文件导出配置摘要
	exportConfig := GetExportConfig()
	fmt.Printf("文件导出: 路径=%s, 保留天数=%d\n", exportConfig.Path, exportConfig.RetentionDays)

	fmt.Println("===============")
}

// ValidateConfig 验证配置的有效性
func ValidateConfig() error {
	// 验证亚丁ATS配置
	adenATS := GetAdenATSConfig()
	if adenATS.BaseURL == "" || adenATS.Username == "" {
		return fmt.Errorf("亚丁ATS系统配置无效")
	}

	// 验证数据处理配置
	dataProcess := GetDataProcessConfig()
	if dataProcess.WorkerNum <= 0 || dataProcess.BatchSize <= 0 {
		return fmt.Errorf("数据处理配置无效")
	}

	// 验证文件导出配置
	exportConfig := GetExportConfig()
	if exportConfig.Path == "" {
		return fmt.Errorf("文件导出配置无效")
	}

	return nil
}
