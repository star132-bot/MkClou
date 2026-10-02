package product

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mkclou/server/internal/pkg/response"
)

// 商城（PRD 10 · MKT）：首页、搜索、热门。商品对外展示规则见 MKT 第 3 节：
// 商品已上架、未删除，且店铺未被封禁。

const (
	hotKey        = "mk:market:hot" // 有序集合：商品 ID → 热度
	hotWindow     = 7 * 24 * time.Hour
	hotKeep       = 100 // 保留前 100 个，首页取前 8 个
	homeHotSize   = 8
	homeLatest    = 12
	searchPerPage = 24
	maxQueryLen   = 50
)

// ShopBrief 是商品卡片上显示的店铺信息。
type ShopBrief struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Card 是商城中的商品卡片：商品摘要 + 所属店铺。
type Card struct {
	Summary
	Shop ShopBrief `json:"shop"`
}

type Home struct {
	Hot    []Card `json:"hot"`
	Latest []Card `json:"latest"`
}

// SearchInput 是搜索条件（MKT-02）。
type SearchInput struct {
	Query    string
	Category string
	Price    string // free / 0-50 / 50-200 / 200+
	Sort     string // default / sales / latest / price_asc / price_desc
	Page     int
}

type scoredID struct {
	ID    uint64
	Score float64
}

// MarketRepository 是商城查询接口。
type MarketRepository interface {
	Latest(ctx context.Context, limit int, exclude []uint64) ([]Product, error)
	VisibleByIDs(ctx context.Context, ids []uint64) ([]Product, error)
	Search(ctx context.Context, q searchQuery) ([]Product, int64, error)
	HotScores(ctx context.Context, since time.Time, limit int) ([]scoredID, error)
	ShopBriefs(ctx context.Context, shopIDs []uint64) (map[uint64]ShopBrief, error)
}

type Market struct {
	products *Service
	repo     MarketRepository
	rdb      *redis.Client
	log      *zap.Logger
	now      func() time.Time
}

func NewMarket(products *Service, repo MarketRepository, rdb *redis.Client, log *zap.Logger) *Market {
	return &Market{products: products, repo: repo, rdb: rdb, log: log, now: time.Now}
}

// Home 返回首页数据（MKT-01）：热门商品不足时用最新上架补齐（MKT-03）。
func (m *Market) Home(ctx context.Context) (*Home, error) {
	hot, err := m.hotProducts(ctx, homeHotSize)
	if err != nil {
		return nil, err
	}
	exclude := make([]uint64, len(hot))
	for i, p := range hot {
		exclude[i] = p.ID
	}
	fill, err := m.repo.Latest(ctx, homeHotSize-len(hot)+homeLatest, exclude)
	if err != nil {
		return nil, err
	}
	if missing := homeHotSize - len(hot); missing > 0 {
		n := min(missing, len(fill))
		hot, fill = append(hot, fill[:n]...), fill[n:]
	}
	// “最新上架”包含全部最新商品（与热门可以重复），这里重新查询以保证顺序正确
	latest, err := m.repo.Latest(ctx, homeLatest, nil)
	if err != nil {
		return nil, err
	}
	hotCards, err := m.cards(ctx, hot)
	if err != nil {
		return nil, err
	}
	latestCards, err := m.cards(ctx, latest)
	if err != nil {
		return nil, err
	}
	return &Home{Hot: hotCards, Latest: latestCards}, nil
}

// hotProducts 读取热门榜单并过滤掉已不可见的商品。Redis 不可用或榜单为空时返回空列表。
func (m *Market) hotProducts(ctx context.Context, limit int) ([]Product, error) {
	members, err := m.rdb.ZRevRange(ctx, hotKey, 0, int64(limit*2-1)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			m.log.Warn("read hot products failed", zap.Error(err))
		}
		return nil, nil
	}
	ids := make([]uint64, 0, len(members))
	for _, s := range members {
		if id, err := strconv.ParseUint(s, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	list, err := m.repo.VisibleByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint64]Product, len(list))
	for _, p := range list {
		byID[p.ID] = p
	}
	out := make([]Product, 0, limit)
	for _, id := range ids {
		if p, ok := byID[id]; ok && len(out) < limit {
			out = append(out, p)
		}
	}
	return out, nil
}

// TaskRefreshHot 是计算热门榜单的定时任务类型（每小时）。
const TaskRefreshHot = "cron:market-hot"

