package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	pkgNacos "test/pkg/nacos"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var (
	ins *pkgNacos.NacosCli
	// vLock sync.Mutex
)

const (
	KMS_ACCESS_KEY  = "KMS_ACCESS_KEY"
	kMS_SECRET_KEY  = "KMS_SECRET_KEY"
	NACOS_URL       = "NACOS_URL"
	NACOS_PORT      = "NACOS_PORT"
	NACOS_REGION_ID = "NACOS_REGION_ID"
	NACOS_CACHE_DIR = "NACOS_CACHE_DIR"

	DEF_NACOS_PORT = 8848
)

var NacosKeys = map[string]string{
	"apm": "apm-endpoint@@public@@",

	// MySQL数据库配置
	"mysql.fund":      "cipher-mysql-funddb-ro@@mkt@@mkt",
	"mysql.symbol":    "cipher-mysql-symboldb-ro@@mkt@@mkt",
	"mysql.bond":      "cipher-mysql-bonddb-ro@@mkt@@mkt",
	"mysql.amount":    "cipher-mysql-amountdb-rw@@mkt@@mkt",
	"mysql.public":    "cipher-mysql-publicdb-ro@@mkt@@mkt",
	"mysql.forex":     "cipher-mysql-forexdb-ro@@mkt@@mkt",
	"mysql.bondQuote": "cipher-mysql-bondquote-rw@@mkt@@mkt", // 债券行情数据库

	// MongoDB数据库配置
	"mongo.wealthmanagedb":   "cipher-mongo-wealthmanage-rw@@mkt@@mkt",
	"mongo.wealthmanagedb_r": "cipher-mongo-wealthmanage-ro@@mkt@@mkt",
}

func GetViperCfgFromNacos(key, localKey, cfgType string) error {
	b, err := GetConfigFromNacos(key)
	if err != nil {
		return err
	}
	v := viper.New()
	v.SetConfigType(cfgType)
	err = v.ReadConfig(bytes.NewBufferString(b))
	if err != nil {
		return err
	}
	if localKey == "" {
		return viper.MergeConfigMap(v.AllSettings())
	} else {
		//newSeting := map[string]interface{}{}
		//for k, conf := range v.AllSettings(){
		//	newSeting[fmt.Sprintf("%s.%s", localKey, k)] = conf
		//}
		//fmt.Println(newSeting)

		return viper.MergeConfigMap(map[string]interface{}{localKey: v.AllSettings()})
	}
}

func NewNacosClientInsFromEnv(app string) error {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("cant read env file! for develop better read form env file")
	}
	var port int32 = DEF_NACOS_PORT //default nacos port
	url := os.Getenv(NACOS_URL)
	if url == "" {
		return errors.New("cant get env var NACOS_URL ")
	}
	p := os.Getenv(NACOS_PORT)
	if p != "" {
		nacosPort, er := strconv.Atoi(p)
		if er != nil {
			return er
		}
		port = int32(nacosPort)
	}

	dir := os.Getenv(NACOS_CACHE_DIR)
	if dir == "" {
		dir = "/data/logs"
	}
	fmt.Println("nacos:", url)
	ins = pkgNacos.NewNacosOpts(
		pkgNacos.WithAddr(os.Getenv(NACOS_URL)),
		pkgNacos.WithPort(port),
		pkgNacos.WithCacheDir(dir),
		pkgNacos.WithLogLevel("warn"),
		pkgNacos.WithLogDir(dir+"/"+app),
		pkgNacos.WithKmsAK(os.Getenv(KMS_ACCESS_KEY)),
		pkgNacos.WithKmsSK(os.Getenv(kMS_SECRET_KEY)),
		pkgNacos.WithRegionId(os.Getenv(NACOS_REGION_ID)),
	)
	return nil
}

func GetCfgByNacosKey(nacosKey, key, cfgType string, cfg interface{}) error {
	b, err := GetConfigFromNacos(nacosKey)
	if err != nil {
		return err
	}
	v := viper.New()
	v.SetConfigType(cfgType)
	err = v.ReadConfig(bytes.NewBufferString(b))
	if err != nil {
		return err
	}
	if key == "" {
		return viper.Unmarshal(cfg)
	}
	return viper.Sub(key).Unmarshal(cfg)
}

func CloseNacosConns() {
	for _, v := range ins.NacosCliMap {
		v.CloseClient()
	}
}

// get config string  so you can Unmarshal yourself !
func GetConfigFromNacos(key string) (string, error) {
	return GetConfigFromNacosSep(key, "@@")
}

func GetConfigFromNacosSep(key, sep string) (string, error) {
	arr := strings.Split(key, sep)
	return ins.GetCfgFromNacos(arr[0], arr[1], arr[2])
}
