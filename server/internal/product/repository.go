package product

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var (
	errNotFound  = errors.New("product not found")
	errDuplicate = errors.New("product public id already exists")
	errVersion   = errors.New("product version conflict")
)

// ListFilter 是商家商品列表的筛选条件。
type ListFilter struct {
	Status   string // 为空表示全部
	Query    string
	Page     int
	PageSize int
}

// Repository 是商品数据访问接口。商家侧方法都要求传入 shopID，从机制上防止越权（数据库设计 1.3）。
type Repository interface {
	Create(ctx context.Context, p *Product) error
	Find(ctx context.Context, shopID uint64, publicID string) (*Product, error)
	// FindPublic 按对外 ID 查找未删除的商品，不限店铺（买家端使用）
	FindPublic(ctx context.Context, publicID string) (*Product, error)
	List(ctx context.Context, shopID uint64, f ListFilter) ([]Product, int64, error)
	Images(ctx context.Context, productIDs ...uint64) (map[uint64][]Image, error)
	// Save 按乐观锁更新商品；images 非 nil 时同时替换封面图。版本不一致返回 errVersion
	Save(ctx context.Context, id uint64, version int, fields map[string]any, images []Image) error
	UpdateFields(ctx context.Context, id uint64, fields map[string]any) error
	SoftDelete(ctx context.Context, id uint64) error
	HasOrders(ctx context.Context, id uint64) (bool, error)
	// OnSaleByShop 返回店铺已上架的商品，按商家排序与上架时间排列
	OnSaleByShop(ctx context.Context, shopID uint64, offset, limit int, excludeID uint64) ([]Product, error)
	ContainsBlockedWord(ctx context.Context, text string) (bool, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Create(ctx context.Context, p *Product) error {
	err := r.db.WithContext(ctx).Create(p).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return errDuplicate
	}
	return err
}

func (r *gormRepository) first(q *gorm.DB) (*Product, error) {
	var p Product
	err := q.First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) Find(ctx context.Context, shopID uint64, publicID string) (*Product, error) {
	return r.first(r.db.WithContext(ctx).Where("shop_id = ? AND public_id = ?", shopID, publicID))
}

func (r *gormRepository) FindPublic(ctx context.Context, publicID string) (*Product, error) {
	return r.first(r.db.WithContext(ctx).Where("public_id = ?", publicID))
}

func (r *gormRepository) List(ctx context.Context, shopID uint64, f ListFilter) ([]Product, int64, error) {
	q := r.db.WithContext(ctx).Model(&Product{}).Where("shop_id = ?", shopID)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Query != "" {
		q = q.Where("name LIKE ?", "%"+escapeLike(f.Query)+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Product
	err := q.Order("updated_at DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&items).Error
	return items, total, err
}

func escapeLike(s string) string {
	r := make([]rune, 0, len(s))
	for _, c := range s {
		if c == '%' || c == '_' || c == '\\' {
			r = append(r, '\\')
		}
		r = append(r, c)
	}
	return string(r)
}

func (r *gormRepository) Images(ctx context.Context, productIDs ...uint64) (map[uint64][]Image, error) {
	out := make(map[uint64][]Image, len(productIDs))
	if len(productIDs) == 0 {
		return out, nil
	}
	var rows []Image
	if err := r.db.WithContext(ctx).Where("product_id IN ?", productIDs).
		Order("product_id, sort_order").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, img := range rows {
		out[img.ProductID] = append(out[img.ProductID], img)
	}
	return out, nil
}

func (r *gormRepository) Save(ctx context.Context, id uint64, version int, fields map[string]any, images []Image) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fields["version"] = gorm.Expr("version + 1")
		res := tx.Model(&Product{}).Where("id = ? AND version = ?", id, version).Updates(fields)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errVersion
		}
		if images == nil {
			return nil
		}
		if err := tx.Where("product_id = ?", id).Delete(&Image{}).Error; err != nil {
			return err
		}
		if len(images) == 0 {
			return nil
		}
		return tx.Create(&images).Error
	})
}

func (r *gormRepository) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&Product{ID: id}).Updates(fields).Error
}

func (r *gormRepository) SoftDelete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&Product{ID: id}).Error
}

func (r *gormRepository) HasOrders(ctx context.Context, id uint64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("orders").Where("product_id = ?", id).Limit(1).Count(&n).Error
	return n > 0, err
}

func (r *gormRepository) OnSaleByShop(ctx context.Context, shopID uint64, offset, limit int, excludeID uint64) ([]Product, error) {
	var items []Product
	err := r.db.WithContext(ctx).
		Where("shop_id = ? AND status = ? AND id <> ?", shopID, StatusOnSale, excludeID).
		Order("sort_order ASC, published_at DESC, id DESC").
		Offset(offset).Limit(limit).Find(&items).Error
	return items, err
}

func (r *gormRepository) ContainsBlockedWord(ctx context.Context, text string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("sensitive_words").
		Where("level = ? AND LOCATE(word, ?) > 0", "BLOCK", text).
		Limit(1).Count(&n).Error
	return n > 0, err
}
