// Package server 组装 HTTP 路由与中间件。
package server

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"mkclou/server/internal/health"
	"mkclou/server/internal/pkg/captcha"
	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/mailer"
	"mkclou/server/internal/pkg/middleware"
	"mkclou/server/internal/pkg/ratelimit"
	"mkclou/server/internal/user"
)

type Deps struct {
	Config *config.Config
	Log    *zap.Logger
	DB     *gorm.DB
	Redis  *redis.Client
	Mail   mailer.Queue
}

func NewRouter(d Deps) *gin.Engine {
	if d.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.HandleMethodNotAllowed = true
	// 只信任本机反向代理（Nginx）传递的客户端 IP
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	r.Use(
		middleware.RequestID(d.Log),
		middleware.Recovery(d.Log),
		middleware.AccessLog(d.Log),
		middleware.CORS(d.Config.App.CORSOrigins),
	)
	r.NoRoute(middleware.NoRoute)
	r.NoMethod(middleware.NoRoute)

	health.NewHandler(d.DB, d.Redis).Register(r)

	limiter := ratelimit.New(d.Redis)
	captchaSvc := captcha.New(d.Redis)

	// 业务接口统一挂在 /api/v1 下，各模块按 docs/api-list.md 逐步注册
	api := r.Group("/api/v1", limiter.ByIP(ratelimit.PerMinute("global", 300)))

	userSvc := user.NewService(user.Deps{
		Repo: user.NewRepository(d.DB), Redis: d.Redis, Limiter: limiter,
		Captcha: captchaSvc, Mail: d.Mail, Config: d.Config, Log: d.Log,
	})
	user.NewHandler(userSvc, captchaSvc, limiter, d.Config.Auth).Register(api)

	return r
}