// RefreshHot 重新计算热门榜单（每小时由 worker 执行）：热度 = 近 7 天成交 × 3 + 近 7 天新增收藏。
// 先写临时 key 再原子替换，计算失败时保留上一次的结果。
func (m *Market) RefreshHot(ctx context.Context) (int, error) {
	scores, err := m.repo.HotScores(ctx, m.now().Add(-hotWindow), hotKeep)
	if err != nil {
		return 0, err
	}
	if len(scores) == 0 {
		return 0, m.rdb.Del(ctx, hotKey).Err()
	}
	tmp := hotKey + ":tmp"
	members := make([]redis.Z, len(scores))
	for i, s := range scores {
		members[i] = redis.Z{Score: s.Score, Member: strconv.FormatUint(s.ID, 10)}
	}
	pipe := m.rdb.TxPipeline()
	pipe.Del(ctx, tmp)
	pipe.ZAdd(ctx, tmp, members...)
	pipe.Rename(ctx, tmp, hotKey)
	_, err = pipe.Exec(ctx)
	return len(scores), err
}

// Search 搜索商品（MKT-02）。
func (m *Market) Search(ctx context.Context, in SearchInput) (*response.Page[Card], error) {
	q := buildSearchQuery(in)
	list, total, err := m.repo.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	cards, err := m.cards(ctx, list)
	if err != nil {
		return nil, err
	}
	return &response.Page[Card]{Items: cards, Total: total, Page: q.page, PageSize: searchPerPage}, nil
}

func (m *Market) cards(ctx context.Context, list []Product) ([]Card, error) {
	summaries, err := m.products.summaries(ctx, list)
	if err != nil {
		return nil, err
	}
	shopIDs := make([]uint64, 0, len(list))
	for _, p := range list {
		shopIDs = append(shopIDs, p.ShopID)
	}
	shops, err := m.repo.ShopBriefs(ctx, shopIDs)
	if err != nil {
		return nil, err
	}
	out := make([]Card, len(list))
	for i, p := range list {
		out[i] = Card{Summary: summaries[i], Shop: shops[p.ShopID]}
	}
	return out, nil
}

// ---------- 搜索条件 ----------

type searchQuery struct {
	// terms 是长度 ≥ 2 的关键词，用全文索引匹配；short 是单字关键词，用 LIKE 匹配（ngram 最小切分为 2 字）
	terms    []string
	short    []string
	raw      string
	category string
	minPrice *int
	maxPrice *int
	sort     string
	page     int
}

// ftOperators 是 MySQL 布尔全文检索的运算符，去掉后避免用户输入改变查询语义。
var ftOperators = strings.NewReplacer(`+`, " ", `-`, " ", `<`, " ", `>`, " ", `(`, " ", `)`, " ", `~`, " ", `*`, " ", `"`, " ", `@`, " ", `'`, " ")

func buildSearchQuery(in SearchInput) searchQuery {
	q := searchQuery{sort: in.Sort, page: max(in.Page, 1)}
	raw := strings.TrimSpace(in.Query)
	if utf8.RuneCountInString(raw) > maxQueryLen {
		raw = string([]rune(raw)[:maxQueryLen])
	}
	q.raw = raw
	for _, t := range strings.Fields(ftOperators.Replace(raw)) {
		if utf8.RuneCountInString(t) >= 2 {
			q.terms = append(q.terms, t)
		} else {
			q.short = append(q.short, t)
		}
	}
	if validCategory(in.Category) {
		q.category = in.Category
	}
	ptr := func(v int) *int { return &v }
	switch in.Price {
	case "free":
		q.minPrice, q.maxPrice = ptr(0), ptr(0)
	case "0-50":
		q.minPrice, q.maxPrice = ptr(1), ptr(5000)
	case "50-200":
		q.minPrice, q.maxPrice = ptr(5001), ptr(20000)
	case "200+":
		q.minPrice = ptr(20001)
	}
	switch in.Sort {
	case "sales", "latest", "price_asc", "price_desc":
	default:
		q.sort = "default"
	}
	return q
}

// ---------- 数据访问 ----------

type gormMarketRepository struct {
	db *gorm.DB
}

func NewMarketRepository(db *gorm.DB) MarketRepository { return &gormMarketRepository{db: db} }

// visible 是商城可见商品的公共条件。
func (r *gormMarketRepository) visible(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&Product{}).
		Joins("JOIN shops ON shops.id = products.shop_id").
		Where("products.status = ? AND shops.status <> ?", StatusOnSale, "BANNED")
}

