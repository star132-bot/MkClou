package shop

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/errcode"
)

// ---------- 测试替身 ----------

type payment struct{ env, status string }

type fakeRepo struct {
	mu        sync.Mutex
	nextID    uint64
	shops     map[uint64]*Shop
	redirects map[string]SlugRedirect
	payments  map[uint64]payment
	onSale    map[uint64]bool
	blocked   []string
	// raceSlug 模拟检查通过后、写入前链接被其他商家抢先占用
	raceSlug string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		shops: map[uint64]*Shop{}, redirects: map[string]SlugRedirect{},
		payments: map[uint64]payment{}, onSale: map[uint64]bool{},
	}
}

func (r *fakeRepo) Create(_ context.Context, s *Shop) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s.Slug == r.raceSlug {
		return errDuplicate
	}
	for _, x := range r.shops {
		if x.Slug == s.Slug || x.MerchantID == s.MerchantID {
			return errDuplicate
		}
	}
	r.nextID++
	s.ID = r.nextID
	s.CreatedAt = time.Now()
	cp := *s
	r.shops[s.ID] = &cp
	return nil
}

func (r *fakeRepo) find(match func(*Shop) bool) (*Shop, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.shops {
		if match(s) {
			cp := *s
			return &cp, nil
		}
	}
	return nil, errNotFound
}

func (r *fakeRepo) FindByMerchant(_ context.Context, id uint64) (*Shop, error) {
	return r.find(func(s *Shop) bool { return s.MerchantID == id })
}

func (r *fakeRepo) FindByID(_ context.Context, id uint64) (*Shop, error) {
	return r.find(func(s *Shop) bool { return s.ID == id })
}

func (r *fakeRepo) FindBySlug(_ context.Context, slug string) (*Shop, error) {
	return r.find(func(s *Shop) bool { return s.Slug == slug })
}

func (r *fakeRepo) FindRedirect(_ context.Context, old string, now time.Time) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rd, ok := r.redirects[old]
	if !ok || !rd.ExpiresAt.After(now) {
		return "", errNotFound
	}
	return r.shops[rd.ShopID].Slug, nil
}

func (r *fakeRepo) TakenSlugs(_ context.Context, cands []string, except uint64, now time.Time) (map[string]bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	taken := map[string]bool{}
	for _, c := range cands {
		for _, s := range r.shops {
			if s.Slug == c && s.ID != except {
				taken[c] = true
			}
		}
		if rd, ok := r.redirects[c]; ok && rd.ShopID != except && rd.ExpiresAt.After(now) {
			taken[c] = true
		}
	}
	return taken, nil
}

func (r *fakeRepo) Update(_ context.Context, id uint64, f map[string]any, rd *SlugRedirect) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.shops[id]
	if rd != nil {
		delete(r.redirects, f["slug"].(string))
		r.redirects[rd.OldSlug] = *rd
	}
	for k, v := range f {
		switch k {
		case "name":
			s.Name = v.(string)
		case "slug":
			s.Slug = v.(string)
		case "slug_changed_at":
			t := v.(time.Time)
			s.SlugChangedAt = &t
		case "description":
			s.Description = v.(string)
		case "status":
			s.Status = v.(string)
		case "pause_note":
			if v == nil {
				s.PauseNote = nil
			} else {
				n := v.(string)
				s.PauseNote = &n
			}
		}
	}
	return nil
}

func (r *fakeRepo) MarkShared(_ context.Context, id uint64, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.shops[id].SharedAt = &at
	return nil
}

func (r *fakeRepo) PaymentConfig(_ context.Context, id uint64) (string, string, bool, error) {
	p, ok := r.payments[id]
	return p.env, p.status, ok, nil
}

func (r *fakeRepo) HasProductOnSale(_ context.Context, id uint64) (bool, error) {
	return r.onSale[id], nil
}

