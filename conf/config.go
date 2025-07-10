package config

import (
	"github.com/spf13/viper"
	"sync"
)

type APPConfig struct {
	ApplVer string `yaml:"appVer"`
	Mode    string `yaml:"mode"`
	Addr    string `yaml:"addr"`
	Prefork bool   `yaml:"prefork"`
}

func GetCfg(key string, cfg interface{}) error {
	if key == "" {
		return viper.Unmarshal(cfg)
	}
	return viper.Sub(key).Unmarshal(cfg)
}

func GetCfgStr(key string) string {
	return viper.GetString(key)
}

type ApmConfig struct {
	Endpoint string `yaml:"endpoint"`
}

type LogCfg struct {
	Path  string `json:"path"`
	Level string `json:"level"`
	Size  int    `json:"size"`
	Count int    `json:"count"`
}

type UserHttpConf struct {
	Url     string `yaml:"url"`
	Timeout int    `yaml:"timeout"`
}

var (
	UserHttp     *UserHttpConf
	onceUserHttp sync.Once

	onceOSSConfig sync.Once
	ossConfigInfo *OSSConfig
)

func GetUserHttpConf() *UserHttpConf {
	onceUserHttp.Do(func() {
		UserHttp = &UserHttpConf{}
		_ = GetCfg("userHttpConf", &UserHttp)
	})
	return UserHttp
}

type OSSConfig struct {
	EndPoint         string `yaml:"endPoint"`
	ExpediteEndPoint string `yaml:"expediteEndPoint"`
	UseExpedite      bool   `yaml:"useExpedite"`
}

func (conf *OSSConfig) GetFilePathPrefix() string {
	if conf.UseExpedite == true {
		return conf.ExpediteEndPoint
	} else {
		return conf.EndPoint
	}
}

func GetOSSConfig() *OSSConfig {
	onceOSSConfig.Do(func() {
		ossConfigInfo = &OSSConfig{}
		_ = GetCfg("ossConfig", &ossConfigInfo)
	})
	return ossConfigInfo
}

type FLPConfig struct {
	Host               string `json:"host"`
	GetFundHoliday     string `json:"getFundHoliday"`
	GetFundProductInfo string `json:"getFundProductInfo"`
	BondHost           string `json:"bondHost"`
	BondRealTimePrice  string `json:"bondRealTimePrice"`
}

var (
	flpConfig     *FLPConfig
	onceFLPConfig sync.Once
)

func GetFlpConfig() *FLPConfig {
	onceFLPConfig.Do(func() {
		flpConfig = &FLPConfig{}
		_ = GetCfg("flpConfig", &flpConfig)
	})
	return flpConfig
}
