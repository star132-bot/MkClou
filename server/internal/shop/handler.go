package shop

import (
	"github.com/gin-gonic/gin"

	"mkclou/server/internal/pkg/auth"
	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/ratelimit"
	"mkclou/server/internal/pkg/response"
)

type Handler struct {
	svc     *Service
	limiter *ratelimit.Limiter
}

func NewHandler(svc *Service, limiter *ratelimit.Limiter) *Handler {
	return &Handler{svc: svc, limiter: limiter}
}

// Register 注册路由（接口清单 #18 ～ #24、#72）。requireMerchant 是账号模块提供的鉴权中间件。
func (h *Handler) Register(api *gin.RouterGroup, requireMerchant gin.HandlerFunc) {
	m := api.Group("/shop", requireMerchant)
	m.GET("/slug-availability", h.limiter.ByIP(ratelimit.PerMinute("slug-check", 60)), h.checkSlug)
	m.POST("", h.create)
	m.GET("", h.get)
	m.PATCH("", h.update)
	m.PUT("/status", h.setStatus)
	m.GET("/onboarding", h.onboarding)
	m.POST("/onboarding/shared", h.markShared)

	api.GET("/public/shops/:slug", h.public)
}

func merchantID(c *gin.Context) uint64 {
	id, _ := auth.IdentityFrom(c)
	return id.MerchantID
}

// bind 解析 JSON 请求体，格式错误统一返回参数错误。
func bind(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.Error(c, errcode.InvalidParams.WithMessage("请求格式不正确"))
		return false
	}
	return true
}

func (h *Handler) checkSlug(c *gin.Context) {
	r, err := h.svc.CheckSlug(c.Request.Context(), merchantID(c), c.Query("slug"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, r)
}

func (h *Handler) create(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Create(c.Request.Context(), merchantID(c), CreateInput{Name: req.Name, Slug: req.Slug})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), merchantID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) update(c *gin.Context) {
	var req struct {
		Name         *string       `json:"name"`
		Slug         *string       `json:"slug"`
		Description  *string       `json:"description"`
		ContactEmail *string       `json:"contactEmail"`
		SocialLinks  *[]SocialLink `json:"socialLinks"`
		Theme        *Theme        `json:"theme"`
	}
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Update(c.Request.Context(), merchantID(c), UpdateInput{
		Name: req.Name, Slug: req.Slug, Description: req.Description,
		ContactEmail: req.ContactEmail, SocialLinks: req.SocialLinks, Theme: req.Theme,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) setStatus(c *gin.Context) {
	var req struct {
		Status    string `json:"status"`
		PauseNote string `json:"pauseNote"`
	}
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.SetStatus(c.Request.Context(), merchantID(c), req.Status, req.PauseNote)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

func (h *Handler) onboarding(c *gin.Context) {
	o, err := h.svc.Onboarding(c.Request.Context(), merchantID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, o)
}

func (h *Handler) markShared(c *gin.Context) {
	if err := h.svc.MarkShared(c.Request.Context(), merchantID(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) public(c *gin.Context) {
	r, err := h.svc.Public(c.Request.Context(), c.Param("slug"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, r)
}