func (r *fakeRepo) ContainsBlockedWord(_ context.Context, text string) (bool, error) {
	for _, w := range r.blocked {
		if strings.Contains(text, w) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) setStatus(slug, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.shops {
		if s.Slug == slug {
			s.Status = status
		}
	}
}

type fakeMerchants map[uint64]bool // merchantID → 邮箱是否已验证

func (m fakeMerchants) Account(_ context.Context, id uint64) (string, bool, error) {
	return "Seller@Example.com", m[id], nil
}

type fakeAssets struct{}

func (fakeAssets) PublicURL(key string) string { return "http://assets.test/" + key }

type env struct {
	svc   *Service
	repo  *fakeRepo
	redis *miniredis.Miniredis
	clock *time.Time
}

func setup(t *testing.T) *env {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := newFakeRepo()
	cfg := &config.Config{
		S3: config.S3Config{Endpoint: "http://127.0.0.1:9002", PublicBucket: "mkclou-public"},
		Shop: config.ShopConfig{
			ReservedSlugs:      []string{"admin", "dashboard", "Shop"},
			SlugChangeInterval: 30 * 24 * time.Hour,
			SlugRedirectTTL:    90 * 24 * time.Hour,
			PublicCacheTTL:     5 * time.Minute,
		},
	}
	svc := NewService(Deps{Repo: repo, Merchants: fakeMerchants{1: true}, Redis: rdb, Assets: fakeAssets{}, Config: cfg, Log: zap.NewNop()})
	clock := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	e := &env{svc: svc, repo: repo, redis: mr, clock: &clock}
	svc.now = func() time.Time { return *e.clock }
	return e
}

func (e *env) advance(d time.Duration) { *e.clock = e.clock.Add(d) }

func ctx() context.Context { return context.Background() }

func ptr[T any](v T) *T { return &v }

func requireCode(t *testing.T, err error, want *errcode.Error) *errcode.Error {
	t.Helper()
	var e *errcode.Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, want.Code, e.Code, e.Message)
	return e
}

func fields(e *errcode.Error) map[string]string {
	d, _ := e.Data.(map[string]any)
	f, _ := d["fields"].(map[string]string)
	return f
}

// ---------- 创建店铺 ----------

func TestCreate(t *testing.T) {
	e := setup(t)
	v, err := e.svc.Create(ctx(), 1, CreateInput{Name: "  阿杰的工具铺 ", Slug: " AJie-Tools "})
	require.NoError(t, err)
	assert.Equal(t, "ajie-tools", v.Slug)
	assert.Equal(t, "阿杰的工具铺", v.Name)
	assert.Equal(t, "Seller@Example.com", v.ContactEmail, "联系邮箱默认为注册邮箱")
	assert.Equal(t, DefaultTheme(), v.Theme)
	assert.Equal(t, StatusOpen, v.Status)
	assert.True(t, v.SlugChangeAllowed)
	assert.NotNil(t, v.SocialLinks)

	_, err = e.svc.Create(ctx(), 1, CreateInput{Name: "第二家店", Slug: "second"})
	requireCode(t, err, errcode.StateConflict)
}

func TestCreateValidation(t *testing.T) {
	e := setup(t)
	e.repo.blocked = []string{"违禁"}
	cases := []struct {
		name, slug, field string
	}{
		{"阿", "valid-slug", "name"},
		{strings.Repeat("长", 21), "valid-slug", "name"},
		{"含有违禁词的店", "valid-slug", "name"},
		{"正常店名", "ab", "slug"},
		{"正常店名", "-abc", "slug"},
		{"正常店名", "abc-", "slug"},
		{"正常店名", "a--b", "slug"},
		{"正常店名", "中文链接", "slug"},
		{"正常店名", "admin", "slug"},
		{"正常店名", "shop", "slug"}, // 配置中的保留词大小写不敏感
		{"正常店名", strings.Repeat("a", 21), "slug"},
	}
	for _, c := range cases {
		_, err := e.svc.Create(ctx(), 1, CreateInput{Name: c.name, Slug: c.slug})
		ce := requireCode(t, err, errcode.InvalidParams)
		assert.Contains(t, fields(ce), c.field, "name=%q slug=%q", c.name, c.slug)
	}
}

func TestCreateSlugTaken(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Create(ctx(), 1, CreateInput{Name: "第一家", Slug: "tools"})
	require.NoError(t, err)

	_, err = e.svc.Create(ctx(), 2, CreateInput{Name: "第二家", Slug: "tools"})
	ce := requireCode(t, err, errcode.SlugUnavailable)
	assert.Contains(t, fields(ce), "slug")
	assert.Len(t, ce.Data.(map[string]any)["suggestions"], 3)
}

func TestCreateConcurrentSlugRace(t *testing.T) {
	e := setup(t)
	e.repo.raceSlug = "hot-slug"
	_, err := e.svc.Create(ctx(), 2, CreateInput{Name: "第二家", Slug: "hot-slug"})
	ce := requireCode(t, err, errcode.SlugUnavailable)
	assert.Equal(t, "该链接刚刚被占用，请换一个", ce.Message)
}

