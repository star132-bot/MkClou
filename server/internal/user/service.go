package user

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"mkclou/server/internal/pkg/auth"
	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/mailer"
	"mkclou/server/internal/pkg/ratelimit"
)

// 登录保护阈值（PRD AUTH-08）
const (
	captchaAfterFailures = 3
	lockAfterFailures    = 10
	failureWindow        = 15 * time.Minute
	lockDuration         = 15 * time.Minute
)

// 邮件发送频率（PRD AUTH-02、SEC-01）
var (
	mailCooldown = ratelimit.PerMinute("mail-cooldown", 1)
	mailDaily    = ratelimit.PerDay("mail-daily", 10)
)

// CaptchaVerifier 校验图形验证码。
type CaptchaVerifier interface {
	Verify(id, answer string) bool
}

type Service struct {
	repo     Repository
	sessions *sessionStore
	tokens   *auth.TokenManager
	rdb      *redis.Client
	limiter  *ratelimit.Limiter
	captcha  CaptchaVerifier
	mail     mailer.Queue
	cfg      *config.Config
	log      *zap.Logger
	now      func() time.Time
	// dummyHash 用于邮箱不存在时也执行一次 bcrypt 比较，使响应时间一致，防止通过耗时判断邮箱是否注册
	dummyHash []byte
}

type Deps struct {
	Repo    Repository
	Redis   *redis.Client
	Limiter *ratelimit.Limiter
	Captcha CaptchaVerifier
	Mail    mailer.Queue
	Config  *config.Config
	Log     *zap.Logger
}

func NewService(d Deps) *Service {
	cost := d.Config.Auth.BcryptCost
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), cost)
	return &Service{
		repo:      d.Repo,
		sessions:  newSessionStore(d.Redis, d.Config.Auth.RefreshTTL, d.Config.Auth.RememberTTL),
		tokens:    auth.NewTokenManager(d.Config.Auth.JWTSecret, d.Config.Auth.AccessTTL),
		rdb:       d.Redis,
		limiter:   d.Limiter,
		captcha:   d.Captcha,
		mail:      d.Mail,
		cfg:       d.Config,
		log:       d.Log,
		now:       time.Now,
		dummyHash: dummy,
	}
}

// AuthResult 是登录、注册、续期的结果。RefreshToken 由 handler 写入 HttpOnly Cookie，不出现在响应体中。
type AuthResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	RefreshTTL   time.Duration
	Merchant     *Merchant
}

type ClientInfo struct {
	UserAgent string
	IP        string
}

// ---------- 注册与登录 ----------

type RegisterInput struct {
	Email      string
	Password   string
	AgreeTerms bool
	Client     ClientInfo
}

// Register 注册并自动登录，异步发送验证邮件（PRD AUTH-01）。
func (s *Service) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	email := NormalizeEmail(in.Email)
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(in.Password, email); err != nil {
		return nil, err
	}
	if !in.AgreeTerms {
		return nil, fieldError("agreeTerms", "请先阅读并同意用户协议和隐私政策")
	}

	hash, err := s.hashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	m := &Merchant{Email: email, PasswordHash: hash, Status: StatusActive}
	if err := s.repo.Create(ctx, m); err != nil {
		if errors.Is(err, errDuplicate) {
			return nil, errcode.EmailRegistered
		}
		return nil, err
	}

	// 验证邮件发送失败不影响注册结果，用户可在后台横幅中重新发送
	if err := s.sendVerification(ctx, m); err != nil {
		s.log.Warn("send verification email", zap.Uint64("merchantId", m.ID), zap.Error(err))
	}
	return s.startSession(ctx, m, false, in.Client)
}

type LoginInput struct {
	Email       string
	Password    string
	Remember    bool
	CaptchaID   string
	CaptchaCode string
	Client      ClientInfo
}

func failKey(email string) string { return "mk:login:fail:" + email }
func lockKey(email string) string { return "mk:login:lock:" + email }

