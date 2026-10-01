// Package auth 提供 Access Token（JWT）签发与校验，以及随机令牌的生成与哈希。
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "mkclou"

// ErrInvalidToken 表示令牌无效或已过期。
var ErrInvalidToken = errors.New("invalid token")

// Claims 是商家 Access Token 的载荷。sid 关联 Redis 中的会话，用于即时吊销。
type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// MerchantID 从 subject 中解析商家 ID。
func (c *Claims) MerchantID() (uint64, error) {
	return strconv.ParseUint(c.Subject, 10, 64)
}

// TokenManager 负责签发和校验 Access Token（HS256）。
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// Issue 签发 Access Token，返回令牌与过期时间。
func (m *TokenManager) Issue(merchantID uint64, sessionID string) (string, time.Time, error) {
	now := m.now()
	exp := now.Add(m.ttl)
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatUint(merchantID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return token, exp, nil
}

// Parse 校验签名、签发者与有效期，返回载荷。
func (m *TokenManager) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), // 防止 alg=none 等算法替换攻击
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || claims.SessionID == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// NewOpaqueToken 生成 32 字节随机令牌（URL 安全的 Base64），用于 Refresh Token、邮件链接等。
func NewOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// NewID 生成 16 字节随机 ID（十六进制），用于会话 ID。
func NewID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken 返回令牌的 SHA-256 十六进制摘要。服务端只保存哈希，数据泄露时令牌本身不可用。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
