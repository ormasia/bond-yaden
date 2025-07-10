package config

import (
	"log"

	"github.com/spf13/viper"
)

func InitFromLocalFile(fileName, fileType string) {
	viper.AddConfigPath("./conf/")    // 配置文件在 pkg 目录下
	viper.AddConfigPath("./conf/")    // 备用路径
	viper.AddConfigPath("/data/conf") // 生产环境路径
	viper.AddConfigPath("./app/conf") // 备用路径
	viper.SetConfigType(fileType)
	viper.SetConfigName(fileName)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("readInConfig err:%s", err.Error())
	}
}
