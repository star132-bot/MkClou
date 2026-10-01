package user

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"mkclou/server/internal/pkg/auth"
)

// Redis 会话设计（docs/database-design.md 第 4 节）：
//
//	mk:sess:{mid}:{sid}     Hash   会话信息，TTL = 会话有效期
//	mk:sess:idx:{mid}       Set    该商家的全部会话 ID
//	mk:rt:{hash}            String Refresh Token 哈希 → "mid:sid"
//	mk:rt:used:{hash}       String 已轮换的旧 Token → "mid:sid:轮换时间"，用于检测重放
const reuseGrace = 10 * time.Second // 多标签页并发续期的宽限期

var (
	errRefreshInvalid = errors.New("refresh token invalid")
	errRefreshReused  = errors.New("refresh token reused")
)

type Session struct {
	ID           string
	MerchantID   uint64
	UserAgent    string
	IP           string
	Remember     bool
	CreatedAt    time.Time
	LastActiveAt time.Time
	ExpiresAt    time.Time
}

type sessionStore struct {
	rdb         *redis.Client
	refreshTTL  time.Duration
	rememberTTL time.Duration
	now         func() time.Time
}

func newSessionStore(rdb *redis.Client, refreshTTL, rememberTTL time.Duration) *sessionStore {
	return &sessionStore{rdb: rdb, refreshTTL: refreshTTL, rememberTTL: rememberTTL, now: time.Now}
}

func sessKey(mid uint64, sid string) string { return fmt.Sprintf("mk:sess:%d:%s", mid, sid) }
func idxKey(mid uint64) string              { return fmt.Sprintf("mk:sess:idx:%d", mid) }
func rtKey(hash string) string              { return "mk:rt:" + hash }
func usedKey(hash string) string            { return "mk:rt:used:" + hash }

func (s *sessionStore) ttl(remember bool) time.Duration {
	if remember {
		return s.rememberTTL
	}
	return s.refreshTTL
}

// Create 创建会话并返回 Refresh Token 明文（只在此时出现一次）。
func (s *sessionStore) Create(ctx context.Context, mid uint64, remember bool, ua, ip string) (*Session, string, error) {
	sid, err := auth.NewID()
	if err != nil {
		return nil, "", err
	}
	token, err := auth.NewOpaqueToken()
	if err != nil {
		return nil, "", err
	}
	now := s.now()
	ttl := s.ttl(remember)
	sess := &Session{ID: sid, MerchantID: mid, UserAgent: truncate(ua, 255), IP: ip, Remember: remember,
		CreatedAt: now, LastActiveAt: now, ExpiresAt: now.Add(ttl)}

	hash := auth.HashToken(token)
	_, err = s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.HSet(ctx, sessKey(mid, sid), map[string]any{
			"rt": hash, "ua": sess.UserAgent, "ip": ip, "remember": boolStr(remember),
			"created": now.UnixMilli(), "active": now.UnixMilli(), "expires": sess.ExpiresAt.UnixMilli(),
		})
		p.Expire(ctx, sessKey(mid, sid), ttl)
		p.SAdd(ctx, idxKey(mid), sid)
		p.Set(ctx, rtKey(hash), fmt.Sprintf("%d:%s", mid, sid), ttl)
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return sess, token, nil
}

// Rotate 用 Refresh Token 换取新的 Refresh Token（轮换）。
// GETDEL 保证同一个 Token 只有一个请求能轮换成功；旧 Token 再次出现且超过宽限期，判定为被盗用。
func (s *sessionStore) Rotate(ctx context.Context, token, ua, ip string) (*Session, string, error) {
	hash := auth.HashToken(token)
	owner, err := s.rdb.GetDel(ctx, rtKey(hash)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, "", s.checkReuse(ctx, hash)
	}
	if err != nil {
		return nil, "", err
	}

	mid, sid, ok := parseOwner(owner)
	if !ok {
		return nil, "", errRefreshInvalid
	}
	vals, err := s.rdb.HGetAll(ctx, sessKey(mid, sid)).Result()
	if err != nil {
		return nil, "", err
	}
	if len(vals) == 0 { // 会话已被吊销
		return nil, "", errRefreshInvalid
	}

	newToken, err := auth.NewOpaqueToken()
	if err != nil {
		return nil, "", err
	}
	newHash := auth.HashToken(newToken)
	now := s.now()
	remember := vals["remember"] == "1"
	ttl := s.ttl(remember)

	_, err = s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Set(ctx, usedKey(hash), fmt.Sprintf("%d:%s:%d", mid, sid, now.UnixMilli()), s.rememberTTL)
		p.Set(ctx, rtKey(newHash), owner, ttl)
		p.HSet(ctx, sessKey(mid, sid), map[string]any{
			"rt": newHash, "ua": truncate(ua, 255), "ip": ip, "active": now.UnixMilli(), "expires": now.Add(ttl).UnixMilli(),
		})
		p.Expire(ctx, sessKey(mid, sid), ttl) // 滑动过期：活跃用户不会被强制登出
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	sess := sessionFromHash(mid, sid, vals)
	sess.LastActiveAt, sess.ExpiresAt, sess.UserAgent, sess.IP = now, now.Add(ttl), truncate(ua, 255), ip
	return sess, newToken, nil
}