// ---------- 链接可用性 ----------

func TestCheckSlug(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Create(ctx(), 1, CreateInput{Name: "第一家", Slug: "tools"})
	require.NoError(t, err)

	r, err := e.svc.CheckSlug(ctx(), 2, "fresh-name")
	require.NoError(t, err)
	assert.True(t, r.Available)
	assert.Empty(t, r.Suggestions)

	r, err = e.svc.CheckSlug(ctx(), 2, "Tools")
	require.NoError(t, err)
	assert.False(t, r.Available)
	assert.Equal(t, "该链接已被占用", r.Reason)
	require.Len(t, r.Suggestions, 3)
	for _, s := range r.Suggestions {
		assert.Empty(t, slugFormatError(s, e.svc.reserved), s)
		assert.NotEqual(t, "tools", s)
		assert.LessOrEqual(t, len(s), slugMaxLen)
	}

	// 自己当前的链接视为可用
	r, err = e.svc.CheckSlug(ctx(), 1, "tools")
	require.NoError(t, err)
	assert.True(t, r.Available)

	r, err = e.svc.CheckSlug(ctx(), 2, "admin")
	require.NoError(t, err)
	assert.False(t, r.Available)
	assert.Contains(t, r.Reason, "保留")
	assert.Len(t, r.Suggestions, 3)
}

func TestSuggestFromLongOrMessyInput(t *testing.T) {
	e := setup(t)
	for _, in := range []string{"--A__very  long shop name that exceeds--", "a", "中文"} {
		r, err := e.svc.CheckSlug(ctx(), 1, in)
		require.NoError(t, err)
		require.Len(t, r.Suggestions, 3, in)
		for _, s := range r.Suggestions {
			assert.Empty(t, slugFormatError(s, e.svc.reserved), "input %q → %q", in, s)
		}
	}
}

// ---------- 修改链接 ----------

func TestChangeSlugRules(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Create(ctx(), 1, CreateInput{Name: "第一家", Slug: "old-name"})
	require.NoError(t, err)

	v, err := e.svc.Update(ctx(), 1, UpdateInput{Slug: ptr("new-name")})
	require.NoError(t, err)
	assert.Equal(t, "new-name", v.Slug)
	assert.False(t, v.SlugChangeAllowed)
	require.NotNil(t, v.NextSlugChangeAt)
	assert.Equal(t, e.clock.Add(30*24*time.Hour), *v.NextSlugChangeAt)

	// 旧链接跳转到新链接
	pub, err := e.svc.Public(ctx(), "old-name")
	require.NoError(t, err)
	require.NotNil(t, pub.RedirectTo)
	assert.Equal(t, "new-name", *pub.RedirectTo)
	assert.Nil(t, pub.Shop)

	// 旧链接在 90 天内不能被他人使用
	r, err := e.svc.CheckSlug(ctx(), 2, "old-name")
	require.NoError(t, err)
	assert.False(t, r.Available)

	// 30 天内不能再次修改；提交相同链接不算修改
	_, err = e.svc.Update(ctx(), 1, UpdateInput{Slug: ptr("third-name")})
	requireCode(t, err, errcode.SlugChangeTooSoon)
	_, err = e.svc.Update(ctx(), 1, UpdateInput{Slug: ptr("NEW-name"), Name: ptr("改个名")})
	require.NoError(t, err)

	// 30 天后可以改回自己以前的链接
	e.advance(31 * 24 * time.Hour)
	v, err = e.svc.Update(ctx(), 1, UpdateInput{Slug: ptr("old-name")})
	require.NoError(t, err)
	assert.Equal(t, "old-name", v.Slug)
	pub, err = e.svc.Public(ctx(), "old-name")
	require.NoError(t, err)
	require.NotNil(t, pub.Shop)
	assert.Equal(t, "改个名", pub.Shop.Name)

	// 90 天后旧链接过期，可被他人使用
	e.advance(91 * 24 * time.Hour)
	r, err = e.svc.CheckSlug(ctx(), 2, "new-name")
	require.NoError(t, err)
	assert.True(t, r.Available)
	_, err = e.svc.Public(ctx(), "new-name")
	requireCode(t, err, errcode.NotFound)
}

// ---------- 基本信息与装修 ----------

