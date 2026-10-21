package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-agent/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	mysqlDB   *gorm.DB
	mysqlOnce sync.Once
	mysqlErr  error
)

// InitMySQL 初始化 MySQL 连接（gorm）。
// 采用 sync.Once 单例，失败时返回 error 供上层降级。
// 与 InitRedis 保持一致的降级语义：连接不上不 panic，由调用方决定是否降级为 Noop。
func InitMySQL(ctx context.Context) (*gorm.DB, error) {
	mysqlOnce.Do(func() {
		conf := config.Cfg.MySQLConf

		if conf.Host == "" || conf.Database == "" {
			mysqlErr = fmt.Errorf("mysql config incomplete: host=%q database=%q", conf.Host, conf.Database)
			return
		}

		// username:password@tcp(host:port)/database?...
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			conf.Username, conf.Password, conf.Host, conf.Port, conf.Database,
		)

		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			// 长期记忆写入不是热点路径，用 Warn 级别即可，避免刷屏
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			mysqlErr = fmt.Errorf("open mysql failed: %w", err)
			return
		}

		// 测试连接
		sqlDB, err := db.DB()
		if err != nil {
			mysqlErr = fmt.Errorf("get sql.DB failed: %w", err)
			return
		}

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(pingCtx); err != nil {
			mysqlErr = fmt.Errorf("ping mysql failed: %w", err)
			return
		}

		// 连接池配置
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetMaxOpenConns(50)
		sqlDB.SetConnMaxLifetime(time.Hour)

		mysqlDB = db
	})

	return mysqlDB, mysqlErr
}

// GetMySQL 返回已初始化的 *gorm.DB，未初始化或初始化失败时返回 nil。
// 调用方应判空以决定是否降级。
func GetMySQL() *gorm.DB {
	return mysqlDB
}
