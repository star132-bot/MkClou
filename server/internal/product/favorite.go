package product

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mkclou/server/internal/pkg/errcode"
	"mkclou/server/internal/pkg/response"
)

// 收藏（PRD MKT-05、MKT-06）。收藏者为统一账号（merchants 表），见 MKT-08。

const (
	maxFavorites     = 1000
	favoritesPerPage = 24
)

// Favorite 对应 favorites 表。
type Favorite struct {
	ID         uint64 `gorm:"primaryKey"`
	MerchantID uint64
	ProductID  uint64
	CreatedAt  time.Time
}

// FavoriteItem 是“我的收藏”中的一项。商品失效时 Available 为 false，Reason 说明原因。
type FavoriteItem struct {
	Card
	Available   bool      `json:"available"`
	Reason      string    `json:"reason"`
	FavoritedAt time.Time `json:"favoritedAt"`
}

type favoriteRow struct {
	Product
	ShopStatus  string
	FavoritedAt time.Time
}

// FavoriteRepository 是收藏数据访问接口。
type FavoriteRepository interface {
	// Add 收藏商品并增加收藏计数；已收藏时 added 为 false
	Add(ctx context.Context, merchantID, productID uint64, at time.Time) (added bool, err error)
	Remove(ctx context.Context, merchantID, productID uint64) error
	Count(ctx context.Context, merchantID uint64) (int64, error)
	List(ctx context.Context, merchantID uint64, offset, limit int) ([]favoriteRow, error)
	// Favorited 返回 productIDs 中已被该账号收藏的
	Favorited(ctx context.Context, merchantID uint64, productIDs []uint64) (map[uint64]bool, error)
	// ProductIDs 把对外 ID 转换为内部 ID（含已删除的商品，用于取消收藏）
	ProductIDs(ctx context.Context, publicIDs []string) (map[string]uint64, error)
}

type Favorites struct {
	market *Market
	repo   FavoriteRepository
	now    func() time.Time
}

func NewFavorites(market *Market, repo FavoriteRepository) *Favorites {
	return &Favorites{market: market, repo: repo, now: time.Now}
}

// Add 收藏商品。只能收藏商城中可见的商品；重复收藏无副作用。返回最新的收藏人数。
func (f *Favorites) Add(ctx context.Context, merchantID uint64, publicID string) (int, error) {
	list, err := f.visible(ctx, publicID)
	if err != nil {
		return 0, err
	}
	p := list[0]
	n, err := f.repo.Count(ctx, merchantID)
	if err != nil {
		return 0, err
	}
	if n >= maxFavorites {
		return 0, errcode.StateConflict.WithMessage("收藏已达上限，请先整理收藏夹")
	}
	added, err := f.repo.Add(ctx, merchantID, p.ID, f.now())
	if err != nil {
		return 0, err
	}
	if added {
		p.FavoriteCount++
	}
	return p.FavoriteCount, nil
}

func (f *Favorites) visible(ctx context.Context, publicID string) ([]Product, error) {
	ids, err := f.repo.ProductIDs(ctx, []string{publicID})
	if err != nil {
		return nil, err
	}
	id, ok := ids[publicID]
	if !ok {
		return nil, errcode.NotFound.WithMessage("商品不存在")
	}
	list, err := f.market.repo.VisibleByIDs(ctx, []uint64{id})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errcode.StateConflict.WithMessage("商品已下架，无法收藏")
	}
	return list, nil
}

// Remove 取消收藏，商品已失效时也可以取消。未收藏时无副作用。
func (f *Favorites) Remove(ctx context.Context, merchantID uint64, publicID string) error {
	ids, err := f.repo.ProductIDs(ctx, []string{publicID})
	if err != nil {
		return err
	}
	id, ok := ids[publicID]
	if !ok {
		return nil
	}
	return f.repo.Remove(ctx, merchantID, id)
}

// Status 返回给定商品中已收藏的对外 ID，用于商品卡片显示收藏状态。
func (f *Favorites) Status(ctx context.Context, merchantID uint64, publicIDs []string) ([]string, error) {
	if len(publicIDs) > 100 {
		publicIDs = publicIDs[:100]
	}
	ids, err := f.repo.ProductIDs(ctx, publicIDs)
	if err != nil {
		return nil, err
	}
	internal := make([]uint64, 0, len(ids))
	for _, id := range ids {
		internal = append(internal, id)
	}
	fav, err := f.repo.Favorited(ctx, merchantID, internal)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(fav))
	for pub, id := range ids {
		if fav[id] {
			out = append(out, pub)
		}
	}
	return out, nil
}