func (r *gormMarketRepository) Latest(ctx context.Context, limit int, exclude []uint64) ([]Product, error) {
	if limit <= 0 {
		return nil, nil
	}
	q := r.visible(ctx)
	if len(exclude) > 0 {
		q = q.Where("products.id NOT IN ?", exclude)
	}
	var list []Product
	err := q.Order("products.published_at DESC, products.id DESC").Limit(limit).Find(&list).Error
	return list, err
}

func (r *gormMarketRepository) VisibleByIDs(ctx context.Context, ids []uint64) ([]Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []Product
	err := r.visible(ctx).Where("products.id IN ?", ids).Find(&list).Error
	return list, err
}

func (r *gormMarketRepository) Search(ctx context.Context, sq searchQuery) ([]Product, int64, error) {
	q := r.visible(ctx)
	if sq.category != "" {
		q = q.Where("products.category = ?", sq.category)
	}
	if sq.minPrice != nil {
		q = q.Where("products.price >= ?", *sq.minPrice)
	}
	if sq.maxPrice != nil {
		q = q.Where("products.price <= ?", *sq.maxPrice)
	}

	var match string
	if len(sq.terms) > 0 {
		parts := make([]string, len(sq.terms))
		for i, t := range sq.terms {
			parts[i] = `+"` + t + `"` // 每个词都必须出现，词内按短语匹配
		}
		match = strings.Join(parts, " ")
	}
	if sq.raw != "" {
		// 商品名称与卖点走全文索引；店铺名称用 LIKE 匹配完整关键词
		cond := r.db.Where("shops.name LIKE ?", "%"+escapeLike(sq.raw)+"%")
		if match != "" {
			cond = cond.Or("MATCH(products.name, products.tagline) AGAINST(? IN BOOLEAN MODE)", match)
		}
		for _, s := range sq.short {
			cond = cond.Or("products.name LIKE ?", "%"+escapeLike(s)+"%")
		}
		q = q.Where(cond)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序最后都按上架时间、ID 兜底，保证分页稳定
	const tail = "products.published_at DESC, products.id DESC"
	order := clause.Expr{WithoutParentheses: true}
	switch sq.sort {
	case "sales":
		order.SQL = "products.sales_count DESC, " + tail
	case "latest":
		order.SQL = tail
	case "price_asc":
		order.SQL = "products.price ASC, " + tail
	case "price_desc":
		order.SQL = "products.price DESC, " + tail
	default:
		// 综合：有关键词时先按相关度，再按热度（销量 × 3 + 收藏）
		order.SQL = "products.sales_count * 3 + products.favorite_count DESC, " + tail
		if match != "" {
			order.SQL = "MATCH(products.name, products.tagline) AGAINST(? IN BOOLEAN MODE) DESC, " + order.SQL
			order.Vars = []any{match}
		}
	}
	var list []Product
	err := q.Select("products.*").Order(clause.OrderBy{Expression: order}).
		Offset((sq.page - 1) * searchPerPage).Limit(searchPerPage).Find(&list).Error
	return list, total, err
}

func (r *gormMarketRepository) HotScores(ctx context.Context, since time.Time, limit int) ([]scoredID, error) {
	var rows []scoredID
	err := r.db.WithContext(ctx).Raw(`
SELECT p.id AS id, COALESCE(o.cnt, 0) * 3 + COALESCE(f.cnt, 0) AS score
FROM products p
JOIN shops s ON s.id = p.shop_id
LEFT JOIN (
  SELECT product_id, COUNT(*) AS cnt FROM orders
  WHERE paid_at >= ? AND is_test = 0 AND status IN ('PAID', 'DELIVERED', 'DELIVERY_FAILED')
  GROUP BY product_id
) o ON o.product_id = p.id
LEFT JOIN (
  SELECT product_id, COUNT(*) AS cnt FROM favorites WHERE created_at >= ? GROUP BY product_id
) f ON f.product_id = p.id
WHERE p.status = ? AND p.deleted_at IS NULL AND s.status <> 'BANNED'
HAVING score > 0
ORDER BY score DESC, p.published_at DESC
LIMIT ?`, since, since, StatusOnSale, limit).Scan(&rows).Error
	return rows, err
}

func (r *gormMarketRepository) ShopBriefs(ctx context.Context, shopIDs []uint64) (map[uint64]ShopBrief, error) {
	out := make(map[uint64]ShopBrief, len(shopIDs))
	if len(shopIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID   uint64
		Slug string
		Name string
	}
	if err := r.db.WithContext(ctx).Table("shops").Select("id, slug, name").Where("id IN ?", shopIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, s := range rows {
		out[s.ID] = ShopBrief{Slug: s.Slug, Name: s.Name}
	}
	return out, nil
}
