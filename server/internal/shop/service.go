package shop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"mkclou/server/internal/pkg/config"
	"mkclou/server/internal/pkg/errcode"
)

// Merchants 读取商家账号信息。由账号模块实现，避免店铺模块直接依赖 user 包。
type Merchants interface {
	// Account 返回商家的登录邮箱与邮箱验证状态
	Account(ctx context.Context, merchantID uint64) (email string, verified bool, err error)
}

// Assets 把对象存储 key 转换为访问地址，由 storage.Client 实现。
type Assets interface {
	PublicURL(key string) string
}

type Service struct {
	repo      Repository
	merchants Merchants
	rdb       *redis.Client
	assets    Assets
	cfg       config.ShopConfig
	log       *zap.Logger
	reserved  map[string]bool
	now       func() time.Time
}

type Deps struct {
	Repo      Repository
	Merchants Merchants
	Redis     *redis.Client
	Assets    Assets
	Config    *config.Config
	Log       *zap.Logger
}

func NewService(d Deps) *Service {
	reserved := make(map[string]bool, len(d.Config.Shop.ReservedSlugs))
	for _, w := range d.Config.Shop.ReservedSlugs {
		reserved[NormalizeSlug(w)] = true
	}
	return &Service{
		repo:      d.Repo,
		merchants: d.Merchants,
		rdb:       d.Redis,
		assets:    d.Assets,
		cfg:       d.Config.Shop,
		log:       d.Log,
		reserved:  reserved,
		now:       time.Now,
	}
}

// ---------- 店铺链接 ----------

// SlugCheck 是链接可用性检查结果（接口 #18）。
type SlugCheck struct {
	Available   bool     `json:"available"`
	Reason      string   `json:"reason"`
	Suggestions []string `json:"suggestions"`
}

// CheckSlug 检查链接是否可用；不可用时推荐 3 个可用的备选（PRD SHOP-01）。
// 商家修改自己的链接时，当前链接和自己的旧链接视为可用。
func (s *Service) CheckSlug(ctx context.Context, merchantID uint64, raw string) (*SlugCheck, error) {
	slug := NormalizeSlug(raw)
	exceptID, err := s.ownShopID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if reason := slugFormatError(slug, s.reserved); reason != "" {
		return &SlugCheck{Reason: reason, Suggestions: s.suggest(ctx, slug, exceptID)}, nil
	}
	taken, err := s.repo.TakenSlugs(ctx, []string{slug}, exceptID, s.now())
	if err != nil {
		return nil, err
	}
	if taken[slug] {
		return &SlugCheck{Reason: "该链接已被占用", Suggestions: s.suggest(ctx, slug, exceptID)}, nil
	}
	return &SlugCheck{Available: true, Suggestions: []string{}}, nil
}

