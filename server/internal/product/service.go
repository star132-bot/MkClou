package product

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/response"
	"mkclou/server/internal/shop"
)

// newBadgeWindow 是“新品”角标的时长（PRD SF-01：上架 7 天内）。
const newBadgeWindow = 7 * 24 * time.Hour

// Shops 提供店铺信息，由店铺模块实现。
type Shops interface {
	Seller(ctx context.Context, merchantID uint64) (*shop.Ref, error)
	RefByID(ctx context.Context, id uint64) (*shop.Ref, error)
	RefBySlug(ctx context.Context, slug string) (*shop.Ref, error)
}

// Images 校验与认领上传的图片，由上传模块实现。
type Images interface {
	OwnedBy(ctx context.Context, shopID uint64, key string) (bool, error)
	Claim(ctx context.Context, keys ...string)
}

// Assets 把对象 key 转换为访问地址。
type Assets interface {
	PublicURL(key string) string
}

type Service struct {
	repo   Repository
	shops  Shops
	images Images
	assets Assets
	log    *zap.Logger
	now    func() time.Time
}

type Deps struct {
	Repo   Repository
	Shops  Shops
	Images Images
	Assets Assets
	Log    *zap.Logger
}

func NewService(d Deps) *Service {
	return &Service{repo: d.Repo, shops: d.Shops, images: d.Images, assets: d.Assets, log: d.Log, now: time.Now}
}

// ---------- 创建与查询 ----------

type CreateInput struct {
	Name         string
	DeliveryType string
	Price        int
}

