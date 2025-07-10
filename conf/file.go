package config

import (
	"log"

	"github.com/spf13/viper"
)

func InitFromLocalFile(fileName, fileType string) {
	viper.AddConfigPath("./config/dev-k8s") //for local dev!!! delete if you dont like it
	viper.AddConfigPath("/data/conf")
	viper.AddConfigPath("./app/conf")
	viper.SetConfigType(fileType)
	viper.SetConfigName(fileName)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("readInConfig err:%s", err.Error())
	}
}
