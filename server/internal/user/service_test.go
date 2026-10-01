package user

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"mkclou/server/internal/pkg/auth"
	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/mailer"
	"mkclou/server/internal/pkg/ratelimit"
)

// ---------- 测试替身 ----------

type fakeRepo struct {
	mu     sync.Mutex
	nextID uint64
	byID   map[uint64]*Merchant
}

func newFakeRepo() *fakeRepo { return &fakeRepo{byID: map[uint64]*Merchant{}} }

func (r *fakeRepo) Create(_ context.Context, m *Merchant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.byID {
		if x.Email == m.Email {
			return errDuplicate
		}
	}
	r.nextID++
	m.ID = r.nextID
	m.CreatedAt = time.Now()
	cp := *m
	r.byID[m.ID] = &cp
	return nil
}

func (r *fakeRepo) FindByID(_ context.Context, id uint64) (*Merchant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.byID[id]
	if !ok {
		return nil, errNotFound
	}
	cp := *m
	return &cp, nil
}

func (r *fakeRepo) FindByEmail(_ context.Context, email string) (*Merchant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.byID {
		if m.Email == email {
			cp := *m
			return &cp, nil
		}
	}
	return nil, errNotFound
}

func (r *fakeRepo) UpdatePassword(_ context.Context, id uint64, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[id].PasswordHash = hash
	return nil
}

func (r *fakeRepo) MarkEmailVerified(_ context.Context, id uint64, at time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byID[id].EmailVerifiedAt != nil {
		return false, nil
	}
	r.byID[id].EmailVerifiedAt = &at
	return true, nil
}

func (r *fakeRepo) UpdateLastLogin(_ context.Context, id uint64, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[id].LastLoginAt = &at
	return nil
}

func (r *fakeRepo) UpdateNickname(_ context.Context, id uint64, n string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[id].Nickname = n
	return nil
}

type fakeMail struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (f *fakeMail) Enqueue(_ context.Context, m mailer.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
}

func (f *fakeMail) last(t *testing.T, template string) mailer.Message {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.sent) - 1; i >= 0; i-- {
		if f.sent[i].Template == template {
			return f.sent[i]
		}
	}
	t.Fatalf("no %s email sent", template)
	return mailer.Message{}
}

func (f *fakeMail) count(template string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, m := range f.sent {
		if m.Template == template {
			n++
		}
	}
	return n
}

type fakeCaptcha struct{ answer string }

func (f fakeCaptcha) Verify(_, answer string) bool { return answer == f.answer }

// ---------- 测试环境 ----------

type env struct {
	svc  *Service
	repo *fakeRepo
	mail *fakeMail
	mr   *miniredis.Miniredis
	ctx  context.Context
}

const (
	testEmail    = "seller@example.com"
	testPassword = "Passw0rd!"
	testClient   = "Mozilla/5.0 test"
)

func setup(t *testing.T) *env {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg := &config.Config{
		App: config.AppConfig{PublicURL: "http://localhost:3000"},
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-at-least-32-characters-long", AccessTTL: 15 * time.Minute,
			RefreshTTL: 7 * 24 * time.Hour, RememberTTL: 30 * 24 * time.Hour,
			BcryptCost: bcrypt.MinCost, VerifyEmailTTL: 24 * time.Hour, ResetPasswordTTL: 30 * time.Minute,
		},
	}
	repo, mail := newFakeRepo(), &fakeMail{}
	svc := NewService(Deps{
		Repo: repo, Redis: rdb, Limiter: ratelimit.New(rdb), Captcha: fakeCaptcha{answer: "ok"},
		Mail: mail, Config: cfg, Log: zap.NewNop(),
	})
	return &env{svc: svc, repo: repo, mail: mail, mr: mr, ctx: context.Background()}
}

func (e *env) register(t *testing.T) *AuthResult {
	t.Helper()
	r, err := e.svc.Register(e.ctx, RegisterInput{Email: testEmail, Password: testPassword, AgreeTerms: true,
		Client: ClientInfo{UserAgent: testClient, IP: "1.1.1.1"}})
	require.NoError(t, err)
	return r
}

func (e *env) login(password string, extra ...func(*LoginInput)) (*AuthResult, error) {
	in := LoginInput{Email: testEmail, Password: password, Client: ClientInfo{UserAgent: testClient, IP: "1.1.1.1"}}
	for _, f := range extra {
		f(&in)
	}
	return e.svc.Login(e.ctx, in)
}

