package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// LoyaltyConfig holds the point-earning rule for one tenant/company. Nil
// TenantID means "the company-wide default" (see [[tenant-scoping]] pattern
// used across the app - a nil TenantID is not scoped to any specific tenant).
type LoyaltyConfig struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	// RupiahPerPoint: how many Rupiah of spend earns 1 point. Default 1000
	// (Rp 1.000 = 1 poin), matching the requested default.
	RupiahPerPoint float64 `gorm:"type:decimal(15,2);not null;default:1000" json:"rupiahPerPoint"`
}

func (LoyaltyConfig) TableName() string {
	return "loyalty_configs"
}

// LoyaltyMember is a customer's virtual membership card, looked up by phone
// number at the POS.
type LoyaltyMember struct {
	types.BaseEntity
	CompanyID     *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	MemberCode    string     `gorm:"type:varchar(30);uniqueIndex;not null" json:"memberCode"`
	PhoneNumber   string     `gorm:"type:varchar(30);not null;index" json:"phoneNumber"`
	Name          string     `gorm:"type:varchar(150)" json:"name"`
	PointsBalance int        `gorm:"not null;default:0" json:"pointsBalance"`
}

func (LoyaltyMember) TableName() string {
	return "loyalty_members"
}

// LoyaltyTransaction is one point movement (earn from a purchase, manual
// adjustment, or a future redemption) on a member's card.
type LoyaltyTransaction struct {
	types.BaseEntity
	CompanyID        *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID         *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	MemberID         uuid.UUID  `gorm:"type:uuid;index;not null" json:"memberId"`
	Type             string     `gorm:"type:varchar(20);not null" json:"type"` // earn | adjust | redeem
	Points           int        `gorm:"not null" json:"points"`
	BalanceAfter     int        `gorm:"not null" json:"balanceAfter"`
	Description      string     `gorm:"type:varchar(255)" json:"description,omitempty"`
	SalesOrderID     *uuid.UUID `gorm:"type:uuid;index" json:"salesOrderId,omitempty"`
	SalesOrderNumber string     `gorm:"type:varchar(50)" json:"salesOrderNumber,omitempty"`
}

func (LoyaltyTransaction) TableName() string {
	return "loyalty_transactions"
}

type LoyaltyRepository interface {
	GetConfig(ctx context.Context) (*LoyaltyConfig, error)
	UpsertConfig(ctx context.Context, cfg *LoyaltyConfig) error

	GetMemberByPhone(ctx context.Context, phone string) (*LoyaltyMember, error)
	GetMemberByID(ctx context.Context, id uuid.UUID) (*LoyaltyMember, error)
	CreateMember(ctx context.Context, m *LoyaltyMember) error
	UpdateMemberBalance(ctx context.Context, id uuid.UUID, newBalance int) error
	ListMembers(ctx context.Context, query types.PaginationQuery) ([]LoyaltyMember, int64, error)
	CountMembersWithCodePrefix(ctx context.Context, prefix string) (int64, error)

	CreateTransaction(ctx context.Context, tx *LoyaltyTransaction) error
	ListTransactionsByMember(ctx context.Context, memberID uuid.UUID) ([]LoyaltyTransaction, error)
}
