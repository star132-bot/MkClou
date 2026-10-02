// 异步任务 worker 入口（Asynq）：关单、交付、发送邮件、定时补偿等，任务清单见 docs/api-list.md 第 3 节。
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/database"
	"mkclou/server/internal/pkg/logger"
	"mkclou/server/internal/pkg/mailer"
	"mkclou/server/internal/pkg/storage"
	"mkclou/server/internal/product"
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

	db, err := database.NewMySQL(cfg.MySQL, false)
	if err != nil {
		return err
	}
	defer database.CloseMySQL(db, log)
	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()

	// 热门榜单只用到商城查询，不需要商品服务的店铺与图片依赖
	market := product.NewMarket(
		product.NewService(product.Deps{Assets: storage.New(cfg.S3), Log: log}),
		product.NewMarketRepository(db), rdb, log,
	)

	redisOpt := asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB}
	const concurrency = 10
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: concurrency,
		// 交付与支付相关任务优先处理
		Queues:         map[string]int{"critical": 6, "default": 3, "low": 1},
		RetryDelayFunc: mailer.RetryDelay,
	})

	mux := asynq.NewServeMux()
	mux.Handle(mailer.TypeSendEmail, mailer.NewHandler(mailer.NewSMTPSender(cfg.Mail), log))
	mux.HandleFunc(product.TaskRefreshHot, func(ctx context.Context, _ *asynq.Task) error {
		n, err := market.RefreshHot(ctx)
		if err == nil {
			log.Info("hot products refreshed", zap.Int("count", n))
		}
		return err
	})
	// 其他模块的任务处理器在开发对应模块时注册，例如：
	// mux.HandleFunc(order.TaskClose, orderWorker.HandleClose)

	// 定时任务（docs/api-list.md 第 3 节）
	scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{Location: time.UTC})
	if _, err := scheduler.Register("@hourly", asynq.NewTask(product.TaskRefreshHot, nil), asynq.Queue("low"), asynq.MaxRetry(1)); err != nil {
		return fmt.Errorf("register scheduler: %w", err)
	}
	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}
	defer scheduler.Shutdown()

	// 启动时立即计算一次，避免首页热门在第一个整点前为空
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if _, err := market.RefreshHot(ctx); err != nil {
			log.Warn("initial hot products refresh failed", zap.Error(err))
		}
	}()

	log.Info("worker starting", zap.Int("concurrency", concurrency))
	return srv.Run(mux) // 内部处理 SIGINT / SIGTERM 并优雅退出
}