func TestUpdateFields(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Create(ctx(), 1, CreateInput{Name: "第一家", Slug: "tools"})
	require.NoError(t, err)

	v, err := e.svc.Update(ctx(), 1, UpdateInput{
		Description:  ptr("  分享真实项目里的工程模板  "),
		ContactEmail: ptr(" Help@Example.com "),
		SocialLinks: &[]SocialLink{
			{Type: "github", URL: "https://github.com/x"},
			{Type: "website", URL: "  "}, // 空链接被忽略
		},
		Theme: &Theme{Color: "#2563eb", CardRatio: "16:9", Layout: "list", Mode: "system"},
	})
	require.NoError(t, err)
	assert.Equal(t, "分享真实项目里的工程模板", v.Description)
	assert.Equal(t, "help@example.com", v.ContactEmail)
	assert.Equal(t, []SocialLink{{Type: "github", URL: "https://github.com/x"}}, v.SocialLinks)
	assert.Equal(t, Theme{Color: "#2563EB", CardRatio: "16:9", Layout: "list", Mode: "system"}, v.Theme)

	bad := []struct {
		in    UpdateInput
		field string
	}{
		{UpdateInput{Description: ptr(strings.Repeat("字", 201))}, "description"},
		{UpdateInput{ContactEmail: ptr("not-an-email")}, "contactEmail"},
		{UpdateInput{SocialLinks: &[]SocialLink{{Type: "github", URL: "javascript:alert(1)"}}}, "socialLinks"},
		{UpdateInput{SocialLinks: &[]SocialLink{{Type: "myspace", URL: "https://x.com"}}}, "socialLinks"},
		{UpdateInput{SocialLinks: &[]SocialLink{
			{"github", "https://a.com"}, {"github", "https://b.com"}, {"github", "https://c.com"},
			{"github", "https://d.com"}, {"github", "https://e.com"},
		}}, "socialLinks"},
		{UpdateInput{Theme: &Theme{Color: "#FFE066", CardRatio: "4:3", Layout: "grid", Mode: "light"}}, "theme.color"},
		{UpdateInput{Theme: &Theme{Color: "red", CardRatio: "4:3", Layout: "grid", Mode: "light"}}, "theme.color"},
		{UpdateInput{Theme: &Theme{Color: "#5B5BD6", CardRatio: "3:2", Layout: "grid", Mode: "light"}}, "theme.cardRatio"},
		{UpdateInput{Theme: &Theme{Color: "#5B5BD6", CardRatio: "4:3", Layout: "masonry", Mode: "light"}}, "theme.layout"},
		{UpdateInput{Theme: &Theme{Color: "#5B5BD6", CardRatio: "4:3", Layout: "grid", Mode: "auto"}}, "theme.mode"},
	}
	for i, c := range bad {
		_, err := e.svc.Update(ctx(), 1, c.in)
		ce := requireCode(t, err, errcode.InvalidParams)
		assert.Contains(t, fields(ce), c.field, "case %d", i)
	}
}

func TestPresetColorsPassContrast(t *testing.T) {
	assert.Len(t, PresetColors, 8)
	for _, c := range PresetColors {
		assert.GreaterOrEqual(t, ContrastWithWhite(c), minBrandContrast, c)
	}
	assert.InDelta(t, 21.0, ContrastWithWhite("#000000"), 0.01)
	assert.InDelta(t, 1.0, ContrastWithWhite("#FFFFFF"), 0.01)
	assert.Zero(t, ContrastWithWhite("#FFF"))
}

func TestShopNotCreated(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Get(ctx(), 1)
	requireCode(t, err, errcode.ShopNotCreated)
	_, err = e.svc.Update(ctx(), 1, UpdateInput{Name: ptr("名字")})
	requireCode(t, err, errcode.ShopNotCreated)
	sum, err := e.svc.Summary(ctx(), 1)
	require.NoError(t, err)
	assert.Nil(t, sum)
}

// ---------- 店铺状态 ----------

