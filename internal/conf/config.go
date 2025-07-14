package config

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
)

// APPConfig 应用基础配置
type APPConfig struct {
	Name    string `yaml:"name"`
	Mode    string `yaml:"mode"`
	Addr    string `yaml:"addr"`
	Ver     string `yaml:"ver"`
	Prefork bool   `yaml:"prefork"`
}

// LogCfg 日志配置
type LogCfg struct {
	Path  string `yaml:"path"`
	Level string `yaml:"level"`
	Size  int    `yaml:"size"`
	Count int    `yaml:"count"`
}

// AdenATSConfig 亚丁ATS系统配置
type AdenATSConfig struct {
	BaseURL              string `yaml:"baseURL"`
	WssURL               string `yaml:"wssURL"`
	Username             string `yaml:"username"`
	Password             string `yaml:"password"`
	SmsCode              string `yaml:"smsCode"`
	ClientId             string `yaml:"clientId"`
	PublicKey            string `yaml:"publicKey"`
	Timeout              int    `yaml:"timeout"`
	Heartbeat            int    `yaml:"heartbeat"`
	ReconnectInterval    int    `yaml:"reconnectInterval"`
	MaxReconnectAttempts int    `yaml:"maxReconnectAttempts"`
}

// DataProcessConfig 数据处理配置
type DataProcessConfig struct {
	RawBufferSize        int `yaml:"rawBufferSize"`
	ParsedBufferSize     int `yaml:"parsedBufferSize"`
	DeadBufferSize       int `yaml:"deadBufferSize"`
	WorkerNum            int `yaml:"workerNum"`
	ParserWorkerNum      int `yaml:"parserWorkerNum"`
	DbWorkerNum          int `yaml:"dbWorkerNum"`
	BatchSize            int `yaml:"batchSize"`
	FlushDelayMs         int `yaml:"flushDelayMs"`
	DataRetentionDays    int `yaml:"dataRetentionDays"`
	CleanupIntervalHours int `yaml:"cleanupIntervalHours"`
}

// ExportConfig 文件导出配置
type ExportConfig struct {
	Path          string `yaml:"path"`
	URLPrefix     string `yaml:"urlPrefix"`
	RetentionDays int    `yaml:"retentionDays"`
}
type OSSConfig struct {
	URL     string `yaml:"url"`
	Timeout int    `yaml:"timeout"` // 超时时间（秒）
}

// 配置获取函数
func GetCfg(key string, cfg interface{}) error {
	if key == "" {
		return viper.Unmarshal(cfg)
	}
	sub := viper.Sub(key)
	if sub == nil {
		return fmt.Errorf("配置键 '%s' 不存在", key)
	}
	return sub.Unmarshal(cfg)
}

func GetCfgStr(key string) string {
	return viper.GetString(key)
}

// 单例模式配置实例
var (
	adenATSConfig *AdenATSConfig
	onceAdenATS   sync.Once

	dataProcessConfig *DataProcessConfig
	onceDataProcess   sync.Once

	exportConfig *ExportConfig
	onceExport   sync.Once
)

// GetAdenATSConfig 获取亚丁ATS配置
func GetAdenATSConfig() *AdenATSConfig {
	onceAdenATS.Do(func() {
		adenATSConfig = &AdenATSConfig{}
		if err := GetCfg("adenATS", adenATSConfig); err != nil {
			fmt.Printf("警告: 获取亚丁ATS配置失败: %v\n", err)
		}
	})
	return adenATSConfig
}

// GetDataProcessConfig 获取数据处理配置
func GetDataProcessConfig() *DataProcessConfig {
	onceDataProcess.Do(func() {
		dataProcessConfig = &DataProcessConfig{}
		if err := GetCfg("dataProcess", dataProcessConfig); err != nil {
			fmt.Printf("警告: 获取数据处理配置失败: %v\n", err)
		}
	})
	return dataProcessConfig
}

// GetExportConfig 获取文件导出配置
func GetExportConfig() *ExportConfig {
	onceExport.Do(func() {
		exportConfig = &ExportConfig{}
		if err := GetCfg("export", exportConfig); err != nil {
			fmt.Printf("警告: 获取文件导出配置失败: %v\n", err)
		}
	})
	return exportConfig
}
