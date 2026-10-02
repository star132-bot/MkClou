// Package shop 实现店铺模块（PRD 02，接口清单 #18 ～ #24、#72）。
//
// 分层：handler（HTTP）→ service（业务规则）→ repository（MySQL）；买家端店铺信息使用 Redis 缓存（Cache Aside）。
// 收款配置（SHOP-04，接口 #25 ～ #27）与图片上传（#28）在后续迭代中实现。
package shop

import "time"

const (
	StatusOpen   = "OPEN"
	StatusPaused = "PAUSED"
	StatusBanned = "BANNED"
)

// 收款配置的环境与状态（shop_payment_configs 表）
const (
	PaymentEnvSandbox   = "SANDBOX"
	PaymentStatusActive = "ACTIVE"
)

// Theme 是店铺装修配置，整体存于 shops.theme（JSON）。
type Theme struct {
	Color     string `json:"color"`
	CardRatio string `json:"cardRatio"` // 4:3 / 16:9 / 1:1
	Layout    string `json:"layout"`    // grid / list
	Mode      string `json:"mode"`      // light / dark / system
}

// DefaultTheme 是新店铺的装修配置（PRD SHOP-03 默认值，主题色为设计规范品牌色）。
func DefaultTheme() Theme {
	return Theme{Color: "#5B5BD6", CardRatio: "4:3", Layout: "grid", Mode: "light"}
}

type SocialLink struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Shop 对应 shops 表（docs/database-design.md 3.2）。
type Shop struct {
	ID            uint64 `gorm:"primaryKey"`
	MerchantID    uint64
	Slug          string
	Name          string
	Description   string
	AvatarKey     *string
	CoverKey      *string
	ContactEmail  string
	SocialLinks   []SocialLink `gorm:"serializer:json"`
	Theme         Theme        `gorm:"serializer:json"`
	Status        string
	PauseNote     *string
	SlugChangedAt *time.Time
	SharedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SlugRedirect 对应 shop_slug_redirects 表：修改链接后旧链接在有效期内跳转到新链接。
type SlugRedirect struct {
	OldSlug   string `gorm:"primaryKey"`
	ShopID    uint64
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (SlugRedirect) TableName() string { return "shop_slug_redirects" }

// ---------- 返回给前端的结构 ----------

// View 是商家后台看到的店铺信息（接口 #20）。
type View struct {
	Slug              string       `json:"slug"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	AvatarURL         *string      `json:"avatarUrl"`
	CoverURL          *string      `json:"coverUrl"`
	ContactEmail      string       `json:"contactEmail"`
	SocialLinks       []SocialLink `json:"socialLinks"`
	Theme             Theme        `json:"theme"`
	Status            string       `json:"status"`
	PauseNote         *string      `json:"pauseNote"`
	SlugChangeAllowed bool         `json:"slugChangeAllowed"`
	// NextSlugChangeAt 是下一次可以修改链接的时间；当前可修改时为 null
	NextSlugChangeAt *time.Time `json:"nextSlugChangeAt"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// Summary 是 /me 接口中的店铺概要（接口 #10）。
type Summary struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PublicView 是买家端看到的店铺信息（接口 #72），与 web/src/lib/storefront/types.ts 的 PublicShop 对应。
type PublicView struct {
	Slug         string       `json:"slug"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	AvatarURL    *string      `json:"avatarUrl"`
	CoverURL     *string      `json:"coverUrl"`
	ContactEmail string       `json:"contactEmail"`
	SocialLinks  []SocialLink `json:"socialLinks"`
	Theme        Theme        `json:"theme"`
	Status       string       `json:"status"`
	PauseNote    *string      `json:"pauseNote"`
	IsTestMode   bool         `json:"isTestMode"`
}

// Ref 是其他模块（商品、上传）需要的店铺信息。
type Ref struct {
	ID           uint64
	Slug         string
	Status       string
	PaymentReady bool
	// EmailVerified 是店主邮箱是否已验证（上架商品的前提），只在 Seller 中填充
	EmailVerified bool
	Public        *PublicView
}

// Onboarding 是新手清单的完成情况（接口 #23，PRD SHOP-05）。
type Onboarding struct {
	EmailVerified bool `json:"emailVerified"`
	PaymentReady  bool `json:"paymentReady"`
	HasProduct    bool `json:"hasProduct"`
	Shared        bool `json:"shared"`
}
