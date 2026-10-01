package user

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	errNotFound  = errors.New("merchant not found")
	errDuplicate = errors.New("email already registered")
)

// Repository 是商家数据访问接口。service 只依赖接口，便于单元测试替换。
type Repository interface {
	Create(ctx context.Context, m *Merchant) error
	FindByID(ctx context.Context, id uint64) (*Merchant, error)
	FindByEmail(ctx context.Context, email string) (*Merchant, error)
	UpdatePassword(ctx context.Context, id uint64, hash string) error
	MarkEmailVerified(ctx context.Context, id uint64, at time.Time) (updated bool, err error)
	UpdateLastLogin(ctx context.Context, id uint64, at time.Time) error
	UpdateNickname(ctx context.Context, id uint64, nickname string) error
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Create(ctx context.Context, m *Merchant) error {
	err := r.db.WithContext(ctx).Create(m).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return errDuplicate // 并发注册同一邮箱时由唯一索引兜底
	}
	return err
}

func (r *gormRepository) FindByID(ctx context.Context, id uint64) (*Merchant, error) {
	return r.first(ctx, "id = ?", id)
}

func (r *gormRepository) FindByEmail(ctx context.Context, email string) (*Merchant, error) {
	return r.first(ctx, "email = ?", email)
}

func (r *gormRepository) first(ctx context.Context, query string, arg any) (*Merchant, error) {
	var m Merchant
	err := r.db.WithContext(ctx).Where(query, arg).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *gormRepository) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	return r.db.WithContext(ctx).Model(&Merchant{ID: id}).Update("password_hash", hash).Error
}

// MarkEmailVerified 只在尚未验证时写入，返回是否发生了更新（用于区分“已验证过”）。
func (r *gormRepository) MarkEmailVerified(ctx context.Context, id uint64, at time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&Merchant{}).
		Where("id = ? AND email_verified_at IS NULL", id).
		Update("email_verified_at", at)
	return res.RowsAffected > 0, res.Error
}

func (r *gormRepository) UpdateLastLogin(ctx context.Context, id uint64, at time.Time) error {
	return r.db.WithContext(ctx).Model(&Merchant{ID: id}).UpdateColumn("last_login_at", at).Error
}

func (r *gormRepository) UpdateNickname(ctx context.Context, id uint64, nickname string) error {
	return r.db.WithContext(ctx).Model(&Merchant{ID: id}).Update("nickname", nickname).Error
}
