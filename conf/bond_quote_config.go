package config

import (
    "sync"
)

// BondQuoteDBConfig 债券行情数据库配置
type BondQuoteDBConfig struct {
    Host            string `yaml:"host"`
    Port            int    `yaml:"port"`
    User            string `yaml:"user"`
    Password        string `yaml:"password"`
    Schema          string `yaml:"schema"`
    MaxIdleConn     int    `yaml:"maxIdleConn"`
    MaxOpenConn     int    `yaml:"maxOpenConn"`
    ConnMaxLifetime int    `yaml:"connMaxLifetime"`
}

// AdenATSConfig 亚丁ATS系统配置
type AdenATSConfig struct {
    BaseURL   string `yaml:"baseURL"`
    WssURL    string `yaml:"wssURL"`
    Username  string `yaml:"username"`
    Password  string `yaml:"password"`
    SmsCode   string `yaml:"smsCode"`
    ClientId  string `yaml:"clientId"`
    PublicKey string `yaml:"publicKey"`
}

// DataProcessConfig 数据处理配置
type DataProcessConfig struct {
    RawBufferSize    int `yaml:"rawBufferSize"`
    ParsedBufferSize int `yaml:"parsedBufferSize"`
    WorkerNum        int `yaml:"workerNum"`
    BatchSize        int `yaml:"batchSize"`
    FlushDelayMs     int `yaml:"flushDelayMs"`
}

// ExportConfig 文件导出配置
type ExportConfig struct {
    Path          string `yaml:"path"`
    URLPrefix     string `yaml:"urlPrefix"`
    RetentionDays int    `yaml:"retentionDays"`
}

// FiberConfig Fiber服务器配置
type FiberConfig struct {
    Port         int `yaml:"port"`
    ReadTimeout  int `yaml:"readTimeout"`
    WriteTimeout int `yaml:"writeTimeout"`
    IdleTimeout  int `yaml:"idleTimeout"`
}

var (
    bondQuoteDBConfig     *BondQuoteDBConfig
    onceBondQuoteDBConfig sync.Once

    adenATSConfig     *AdenATSConfig
    onceAdenATSConfig sync.Once

    dataProcessConfig     *DataProcessConfig
    onceDataProcessConfig sync.Once

    exportConfig     *ExportConfig
    onceExportConfig sync.Once

    fiberConfig     *FiberConfig
    onceFiberConfig sync.Once
)

// GetBondQuoteDBConfig 获取债券行情数据库配置
func GetBondQuoteDBConfig() *BondQuoteDBConfig {
    onceBondQuoteDBConfig.Do(func() {
        bondQuoteDBConfig = &BondQuoteDBConfig{
            Host:            "localhost",
            Port:            3306,
            User:            "root",
            Password:        "password",
            Schema:          "bond_quote_db",
            MaxIdleConn:     5,
            MaxOpenConn:     20,
            ConnMaxLifetime: 300,
        }
        _ = GetCfg("mysql.bondQuote", bondQuoteDBConfig)
    })
    return bondQuoteDBConfig
}

// GetAdenATSConfig 获取亚丁ATS系统配置
func GetAdenATSConfig() *AdenATSConfig {
    onceAdenATSConfig.Do(func() {
        adenATSConfig = &AdenATSConfig{
            BaseURL:   "https://adenapi.cstm.adenfin.com",
            WssURL:    "wss://adenapi.cstm.adenfin.com/message-gateway/message/atsapi/ws",
            Username:  "ATSTEST10001",
            Password:  "Abc12345",
            SmsCode:   "1234",
            ClientId:  "30021",
            PublicKey: "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCTCY4u102mtUlVEyUMlXOkflPLdWN+ez5IDcLiNzw2ZkiEY17U4lk8iMx7yTEO/ZWCKIEdQV+U6tplJ98X3I/Py/DzWd1L7IPE6mZgclfcXg+P4ocaHPsKgAodc4G1W9jTu2d6obL3d33USCD0soGYE6fkf8hk7EPKhgNf4iUPCwIDAQAB",
        }
        _ = GetCfg("adenATS", adenATSConfig)
    })
    return adenATSConfig
}

// GetDataProcessConfig 获取数据处理配置
func GetDataProcessConfig() *DataProcessConfig {
    onceDataProcessConfig.Do(func() {
        dataProcessConfig = &DataProcessConfig{
            RawBufferSize:    20000,
            ParsedBufferSize: 4000,
            WorkerNum:        8,
            BatchSize:        300,
            FlushDelayMs:     100,
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

// GetFiberConfig 获取Fiber服务器配置
func GetFiberConfig() *FiberConfig {
    onceFiberConfig.Do(func() {
        fiberConfig = &FiberConfig{
            Port:         8080,
            ReadTimeout:  10,
            WriteTimeout: 30,
            IdleTimeout:  120,
        }
        _ = GetCfg("fiber", fiberConfig)
    })
    return fiberConfig
}