// tokenFromLink 从邮件正文中取出链接里的 token 参数。
func tokenFromLink(t *testing.T, msg mailer.Message) string {
	t.Helper()
	i := strings.Index(msg.Text, "http://")
	require.GreaterOrEqual(t, i, 0)
	u, err := url.Parse(strings.Fields(msg.Text[i:])[0])
	require.NoError(t, err)
	return u.Query().Get("token")
}

func assertCode(t *testing.T, err error, want *errcode.Error) {
	t.Helper()
	var e *errcode.Error
	require.ErrorAs(t, err, &e)
	assert.Equal(t, want.Code, e.Code, e.Message)
}

// ---------- 注册 ----------

func TestRegister_Success(t *testing.T) {
	e := setup(t)
	r, err := e.svc.Register(e.ctx, RegisterInput{Email: "  Seller@Example.COM ", Password: testPassword, AgreeTerms: true})
	require.NoError(t, err)

	assert.Equal(t, testEmail, r.Merchant.Email, "邮箱应去空格并转小写")
	assert.NotEmpty(t, r.AccessToken)
	assert.NotEmpty(t, r.RefreshToken)
	assert.False(t, r.Merchant.EmailVerified())

	stored, _ := e.repo.FindByEmail(e.ctx, testEmail)
	assert.NotEqual(t, testPassword, stored.PasswordHash, "密码不能明文存储")
	assert.Equal(t, 1, e.mail.count("VERIFY_EMAIL"))
}

func TestRegister_Validation(t *testing.T) {
	cases := []struct {
		name, email, password string
		agree                 bool
		field                 string
	}{
		{"邮箱格式错误", "not-an-email", testPassword, true, "email"},
		{"邮箱缺少域名后缀", "a@localhost", testPassword, true, "email"},
		{"临时邮箱", "a@mailinator.com", testPassword, true, "email"},
		{"密码太短", testEmail, "a1b2", true, "password"},
		{"密码没有数字", testEmail, "abcdefgh", true, "password"},
		{"密码没有字母", testEmail, "12345678", true, "password"},
		{"未同意协议", testEmail, testPassword, false, "agreeTerms"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := setup(t)
			_, err := e.svc.Register(e.ctx, RegisterInput{Email: c.email, Password: c.password, AgreeTerms: c.agree})
			var ce *errcode.Error
			require.ErrorAs(t, err, &ce)
			assert.Equal(t, errcode.InvalidParams.Code, ce.Code)
			assert.Contains(t, ce.Data.(map[string]any)["fields"], c.field)
		})
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	e := setup(t)
	e.register(t)
	_, err := e.svc.Register(e.ctx, RegisterInput{Email: "SELLER@example.com", Password: testPassword, AgreeTerms: true})
	assertCode(t, err, errcode.EmailRegistered)
}

// ---------- 登录与保护 ----------

func TestLogin_Success(t *testing.T) {
	e := setup(t)
	e.register(t)
	r, err := e.login(testPassword)
	require.NoError(t, err)

	id, err := e.svc.Authenticate(e.ctx, r.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, r.Merchant.ID, id.MerchantID)
}

func TestLogin_UnknownEmailAndWrongPasswordLookTheSame(t *testing.T) {
	e := setup(t)
	e.register(t)

	_, errWrong := e.login("Wrong-pass1")
	_, errUnknown := e.svc.Login(e.ctx, LoginInput{Email: "nobody@example.com", Password: "Wrong-pass1"})
	assertCode(t, errWrong, errcode.BadCredentials)
	assertCode(t, errUnknown, errcode.BadCredentials)
	assert.Equal(t, errWrong.Error(), errUnknown.Error(), "不能区分邮箱不存在和密码错误")
}

func TestLogin_CaptchaAfterThreeFailures(t *testing.T) {
	e := setup(t)
	e.register(t)
	for i := 0; i < 3; i++ {
		_, _ = e.login("Wrong-pass1")
	}

	_, err := e.login(testPassword)
	assertCode(t, err, errcode.CaptchaRequired)

	_, err = e.login(testPassword, func(in *LoginInput) { in.CaptchaID, in.CaptchaCode = "c1", "bad" })
	assertCode(t, err, errcode.CaptchaRequired)

	_, err = e.login(testPassword, func(in *LoginInput) { in.CaptchaID, in.CaptchaCode = "c1", "ok" })
	require.NoError(t, err)

	_, err = e.login(testPassword)
	require.NoError(t, err, "登录成功后失败计数清零，不再需要验证码")
}

