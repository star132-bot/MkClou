// Package database 负责创建 MySQL（GORM）与 Redis 连接。
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"mkclou/server/internal/pkg/config"
)

// NewMySQL 创建 GORM 连接并验证可用性。debug 为 true 时打印 SQL。
func NewMySQL(cfg config.MySQLConfig, debug bool) (*gorm.DB, error) {
	logLevel := gormlogger.Warn
	if debug {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(logLevel),
		NowFunc:                func() time.Time { return time.Now().UTC() },
		SkipDefaultTransaction: true, // 单条写操作不额外开启事务，需要事务时显式使用
		TranslateError:         true, // 将唯一键冲突等转换为 gorm.ErrDuplicatedKey
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql %s:%d: %w", cfg.Host, cfg.Port, err)
	}
	return db, nil
}

// NewRedis 创建 Redis 客户端并验证可用性。
func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis %s: %w", cfg.Addr, err)
	}
	return rdb, nil
}

// CloseMySQL 关闭底层连接池，错误仅记录日志。
func CloseMySQL(db *gorm.DB, log *zap.Logger) {
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Warn("close mysql", zap.Error(err))
		}
	}
}