// Login 登录（PRD AUTH-03、AUTH-08）：失败 3 次要求图形验证码，10 次锁定 15 分钟。
func (s *Service) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	email := NormalizeEmail(in.Email)
	if email == "" || in.Password == "" {
		return nil, errcode.BadCredentials
	}

	if ttl, err := s.rdb.TTL(ctx, lockKey(email)).Result(); err == nil && ttl > 0 {
		return nil, lockedError(ttl)
	}

	failures, _ := s.rdb.Get(ctx, failKey(email)).Int()
	if failures >= captchaAfterFailures {
		if in.CaptchaID == "" {
			return nil, errcode.CaptchaRequired
		}
		if !s.captcha.Verify(in.CaptchaID, in.CaptchaCode) {
			return nil, errcode.CaptchaRequired.WithMessage("图形验证码错误，请重新输入")
		}
	}

	m, err := s.repo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, errNotFound) {
		return nil, err
	}
	if m == nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(in.Password))
		return nil, s.recordFailure(ctx, email, nil)
	}
	if bcrypt.CompareHashAndPassword([]byte(m.PasswordHash), []byte(in.Password)) != nil {
		return nil, s.recordFailure(ctx, email, m)
	}
	if m.Status == StatusBanned {
		return nil, disabledError(m)
	}

	s.rdb.Del(ctx, failKey(email))
	if err := s.repo.UpdateLastLogin(ctx, m.ID, s.now()); err != nil {
		s.log.Warn("update last login", zap.Error(err))
	}
	return s.startSession(ctx, m, in.Remember, in.Client)
}

// recordFailure 记录一次失败并返回对应错误。邮箱不存在时同样计数，避免通过行为差异判断邮箱是否注册。
func (s *Service) recordFailure(ctx context.Context, email string, m *Merchant) error {
	n, err := s.rdb.Incr(ctx, failKey(email)).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		s.rdb.Expire(ctx, failKey(email), failureWindow)
	}

	if n >= lockAfterFailures {
		s.rdb.Set(ctx, lockKey(email), "1", lockDuration)
		s.rdb.Del(ctx, failKey(email))
		if m != nil {
			if msg, err := mailer.AccountLocked(m.Email, s.link("/forgot-password", nil)); err == nil {
				if err := s.mail.Enqueue(ctx, msg); err != nil {
					s.log.Warn("enqueue lock alert", zap.Error(err))
				}
			}
		}
		return lockedError(lockDuration)
	}
	if n >= captchaAfterFailures {
		return errcode.BadCredentials.WithData(map[string]any{"captchaRequired": true})
	}
	return errcode.BadCredentials
}

func lockedError(ttl time.Duration) *errcode.Error {
	min := int(ttl.Minutes()) + 1
	return errcode.AccountLocked.
		WithMessage(fmt.Sprintf("登录尝试过多，请 %d 分钟后再试，或找回密码", min)).
		WithData(map[string]int{"retryAfter": int(ttl.Seconds())})
}

func disabledError(m *Merchant) *errcode.Error {
	msg := "账号已被停用"
	if m.BanReason != nil && *m.BanReason != "" {
		msg += "，原因：" + *m.BanReason
	}
	return errcode.AccountDisabled.WithMessage(msg)
}

