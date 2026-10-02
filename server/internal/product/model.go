// Package product 实现商品模块（PRD 03，接口清单 #34 ～ #42、#73、#74）。
//
// 本迭代支持“链接”与“文本”两种交付类型；“文件”（分片上传）与“卡密”（导入与库存）在后续迭代实现。
package product

import (
	"time"

	"gorm.io/gorm"
)

const (
	StatusDraft         = "DRAFT"
	StatusPendingReview = "PENDING_REVIEW"
	StatusOnSale        = "ON_SALE"
	StatusOffSale       = "OFF_SALE"
	StatusBanned        = "BANNED"
)

const (
	DeliveryFile = "FILE"
	DeliveryLink = "LINK"
	DeliveryText = "TEXT"
	DeliveryCard = "CARD"
)

// SupportedDeliveryTypes 是当前可以创建的交付类型。
var SupportedDeliveryTypes = map[string]bool{DeliveryLink: true, DeliveryText: true}

// Category 是平台一级分类（PRD MKT-04）。
type Category struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

var Categories = []Category{
	{"design", "设计素材"},
	{"software", "软件工具"},
	{"course", "教程课程"},
	{"template", "效率模板"},
	{"membership", "会员兑换"},
	{"other", "其他"},
}

func validCategory(v string) bool {
	for _, c := range Categories {
		if c.Value == v {
			return true
		}
	}
	return false
}

type FAQ struct {
	Q string `json:"q"`
	A string `json:"a"`
}

// Detail 是结构化的商品描述（PRD-02 分组 3），平台按固定版式渲染。
type Detail struct {
	Includes []string `json:"includes"`
	Audience string   `json:"audience"`
	FAQs     []FAQ    `json:"faqs"`
	Notice   string   `json:"notice"`
}

type Link struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Code string `json:"code"`
}

// DeliveryConfig 是交付内容（链接、文本、卡密说明）与交付附言。只在支付成功后对买家可见。
type DeliveryConfig struct {
	Links            []Link `json:"links"`
	Text             string `json:"text"`
	CardInstructions string `json:"cardInstructions"`
	Note             string `json:"note"`
}

// Product 对应 products 表（docs/database-design.md 3.3）。
type Product struct {
	ID                uint64 `gorm:"primaryKey"`
	PublicID          string
	ShopID            uint64
	Name              string
	Tagline           string
	Price             int
	OriginalPrice     *int
	DeliveryType      string
	Category          *string
	Status            string
	DescriptionMD     *string        `gorm:"column:description_md"`
	Detail            Detail         `gorm:"serializer:json"`
	DeliveryConfig    DeliveryConfig `gorm:"serializer:json"`
	MaxPerOrder       int
	DownloadLimit     *int
	LowStockThreshold int
	StockAvailable    int
	SalesCount        int
	FavoriteCount     int
	SortOrder         int
	BanReason         *string
	Version           int
	PublishedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt
}

func (p *Product) Description() string {
	if p.DescriptionMD == nil {
		return ""
	}
	return *p.DescriptionMD
}

// Image 对应 product_images 表，sort_order 为 0 的是主图。
type Image struct {
	ID        uint64 `gorm:"primaryKey"`
	ProductID uint64
	ObjectKey string
	Width     int
	Height    int
	SortOrder int
	CreatedAt time.Time
}

func (Image) TableName() string { return "product_images" }

// ---------- 商家端视图 ----------

type ImageView struct {
	Key    string `json:"key"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// EditView 是编辑页使用的完整商品信息（接口 #36）。
type EditView struct {
	PublicID       string         `json:"publicId"`
	Name           string         `json:"name"`
	Tagline        string         `json:"tagline"`
	Category       *string        `json:"category"`
	Price          int            `json:"price"`
	OriginalPrice  *int           `json:"originalPrice"`
	DeliveryType   string         `json:"deliveryType"`
	Status         string         `json:"status"`
	DescriptionMD  string         `json:"descriptionMd"`
	Detail         Detail         `json:"detail"`
	DeliveryConfig DeliveryConfig `json:"deliveryConfig"`
	MaxPerOrder    int            `json:"maxPerOrder"`
	Images         []ImageView    `json:"images"`
	SalesCount     int            `json:"salesCount"`
	BanReason      *string        `json:"banReason"`
	Version        int            `json:"version"`
	// DeliveryTypeLocked 为 true 表示商品上架过，不能再修改交付类型
	DeliveryTypeLocked bool       `json:"deliveryTypeLocked"`
	PublishedAt        *time.Time `json:"publishedAt"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// ListItem 是商家商品列表的一行（接口 #34）。
type ListItem struct {
	PublicID      string     `json:"publicId"`
	Name          string     `json:"name"`
	Cover         *ImageView `json:"cover"`
	Category      *string    `json:"category"`
	DeliveryType  string     `json:"deliveryType"`
	Price         int        `json:"price"`
	OriginalPrice *int       `json:"originalPrice"`
	Status        string     `json:"status"`
	SalesCount    int        `json:"salesCount"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// Issue 是上架检查未通过的一项（错误码 40001 的 data.issues）。
type Issue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ---------- 买家端视图（与 web/src/lib/storefront/types.ts 对应） ----------

type PublicImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Alt    string `json:"alt"`
}

// Summary 是商品卡片（店铺页、商城首页、搜索结果）。
type Summary struct {
	PublicID      string      `json:"publicId"`
	Name          string      `json:"name"`
	Price         int         `json:"price"`
	OriginalPrice *int        `json:"originalPrice"`
	Cover         PublicImage `json:"cover"`
	SoldOut       bool        `json:"soldOut"`
	IsNew         bool        `json:"isNew"`
	FavoriteCount int         `json:"favoriteCount"`
}

type PublicDelivery struct {
	Type      string `json:"type"`
	FileCount *int   `json:"fileCount"`
	TotalSize *int64 `json:"totalSize"`
}

// Public 是商品详情（接口 #74）。不包含交付内容。
type Public struct {
	Summary
	Tagline       string         `json:"tagline"`
	Status        string         `json:"status"`
	Category      *string        `json:"category"`
	Images        []PublicImage  `json:"images"`
	DescriptionMD string         `json:"descriptionMd"`
	Detail        Detail         `json:"detail"`
	Delivery      PublicDelivery `json:"delivery"`
	MaxPerOrder   int            `json:"maxPerOrder"`
	StockHint     *int           `json:"stockHint"`
	Purchasable   bool           `json:"purchasable"`
}
