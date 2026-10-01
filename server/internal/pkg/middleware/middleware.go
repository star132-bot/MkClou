// Package middleware 提供全局 HTTP 中间件。
package middleware

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/response"
)

const (
	headerRequestID = "X-Request-ID"
	loggerKey       = "logger"
)

// Logger 取出带 traceId 的请求级 logger；未经过 RequestID 中间件时返回 fallback。
func Logger(c *gin.Context, fallback *zap.Logger) *zap.Logger {
	if l, ok := c.Get(loggerKey); ok {
		return l.(*zap.Logger)
	}
	return fallback
}

// RequestID 为每个请求生成 traceId（或沿用上游传入的值），写入响应头，并派生请求级 logger。
func RequestID(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(headerRequestID)
		if id == "" || len(id) > 64 {
			id = strings.ReplaceAll(uuid.NewString(), "-", "")
		}
		c.Set(response.TraceIDKey, id)
		c.Set(loggerKey, base.With(zap.String("traceId", id)))
		c.Header(headerRequestID, id)
		c.Next()
	}
}

// AccessLog 记录每个请求的方法、路径、状态码和耗时。只记录路由模板，不记录查询参数，避免令牌等敏感信息进入日志。
func AccessLog(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "(no route)"
		}
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("route", route),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		}
		l := Logger(c, base)
		switch {
		case c.Writer.Status() >= http.StatusInternalServerError:
			l.Error("request", fields...)
		case c.Writer.Status() >= http.StatusBadRequest:
			l.Warn("request", fields...)
		default:
			l.Info("request", fields...)
		}
	}
}

// Recovery 捕获 panic，记录堆栈并返回统一的 500 响应。
func Recovery(base *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		Logger(c, base).Error("panic recovered", zap.Any("panic", recovered), zap.Stack("stack"))
		response.Error(c, errcode.Internal)
	})
}

// CORS 只允许配置中的来源跨域访问，并允许携带 Cookie（Refresh Token、买家会话）。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && slices.Contains(allowedOrigins, origin) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Order-Token, X-Request-ID")
			h.Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After")
			h.Set("Access-Control-Max-Age", "600")
			h.Add("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// NoRoute 统一处理未匹配的路由。
func NoRoute(c *gin.Context) {
	response.Error(c, errcode.NotFound)
}