func (s *Service) startSession(ctx context.Context, m *Merchant, remember bool, c ClientInfo) (*AuthResult, error) {
	sess, refresh, err := s.sessions.Create(ctx, m.ID, remember, c.UserAgent, c.IP)
	if err != nil {
		return nil, err
	}
	access, exp, err := s.tokens.Issue(m.ID, sess.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{AccessToken: access, ExpiresAt: exp, RefreshToken: refresh,
		RefreshTTL: sess.ExpiresAt.Sub(s.now()), Merchant: m}, nil
}

// ---------- 会话 ----------

// Refresh 轮换 Refresh Token 并签发新的 Access Token（PRD AUTH-06）。
func (s *Service) Refresh(ctx context.Context, refreshToken string, c ClientInfo) (*AuthResult, error) {
	if refreshToken == "" {
		return nil, errcode.Unauthorized
	}
	sess, newRefresh, err := s.sessions.Rotate(ctx, refreshToken, c.UserAgent, c.IP)
	switch {
	case errors.Is(err, errRefreshReused):
		s.log.Warn("refresh token reuse detected, all sessions revoked", zap.String("ip", c.IP))
		return nil, errcode.Unauthorized.WithMessage("登录状态异常，请重新登录")
	case errors.Is(err, errRefreshInvalid):
		return nil, errcode.Unauthorized
	case err != nil:
		return nil, err
	}

	m, err := s.repo.FindByID(ctx, sess.MerchantID)
	if err != nil || m.Status == StatusBanned {
		_ = s.sessions.Revoke(ctx, sess.MerchantID, sess.ID)
		if m != nil {
			return nil, disabledError(m)
		}
		return nil, errcode.Unauthorized
	}

	access, exp, err := s.tokens.Issue(m.ID, sess.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{AccessToken: access, ExpiresAt: exp, RefreshToken: newRefresh,
		RefreshTTL: sess.ExpiresAt.Sub(s.now()), Merchant: m}, nil
}

// Logout 吊销 Refresh Token 所属的会话。Token 无效时视为已登出（幂等）。
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	mid, sid, err := s.sessions.Owner(ctx, refreshToken)
	if errors.Is(err, errRefreshInvalid) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.sessions.Revoke(ctx, mid, sid)
}

// Authenticate 校验 Access Token 且会话仍然存在，供鉴权中间件使用。
func (s *Service) Authenticate(ctx context.Context, accessToken string) (auth.Identity, error) {
	claims, err := s.tokens.Parse(accessToken)
	if err != nil {
		return auth.Identity{}, errcode.Unauthorized
	}
	mid, err := claims.MerchantID()
	if err != nil {
		return auth.Identity{}, errcode.Unauthorized
	}
	ok, err := s.sessions.Exists(ctx, mid, claims.SessionID)
	if err != nil {
		return auth.Identity{}, err
	}
	if !ok { // 会话已被吊销：登出、改密码、退出其他设备、封禁
		return auth.Identity{}, errcode.Unauthorized
	}
	return auth.Identity{MerchantID: mid, SessionID: claims.SessionID}, nil
}

type SessionView struct {
	ID           string    `json:"id"`
	UserAgent    string    `json:"userAgent"`
	IP           string    `json:"ip"`
	LastActiveAt time.Time `json:"lastActiveAt"`
	CreatedAt    time.Time `json:"createdAt"`
	Current      bool      `json:"current"`
}

func (s *Service) Sessions(ctx context.Context, id auth.Identity) ([]SessionView, error) {
	list, err := s.sessions.List(ctx, id.MerchantID)
	if err != nil {
		return nil, err
	}
	out := make([]SessionView, 0, len(list))
	for _, x := range list {
		out = append(out, SessionView{ID: x.ID, UserAgent: x.UserAgent, IP: x.IP,
			LastActiveAt: x.LastActiveAt.UTC(), CreatedAt: x.CreatedAt.UTC(), Current: x.ID == id.SessionID})
	}
	return out, nil
}

// RevokeSession 退出指定设备。不允许通过此接口退出当前设备（应使用登出）。
func (s *Service) RevokeSession(ctx context.Context, id auth.Identity, sid string) error {
	if sid == id.SessionID {
		return errcode.StateConflict.WithMessage("不能在这里退出当前设备，请使用“退出登录”")
	}
	ok, err := s.sessions.Exists(ctx, id.MerchantID, sid)
	if err != nil {
		return err
	}
	if !ok {
		return errcode.NotFound
	}
	return s.sessions.Revoke(ctx, id.MerchantID, sid)
}

func (s *Service) RevokeOtherSessions(ctx context.Context, id auth.Identity) error {
	return s.sessions.RevokeAll(ctx, id.MerchantID, id.SessionID)
}

// ---------- 账号信息 ----------

func (s *Service) Me(ctx context.Context, mid uint64) (*Merchant, error) {
	m, err := s.repo.FindByID(ctx, mid)
	if errors.Is(err, errNotFound) {
		return nil, errcode.Unauthorized
	}
	return m, err
}

func (s *Service) UpdateNickname(ctx context.Context, mid uint64, nickname string) error {
	if err := validateNickname(nickname); err != nil {
		return err
	}
	return s.repo.UpdateNickname(ctx, mid, nickname)
}

// ChangePassword 修改密码后其他设备全部下线（PRD AUTH-07）。
func (s *Service) ChangePassword(ctx context.Context, id auth.Identity, current, next string) error {
	m, err := s.Me(ctx, id.MerchantID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(m.PasswordHash), []byte(current)) != nil {
		return fieldError("currentPassword", "当前密码不正确")
	}
	if err := validatePassword(next, m.Email); err != nil {
		return err
	}
	if current == next {
		return fieldError("newPassword", "新密码不能与当前密码相同")
	}
	hash, err := s.hashPassword(next)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, m.ID, hash); err != nil {
		return err
	}
	return s.sessions.RevokeAll(ctx, m.ID, id.SessionID)
}

// ---------- 邮箱验证 ----------

func verifyKey(hash string) string { return "mk:email:verify:" + hash }
func resetKey(hash string) string  { return "mk:pwd:reset:" + hash }

