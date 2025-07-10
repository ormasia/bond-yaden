package config

import (
	"sync"
)

// AdenATSConfig 亚丁ATS系统配置
type AdenATSConfig struct {
	BaseURL              string `yaml:"baseURL"`
	WssURL               string `yaml:"wssURL"`
	Username             string `yaml:"username"`
	Password             string `yaml:"password"`
	SmsCode              string `yaml:"smsCode"`
	ClientId             string `yaml:"clientId"`
	PublicKey            string `yaml:"publicKey"`
	Timeout              int    `yaml:"timeout"`              // 连接超时时间（秒）
	Heartbeat            int    `yaml:"heartbeat"`            // 心跳间隔（毫秒）
	ReconnectInterval    int    `yaml:"reconnectInterval"`    // 重连间隔（毫秒）
	MaxReconnectAttempts int    `yaml:"maxReconnectAttempts"` // 最大重连次数
}

// DataProcessConfig 数据处理配置
type DataProcessConfig struct {
	// 缓冲区配置
	RawBufferSize    int `yaml:"rawBufferSize"`    // 原始数据缓冲区大小
	ParsedBufferSize int `yaml:"parsedBufferSize"` // 解析后数据缓冲区大小
	DeadBufferSize   int `yaml:"deadBufferSize"`   // 死信队列缓冲区大小
	// 工作线程配置
	WorkerNum       int `yaml:"workerNum"`       // 数据处理工作线程数
	ParserWorkerNum int `yaml:"parserWorkerNum"` // 数据解析工作线程数
	DbWorkerNum     int `yaml:"dbWorkerNum"`     // 数据库写入工作线程数
	// 批处理配置
	BatchSize    int `yaml:"batchSize"`    // 批处理大小
	FlushDelayMs int `yaml:"flushDelayMs"` // 刷新延迟（毫秒）
	// 数据清理配置
	DataRetentionDays    int `yaml:"dataRetentionDays"`    // 数据保留天数
	CleanupIntervalHours int `yaml:"cleanupIntervalHours"` // 清理任务执行间隔（小时）
}

// ExportConfig 文件导出配置
type ExportConfig struct {
	Path          string `yaml:"path"`
	URLPrefix     string `yaml:"urlPrefix"`
	RetentionDays int    `yaml:"retentionDays"`
}

var (
	adenATSConfig     *AdenATSConfig
	onceAdenATSConfig sync.Once

	dataProcessConfig     *DataProcessConfig
	onceDataProcessConfig sync.Once

	exportConfig     *ExportConfig
	onceExportConfig sync.Once
)

// GetAdenATSConfig 获取亚丁ATS系统配置
func GetAdenATSConfig() *AdenATSConfig {
	onceAdenATSConfig.Do(func() {
		adenATSConfig = &AdenATSConfig{
			BaseURL:              "https://adenapi.cstm.adenfin.com",
			WssURL:               "wss://adenapi.cstm.adenfin.com/message-gateway/message/atsapi/ws",
			Username:             "ATSTEST10001",
			Password:             "Abc12345",
			SmsCode:              "1234",
			ClientId:             "30021",
			PublicKey:            "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCTCY4u102mtUlVEyUMlXOkflPLdWN+ez5IDcLiNzw2ZkiEY17U4lk8iMx7yTEO/ZWCKIEdQV+U6tplJ98X3I/Py/DzWd1L7IPE6mZgclfcXg+P4ocaHPsKgAodc4G1W9jTu2d6obL3d33USCD0soGYE6fkf8hk7EPKhgNf4iUPCwIDAQAB",
			Timeout:              30,
			Heartbeat:            20000,
			ReconnectInterval:    5000,
			MaxReconnectAttempts: 10,
		}
		_ = GetCfg("adenATS", adenATSConfig)
	})
	return adenATSConfig
}

// GetDataProcessConfig 获取数据处理配置
func GetDataProcessConfig() *DataProcessConfig {
	onceDataProcessConfig.Do(func() {
		dataProcessConfig = &DataProcessConfig{
			RawBufferSize:        20000,
			ParsedBufferSize:     4000,
			DeadBufferSize:       1000,
			WorkerNum:            8,
			ParserWorkerNum:      4,
			DbWorkerNum:          2,
			BatchSize:            300,
			FlushDelayMs:         100,
			DataRetentionDays:    30,
			CleanupIntervalHours: 24,
		}
		_ = GetCfg("dataProcess", dataProcessConfig)
	})
	return dataProcessConfig
}

// GetExportConfig 获取文件导出配置
func GetExportConfig() *ExportConfig {
	onceExportConfig.Do(func() {
		exportConfig = &ExportConfig{
			Path:          "/data/export/bond_quote",
			URLPrefix:     "http://localhost:8081/download",
			RetentionDays: 7,
		}
		_ = GetCfg("export", exportConfig)
	})
	return exportConfig
}