func (s *Service) ownShopID(ctx context.Context, merchantID uint64) (uint64, error) {
	sh, err := s.repo.FindByMerchant(ctx, merchantID)
	if errors.Is(err, errNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return sh.ID, nil
}

// suggest 基于输入生成 3 个格式正确且未被占用的链接。查询失败时返回空列表，不影响主流程。
func (s *Service) suggest(ctx context.Context, slug string, exceptShopID uint64) []string {
	base := sanitizeSlugBase(slug)
	if len(base) < slugMinLen {
		base = "shop" + base
	}
	candidates := make([]string, 0, 12)
	seen := map[string]bool{}
	add := func(c string) {
		if !seen[c] && slugFormatError(c, s.reserved) == "" {
			seen[c] = true
			candidates = append(candidates, c)
		}
	}
	for _, suffix := range []string{"-shop", "-studio", "-store"} {
		add(trimTo(base, slugMaxLen-len(suffix)) + suffix)
	}
	for range 9 {
		suffix := fmt.Sprintf("%d", 10+rand.IntN(990)) //nolint:gosec // 仅用于生成备选链接
		add(trimTo(base, slugMaxLen-len(suffix)) + suffix)
	}

	taken, err := s.repo.TakenSlugs(ctx, candidates, exceptShopID, s.now())
	if err != nil {
		s.log.Warn("suggest slugs failed", zap.Error(err))
		return []string{}
	}
	out := make([]string, 0, 3)
	for _, c := range candidates {
		if !taken[c] {
			out = append(out, c)
			if len(out) == 3 {
				break
			}
		}
	}
	return out
}

// sanitizeSlugBase 把任意输入整理为只含小写字母、数字和单个连字符的字符串。
func sanitizeSlugBase(in string) string {
	var b strings.Builder
	lastDash := true // 去掉开头的连字符
	for _, r := range strings.ToLower(in) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func trimTo(s string, n int) string {
	if len(s) > n {
		s = s[:n]
	}
	return strings.TrimRight(s, "-")
}

// ---------- 创建与查询 ----------

type CreateInput struct {
	Name string
	Slug string
}

// Create 创建店铺（PRD SHOP-01）。一个商家只能有一个店铺；联系邮箱默认为注册邮箱。
func (s *Service) Create(ctx context.Context, merchantID uint64, in CreateInput) (*View, error) {
	name := strings.TrimSpace(in.Name)
	slug := NormalizeSlug(in.Slug)
	if err := s.checkName(ctx, name); err != nil {
		return nil, err
	}
	if reason := slugFormatError(slug, s.reserved); reason != "" {
		return nil, fieldError("slug", reason)
	}
	if _, err := s.repo.FindByMerchant(ctx, merchantID); err == nil {
		return nil, errcode.StateConflict.WithMessage("你已经创建过店铺了")
	} else if !errors.Is(err, errNotFound) {
		return nil, err
	}
	if err := s.ensureSlugFree(ctx, slug, 0); err != nil {
		return nil, err
	}
	email, _, err := s.merchants.Account(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	sh := &Shop{
		MerchantID:   merchantID,
		Slug:         slug,
		Name:         name,
		ContactEmail: email,
		SocialLinks:  []SocialLink{},
		Theme:        DefaultTheme(),
		Status:       StatusOpen,
	}
	if err := s.repo.Create(ctx, sh); err != nil {
		if !errors.Is(err, errDuplicate) {
			return nil, err
		}
		// 区分是重复提交（该商家已有店铺）还是链接被并发占用
		if _, ferr := s.repo.FindByMerchant(ctx, merchantID); ferr == nil {
			return nil, errcode.StateConflict.WithMessage("你已经创建过店铺了")
		}
		return nil, slugJustTaken()
	}
	s.invalidate(ctx, slug)
	return s.view(sh), nil
}

func slugJustTaken() *errcode.Error {
	return errcode.SlugUnavailable.WithMessage("该链接刚刚被占用，请换一个").WithData(map[string]any{
		"fields": map[string]string{"slug": "该链接刚刚被占用，请换一个"},
	})
}

func (s *Service) ensureSlugFree(ctx context.Context, slug string, exceptShopID uint64) error {
	taken, err := s.repo.TakenSlugs(ctx, []string{slug}, exceptShopID, s.now())
	if err != nil {
		return err
	}
	if taken[slug] {
		return errcode.SlugUnavailable.WithMessage("该链接已被占用，请换一个").WithData(map[string]any{
			"fields":      map[string]string{"slug": "该链接已被占用，请换一个"},
			"suggestions": s.suggest(ctx, slug, exceptShopID),
		})
	}
	return nil
}

func (s *Service) checkName(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	blocked, err := s.repo.ContainsBlockedWord(ctx, name)
	if err != nil {
		return err
	}
	if blocked {
		return fieldError("name", "店铺名称包含不允许使用的词，请修改")
	}
	return nil
}

// Get 返回商家自己的店铺（接口 #20）。
func (s *Service) Get(ctx context.Context, merchantID uint64) (*View, error) {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	return s.view(sh), nil
}

// Summary 返回 /me 中的店铺概要；未创建店铺时返回 nil。
func (s *Service) Summary(ctx context.Context, merchantID uint64) (*Summary, error) {
	sh, err := s.repo.FindByMerchant(ctx, merchantID)
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Summary{Slug: sh.Slug, Name: sh.Name, Status: sh.Status}, nil
}

func (s *Service) mine(ctx context.Context, merchantID uint64) (*Shop, error) {
	sh, err := s.repo.FindByMerchant(ctx, merchantID)
	if errors.Is(err, errNotFound) {
		return nil, errcode.ShopNotCreated
	}
	return sh, err
}

// ---------- 修改 ----------

// UpdateInput 中为 nil 的字段表示不修改（PATCH 语义）。
type UpdateInput struct {
	Name         *string
	Slug         *string
	Description  *string
	ContactEmail *string
	SocialLinks  *[]SocialLink
	Theme        *Theme
}

// Update 修改基本信息与装修配置（PRD SHOP-02、SHOP-03）。
func (s *Service) Update(ctx context.Context, merchantID uint64, in UpdateInput) (*View, error) {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if sh.Status == StatusBanned {
		return nil, errcode.StateConflict.WithMessage("店铺已被封禁，无法修改")
	}

	now := s.now()
	fields := map[string]any{}
	var redirect *SlugRedirect
	oldSlug := sh.Slug

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if err := s.checkName(ctx, name); err != nil {
			return nil, err
		}
		fields["name"], sh.Name = name, name
	}
	if in.Slug != nil && NormalizeSlug(*in.Slug) != sh.Slug {
		slug := NormalizeSlug(*in.Slug)
		if next := s.nextSlugChange(sh); next != nil && now.Before(*next) {
			return nil, errcode.SlugChangeTooSoon.WithMessage(
				fmt.Sprintf("店铺链接 30 天内只能修改一次，%s 后可再次修改", next.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")),
			).WithData(map[string]any{"nextSlugChangeAt": next.UTC()})
		}
		if reason := slugFormatError(slug, s.reserved); reason != "" {
			return nil, fieldError("slug", reason)
		}
		if err := s.ensureSlugFree(ctx, slug, sh.ID); err != nil {
			return nil, err
		}
		redirect = &SlugRedirect{OldSlug: sh.Slug, ShopID: sh.ID, ExpiresAt: now.Add(s.cfg.SlugRedirectTTL), CreatedAt: now}
		fields["slug"], sh.Slug = slug, slug
		fields["slug_changed_at"], sh.SlugChangedAt = now, &now
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if err := validateDescription(desc); err != nil {
			return nil, err
		}
		fields["description"], sh.Description = desc, desc
	}
	if in.ContactEmail != nil {
		email := strings.ToLower(strings.TrimSpace(*in.ContactEmail))
		if err := validateContactEmail(email); err != nil {
			return nil, err
		}
		fields["contact_email"], sh.ContactEmail = email, email
	}
	if in.SocialLinks != nil {
		links := normalizeLinks(*in.SocialLinks)
		if err := validateSocialLinks(links); err != nil {
			return nil, err
		}
		fields["social_links"], sh.SocialLinks = jsonValue(links), links
	}
	if in.Theme != nil {
		t := *in.Theme
		t.Color = strings.ToUpper(t.Color)
		if err := validateTheme(t); err != nil {
			return nil, err
		}
		fields["theme"], sh.Theme = jsonValue(t), t
	}

	if len(fields) == 0 {
		return s.view(sh), nil
	}
	if err := s.repo.Update(ctx, sh.ID, fields, redirect); err != nil {
		if errors.Is(err, errDuplicate) {
			return nil, slugJustTaken()
		}
		return nil, err
	}
	s.invalidate(ctx, oldSlug, sh.Slug)
	return s.view(sh), nil
}

func normalizeLinks(in []SocialLink) []SocialLink {
	out := make([]SocialLink, 0, len(in))
	for _, l := range in {
		l.Type = strings.TrimSpace(l.Type)
		l.URL = strings.TrimSpace(l.URL)
		if l.URL != "" {
			out = append(out, l)
		}
	}
	return out
}

// jsonValue 把 JSON 列的值序列化为字符串。按 map 更新时 GORM 不会调用字段的 serializer。
func jsonValue(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// SetStatus 暂停营业 / 恢复营业（PRD SHOP-06）。封禁状态只能由平台管理员解除。
func (s *Service) SetStatus(ctx context.Context, merchantID uint64, status, note string) (*View, error) {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if sh.Status == StatusBanned {
		return nil, errcode.StateConflict.WithMessage("店铺已被封禁，如有疑问请联系平台客服")
	}
	fields := map[string]any{"status": status}
	switch status {
	case StatusOpen:
		fields["pause_note"], sh.PauseNote = nil, nil
	case StatusPaused:
		note = strings.TrimSpace(note)
		if err := validatePauseNote(note); err != nil {
			return nil, err
		}
		if note == "" {
			fields["pause_note"], sh.PauseNote = nil, nil
		} else {
			fields["pause_note"], sh.PauseNote = note, &note
		}
	default:
		return nil, fieldError("status", "不支持的店铺状态")
	}
	sh.Status = status
	if err := s.repo.Update(ctx, sh.ID, fields, nil); err != nil {
		return nil, err
	}
	s.invalidate(ctx, sh.Slug)
	return s.view(sh), nil
}

// ---------- 新手清单 ----------

// Onboarding 返回新手清单完成情况（PRD SHOP-05）。
func (s *Service) Onboarding(ctx context.Context, merchantID uint64) (*Onboarding, error) {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	_, verified, err := s.merchants.Account(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	_, payStatus, found, err := s.repo.PaymentConfig(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	hasProduct, err := s.repo.HasProductOnSale(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	return &Onboarding{
		EmailVerified: verified,
		PaymentReady:  found && payStatus == PaymentStatusActive,
		HasProduct:    hasProduct,
		Shared:        sh.SharedAt != nil,
	}, nil
}

// MarkShared 标记“已分享店铺”，重复调用无副作用。
func (s *Service) MarkShared(ctx context.Context, merchantID uint64) error {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return err
	}
	if sh.SharedAt != nil {
		return nil
	}
	return s.repo.MarkShared(ctx, sh.ID, s.now())
}

// ---------- 供其他模块使用 ----------

// ShopID 返回商家的店铺 ID；未开店时返回 errcode.ShopNotCreated。
func (s *Service) ShopID(ctx context.Context, merchantID uint64) (uint64, error) {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return 0, err
	}
	return sh.ID, nil
}

// Seller 返回商家的店铺信息与邮箱验证状态，用于商品上架检查。
func (s *Service) Seller(ctx context.Context, merchantID uint64) (*Ref, error) {
	sh, err := s.mine(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	ref, err := s.ref(ctx, sh)
	if err != nil {
		return nil, err
	}
	_, ref.EmailVerified, err = s.merchants.Account(ctx, merchantID)
	return ref, err
}

// RefByID 按店铺 ID 返回店铺信息（含买家端视图），不存在时返回 errcode.NotFound。
func (s *Service) RefByID(ctx context.Context, id uint64) (*Ref, error) {
	sh, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, errNotFound) {
		return nil, errcode.NotFound.WithMessage("店铺不存在")
	}
	if err != nil {
		return nil, err
	}
	return s.ref(ctx, sh)
}

// RefBySlug 按当前链接返回店铺信息（不处理旧链接跳转），不存在时返回 errcode.NotFound。
func (s *Service) RefBySlug(ctx context.Context, slug string) (*Ref, error) {
	sh, err := s.repo.FindBySlug(ctx, NormalizeSlug(slug))
	if errors.Is(err, errNotFound) {
		return nil, errcode.NotFound.WithMessage("店铺不存在")
	}
	if err != nil {
		return nil, err
	}
	return s.ref(ctx, sh)
}

func (s *Service) ref(ctx context.Context, sh *Shop) (*Ref, error) {
	v, err := s.publicView(ctx, sh)
	if err != nil {
		return nil, err
	}
	_, status, found, err := s.repo.PaymentConfig(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	return &Ref{
		ID: sh.ID, Slug: sh.Slug, Status: sh.Status,
		PaymentReady: found && status == PaymentStatusActive, Public: v,
	}, nil
}

// ---------- 买家端 ----------

// PublicResult 是买家访问 /s/{slug} 的结果：店铺信息，或旧链接需要跳转到的新链接（PRD 02 3.2）。
type PublicResult struct {
	Shop       *PublicView `json:"shop"`
	RedirectTo *string     `json:"redirectTo"`
}

func publicCacheKey(slug string) string { return "mk:shop:" + slug }

// Public 返回买家端店铺信息，使用 Redis 缓存（Cache Aside，数据库设计第 4 节）。
func (s *Service) Public(ctx context.Context, raw string) (*PublicResult, error) {
	slug := NormalizeSlug(raw)
	if slug == "" || len(slug) > slugMaxLen {
		return nil, errcode.NotFound.WithMessage("店铺不存在")
	}

	key := publicCacheKey(slug)
	if b, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
		var v PublicView
		if json.Unmarshal(b, &v) == nil {
			return &PublicResult{Shop: &v}, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		s.log.Warn("read shop cache failed", zap.Error(err)) // 缓存不可用时直接查库
	}

	sh, err := s.repo.FindBySlug(ctx, slug)
	if errors.Is(err, errNotFound) {
		to, rerr := s.repo.FindRedirect(ctx, slug, s.now())
		if errors.Is(rerr, errNotFound) {
			return nil, errcode.NotFound.WithMessage("店铺不存在")
		}
		if rerr != nil {
			return nil, rerr
		}
		return &PublicResult{RedirectTo: &to}, nil
	}
	if err != nil {
		return nil, err
	}

	v, err := s.publicView(ctx, sh)
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(v); err == nil {
		if err := s.rdb.Set(ctx, key, b, s.cfg.PublicCacheTTL).Err(); err != nil {
			s.log.Warn("write shop cache failed", zap.Error(err))
		}
	}
	return &PublicResult{Shop: v}, nil
}

func (s *Service) publicView(ctx context.Context, sh *Shop) (*PublicView, error) {
	if sh.Status == StatusBanned {
		// 已封禁的店铺只显示“该店铺已关闭”，不再对外展示店铺内容
		return &PublicView{
			Slug: sh.Slug, Name: sh.Name, SocialLinks: []SocialLink{}, Theme: DefaultTheme(), Status: StatusBanned,
		}, nil
	}
	env, status, found, err := s.repo.PaymentConfig(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	return &PublicView{
		Slug:         sh.Slug,
		Name:         sh.Name,
		Description:  sh.Description,
		AvatarURL:    s.assetURL(sh.AvatarKey),
		CoverURL:     s.assetURL(sh.CoverKey),
		ContactEmail: sh.ContactEmail,
		SocialLinks:  orEmpty(sh.SocialLinks),
		Theme:        sh.Theme,
		Status:       sh.Status,
		PauseNote:    sh.PauseNote,
		IsTestMode:   found && status == PaymentStatusActive && env == PaymentEnvSandbox,
	}, nil
}

// invalidate 删除买家端店铺缓存（先更新数据库，再删除缓存）。删除失败只记录日志，缓存会在 TTL 后过期。
func (s *Service) invalidate(ctx context.Context, slugs ...string) {
	keys := make([]string, len(slugs))
	for i, slug := range slugs {
		keys[i] = publicCacheKey(slug)
	}
	if err := s.rdb.Del(ctx, keys...).Err(); err != nil {
		s.log.Warn("invalidate shop cache failed", zap.Strings("keys", keys), zap.Error(err))
	}
}

// ---------- 视图 ----------

func (s *Service) view(sh *Shop) *View {
	next := s.nextSlugChange(sh)
	allowed := next == nil || !s.now().Before(*next)
	if allowed {
		next = nil
	}
	return &View{
		Slug:              sh.Slug,
		Name:              sh.Name,
		Description:       sh.Description,
		AvatarURL:         s.assetURL(sh.AvatarKey),
		CoverURL:          s.assetURL(sh.CoverKey),
		ContactEmail:      sh.ContactEmail,
		SocialLinks:       orEmpty(sh.SocialLinks),
		Theme:             sh.Theme,
		Status:            sh.Status,
		PauseNote:         sh.PauseNote,
		SlugChangeAllowed: allowed,
		NextSlugChangeAt:  next,
		CreatedAt:         sh.CreatedAt.UTC(),
	}
}

// nextSlugChange 返回下一次允许修改链接的时间；从未修改过时返回 nil。
func (s *Service) nextSlugChange(sh *Shop) *time.Time {
	if sh.SlugChangedAt == nil {
		return nil
	}
	t := sh.SlugChangedAt.Add(s.cfg.SlugChangeInterval).UTC()
	return &t
}

// assetURL 把公有桶中的对象 key 转换为访问地址。
func (s *Service) assetURL(key *string) *string {
	if key == nil || *key == "" {
		return nil
	}
	u := s.assets.PublicURL(*key)
	return &u
}

func orEmpty(links []SocialLink) []SocialLink {
	if links == nil {
		return []SocialLink{}
	}
	return links
}
