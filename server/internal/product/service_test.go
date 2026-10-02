package product

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/shop"
)

// ---------- 测试替身 ----------

type fakeRepo struct {
	mu       sync.Mutex
	nextID   uint64
	products map[uint64]*Product
	images   map[uint64][]Image
	orders   map[uint64]bool
	blocked  []string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{products: map[uint64]*Product{}, images: map[uint64][]Image{}, orders: map[uint64]bool{}}
}

func (r *fakeRepo) Create(_ context.Context, p *Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	p.ID = r.nextID
	p.CreatedAt, p.UpdatedAt = time.Now(), time.Now()
	cp := *p
	r.products[p.ID] = &cp
	return nil
}

func (r *fakeRepo) get(match func(*Product) bool) (*Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.products {
		if p.DeletedAt.Valid {
			continue
		}
		if match(p) {
			cp := *p
			return &cp, nil
		}
	}
	return nil, errNotFound
}

func (r *fakeRepo) Find(_ context.Context, shopID uint64, publicID string) (*Product, error) {
	return r.get(func(p *Product) bool { return p.ShopID == shopID && p.PublicID == publicID })
}

func (r *fakeRepo) FindPublic(_ context.Context, publicID string) (*Product, error) {
	return r.get(func(p *Product) bool { return p.PublicID == publicID })
}

