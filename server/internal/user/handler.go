package user

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mkclou/server/internal/pkg/auth"
	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/ratelimit"
	"mkclou/server/internal/pkg/response"
)

// CaptchaGenerator 生成图形验证码。
type CaptchaGenerator interface {
	Generate() (id, image string, err error)
}

type Handler struct {
	svc     *Service
	captcha CaptchaGenerator
	limiter *ratelimit.Limiter
	cfg     config.AuthConfig
}

func NewHandler(svc *Service, captcha CaptchaGenerator, limiter *ratelimit.Limiter, cfg config.AuthConfig) *Handler {
	return &Handler{svc: svc, captcha: captcha, limiter: limiter, cfg: cfg}
}

// Register 注册路由（接口清单 #1 ～ #15）。
func (h *Handler) Register(api *gin.RouterGroup) {
	a := api.Group("/auth")
	a.POST("/register", h.limiter.ByIP(ratelimit.PerHour("register", 5)), h.register)
	a.POST("/email/verify", h.limiter.ByIP(ratelimit.PerMinute("email-verify", 20)), h.verifyEmail)
	a.GET("/captcha", h.limiter.ByIP(ratelimit.PerMinute("captcha", 30)), h.getCaptcha)
	a.POST("/login", h.limiter.ByIP(ratelimit.PerMinute("login", 20)), h.login)
	a.POST("/refresh", h.limiter.ByIP(ratelimit.PerMinute("refresh", 60)), h.refresh)
	a.POST("/logout", h.logout)
	a.POST("/password/forgot", h.limiter.ByIP(ratelimit.PerHour("pwd-forgot", 20)), h.forgotPassword)
	a.POST("/password/reset", h.limiter.ByIP(ratelimit.PerHour("pwd-reset", 20)), h.resetPassword)

	authed := api.Group("", h.RequireMerchant())
	authed.POST("/auth/email/resend", h.resendVerification)
	authed.GET("/me", h.me)
	authed.PATCH("/me", h.updateMe)
	authed.PUT("/me/password", h.changePassword)
	authed.GET("/me/sessions", h.listSessions)
	authed.DELETE("/me/sessions/:sessionId", h.revokeSession)
	authed.DELETE("/me/sessions", h.revokeOtherSessions)
}

// RequireMerchant 鉴权中间件：校验 Access Token 并确认会话未被吊销。其他模块的商家接口复用此中间件。
func (h *Handler) RequireMerchant() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			response.Error(c, errcode.Unauthorized)
			return
		}
		id, err := h.svc.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.Error(c, err)
			return
		}
		auth.SetIdentity(c, id)
		c.Next()
	}
}

// ---------- 请求与响应 ----------

type authResponse struct {
	AccessToken string       `json:"accessToken"`
	ExpiresAt   time.Time    `json:"expiresAt"`
	Merchant    MerchantView `json:"merchant"`
}

func (h *Handler) respondAuth(c *gin.Context, r *AuthResult) {
	h.setRefreshCookie(c, r.RefreshToken, r.RefreshTTL)
	response.OK(c, authResponse{AccessToken: r.AccessToken, ExpiresAt: r.ExpiresAt.UTC(), Merchant: r.Merchant.View()})
}

// Refresh Token 只放在 HttpOnly Cookie 中，前端 JS 无法读取；路径限定为 /api/v1/auth，其他接口不会携带它。
func (h *Handler) setRefreshCookie(c *gin.Context, token string, ttl time.Duration) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: h.cfg.RefreshCookie, Value: token, Path: "/api/v1/auth",
		MaxAge: int(ttl.Seconds()), HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: h.cfg.RefreshCookie, Value: "", Path: "/api/v1/auth",
		MaxAge: -1, HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) refreshToken(c *gin.Context) string {
	v, _ := c.Cookie(h.cfg.RefreshCookie)
	return v
}

func client(c *gin.Context) ClientInfo {
	return ClientInfo{UserAgent: c.Request.UserAgent(), IP: c.ClientIP()}
}

func identity(c *gin.Context) auth.Identity {
	id, _ := auth.IdentityFrom(c)
	return id
}

// bind 解析 JSON 请求体，格式错误统一返回参数错误。
func bind(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.Error(c, errcode.InvalidParams.WithMessage("请求格式不正确"))
		return false
	}
	return true
}

// ---------- 接口实现 ----------

func (h *Handler) register(c *gin.Context) {
	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		AgreeTerms bool   `json:"agreeTerms"`
	}
	if !bind(c, &req) {
		return
	}
	r, err := h.svc.Register(c.Request.Context(), RegisterInput{
		Email: req.Email, Password: req.Password, AgreeTerms: req.AgreeTerms, Client: client(c),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	h.respondAuth(c, r)
}

func (h *Handler) login(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		Remember    bool   `json:"remember"`
		CaptchaID   string `json:"captchaId"`
		CaptchaCode string `json:"captchaCode"`
	}
	if !bind(c, &req) {
		return
	}
	r, err := h.svc.Login(c.Request.Context(), LoginInput{
		Email: req.Email, Password: req.Password, Remember: req.Remember,
		CaptchaID: req.CaptchaID, CaptchaCode: req.CaptchaCode, Client: client(c),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	h.respondAuth(c, r)
}

func (h *Handler) getCaptcha(c *gin.Context) {
	id, image, err := h.captcha.Generate()
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"captchaId": id, "image": image})
}

func (h *Handler) refresh(c *gin.Context) {
	r, err := h.svc.Refresh(c.Request.Context(), h.refreshToken(c), client(c))
	if err != nil {
		h.clearRefreshCookie(c)
		response.Error(c, err)
		return
	}
	h.respondAuth(c, r)
}

func (h *Handler) logout(c *gin.Context) {
	if err := h.svc.Logout(c.Request.Context(), h.refreshToken(c)); err != nil {
		response.Error(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.OK(c, nil)
}

func (h *Handler) verifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if !bind(c, &req) {
		return
	}
	already, err := h.svc.VerifyEmail(c.Request.Context(), req.Token)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"alreadyVerified": already})
}

func (h *Handler) resendVerification(c *gin.Context) {
	if err := h.svc.ResendVerification(c.Request.Context(), identity(c).MerchantID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) forgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) resetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		response.Error(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.OK(c, nil)
}

func (h *Handler) me(c *gin.Context) {
	m, err := h.svc.Me(c.Request.Context(), identity(c).MerchantID)
	if err != nil {
		response.Error(c, err)
		return
	}
	// shop 字段在店铺模块完成后填充（接口清单 #10）
	response.OK(c, gin.H{"merchant": m.View(), "shop": nil})
}

func (h *Handler) updateMe(c *gin.Context) {
	var req struct {
		Nickname string `json:"nickname"`
	}
	if !bind(c, &req) {
		return
	}
	if err := h.svc.UpdateNickname(c.Request.Context(), identity(c).MerchantID, strings.TrimSpace(req.Nickname)); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) changePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), identity(c), req.CurrentPassword, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) listSessions(c *gin.Context) {
	list, err := h.svc.Sessions(c.Request.Context(), identity(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"items": list})
}

func (h *Handler) revokeSession(c *gin.Context) {
	if err := h.svc.RevokeSession(c.Request.Context(), identity(c), c.Param("sessionId")); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) revokeOtherSessions(c *gin.Context) {
	if err := h.svc.RevokeOtherSessions(c.Request.Context(), identity(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}