func TestLogin_LockAfterTenFailures(t *testing.T) {
	e := setup(t)
	e.register(t)
	withCaptcha := func(in *LoginInput) { in.CaptchaID, in.CaptchaCode = "c", "ok" }

	var err error
	for i := 0; i < 10; i++ {
		_, err = e.login("Wrong-pass1", withCaptcha)
	}
	assertCode(t, err, errcode.AccountLocked)
	assert.Equal(t, 1, e.mail.count("ACCOUNT_LOCKED"), "锁定时发送安全提醒")

	_, err = e.login(testPassword, withCaptcha)
	assertCode(t, err, errcode.AccountLocked)

	e.mr.FastForward(lockDuration + time.Second)
	_, err = e.login(testPassword)
	require.NoError(t, err, "锁定 15 分钟后自动解除")
}

func TestLogin_BannedAccount(t *testing.T) {
	e := setup(t)
	r := e.register(t)
	reason := "售卖违规商品"
	e.repo.byID[r.Merchant.ID].Status = StatusBanned
	e.repo.byID[r.Merchant.ID].BanReason = &reason

	_, err := e.login(testPassword)
	assertCode(t, err, errcode.AccountDisabled)
	assert.Contains(t, err.Error(), reason)
}

// ---------- 续期、登出、会话 ----------

func TestRefresh_RotatesToken(t *testing.T) {
	e := setup(t)
	r := e.register(t)

	r2, err := e.svc.Refresh(e.ctx, r.RefreshToken, ClientInfo{})
	require.NoError(t, err)
	assert.NotEqual(t, r.RefreshToken, r2.RefreshToken)
	_, err = e.svc.Authenticate(e.ctx, r2.AccessToken)
	require.NoError(t, err)

	r3, err := e.svc.Refresh(e.ctx, r2.RefreshToken, ClientInfo{})
	require.NoError(t, err, "新 Token 可以继续续期")
	assert.NotEqual(t, r2.RefreshToken, r3.RefreshToken)
}

func TestRefresh_ReuseWithinGraceIsRejectedWithoutRevoking(t *testing.T) {
	e := setup(t)
	r := e.register(t)
	r2, err := e.svc.Refresh(e.ctx, r.RefreshToken, ClientInfo{})
	require.NoError(t, err)

	// 模拟另一个标签页几乎同时用旧 Token 续期
	_, err = e.svc.Refresh(e.ctx, r.RefreshToken, ClientInfo{})
	assertCode(t, err, errcode.Unauthorized)

	_, err = e.svc.Refresh(e.ctx, r2.RefreshToken, ClientInfo{})
	require.NoError(t, err, "宽限期内的重复使用不应吊销会话")
}

func TestRefresh_ReuseAfterGraceRevokesAllSessions(t *testing.T) {
	e := setup(t)
	r := e.register(t)
	other, err := e.login(testPassword)
	require.NoError(t, err)

	r2, err := e.svc.Refresh(e.ctx, r.RefreshToken, ClientInfo{})
	require.NoError(t, err)

	// 攻击者在宽限期之后使用被盗的旧 Token
	e.svc.sessions.now = func() time.Time { return time.Now().Add(reuseGrace + time.Second) }
	_, err = e.svc.Refresh(e.ctx, r.RefreshToken, ClientInfo{})
	assertCode(t, err, errcode.Unauthorized)

	_, err = e.svc.Refresh(e.ctx, r2.RefreshToken, ClientInfo{})
	assert.Error(t, err, "检测到重放后，合法用户的新 Token 也失效")
	_, err = e.svc.Authenticate(e.ctx, other.AccessToken)
	assert.Error(t, err, "该账号的其他会话全部下线")
}

func TestLogout_InvalidatesSessionImmediately(t *testing.T) {
	e := setup(t)
	r := e.register(t)
	require.NoError(t, e.svc.Logout(e.ctx, r.RefreshToken))

	_, err := e.svc.Authenticate(e.ctx, r.AccessToken)
	assertCode(t, err, errcode.Unauthorized)
	_, err = e.svc.Refresh(e.ctx, r.RefreshToken, ClientInfo{})
	assertCode(t, err, errcode.Unauthorized)
	require.NoError(t, e.svc.Logout(e.ctx, r.RefreshToken), "重复登出是幂等的")
}

func TestSessions_ListAndRevokeOthers(t *testing.T) {
	e := setup(t)
	a := e.register(t)
	b, err := e.login(testPassword)
	require.NoError(t, err)

	idA, err := e.svc.Authenticate(e.ctx, a.AccessToken)
	require.NoError(t, err)
	list, err := e.svc.Sessions(e.ctx, idA)
	require.NoError(t, err)
	require.Len(t, list, 2)
	currents := 0
	for _, s := range list {
		if s.Current {
			currents++
			assert.Equal(t, idA.SessionID, s.ID)
		}
	}
	assert.Equal(t, 1, currents)

	assertCode(t, e.svc.RevokeSession(e.ctx, idA, idA.SessionID), errcode.StateConflict)

	require.NoError(t, e.svc.RevokeOtherSessions(e.ctx, idA))
	_, err = e.svc.Authenticate(e.ctx, b.AccessToken)
	assert.Error(t, err, "其他设备已下线")
	_, err = e.svc.Authenticate(e.ctx, a.AccessToken)
	assert.NoError(t, err, "当前设备保留")
}