func (r *fakeRepo) List(_ context.Context, shopID uint64, f ListFilter) ([]Product, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Product
	for _, p := range r.products {
		if p.ShopID == shopID && !p.DeletedAt.Valid && (f.Status == "" || p.Status == f.Status) &&
			(f.Query == "" || strings.Contains(p.Name, f.Query)) {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, int64(len(out)), nil
}

func (r *fakeRepo) Images(_ context.Context, ids ...uint64) (map[uint64][]Image, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[uint64][]Image{}
	for _, id := range ids {
		out[id] = append([]Image(nil), r.images[id]...)
	}
	return out, nil
}

func (r *fakeRepo) Save(_ context.Context, id uint64, version int, fields map[string]any, images []Image) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.products[id]
	if p.Version != version {
		return errVersion
	}
	p.Version++
	apply(p, fields)
	if images != nil {
		r.images[id] = images
	}
	return nil
}

func (r *fakeRepo) UpdateFields(_ context.Context, id uint64, fields map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	apply(r.products[id], fields)
	return nil
}

func apply(p *Product, f map[string]any) {
	for k, v := range f {
		switch k {
		case "name":
			p.Name = v.(string)
		case "tagline":
			p.Tagline = v.(string)
		case "price":
			p.Price = v.(int)
		case "category":
			p.Category = v.(*string)
		case "delivery_type":
			p.DeliveryType = v.(string)
		case "status":
			p.Status = v.(string)
		case "published_at":
			t := v.(time.Time)
			p.PublishedAt = &t
		case "description_md":
			p.DescriptionMD = v.(*string)
		case "delivery_config":
			// 测试只关心是否配置了交付内容
			p.DeliveryConfig.Links = nil
			p.DeliveryConfig.Text = ""
			if strings.Contains(v.(string), `"url"`) {
				p.DeliveryConfig.Links = []Link{{URL: "https://x"}}
			}
			if strings.Contains(v.(string), `"text":"`) && !strings.Contains(v.(string), `"text":""`) {
				p.DeliveryConfig.Text = "t"
			}
		}
	}
}

func (r *fakeRepo) SoftDelete(_ context.Context, id uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[id].DeletedAt.Valid = true
	return nil
}

func (r *fakeRepo) HasOrders(_ context.Context, id uint64) (bool, error) { return r.orders[id], nil }

func (r *fakeRepo) OnSaleByShop(_ context.Context, shopID uint64, offset, limit int, exclude uint64) ([]Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Product
	for _, p := range r.products {
		if p.ShopID == shopID && p.Status == StatusOnSale && p.ID != exclude && !p.DeletedAt.Valid {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if offset >= len(out) {
		return nil, nil
	}
	out = out[offset:]
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *fakeRepo) ContainsBlockedWord(_ context.Context, text string) (bool, error) {
	for _, w := range r.blocked {
		if strings.Contains(text, w) {
			return true, nil
		}
	}
	return false, nil
}

type fakeShops struct{ refs map[uint64]*shop.Ref } // merchantID == shopID

func (f *fakeShops) Seller(_ context.Context, mid uint64) (*shop.Ref, error) {
	if r, ok := f.refs[mid]; ok {
		return r, nil
	}
	return nil, errcode.ShopNotCreated
}

func (f *fakeShops) RefByID(_ context.Context, id uint64) (*shop.Ref, error) {
	if r, ok := f.refs[id]; ok {
		return r, nil
	}
	return nil, errcode.NotFound
}

func (f *fakeShops) RefBySlug(_ context.Context, slug string) (*shop.Ref, error) {
	for _, r := range f.refs {
		if r.Slug == slug {
			return r, nil
		}
	}
	return nil, errcode.NotFound
}

type fakeImages struct{ owned map[string]uint64 }

func (f *fakeImages) OwnedBy(_ context.Context, shopID uint64, key string) (bool, error) {
	if strings.HasPrefix(key, "img/s1-") { // complete() 每次生成的新图片都属于店铺 1
		return shopID == 1, nil
	}
	return f.owned[key] == shopID, nil
}

func (f *fakeImages) Claim(_ context.Context, keys ...string) {
	for _, k := range keys {
		delete(f.owned, k)
	}
}

type fakeAssets struct{}

func (fakeAssets) PublicURL(key string) string { return "http://assets.test/" + key }

type env struct {
	svc    *Service
	repo   *fakeRepo
	shops  *fakeShops
	images *fakeImages
}

func setup(t *testing.T) *env {
	t.Helper()
	repo := newFakeRepo()
	shops := &fakeShops{refs: map[uint64]*shop.Ref{
		1: {ID: 1, Slug: "one", Status: shop.StatusOpen, EmailVerified: true, PaymentReady: true, Public: &shop.PublicView{Slug: "one"}},
		2: {ID: 2, Slug: "two", Status: shop.StatusOpen, EmailVerified: false, PaymentReady: false, Public: &shop.PublicView{Slug: "two"}},
	}}
	images := &fakeImages{owned: map[string]uint64{"img/a.jpg": 1, "img/b.jpg": 1, "img/other.jpg": 2}}
	svc := NewService(Deps{Repo: repo, Shops: shops, Images: images, Assets: fakeAssets{}, Log: zap.NewNop()})
	return &env{svc: svc, repo: repo, shops: shops, images: images}
}

func ctx() context.Context { return context.Background() }

func ptr[T any](v T) *T { return &v }

func requireCode(t *testing.T, err error, want *errcode.Error) *errcode.Error {
	t.Helper()
	var e *errcode.Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, want.Code, e.Code, e.Message)
	return e
}

func issueFields(e *errcode.Error) []string {
	issues := e.Data.(map[string]any)["issues"].([]Issue)
	out := make([]string, len(issues))
	for i, is := range issues {
		out[i] = is.Field
	}
	return out
}

var imageSeq atomic.Int64

// complete 是一份可以上架的完整内容，每次使用一张新上传的图片。
func complete(version int) UpdateInput {
	key := fmt.Sprintf("img/s1-%d.jpg", imageSeq.Add(1))
	return UpdateInput{
		Version: version, Name: "Figma 模板包", Tagline: "3 套模板", Category: ptr("design"), Price: 2990,
		DeliveryType: DeliveryLink, DescriptionMD: "## 介绍\n好用的模板",
		DeliveryConfig: DeliveryConfig{Links: []Link{{Name: "网盘", URL: "https://pan.example.com/s/1", Code: "ab12"}}},
		MaxPerOrder:    1,
		Images:         []ImageInput{{Key: key, Width: 800, Height: 600}},
	}
}

func (e *env) draft(t *testing.T, merchant uint64) *EditView {
	t.Helper()
	v, err := e.svc.Create(ctx(), merchant, CreateInput{Name: "新商品", DeliveryType: DeliveryLink, Price: 2990})
	require.NoError(t, err)
	return v
}

// ---------- 创建 ----------

func TestCreate(t *testing.T) {
	e := setup(t)
	v := e.draft(t, 1)
	assert.Regexp(t, `^p_[0-9A-Za-z]{12}$`, v.PublicID)
	assert.Equal(t, StatusDraft, v.Status)
	assert.Equal(t, 1, v.Version)
	assert.False(t, v.DeliveryTypeLocked)
	assert.NotNil(t, v.Images)

	_, err := e.svc.Create(ctx(), 1, CreateInput{Name: "文件商品", DeliveryType: DeliveryFile})
	requireCode(t, err, errcode.InvalidParams)
	_, err = e.svc.Create(ctx(), 1, CreateInput{Name: "x", DeliveryType: DeliveryText})
	requireCode(t, err, errcode.InvalidParams)
	_, err = e.svc.Create(ctx(), 1, CreateInput{Name: "价格错误", DeliveryType: DeliveryText, Price: -1})
	requireCode(t, err, errcode.InvalidParams)
	_, err = e.svc.Create(ctx(), 9, CreateInput{Name: "没有店铺", DeliveryType: DeliveryText})
	requireCode(t, err, errcode.ShopNotCreated)

	e.shops.refs[1].Status = shop.StatusBanned
	_, err = e.svc.Create(ctx(), 1, CreateInput{Name: "封禁店铺", DeliveryType: DeliveryText})
	requireCode(t, err, errcode.StateConflict)
}

// ---------- 保存 ----------

func TestUpdateAndImages(t *testing.T) {
	e := setup(t)
	d := e.draft(t, 1)

	in := complete(d.Version)
	in.Images = []ImageInput{{Key: "img/a.jpg", Width: 800, Height: 600}, {Key: "img/b.jpg", Width: 600, Height: 600}, {Key: "img/a.jpg", Width: 1, Height: 1}}
	v, err := e.svc.Update(ctx(), 1, d.PublicID, in)
	require.NoError(t, err)
	assert.Equal(t, 2, v.Version)
	require.Len(t, v.Images, 2, "重复的图片只保留一次")
	assert.Equal(t, "http://assets.test/img/a.jpg", v.Images[0].URL)
	assert.Empty(t, e.images.owned["img/a.jpg"], "被引用后认领，不再被当作未使用的图片")

	// 已属于该商品的图片可以继续保留和重排
	in = complete(v.Version)
	in.Images = []ImageInput{{Key: "img/b.jpg"}, {Key: "img/a.jpg"}}
	v, err = e.svc.Update(ctx(), 1, d.PublicID, in)
	require.NoError(t, err)
	assert.Equal(t, "img/b.jpg", v.Images[0].Key)
	assert.Equal(t, 600, v.Images[0].Width, "保留原图的宽高")

	// 其他店铺上传的图片不能使用
	in = complete(v.Version)
	in.Images = []ImageInput{{Key: "img/other.jpg", Width: 10, Height: 10}}
	_, err = e.svc.Update(ctx(), 1, d.PublicID, in)
	requireCode(t, err, errcode.InvalidParams)

	// 版本不一致（其他页面已保存）
	_, err = e.svc.Update(ctx(), 1, d.PublicID, complete(1))
	requireCode(t, err, errcode.VersionConflict)

	// 其他商家看不到这个商品
	_, err = e.svc.Get(ctx(), 2, d.PublicID)
	requireCode(t, err, errcode.NotFound)
}

func TestUpdateValidation(t *testing.T) {
	e := setup(t)
	d := e.draft(t, 1)
	cases := []struct {
		mutate func(*UpdateInput)
		field  string
	}{
		{func(in *UpdateInput) { in.Name = "a" }, "name"},
		{func(in *UpdateInput) { in.Tagline = strings.Repeat("长", 81) }, "tagline"},
		{func(in *UpdateInput) { in.Category = ptr("food") }, "category"},
		{func(in *UpdateInput) { in.Price = 5_000_001 }, "price"},
		{func(in *UpdateInput) { in.OriginalPrice = ptr(100) }, "originalPrice"},
		{func(in *UpdateInput) { in.DescriptionMD = strings.Repeat("字", 5001) }, "descriptionMd"},
		{func(in *UpdateInput) { in.Detail.Includes = make([]string, 11); fill(in.Detail.Includes) }, "detail.includes"},
		{func(in *UpdateInput) { in.Detail.FAQs = []FAQ{{Q: "只有问题"}} }, "detail.faqs"},
		{func(in *UpdateInput) { in.DeliveryConfig.Links = []Link{{URL: "javascript:alert(1)"}} }, "deliveryConfig.links"},
		{func(in *UpdateInput) { in.MaxPerOrder = 21 }, "maxPerOrder"},
		{func(in *UpdateInput) {
			in.Images = make([]ImageInput, 7)
			for i := range in.Images {
				in.Images[i] = ImageInput{Key: strings.Repeat("k", i+1)}
			}
		}, "images"},
	}
	for i, c := range cases {
		in := complete(d.Version)
		c.mutate(&in)
		_, err := e.svc.Update(ctx(), 1, d.PublicID, in)
		ce := requireCode(t, err, errcode.InvalidParams)
		assert.Contains(t, ce.Data.(map[string]any)["fields"], c.field, "case %d", i)
	}
}

func fill(s []string) {
	for i := range s {
		s[i] = "项"
	}
}

func TestNormalizeDropsIrrelevantDelivery(t *testing.T) {
	in := UpdateInput{
		DeliveryConfig: DeliveryConfig{Links: []Link{{URL: " https://a.com "}, {URL: "  "}}, Text: "文本", Note: " 附言 "},
		Detail:         Detail{Includes: []string{" a ", ""}, FAQs: []FAQ{{Q: " ", A: ""}, {Q: "q", A: "a"}}},
	}
	in.normalize(DeliveryLink)
	assert.Equal(t, []Link{{URL: "https://a.com"}}, in.DeliveryConfig.Links)
	assert.Empty(t, in.DeliveryConfig.Text, "链接类型不保存文本内容")
	assert.Equal(t, "附言", in.DeliveryConfig.Note)
	assert.Equal(t, []string{"a"}, in.Detail.Includes)
	assert.Equal(t, []FAQ{{Q: "q", A: "a"}}, in.Detail.FAQs)
	assert.Equal(t, 1, in.MaxPerOrder)
}

// ---------- 上架 ----------

func TestPublishChecks(t *testing.T) {
	e := setup(t)
	d := e.draft(t, 1)

	_, err := e.svc.Publish(ctx(), 1, d.PublicID)
	ce := requireCode(t, err, errcode.PublishCheckFailed)
	assert.ElementsMatch(t, []string{"category", "images", "descriptionMd", "deliveryConfig.links"}, issueFields(ce))

	v, err := e.svc.Update(ctx(), 1, d.PublicID, complete(d.Version))
	require.NoError(t, err)
	v, err = e.svc.Publish(ctx(), 1, d.PublicID)
	require.NoError(t, err)
	assert.Equal(t, StatusOnSale, v.Status)
	require.NotNil(t, v.PublishedAt)
	assert.True(t, v.DeliveryTypeLocked)

	// 重复上架无副作用
	v2, err := e.svc.Publish(ctx(), 1, d.PublicID)
	require.NoError(t, err)
	assert.Equal(t, StatusOnSale, v2.Status)
}

func TestPublishRequiresAccountAndPayment(t *testing.T) {
	e := setup(t)
	d := e.draft(t, 2) // 店铺 2：邮箱未验证、收款未配置
	in := complete(d.Version)
	in.Images = []ImageInput{{Key: "img/other.jpg", Width: 10, Height: 10}}
	v, err := e.svc.Update(ctx(), 2, d.PublicID, in)
	require.NoError(t, err)

	_, err = e.svc.Publish(ctx(), 2, d.PublicID)
	ce := requireCode(t, err, errcode.PublishCheckFailed)
	assert.ElementsMatch(t, []string{"account", "payment"}, issueFields(ce))

	// 免费商品不需要收款设置
	in = complete(v.Version)
	in.Images = []ImageInput{{Key: "img/other.jpg"}}
	in.Price = 0
	_, err = e.svc.Update(ctx(), 2, d.PublicID, in)
	require.NoError(t, err)
	_, err = e.svc.Publish(ctx(), 2, d.PublicID)
	ce = requireCode(t, err, errcode.PublishCheckFailed)
	assert.Equal(t, []string{"account"}, issueFields(ce))
}

func TestSensitiveWordsGoToReview(t *testing.T) {
	e := setup(t)
	e.repo.blocked = []string{"违禁"}
	d := e.draft(t, 1)
	in := complete(d.Version)
	in.Tagline = "含有违禁词"
	_, err := e.svc.Update(ctx(), 1, d.PublicID, in)
	require.NoError(t, err)
	v, err := e.svc.Publish(ctx(), 1, d.PublicID)
	require.NoError(t, err)
	assert.Equal(t, StatusPendingReview, v.Status)

	// 修改后自动审核通过，转为上架
	in = complete(v.Version)
	v, err = e.svc.Update(ctx(), 1, d.PublicID, in)
	require.NoError(t, err)
	assert.Equal(t, StatusOnSale, v.Status)
}

func TestOnSaleEditsMustStayComplete(t *testing.T) {
	e := setup(t)
	d := e.draft(t, 1)
	v, err := e.svc.Update(ctx(), 1, d.PublicID, complete(d.Version))
	require.NoError(t, err)
	_, err = e.svc.Publish(ctx(), 1, d.PublicID)
	require.NoError(t, err)

	in := complete(v.Version)
	in.DeliveryConfig.Links = nil
	_, err = e.svc.Update(ctx(), 1, d.PublicID, in)
	ce := requireCode(t, err, errcode.PublishCheckFailed)
	assert.Equal(t, []string{"deliveryConfig.links"}, issueFields(ce))

	in = complete(v.Version)
	in.DeliveryType = DeliveryText
	in.DeliveryConfig = DeliveryConfig{Text: "说明"}
	_, err = e.svc.Update(ctx(), 1, d.PublicID, in)
	requireCode(t, err, errcode.DeliveryTypeLocked)
}

// ---------- 下架与删除 ----------

func TestUnpublishAndDelete(t *testing.T) {
	e := setup(t)
	d := e.draft(t, 1)
	_, err := e.svc.Unpublish(ctx(), 1, d.PublicID)
	requireCode(t, err, errcode.StateConflict)

	_, err = e.svc.Update(ctx(), 1, d.PublicID, complete(d.Version))
	require.NoError(t, err)
	_, err = e.svc.Publish(ctx(), 1, d.PublicID)
	require.NoError(t, err)

	requireCode(t, e.svc.Delete(ctx(), 1, d.PublicID), errcode.StateConflict)
	v, err := e.svc.Unpublish(ctx(), 1, d.PublicID)
	require.NoError(t, err)
	assert.Equal(t, StatusOffSale, v.Status)

	p, _ := e.repo.FindPublic(ctx(), d.PublicID)
	e.repo.orders[p.ID] = true
	requireCode(t, e.svc.Delete(ctx(), 1, d.PublicID), errcode.ProductHasOrders)
	e.repo.orders[p.ID] = false
	require.NoError(t, e.svc.Delete(ctx(), 1, d.PublicID))
	_, err = e.svc.Get(ctx(), 1, d.PublicID)
	requireCode(t, err, errcode.NotFound)
}

// ---------- 买家端 ----------

func TestPublicDetail(t *testing.T) {
	e := setup(t)
	d := e.draft(t, 1)
	_, err := e.svc.Update(ctx(), 1, d.PublicID, complete(d.Version))
	require.NoError(t, err)

	// 未上架：只返回名称与状态
	page, err := e.svc.PublicDetail(ctx(), d.PublicID)
	require.NoError(t, err)
	assert.Equal(t, StatusOffSale, page.Product.Status)
	assert.Empty(t, page.Product.DescriptionMD)

	_, err = e.svc.Publish(ctx(), 1, d.PublicID)
	require.NoError(t, err)
	other := e.draft(t, 1)
	v, err := e.svc.Update(ctx(), 1, other.PublicID, complete(other.Version))
	require.NoError(t, err)
	_, err = e.svc.Publish(ctx(), 1, v.PublicID)
	require.NoError(t, err)

	page, err = e.svc.PublicDetail(ctx(), d.PublicID)
	require.NoError(t, err)
	p := page.Product
	assert.Equal(t, StatusOnSale, p.Status)
	assert.Equal(t, "## 介绍\n好用的模板", p.DescriptionMD)
	assert.True(t, p.Purchasable)
	assert.True(t, p.IsNew)
	assert.Regexp(t, `^http://assets\.test/img/s1-\d+\.jpg$`, p.Cover.URL)
	assert.Equal(t, DeliveryLink, p.Delivery.Type)
	assert.NotNil(t, p.Detail.Includes)
	require.Len(t, page.MoreFromShop, 1, "同店其他商品不含当前商品")
	assert.Equal(t, other.PublicID, page.MoreFromShop[0].PublicID)

	e.shops.refs[1].PaymentReady = false
	page, err = e.svc.PublicDetail(ctx(), d.PublicID)
	require.NoError(t, err)
	assert.False(t, page.Product.Purchasable, "收款失效后不可购买")

	e.shops.refs[1].Status = shop.StatusBanned
	_, err = e.svc.PublicDetail(ctx(), d.PublicID)
	requireCode(t, err, errcode.NotFound)

	_, err = e.svc.PublicDetail(ctx(), "p_missing00000")
	requireCode(t, err, errcode.NotFound)
}

func TestShopProductsPagination(t *testing.T) {
	e := setup(t)
	for range 5 {
		d := e.draft(t, 1)
		_, err := e.svc.Update(ctx(), 1, d.PublicID, complete(d.Version))
		require.NoError(t, err)
		_, err = e.svc.Publish(ctx(), 1, d.PublicID)
		require.NoError(t, err)
	}
	e.draft(t, 1) // 草稿不出现在店铺页

	items, next, err := e.svc.ShopProducts(ctx(), "one", "", 3)
	require.NoError(t, err)
	assert.Len(t, items, 3)
	require.NotEmpty(t, next)
	items, next, err = e.svc.ShopProducts(ctx(), "one", next, 3)
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Empty(t, next)

	_, _, err = e.svc.ShopProducts(ctx(), "missing", "", 3)
	requireCode(t, err, errcode.NotFound)
}

func TestListFiltersAndCovers(t *testing.T) {
	e := setup(t)
	d := e.draft(t, 1)
	_, err := e.svc.Update(ctx(), 1, d.PublicID, complete(d.Version))
	require.NoError(t, err)
	e.draft(t, 1)

	page, err := e.svc.List(ctx(), 1, ListFilter{Status: "", Page: 0, PageSize: 0})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 20, page.PageSize)

	page, err = e.svc.List(ctx(), 1, ListFilter{Query: "Figma"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.NotNil(t, page.Items[0].Cover)
	assert.True(t, strings.HasPrefix(page.Items[0].Cover.Key, "img/s1-"))

	page, err = e.svc.List(ctx(), 1, ListFilter{Status: "BOGUS"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total, "未知状态等同于全部")
}