// ResendVerification 重新发送验证邮件：60 秒冷却、每天最多 10 次（PRD 01 4.3）。
func (s *Service) ResendVerification(ctx context.Context, mid uint64) error {
	m, err := s.Me(ctx, mid)
	if err != nil {
		return err
	}
	if m.EmailVerified() {
		return errcode.StateConflict.WithMessage("邮箱已验证")
	}
	if err := s.checkMailQuota(ctx, "verify", m.Email); err != nil {
		return err
	}
	return s.sendVerification(ctx, m)
}

func (s *Service) sendVerification(ctx context.Context, m *Merchant) error {
	token, err := auth.NewOpaqueToken()
	if err != nil {
		return err
	}
	if err := s.rdb.Set(ctx, verifyKey(auth.HashToken(token)), m.ID, s.cfg.Auth.VerifyEmailTTL).Err(); err != nil {
		return err
	}
	msg, err := mailer.VerifyEmail(m.Email, s.link("/verify-email", url.Values{"token": {token}}))
	if err != nil {
		return err
	}
	return s.mail.Enqueue(ctx, msg)
}

// VerifyEmail 使用邮件中的令牌完成验证。令牌一次性；返回 alreadyVerified 区分重复点击。
func (s *Service) VerifyEmail(ctx context.Context, token string) (alreadyVerified bool, err error) {
	if token == "" {
		return false, errcode.TokenInvalid
	}
	v, err := s.rdb.GetDel(ctx, verifyKey(auth.HashToken(token))).Result()
	if errors.Is(err, redis.Nil) {
		return false, errcode.TokenInvalid.WithMessage("验证链接已失效，请重新发送验证邮件")
	}
	if err != nil {
		return false, err
	}
	mid, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return false, errcode.TokenInvalid
	}
	updated, err := s.repo.MarkEmailVerified(ctx, mid, s.now())
	if err != nil {
		return false, err
	}
	return !updated, nil
}

// ---------- 找回密码 ----------

// ForgotPassword 无论邮箱是否存在都返回成功，防止邮箱枚举（PRD SEC-03）。
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = NormalizeEmail(email)
	if err := validateEmail(email); err != nil {
		return err
	}
	// 频率限制按邮箱计算，与邮箱是否存在无关，因此不会泄露注册信息
	if err := s.checkMailQuota(ctx, "reset", email); err != nil {
		return err
	}

	m, err := s.repo.FindByEmail(ctx, email)
	if errors.Is(err, errNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	token, err := auth.NewOpaqueToken()
	if err != nil {
		return err
	}
	if err := s.rdb.Set(ctx, resetKey(auth.HashToken(token)), m.ID, s.cfg.Auth.ResetPasswordTTL).Err(); err != nil {
		return err
	}
	msg, err := mailer.ResetPassword(m.Email, s.link("/reset-password", url.Values{"token": {token}}))
	if err != nil {
		return err
	}
	return s.mail.Enqueue(ctx, msg)
}

// ResetPassword 设置新密码：令牌一次性，成功后全部设备下线并解除登录锁定（PRD AUTH-05）。
func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if token == "" {
		return errcode.TokenInvalid
	}
	key := resetKey(auth.HashToken(token))
	v, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return errcode.TokenInvalid.WithMessage("重置链接已失效，请重新获取")
	}
	if err != nil {
		return err
	}
	mid, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return errcode.TokenInvalid
	}
	m, err := s.repo.FindByID(ctx, mid)
	if err != nil {
		return errcode.TokenInvalid
	}
	if err := validatePassword(password, m.Email); err != nil {
		return err // 密码不合规时保留令牌，用户可修改后重试
	}

	// 先删除令牌再改密码：并发提交时只有一个请求能继续
	if n, err := s.rdb.Del(ctx, key).Result(); err != nil {
		return err
	} else if n == 0 {
		return errcode.TokenInvalid.WithMessage("重置链接已失效，请重新获取")
	}

	hash, err := s.hashPassword(password)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, m.ID, hash); err != nil {
		return err
	}
	s.rdb.Del(ctx, lockKey(m.Email), failKey(m.Email))
	return s.sessions.RevokeAll(ctx, m.ID, "")
}

// ---------- 工具 ----------

func (s *Service) checkMailQuota(ctx context.Context, kind, email string) error {
	key := kind + ":" + email
	if err := s.limiter.Check(ctx, mailCooldown, key); err != nil {
		return err
	}
	return s.limiter.Check(ctx, mailDaily, key)
}

func (s *Service) hashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.Auth.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

func (s *Service) link(path string, q url.Values) string {
	u := s.cfg.App.PublicURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}
