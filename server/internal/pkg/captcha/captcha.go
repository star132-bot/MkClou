// Package captcha 提供图形验证码，答案存储在 Redis 中（PRD AUTH-08）。
package captcha

import (
	"context"
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
)

const ttl = 5 * time.Minute

type Service struct {
	c *base64Captcha.Captcha
}

func New(rdb *redis.Client) *Service {
	// 4 位数字字母，去除易混淆字符由驱动默认处理
	driver := base64Captcha.NewDriverString(48, 140, 2, base64Captcha.OptionShowHollowLine, 4,
		"23456789abcdefghjkmnpqrstuvwxyz", nil, nil, nil)
	return &Service{c: base64Captcha.NewCaptcha(driver, &redisStore{rdb: rdb})}
}

// Generate 返回验证码 ID 与 data URI 格式的图片。
func (s *Service) Generate() (id, image string, err error) {
	id, image, _, err = s.c.Generate()
	return id, image, err
}

// Verify 校验答案（不区分大小写），无论对错都会作废该验证码，防止重复尝试。
func (s *Service) Verify(id, answer string) bool {
	if id == "" || answer == "" {
		return false
	}
	return s.c.Verify(id, strings.ToLower(strings.TrimSpace(answer)), true)
}

type redisStore struct {
	rdb *redis.Client
}

func key(id string) string { return "mk:captcha:" + id }

func (s *redisStore) Set(id, value string) error {
	return s.rdb.Set(context.Background(), key(id), value, ttl).Err()
}

func (s *redisStore) Get(id string, clear bool) string {
	ctx := context.Background()
	var (
		v   string
		err error
	)
	if clear {
		v, err = s.rdb.GetDel(ctx, key(id)).Result()
	} else {
		v, err = s.rdb.Get(ctx, key(id)).Result()
	}
	if err != nil {
		return ""
	}
	return v
}

func (s *redisStore) Verify(id, answer string, clear bool) bool {
	v := s.Get(id, clear)
	return v != "" && v == answer
}
