// 异步任务 worker 入口（Asynq）：关单、交付、发送邮件、定时补偿等，任务清单见 docs/api-list.md 第 3 节。
package main

import (
	"fmt"
	"os"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/logger"
	"mkclou/server/internal/pkg/mailer"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("configs")
	if err != nil {
		return err
	}
	log, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	const concurrency = 10
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB},
		asynq.Config{
			Concurrency: concurrency,
			// 交付与支付相关任务优先处理
			Queues:         map[string]int{"critical": 6, "default": 3, "low": 1},
			RetryDelayFunc: mailer.RetryDelay,
		},
	)

	mux := asynq.NewServeMux()
	mux.Handle(mailer.TypeSendEmail, mailer.NewHandler(mailer.NewSMTPSender(cfg.Mail), log))
	// 其他模块的任务处理器在开发对应模块时注册，例如：
	// mux.HandleFunc(order.TaskClose, orderWorker.HandleClose)

	log.Info("worker starting", zap.Int("concurrency", concurrency))
	return srv.Run(mux) // 内部处理 SIGINT / SIGTERM 并优雅退出
}