func TestSetStatus(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Create(ctx(), 1, CreateInput{Name: "第一家", Slug: "tools"})
	require.NoError(t, err)

	v, err := e.svc.SetStatus(ctx(), 1, StatusPaused, "  国庆休息，10 月 8 日恢复  ")
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, v.Status)
	require.NotNil(t, v.PauseNote)
	assert.Equal(t, "国庆休息，10 月 8 日恢复", *v.PauseNote)

	v, err = e.svc.SetStatus(ctx(), 1, StatusOpen, "")
	require.NoError(t, err)
	assert.Equal(t, StatusOpen, v.Status)
	assert.Nil(t, v.PauseNote)

	_, err = e.svc.SetStatus(ctx(), 1, StatusPaused, strings.Repeat("长", 101))
	requireCode(t, err, errcode.InvalidParams)
	_, err = e.svc.SetStatus(ctx(), 1, StatusBanned, "")
	requireCode(t, err, errcode.InvalidParams)

	e.repo.setStatus("tools", StatusBanned)
	_, err = e.svc.SetStatus(ctx(), 1, StatusOpen, "")
	requireCode(t, err, errcode.StateConflict)
	_, err = e.svc.Update(ctx(), 1, UpdateInput{Name: ptr("改名")})
	requireCode(t, err, errcode.StateConflict)
}

// ---------- 买家端 ----------

func TestPublic(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Create(ctx(), 1, CreateInput{Name: "第一家", Slug: "tools"})
	require.NoError(t, err)

	_, err = e.svc.Public(ctx(), "missing")
	requireCode(t, err, errcode.NotFound)

	r, err := e.svc.Public(ctx(), "TOOLS")
	require.NoError(t, err)
	require.NotNil(t, r.Shop)
	assert.Nil(t, r.RedirectTo)
	assert.Equal(t, "第一家", r.Shop.Name)
	assert.False(t, r.Shop.IsTestMode)
	assert.True(t, e.redis.Exists("mk:shop:tools"), "结果写入缓存")

	// 修改后删除缓存，下次读取到新数据
	_, err = e.svc.Update(ctx(), 1, UpdateInput{Name: ptr("新名字")})
	require.NoError(t, err)
	assert.False(t, e.redis.Exists("mk:shop:tools"))
	r, err = e.svc.Public(ctx(), "tools")
	require.NoError(t, err)
	assert.Equal(t, "新名字", r.Shop.Name)

	// 暂停营业后买家看到暂停说明
	_, err = e.svc.SetStatus(ctx(), 1, StatusPaused, "休息一周")
	require.NoError(t, err)
	r, err = e.svc.Public(ctx(), "tools")
	require.NoError(t, err)
	assert.Equal(t, StatusPaused, r.Shop.Status)
	assert.Equal(t, "休息一周", *r.Shop.PauseNote)
}

func TestPublicTestModeAndBanned(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Create(ctx(), 1, CreateInput{Name: "第一家", Slug: "tools"})
	require.NoError(t, err)
	_, err = e.svc.Update(ctx(), 1, UpdateInput{Description: ptr("简介")})
	require.NoError(t, err)

	e.repo.payments[1] = payment{env: PaymentEnvSandbox, status: PaymentStatusActive}
	r, err := e.svc.Public(ctx(), "tools")
	require.NoError(t, err)
	assert.True(t, r.Shop.IsTestMode)

	e.repo.setStatus("tools", StatusBanned)
	e.redis.FlushAll()
	r, err = e.svc.Public(ctx(), "tools")
	require.NoError(t, err)
	assert.Equal(t, StatusBanned, r.Shop.Status)
	assert.Empty(t, r.Shop.Description, "已封禁的店铺不展示店铺内容")
	assert.Empty(t, r.Shop.ContactEmail)
}

// ---------- 新手清单 ----------

func TestOnboarding(t *testing.T) {
	e := setup(t)
	_, err := e.svc.Onboarding(ctx(), 1)
	requireCode(t, err, errcode.ShopNotCreated)

	_, err = e.svc.Create(ctx(), 1, CreateInput{Name: "第一家", Slug: "tools"})
	require.NoError(t, err)
	o, err := e.svc.Onboarding(ctx(), 1)
	require.NoError(t, err)
	assert.Equal(t, Onboarding{EmailVerified: true}, *o)

	e.repo.payments[1] = payment{env: "PRODUCTION", status: PaymentStatusActive}
	e.repo.onSale[1] = true
	require.NoError(t, e.svc.MarkShared(ctx(), 1))
	require.NoError(t, e.svc.MarkShared(ctx(), 1), "重复标记无副作用")
	o, err = e.svc.Onboarding(ctx(), 1)
	require.NoError(t, err)
	assert.Equal(t, Onboarding{EmailVerified: true, PaymentReady: true, HasProduct: true, Shared: true}, *o)
}