// List 返回“我的收藏”（MKT-06），按收藏时间倒序。失效商品保留并标注原因。
func (f *Favorites) List(ctx context.Context, merchantID uint64, page int) (*response.Page[FavoriteItem], error) {
	page = max(page, 1)
	total, err := f.repo.Count(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	rows, err := f.repo.List(ctx, merchantID, (page-1)*favoritesPerPage, favoritesPerPage)
	if err != nil {
		return nil, err
	}
	products := make([]Product, len(rows))
	for i, r := range rows {
		products[i] = r.Product
	}
	cards, err := f.market.cards(ctx, products)
	if err != nil {
		return nil, err
	}
	items := make([]FavoriteItem, len(rows))
	for i, r := range rows {
		item := FavoriteItem{Card: cards[i], Available: true, FavoritedAt: r.FavoritedAt.UTC()}
		switch {
		case r.DeletedAt.Valid, r.Status != StatusOnSale, r.ShopStatus == "BANNED":
			item.Available, item.Reason = false, "已失效"
		case item.SoldOut:
			item.Reason = "已售罄"
		}
		items[i] = item
	}
	return &response.Page[FavoriteItem]{Items: items, Total: total, Page: page, PageSize: favoritesPerPage}, nil
}

// ---------- 数据访问 ----------

type gormFavoriteRepository struct {
	db *gorm.DB
}

func NewFavoriteRepository(db *gorm.DB) FavoriteRepository { return &gormFavoriteRepository{db: db} }

func (r *gormFavoriteRepository) Add(ctx context.Context, merchantID, productID uint64, at time.Time) (bool, error) {
	added := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.Insert{Modifier: "IGNORE"}).
			Create(&Favorite{MerchantID: merchantID, ProductID: productID, CreatedAt: at})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // 已收藏
		}
		added = true
		return tx.Model(&Product{}).Where("id = ?", productID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1")).Error
	})
	return added, err
}

func (r *gormFavoriteRepository) Remove(ctx context.Context, merchantID, productID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("merchant_id = ? AND product_id = ?", merchantID, productID).Delete(&Favorite{})
		if res.Error != nil || res.RowsAffected == 0 {
			return res.Error
		}
		// 已删除的商品也要更新计数，因此不经过软删除过滤
		return tx.Unscoped().Model(&Product{}).Where("id = ?", productID).
			UpdateColumn("favorite_count", gorm.Expr("GREATEST(favorite_count, 1) - 1")).Error
	})
}

func (r *gormFavoriteRepository) Count(ctx context.Context, merchantID uint64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Favorite{}).Where("merchant_id = ?", merchantID).Count(&n).Error
	return n, err
}

func (r *gormFavoriteRepository) List(ctx context.Context, merchantID uint64, offset, limit int) ([]favoriteRow, error) {
	var rows []favoriteRow
	err := r.db.WithContext(ctx).Unscoped().Table("favorites AS f").
		Select("products.*, shops.status AS shop_status, f.created_at AS favorited_at").
		Joins("JOIN products ON products.id = f.product_id").
		Joins("JOIN shops ON shops.id = products.shop_id").
		Where("f.merchant_id = ?", merchantID).
		Order("f.created_at DESC, f.id DESC").Offset(offset).Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *gormFavoriteRepository) Favorited(ctx context.Context, merchantID uint64, productIDs []uint64) (map[uint64]bool, error) {
	out := make(map[uint64]bool, len(productIDs))
	if len(productIDs) == 0 {
		return out, nil
	}
	var ids []uint64
	if err := r.db.WithContext(ctx).Model(&Favorite{}).
		Where("merchant_id = ? AND product_id IN ?", merchantID, productIDs).
		Pluck("product_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func (r *gormFavoriteRepository) ProductIDs(ctx context.Context, publicIDs []string) (map[string]uint64, error) {
	out := make(map[string]uint64, len(publicIDs))
	if len(publicIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID       uint64
		PublicID string
	}
	if err := r.db.WithContext(ctx).Unscoped().Model(&Product{}).Select("id, public_id").
		Where("public_id IN ?", publicIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.PublicID] = r.ID
	}
	return out, nil
}
