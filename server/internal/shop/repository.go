package shop

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errNotFound = errors.New("shop not found")
	// errDuplicate 表示违反唯一约束：店铺链接被占用，或该商家已有店铺
	errDuplicate = errors.New("shop slug or merchant already exists")
)

// Repository 是店铺数据访问接口。service 只依赖接口，便于单元测试替换。
// 商家侧方法以 merchantID / shopID 定位，shopID 只来自服务端查询，不接受前端传入（PRD SEC-05）。
type Repository interface {
	Create(ctx context.Context, s *Shop) error
	FindByMerchant(ctx context.Context, merchantID uint64) (*Shop, error)
	FindByID(ctx context.Context, id uint64) (*Shop, error)
	FindBySlug(ctx context.Context, slug string) (*Shop, error)
	// FindRedirect 查找未过期的旧链接跳转，返回新链接
	FindRedirect(ctx context.Context, oldSlug string, now time.Time) (newSlug string, err error)
	// TakenSlugs 返回 candidates 中已被占用的链接：其他店铺正在使用，或是其他店铺未过期的旧链接
	TakenSlugs(ctx context.Context, candidates []string, exceptShopID uint64, now time.Time) (map[string]bool, error)
	// Update 修改店铺字段；redirect 非空时表示同时修改了链接，需在同一事务中登记旧链接跳转
	Update(ctx context.Context, shopID uint64, fields map[string]any, redirect *SlugRedirect) error
	MarkShared(ctx context.Context, shopID uint64, at time.Time) error
	// PaymentConfig 返回收款配置的环境与状态；未配置时 found 为 false
	PaymentConfig(ctx context.Context, shopID uint64) (env, status string, found bool, err error)
	HasProductOnSale(ctx context.Context, shopID uint64) (bool, error)
	// ContainsBlockedWord 检查文本是否包含禁用级别的敏感词（sensitive_words 表，由平台管理后台维护）
	ContainsBlockedWord(ctx context.Context, text string) (bool, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Create(ctx context.Context, s *Shop) error {
	err := r.db.WithContext(ctx).Create(s).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return errDuplicate // 并发占用同一链接或重复创建时由唯一索引兜底
	}
	return err
}

func (r *gormRepository) FindByMerchant(ctx context.Context, merchantID uint64) (*Shop, error) {
	return r.first(ctx, "merchant_id = ?", merchantID)
}

func (r *gormRepository) FindByID(ctx context.Context, id uint64) (*Shop, error) {
	return r.first(ctx, "id = ?", id)
}

func (r *gormRepository) FindBySlug(ctx context.Context, slug string) (*Shop, error) {
	return r.first(ctx, "slug = ?", slug)
}

func (r *gormRepository) first(ctx context.Context, query string, arg any) (*Shop, error) {
	var s Shop
	err := r.db.WithContext(ctx).Where(query, arg).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *gormRepository) FindRedirect(ctx context.Context, oldSlug string, now time.Time) (string, error) {
	var slug string
	err := r.db.WithContext(ctx).Table("shop_slug_redirects AS r").
		Select("s.slug").
		Joins("JOIN shops AS s ON s.id = r.shop_id").
		Where("r.old_slug = ? AND r.expires_at > ?", oldSlug, now).
		Limit(1).Scan(&slug).Error
	if err != nil {
		return "", err
	}
	if slug == "" {
		return "", errNotFound
	}
	return slug, nil
}

func (r *gormRepository) TakenSlugs(ctx context.Context, candidates []string, exceptShopID uint64, now time.Time) (map[string]bool, error) {
	taken := make(map[string]bool, len(candidates))
	if len(candidates) == 0 {
		return taken, nil
	}
	var used []string
	if err := r.db.WithContext(ctx).Model(&Shop{}).
		Where("slug IN ? AND id <> ?", candidates, exceptShopID).
		Pluck("slug", &used).Error; err != nil {
		return nil, err
	}
	var redirected []string
	if err := r.db.WithContext(ctx).Model(&SlugRedirect{}).
		Where("old_slug IN ? AND shop_id <> ? AND expires_at > ?", candidates, exceptShopID, now).
		Pluck("old_slug", &redirected).Error; err != nil {
		return nil, err
	}
	for _, s := range append(used, redirected...) {
		taken[s] = true
	}
	return taken, nil
}

func (r *gormRepository) Update(ctx context.Context, shopID uint64, fields map[string]any, redirect *SlugRedirect) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if redirect != nil {
			// 新链接若是自己以前的旧链接（或已过期的他人旧链接），先删除那条跳转记录
			if err := tx.Where("old_slug = ?", fields["slug"]).Delete(&SlugRedirect{}).Error; err != nil {
				return err
			}
			// 旧链接可能曾作为跳转记录存在（已过期），按主键覆盖
			if err := tx.Clauses(clause.OnConflict{
				UpdateAll: true,
			}).Create(redirect).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Shop{ID: shopID}).Updates(fields).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return errDuplicate
	}
	return err
}

func (r *gormRepository) MarkShared(ctx context.Context, shopID uint64, at time.Time) error {
	return r.db.WithContext(ctx).Model(&Shop{}).
		Where("id = ? AND shared_at IS NULL", shopID).
		UpdateColumn("shared_at", at).Error
}

func (r *gormRepository) PaymentConfig(ctx context.Context, shopID uint64) (string, string, bool, error) {
	var row struct {
		Env    string
		Status string
	}
	res := r.db.WithContext(ctx).Table("shop_payment_configs").
		Select("env, status").
		Where("shop_id = ? AND provider = ?", shopID, "ALIPAY").
		Limit(1).Scan(&row)
	if res.Error != nil {
		return "", "", false, res.Error
	}
	return row.Env, row.Status, res.RowsAffected > 0, nil
}

func (r *gormRepository) HasProductOnSale(ctx context.Context, shopID uint64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("products").
		Where("shop_id = ? AND status = ? AND deleted_at IS NULL", shopID, "ON_SALE").
		Limit(1).Count(&n).Error
	return n > 0, err
}

func (r *gormRepository) ContainsBlockedWord(ctx context.Context, text string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("sensitive_words").
		Where("level = ? AND LOCATE(word, ?) > 0", "BLOCK", text).
		Limit(1).Count(&n).Error
	return n > 0, err
}
