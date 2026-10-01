// Package health 提供存活与就绪检查接口（api-list #112、#113）。
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb}
}

func (h *Handler) Register(r gin.IRouter) {
	r.GET("/healthz", h.Live)
	r.GET("/readyz", h.Ready)
}

// Live 进程存活即返回 200。
func (h *Handler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready 检查 MySQL 与 Redis 是否可用，任一不可用返回 503。
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{"mysql": "ok", "redis": "ok"}
	healthy := true

	if sqlDB, err := h.db.DB(); err != nil || sqlDB.PingContext(ctx) != nil {
		checks["mysql"] = "unavailable"
		healthy = false
	}
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		checks["redis"] = "unavailable"
		healthy = false
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"status": map[bool]string{true: "ok", false: "degraded"}[healthy], "checks": checks})
}
