package dataSource

import (
	"sync"
	config "test/internal/conf"
	"test/pkg/db"
	"time"

	"gorm.io/gorm"
)

var (
	mu          sync.Mutex
	connPoolMap = make(map[string]*gorm.DB, 5)
)

func GetDBConn(dbName string) *gorm.DB {
	mu.Lock()
	defer mu.Unlock()
	pool, ok := connPoolMap[dbName]
	if ok {
		return pool
	}
	cfg := &db.MysqlCfg{}
	_ = config.GetCfg("mysql."+dbName, &cfg)
	connPool, er := db.InitMysqlConnPool(cfg)
	if er != nil {
		panic(er)
	}
	conn, err := connPool.DB()
	if err != nil {
		panic(err)
	}
	pCfg := db.DBPoolConfig{}
	_ = config.GetCfg("mysqlDBPool", &pCfg)
	conn.SetMaxIdleConns(pCfg.MaxIdleConn)
	conn.SetMaxOpenConns(pCfg.MaxOpenConn)
	conn.SetConnMaxLifetime(time.Second * time.Duration(pCfg.ConnMaxLifetime))
	if err := conn.Ping(); err != nil {
		conn.Close()
		panic(err)
	}
	connPoolMap[dbName] = connPool
	return connPool
}

func IsDBNoData(err error) bool {
	if err != nil && err.Error() == gorm.ErrRecordNotFound.Error() {
		return true
	}
	return false
}