// Create 创建商品草稿（接口 #35）。
func (s *Service) Create(ctx context.Context, merchantID uint64, in CreateInput) (*EditView, error) {
	seller, err := s.writableShop(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if err := validateName(name); err != nil {
		return nil, err
	}
	if err := checkDeliveryType(in.DeliveryType); err != nil {
		return nil, err
	}
	if in.Price < 0 || in.Price > maxPrice {
		return nil, fieldError("price", "价格为 0（免费）或 ¥0.01～¥50,000.00")
	}

	limit := 5
	p := &Product{
		ShopID:            seller.ID,
		Name:              name,
		Price:             in.Price,
		DeliveryType:      in.DeliveryType,
		Status:            StatusDraft,
		Detail:            Detail{Includes: []string{}, FAQs: []FAQ{}},
		DeliveryConfig:    DeliveryConfig{Links: []Link{}},
		MaxPerOrder:       1,
		DownloadLimit:     &limit,
		LowStockThreshold: 5,
		Version:           1,
	}
	// 对外 ID 随机生成，极小概率冲突时重试
	for range 3 {
		p.PublicID = newPublicID()
		if err = s.repo.Create(ctx, p); !errors.Is(err, errDuplicate) {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return s.editView(p, nil), nil
}

func checkDeliveryType(t string) error {
	switch {
	case SupportedDeliveryTypes[t]:
		return nil
	case t == DeliveryFile || t == DeliveryCard:
		return fieldError("deliveryType", "文件和卡密交付即将支持，目前可以选择链接或文本")
	default:
		return fieldError("deliveryType", "请选择交付类型")
	}
}

const publicIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// newPublicID 生成对外 ID：p_ + 12 位 base62（总览 5.3）。
func newPublicID() string {
	b := make([]byte, 12)
	max := big.NewInt(int64(len(publicIDAlphabet)))
	for i := range b {
		n, _ := rand.Int(rand.Reader, max)
		b[i] = publicIDAlphabet[n.Int64()]
	}
	return "p_" + string(b)
}

// writableShop 返回可以管理商品的店铺：已开店且未被封禁。
func (s *Service) writableShop(ctx context.Context, merchantID uint64) (*shop.Ref, error) {
	seller, err := s.shops.Seller(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if seller.Status == shop.StatusBanned {
		return nil, errcode.StateConflict.WithMessage("店铺已被封禁，无法管理商品")
	}
	return seller, nil
}

func (s *Service) mine(ctx context.Context, merchantID uint64, publicID string) (*shop.Ref, *Product, error) {
	seller, err := s.writableShop(ctx, merchantID)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.repo.Find(ctx, seller.ID, publicID)
	if errors.Is(err, errNotFound) {
		return nil, nil, errcode.NotFound.WithMessage("商品不存在或已删除")
	}
	return seller, p, err
}

// Get 返回编辑页的商品信息（接口 #36）。
func (s *Service) Get(ctx context.Context, merchantID uint64, publicID string) (*EditView, error) {
	_, p, err := s.mine(ctx, merchantID, publicID)
	if err != nil {
		return nil, err
	}
	imgs, err := s.repo.Images(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	return s.editView(p, imgs[p.ID]), nil
}

var listStatuses = map[string]bool{StatusOnSale: true, StatusDraft: true, StatusOffSale: true, StatusPendingReview: true, StatusBanned: true}

// List 返回商家的商品列表（接口 #34）。
func (s *Service) List(ctx context.Context, merchantID uint64, f ListFilter) (*response.Page[ListItem], error) {
	seller, err := s.shops.Seller(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if f.Status != "" && !listStatuses[f.Status] {
		f.Status = ""
	}
	f.Query = strings.TrimSpace(f.Query)
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	items, total, err := s.repo.List(ctx, seller.ID, f)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, len(items))
	for i, p := range items {
		ids[i] = p.ID
	}
	imgs, err := s.repo.Images(ctx, ids...)
	if err != nil {
		return nil, err
	}
	out := make([]ListItem, len(items))
	for i, p := range items {
		out[i] = ListItem{
			PublicID: p.PublicID, Name: p.Name, Category: p.Category, DeliveryType: p.DeliveryType,
			Price: p.Price, OriginalPrice: p.OriginalPrice, Status: p.Status, SalesCount: p.SalesCount,
			UpdatedAt: p.UpdatedAt.UTC(),
		}
		if list := imgs[p.ID]; len(list) > 0 {
			v := s.imageView(list[0])
			out[i].Cover = &v
		}
	}
	return &response.Page[ListItem]{Items: out, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

// ---------- 保存 ----------

type ImageInput struct {
	Key    string `json:"key"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// UpdateInput 是保存商品的完整内容（接口 #37，PUT 语义）。
type UpdateInput struct {
	Version        int
	Name           string
	Tagline        string
	Category       *string
	Price          int
	OriginalPrice  *int
	DeliveryType   string
	DescriptionMD  string
	Detail         Detail
	DeliveryConfig DeliveryConfig
	MaxPerOrder    int
	Images         []ImageInput
}

// Update 保存商品（PRD-02）。已上架的商品修改后需仍满足上架条件，修改即时生效；
// 内容命中敏感词时转为待审核。
func (s *Service) Update(ctx context.Context, merchantID uint64, publicID string, in UpdateInput) (*EditView, error) {
	seller, p, err := s.mine(ctx, merchantID, publicID)
	if err != nil {
		return nil, err
	}
	if in.DeliveryType == "" {
		in.DeliveryType = p.DeliveryType
	}
	if in.DeliveryType != p.DeliveryType {
		if p.PublishedAt != nil {
			return nil, errcode.DeliveryTypeLocked
		}
		if err := checkDeliveryType(in.DeliveryType); err != nil {
			return nil, err
		}
	}
	in.normalize(in.DeliveryType)
	if err := in.validate(); err != nil {
		return nil, err
	}

	current, err := s.repo.Images(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	images, newKeys, err := s.resolveImages(ctx, seller.ID, p.ID, current[p.ID], in.Images)
	if err != nil {
		return nil, err
	}

	next := *p
	next.Name, next.Tagline, next.Category = in.Name, in.Tagline, in.Category
	next.Price, next.OriginalPrice, next.DeliveryType = in.Price, in.OriginalPrice, in.DeliveryType
	next.DescriptionMD = &in.DescriptionMD
	next.Detail, next.DeliveryConfig, next.MaxPerOrder = in.Detail, in.DeliveryConfig, in.MaxPerOrder

	fields := map[string]any{
		"name": next.Name, "tagline": next.Tagline, "category": next.Category,
		"price": next.Price, "original_price": next.OriginalPrice, "delivery_type": next.DeliveryType,
		"description_md": next.DescriptionMD, "detail": jsonValue(next.Detail),
		"delivery_config": jsonValue(next.DeliveryConfig), "max_per_order": next.MaxPerOrder,
	}
	// 已上架（或待审核）的商品保存后仍要满足上架条件，避免买家看到不完整的商品
	if p.Status == StatusOnSale || p.Status == StatusPendingReview {
		if issues := s.publishIssues(seller, &next, len(images)); len(issues) > 0 {
			return nil, errcode.PublishCheckFailed.WithMessage("已上架的商品需要保持内容完整，如需大幅修改请先下架").
				WithData(map[string]any{"issues": issues})
		}
		status, err := s.reviewedStatus(ctx, &next)
		if err != nil {
			return nil, err
		}
		fields["status"], next.Status = status, status
	}

	if err := s.repo.Save(ctx, p.ID, in.Version, fields, images); err != nil {
		if errors.Is(err, errVersion) {
			return nil, errcode.VersionConflict.WithMessage("商品已在其他页面被修改，请刷新后重试")
		}
		return nil, err
	}
	s.images.Claim(ctx, newKeys...)
	next.Version = in.Version + 1
	next.UpdatedAt = s.now()
	return s.editView(&next, images), nil
}

// resolveImages 校验封面图：已属于该商品的图片直接保留，新图片必须是本店铺最近上传的。
func (s *Service) resolveImages(ctx context.Context, shopID, productID uint64, current []Image, in []ImageInput) ([]Image, []string, error) {
	existing := make(map[string]Image, len(current))
	for _, img := range current {
		existing[img.ObjectKey] = img
	}
	seen := map[string]bool{}
	out := make([]Image, 0, len(in))
	var newKeys []string
	now := s.now()
	for i, img := range in {
		if seen[img.Key] {
			continue
		}
		seen[img.Key] = true
		if old, ok := existing[img.Key]; ok {
			out = append(out, Image{ProductID: productID, ObjectKey: old.ObjectKey, Width: old.Width, Height: old.Height, SortOrder: i, CreatedAt: now})
			continue
		}
		owned, err := s.images.OwnedBy(ctx, shopID, img.Key)
		if err != nil {
			return nil, nil, err
		}
		if !owned || img.Width <= 0 || img.Height <= 0 {
			return nil, nil, fieldError("images", "图片已失效，请重新上传")
		}
		out = append(out, Image{ProductID: productID, ObjectKey: img.Key, Width: img.Width, Height: img.Height, SortOrder: i, CreatedAt: now})
		newKeys = append(newKeys, img.Key)
	}
	for i := range out {
		out[i].SortOrder = i
	}
	return out, newKeys, nil
}

// jsonValue 把 JSON 列的值序列化为字符串。按 map 更新时 GORM 不会调用字段的 serializer。
func jsonValue(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// ---------- 上架、下架、删除 ----------

// publishIssues 汇总上架检查（PRD-08）：账号、收款与内容完整性。
func (s *Service) publishIssues(seller *shop.Ref, p *Product, imageCount int) []Issue {
	var issues []Issue
	if !seller.EmailVerified {
		issues = append(issues, Issue{"account", "请先验证邮箱"})
	}
	if p.Price > 0 && !seller.PaymentReady {
		issues = append(issues, Issue{"payment", "请先完成收款设置（免费商品无需设置）"})
	}
	return append(issues, completenessIssues(p, imageCount)...)
}

// reviewedStatus 是通过完整性检查后的状态：内容命中敏感词进入待审核，否则直接上架（PRD MKT 第 3 节）。
func (s *Service) reviewedStatus(ctx context.Context, p *Product) (string, error) {
	blocked, err := s.repo.ContainsBlockedWord(ctx, reviewText(p))
	if err != nil {
		return "", err
	}
	if blocked {
		return StatusPendingReview, nil
	}
	return StatusOnSale, nil
}

// Publish 上架商品（接口 #38）。
func (s *Service) Publish(ctx context.Context, merchantID uint64, publicID string) (*EditView, error) {
	seller, p, err := s.mine(ctx, merchantID, publicID)
	if err != nil {
		return nil, err
	}
	switch p.Status {
	case StatusOnSale, StatusPendingReview:
		return s.Get(ctx, merchantID, publicID)
	case StatusBanned:
		return nil, errcode.StateConflict.WithMessage("商品已被平台下架，请修改后联系平台申诉")
	}
	imgs, err := s.repo.Images(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if issues := s.publishIssues(seller, p, len(imgs[p.ID])); len(issues) > 0 {
		return nil, errcode.PublishCheckFailed.WithData(map[string]any{"issues": issues})
	}
	status, err := s.reviewedStatus(ctx, p)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{"status": status}
	if p.PublishedAt == nil {
		now := s.now()
		fields["published_at"], p.PublishedAt = now, &now
	}
	if err := s.repo.UpdateFields(ctx, p.ID, fields); err != nil {
		return nil, err
	}
	p.Status = status
	return s.editView(p, imgs[p.ID]), nil
}

// Unpublish 下架商品（接口 #39）。未支付订单可继续完成支付（订单模块处理）。
func (s *Service) Unpublish(ctx context.Context, merchantID uint64, publicID string) (*EditView, error) {
	_, p, err := s.mine(ctx, merchantID, publicID)
	if err != nil {
		return nil, err
	}
	switch p.Status {
	case StatusOffSale:
		return s.Get(ctx, merchantID, publicID)
	case StatusOnSale, StatusPendingReview:
	default:
		return nil, errcode.StateConflict.WithMessage("只有已上架的商品可以下架")
	}
	if err := s.repo.UpdateFields(ctx, p.ID, map[string]any{"status": StatusOffSale}); err != nil {
		return nil, err
	}
	return s.Get(ctx, merchantID, publicID)
}

// Delete 软删除商品（接口 #42，PRD-15）：上架中的商品需先下架，有订单的商品不能删除。
func (s *Service) Delete(ctx context.Context, merchantID uint64, publicID string) error {
	_, p, err := s.mine(ctx, merchantID, publicID)
	if err != nil {
		return err
	}
	if p.Status == StatusOnSale || p.Status == StatusPendingReview {
		return errcode.StateConflict.WithMessage("请先下架商品再删除")
	}
	hasOrders, err := s.repo.HasOrders(ctx, p.ID)
	if err != nil {
		return err
	}
	if hasOrders {
		return errcode.ProductHasOrders
	}
	return s.repo.SoftDelete(ctx, p.ID)
}

// ---------- 买家端 ----------

// PublicPage 是商品详情页的数据（接口 #74）。
type PublicPage struct {
	Shop         *shop.PublicView `json:"shop"`
	Product      *Public          `json:"product"`
	MoreFromShop []Summary        `json:"moreFromShop"`
}

// PublicDetail 返回买家看到的商品。未上架的商品只返回名称与状态，页面显示“商品已下架”。
func (s *Service) PublicDetail(ctx context.Context, publicID string) (*PublicPage, error) {
	notFound := errcode.NotFound.WithMessage("商品不存在")
	p, err := s.repo.FindPublic(ctx, publicID)
	if errors.Is(err, errNotFound) {
		return nil, notFound
	}
	if err != nil {
		return nil, err
	}
	ref, err := s.shops.RefByID(ctx, p.ShopID)
	if err != nil {
		return nil, err
	}
	if ref.Status == shop.StatusBanned {
		return nil, notFound
	}
	if p.Status != StatusOnSale {
		return &PublicPage{
			Shop:         ref.Public,
			Product:      &Public{Summary: Summary{PublicID: p.PublicID, Name: p.Name}, Status: StatusOffSale, Images: []PublicImage{}},
			MoreFromShop: []Summary{},
		}, nil
	}

	imgs, err := s.repo.Images(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	more, err := s.repo.OnSaleByShop(ctx, p.ShopID, 0, 6, p.ID)
	if err != nil {
		return nil, err
	}
	moreSummaries, err := s.summaries(ctx, more)
	if err != nil {
		return nil, err
	}

	pub := &Public{
		Summary:       s.summary(p, imgs[p.ID]),
		Tagline:       p.Tagline,
		Status:        p.Status,
		Category:      p.Category,
		Images:        make([]PublicImage, 0, len(imgs[p.ID])),
		DescriptionMD: p.Description(),
		Detail:        p.Detail,
		Delivery:      PublicDelivery{Type: p.DeliveryType},
		MaxPerOrder:   p.MaxPerOrder,
		Purchasable:   p.Price == 0 || ref.PaymentReady,
	}
	for i, img := range imgs[p.ID] {
		pub.Images = append(pub.Images, s.publicImage(img, p.Name, i))
	}
	if pub.Detail.Includes == nil {
		pub.Detail.Includes = []string{}
	}
	if pub.Detail.FAQs == nil {
		pub.Detail.FAQs = []FAQ{}
	}
	return &PublicPage{Shop: ref.Public, Product: pub, MoreFromShop: moreSummaries}, nil
}

// ShopProducts 返回店铺已上架的商品（接口 #73，游标分页）。
func (s *Service) ShopProducts(ctx context.Context, slug, cursor string, limit int) (items []Summary, next string, err error) {
	ref, err := s.shops.RefBySlug(ctx, slug)
	if err != nil {
		return nil, "", err
	}
	if ref.Status == shop.StatusBanned {
		return []Summary{}, "", nil
	}
	if limit < 1 || limit > 48 {
		limit = 24
	}
	offset := decodeCursor(cursor)
	list, err := s.repo.OnSaleByShop(ctx, ref.ID, offset, limit+1, 0)
	if err != nil {
		return nil, "", err
	}
	if len(list) > limit {
		list = list[:limit]
		next = encodeCursor(offset + limit)
	}
	items, err = s.summaries(ctx, list)
	return items, next, err
}

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(c string) int {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(string(b))
	if err != nil || n < 0 || n > 100_000 {
		return 0
	}
	return n
}

// Summaries 把商品转换为卡片，供商城首页与搜索复用。
func (s *Service) summaries(ctx context.Context, list []Product) ([]Summary, error) {
	ids := make([]uint64, len(list))
	for i, p := range list {
		ids[i] = p.ID
	}
	imgs, err := s.repo.Images(ctx, ids...)
	if err != nil {
		return nil, err
	}
	out := make([]Summary, len(list))
	for i := range list {
		out[i] = s.summary(&list[i], imgs[list[i].ID])
	}
	return out, nil
}

func (s *Service) summary(p *Product, imgs []Image) Summary {
	sm := Summary{
		PublicID:      p.PublicID,
		Name:          p.Name,
		Price:         p.Price,
		OriginalPrice: p.OriginalPrice,
		SoldOut:       p.DeliveryType == DeliveryCard && p.StockAvailable == 0,
		IsNew:         p.PublishedAt != nil && s.now().Sub(*p.PublishedAt) < newBadgeWindow,
		FavoriteCount: p.FavoriteCount,
		Cover:         PublicImage{Alt: p.Name},
	}
	if len(imgs) > 0 {
		sm.Cover = s.publicImage(imgs[0], p.Name, 0)
	}
	return sm
}

func (s *Service) publicImage(img Image, name string, i int) PublicImage {
	alt := name
	if i > 0 {
		alt = name + " 图 " + strconv.Itoa(i+1)
	}
	return PublicImage{URL: s.assets.PublicURL(img.ObjectKey), Width: img.Width, Height: img.Height, Alt: alt}
}

// ---------- 视图 ----------

func (s *Service) imageView(img Image) ImageView {
	return ImageView{Key: img.ObjectKey, URL: s.assets.PublicURL(img.ObjectKey), Width: img.Width, Height: img.Height}
}

func (s *Service) editView(p *Product, imgs []Image) *EditView {
	v := &EditView{
		PublicID:           p.PublicID,
		Name:               p.Name,
		Tagline:            p.Tagline,
		Category:           p.Category,
		Price:              p.Price,
		OriginalPrice:      p.OriginalPrice,
		DeliveryType:       p.DeliveryType,
		Status:             p.Status,
		DescriptionMD:      p.Description(),
		Detail:             p.Detail,
		DeliveryConfig:     p.DeliveryConfig,
		MaxPerOrder:        p.MaxPerOrder,
		Images:             make([]ImageView, len(imgs)),
		SalesCount:         p.SalesCount,
		BanReason:          p.BanReason,
		Version:            p.Version,
		DeliveryTypeLocked: p.PublishedAt != nil,
		PublishedAt:        utc(p.PublishedAt),
		CreatedAt:          p.CreatedAt.UTC(),
		UpdatedAt:          p.UpdatedAt.UTC(),
	}
	for i, img := range imgs {
		v.Images[i] = s.imageView(img)
	}
	if v.Detail.Includes == nil {
		v.Detail.Includes = []string{}
	}
	if v.Detail.FAQs == nil {
		v.Detail.FAQs = []FAQ{}
	}
	if v.DeliveryConfig.Links == nil {
		v.DeliveryConfig.Links = []Link{}
	}
	return v
}

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