// ---------- 邮箱验证 ----------

func TestVerifyEmail(t *testing.T) {
	e := setup(t)
	r := e.register(t)
	token := tokenFromLink(t, e.mail.last(t, "VERIFY_EMAIL"))

	already, err := e.svc.VerifyEmail(e.ctx, token)
	require.NoError(t, err)
	assert.False(t, already)
	m, _ := e.repo.FindByID(e.ctx, r.Merchant.ID)
	assert.True(t, m.EmailVerified())

	_, err = e.svc.VerifyEmail(e.ctx, token)
	assertCode(t, err, errcode.TokenInvalid)
}

func TestVerifyEmail_ExpiredLink(t *testing.T) {
	e := setup(t)
	e.register(t)
	token := tokenFromLink(t, e.mail.last(t, "VERIFY_EMAIL"))
	e.mr.FastForward(25 * time.Hour)

	_, err := e.svc.VerifyEmail(e.ctx, token)
	assertCode(t, err, errcode.TokenInvalid)
}

func TestResendVerification_Cooldown(t *testing.T) {
	e := setup(t)
	r := e.register(t)

	require.NoError(t, e.svc.ResendVerification(e.ctx, r.Merchant.ID))
	assertCode(t, e.svc.ResendVerification(e.ctx, r.Merchant.ID), errcode.TooManyRequests)
}

// ---------- 找回与修改密码 ----------

func TestForgotPassword_SameResponseForUnknownEmail(t *testing.T) {
	e := setup(t)
	require.NoError(t, e.svc.ForgotPassword(e.ctx, "nobody@example.com"))
	assert.Equal(t, 0, e.mail.count("RESET_PASSWORD"), "不存在的邮箱不发送邮件，但响应相同")
}

func TestResetPassword_RevokesSessionsAndUnlocks(t *testing.T) {
	e := setup(t)
	r := e.register(t)
	withCaptcha := func(in *LoginInput) { in.CaptchaID, in.CaptchaCode = "c", "ok" }
	for i := 0; i < 10; i++ {
		_, _ = e.login("Wrong-pass1", withCaptcha)
	}

	require.NoError(t, e.svc.ForgotPassword(e.ctx, testEmail))
	token := tokenFromLink(t, e.mail.last(t, "RESET_PASSWORD"))

	assertCode(t, e.svc.ResetPassword(e.ctx, token, "short"), errcode.InvalidParams)
	require.NoError(t, e.svc.ResetPassword(e.ctx, token, "NewPassw0rd"), "密码不合规时令牌保留，可修改后重试")

	_, err := e.svc.Authenticate(e.ctx, r.AccessToken)
	assert.Error(t, err, "重置密码后所有设备下线")
	_, err = e.login("NewPassw0rd")
	require.NoError(t, err, "重置密码解除锁定")
	assertCode(t, e.svc.ResetPassword(e.ctx, token, "Another1pw"), errcode.TokenInvalid)
}

func TestChangePassword_KeepsCurrentSession(t *testing.T) {
	e := setup(t)
	a := e.register(t)
	b, err := e.login(testPassword)
	require.NoError(t, err)
	idA, _ := e.svc.Authenticate(e.ctx, a.AccessToken)

	assertCode(t, e.svc.ChangePassword(e.ctx, idA, "Wrong-pass1", "NewPassw0rd"), errcode.InvalidParams)
	require.NoError(t, e.svc.ChangePassword(e.ctx, idA, testPassword, "NewPassw0rd"))

	_, err = e.svc.Authenticate(e.ctx, a.AccessToken)
	assert.NoError(t, err)
	_, err = e.svc.Authenticate(e.ctx, b.AccessToken)
	assert.Error(t, err)
	_, err = e.login("NewPassw0rd")
	assert.NoError(t, err)
}

func TestAuthenticate_RejectsForgedToken(t *testing.T) {
	e := setup(t)
	e.register(t)
	forged, _, err := auth.NewTokenManager("attacker-secret-at-least-32-characters", time.Minute).Issue(1, "x")
	require.NoError(t, err)

	_, err = e.svc.Authenticate(e.ctx, forged)
	assertCode(t, err, errcode.Unauthorized)
}
