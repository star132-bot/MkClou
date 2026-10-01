// Package ratelimit 基于 Redis 的限流（GCRA 算法，效果等同令牌桶），见 PRD SEC-01。
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"

	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/response"
)

// Rule 描述一条限流规则：Window 时间内最多 Limit 次。
type Rule struct {
	Name   string
	Limit  int
	Window time.Duration
}

func PerMinute(name string, n int) Rule { return Rule{Name: name, Limit: n, Window: time.Minute} }
func PerHour(name string, n int) Rule   { return Rule{Name: name, Limit: n, Window: time.Hour} }
func PerDay(name string, n int) Rule    { return Rule{Name: name, Limit: n, Window: 24 * time.Hour} }

type Limiter struct {
	l *redis_rate.Limiter
}

func New(rdb *redis.Client) *Limiter {
	return &Limiter{l: redis_rate.NewLimiter(rdb)}
}

// Allow 为 key 消耗一次额度。不允许时返回需要等待的时长。
func (l *Limiter) Allow(ctx context.Context, rule Rule, key string) (bool, time.Duration, error) {
	res, err := l.l.Allow(ctx, fmt.Sprintf("mk:rl:%s:%s", rule.Name, key), redis_rate.Limit{
		Rate:   rule.Limit,
		Burst:  rule.Limit,
		Period: rule.Window,
	})
	if err != nil {
		return false, 0, err
	}
	return res.Allowed > 0, res.RetryAfter, nil
}

// Check 是业务层使用的便捷方法：超限时返回带 retryAfter 的 429 业务错误。
func (l *Limiter) Check(ctx context.Context, rule Rule, key string) error {
	ok, retry, err := l.Allow(ctx, rule, key)
	if err != nil {
		return err
	}
	if !ok {
		return TooMany(retry)
	}
	return nil
}

// TooMany 构造包含重试秒数的限流错误。
func TooMany(retry time.Duration) *errcode.Error {
	sec := int(math.Ceil(retry.Seconds()))
	if sec < 1 {
		sec = 1
	}
	return errcode.TooManyRequests.
		WithMessage(fmt.Sprintf("操作太频繁，请 %d 秒后再试", sec)).
		WithData(map[string]int{"retryAfter": sec})
}

// ByIP 返回按客户端 IP 限流的中间件。Redis 不可用时放行（限流不应导致整站不可用），由调用方日志监控。
func (l *Limiter) ByIP(rule Rule) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retry, err := l.Allow(c.Request.Context(), rule, c.ClientIP())
		if err != nil {
			c.Next()
			return
		}
		if !ok {
			response.Error(c, TooMany(retry)) // response.Error 会写入 Retry-After 响应头
			return
		}
		c.Next()
	}
}
