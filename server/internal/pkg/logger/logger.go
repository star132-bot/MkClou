// Package logger 基于 zap 构建结构化日志。
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New 根据级别和格式创建 logger。format 为 json 时输出 JSON（生产环境），否则输出彩色控制台格式。
func New(level, format string) (*zap.Logger, error) {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}

	var cfg zap.Config
	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 开发配置默认给 Warn 级别附带堆栈，4xx 请求日志会被误看成程序崩溃；只在 Error 及以上附带堆栈
	return cfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
}
