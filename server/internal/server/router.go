// Package server 组装 HTTP 路由与中间件。
package server

import (
	"context"

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
	"mkclou/server/internal/pkg/storage"
	"mkclou/server/internal/product"
	"mkclou/server/internal/shop"
	"mkclou/server/internal/upload"
	"mkclou/server/internal/user"
)

type Deps struct {
	Config *config.Config
	Log    *zap.Logger
	DB     *gorm.DB
	Redis  *redis.Client
	Mail   mailer.Queue
	Store  *storage.Client
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

	userRepo := user.NewRepository(d.DB)
	userSvc := user.NewService(user.Deps{
		Repo: userRepo, Redis: d.Redis, Limiter: limiter,
		Captcha: captchaSvc, Mail: d.Mail, Config: d.Config, Log: d.Log,
	})
	shopSvc := shop.NewService(shop.Deps{
		Repo: shop.NewRepository(d.DB), Merchants: merchantAccounts{userRepo}, Redis: d.Redis,
		Assets: d.Store, Config: d.Config, Log: d.Log,
	})
	imageSvc := upload.NewImageService(d.Store, d.Redis)
	productSvc := product.NewService(product.Deps{
		Repo: product.NewRepository(d.DB), Shops: shopSvc, Images: imageSvc, Assets: d.Store, Log: d.Log,
	})

	userHandler := user.NewHandler(userSvc, captchaSvc, limiter, d.Config.Auth, shopSummaries{shopSvc})
	userHandler.Register(api)
	requireMerchant := userHandler.RequireMerchant()
	shop.NewHandler(shopSvc, limiter).Register(api, requireMerchant)
	upload.NewHandler(imageSvc, shopSvc, limiter).Register(api, requireMerchant)
	market := product.NewMarket(productSvc, product.NewMarketRepository(d.DB), d.Redis, d.Log)
	favorites := product.NewFavorites(market, product.NewFavoriteRepository(d.DB))
	product.NewHandler(productSvc, market, favorites).Register(api, requireMerchant)

	return r
}

// 以下适配器连接账号与店铺模块，使两个包互不导入。

type merchantAccounts struct{ repo user.Repository }

func (a merchantAccounts) Account(ctx context.Context, merchantID uint64) (string, bool, error) {
	m, err := a.repo.FindByID(ctx, merchantID)
	if err != nil {
		return "", false, err
	}
	return m.Email, m.EmailVerified(), nil
}

type shopSummaries struct{ svc *shop.Service }

func (a shopSummaries) SummaryOf(ctx context.Context, merchantID uint64) (any, error) {
	s, err := a.svc.Summary(ctx, merchantID)
	if err != nil || s == nil {
		return nil, err // 未创建店铺时返回无类型的 nil，JSON 中为 null
	}
	return s, nil
}