func (s *sessionStore) checkReuse(ctx context.Context, hash string) error {
	v, err := s.rdb.Get(ctx, usedKey(hash)).Result()
	if errors.Is(err, redis.Nil) {
		return errRefreshInvalid
	}
	if err != nil {
		return err
	}
	parts := strings.Split(v, ":")
	if len(parts) != 3 {
		return errRefreshInvalid
	}
	mid, _ := strconv.ParseUint(parts[0], 10, 64)
	rotatedAt, _ := strconv.ParseInt(parts[2], 10, 64)
	if s.now().Sub(time.UnixMilli(rotatedAt)) <= reuseGrace {
		return errRefreshInvalid // 多标签页并发续期，不视为攻击
	}
	if err := s.RevokeAll(ctx, mid, ""); err != nil {
		return err
	}
	return errRefreshReused
}

// Exists 判断会话是否仍然有效，鉴权中间件在每个请求中调用。
func (s *sessionStore) Exists(ctx context.Context, mid uint64, sid string) (bool, error) {
	n, err := s.rdb.Exists(ctx, sessKey(mid, sid)).Result()
	return n == 1, err
}

// Owner 返回 Refresh Token 所属的会话，用于登出。
func (s *sessionStore) Owner(ctx context.Context, token string) (uint64, string, error) {
	v, err := s.rdb.Get(ctx, rtKey(auth.HashToken(token))).Result()
	if errors.Is(err, redis.Nil) {
		return 0, "", errRefreshInvalid
	}
	if err != nil {
		return 0, "", err
	}
	mid, sid, ok := parseOwner(v)
	if !ok {
		return 0, "", errRefreshInvalid
	}
	return mid, sid, nil
}

// List 返回商家的全部有效会话，按最近活动时间倒序；顺带清理索引中已过期的会话 ID。
func (s *sessionStore) List(ctx context.Context, mid uint64) ([]Session, error) {
	sids, err := s.rdb.SMembers(ctx, idxKey(mid)).Result()
	if err != nil {
		return nil, err
	}
	var out []Session
	for _, sid := range sids {
		vals, err := s.rdb.HGetAll(ctx, sessKey(mid, sid)).Result()
		if err != nil {
			return nil, err
		}
		if len(vals) == 0 {
			s.rdb.SRem(ctx, idxKey(mid), sid)
			continue
		}
		out = append(out, *sessionFromHash(mid, sid, vals))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastActiveAt.After(out[j].LastActiveAt) })
	return out, nil
}

// Revoke 吊销单个会话。
func (s *sessionStore) Revoke(ctx context.Context, mid uint64, sid string) error {
	hash, err := s.rdb.HGet(ctx, sessKey(mid, sid), "rt").Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	_, err = s.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if hash != "" {
			p.Del(ctx, rtKey(hash))
		}
		p.Del(ctx, sessKey(mid, sid))
		p.SRem(ctx, idxKey(mid), sid)
		return nil
	})
	return err
}

// RevokeAll 吊销商家的全部会话，except 不为空时保留该会话（用于“退出其他设备”和修改密码）。
func (s *sessionStore) RevokeAll(ctx context.Context, mid uint64, except string) error {
	sids, err := s.rdb.SMembers(ctx, idxKey(mid)).Result()
	if err != nil {
		return err
	}
	for _, sid := range sids {
		if sid == except {
			continue
		}
		if err := s.Revoke(ctx, mid, sid); err != nil {
			return err
		}
	}
	return nil
}

func sessionFromHash(mid uint64, sid string, v map[string]string) *Session {
	ms := func(k string) time.Time {
		n, _ := strconv.ParseInt(v[k], 10, 64)
		return time.UnixMilli(n)
	}
	return &Session{ID: sid, MerchantID: mid, UserAgent: v["ua"], IP: v["ip"], Remember: v["remember"] == "1",
		CreatedAt: ms("created"), LastActiveAt: ms("active"), ExpiresAt: ms("expires")}
}

func parseOwner(v string) (uint64, string, bool) {
	mid, sid, ok := strings.Cut(v, ":")
	if !ok {
		return 0, "", false
	}
	id, err := strconv.ParseUint(mid, 10, 64)
	return id, sid, err == nil && sid != ""
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
