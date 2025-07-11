package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlCfg struct {
	User         string `json:"user"`
	Password     string `json:"password"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	Schema 		 string	`json:"schema"`
}

type DBPoolConfig struct {
	MaxIdleConn     int `json:"maxIdleConn"`
	MaxOpenConn     int `json:"maxOpenConn"`
	ConnMaxLifetime int `json:"connMaxLifetime"`
}


func InitMysqlConnPool(cfg *MysqlCfg) (*gorm.DB, error) {
	dbURI := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Schema)
	connPool, err := gorm.Open(
		mysql.New(mysql.Config{
			DSN:                       dbURI,
			DefaultStringSize:         256,
			DisableDatetimePrecision:  true,
			DontSupportRenameIndex:    true,
			DontSupportRenameColumn:   true,
			SkipInitializeWithVersion: false,
		}),
		&gorm.Config{QueryFields: true})
	if err != nil {
		return nil, err
	}
	return connPool, nil
}
