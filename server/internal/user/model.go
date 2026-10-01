// Package user 实现商家账号与认证（PRD 01，接口清单 #1 ～ #15）。
//
// 分层：handler（HTTP）→ service（业务规则）→ repository（MySQL）/ sessionStore（Redis）。
package user

import (
	"time"

	"gorm.io/gorm"
)

const (
	StatusActive = "ACTIVE"
	StatusBanned = "BANNED"
)

// Merchant 对应 merchants 表（docs/database-design.md 3.1）。
type Merchant struct {
	ID              uint64 `gorm:"primaryKey"`
	Email           string
	PasswordHash    string
	Nickname        string
	EmailVerifiedAt *time.Time
	Status          string
	BanReason       *string
	LastLoginAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt
}

func (m *Merchant) EmailVerified() bool { return m.EmailVerifiedAt != nil }

// MerchantView 是返回给前端的账号信息，不包含内部 ID 与密码哈希。
type MerchantView struct {
	Email         string    `json:"email"`
	Nickname      string    `json:"nickname"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (m *Merchant) View() MerchantView {
	return MerchantView{
		Email:         m.Email,
		Nickname:      m.Nickname,
		EmailVerified: m.EmailVerified(),
		CreatedAt:     m.CreatedAt,
	}
}
