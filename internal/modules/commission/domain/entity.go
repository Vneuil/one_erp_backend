package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// CommissionRule defines how commission is calculated for sales orders
type CommissionRule struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this rule to a branch/subsidiary Tenant within the
	// Company's database (nil for companies with only their default Tenant).
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`
	Scope       string     `gorm:"type:varchar(20);not null" json:"scope"`
	RatePercent float64    `gorm:"type:decimal(5,2);not null;default:0" json:"ratePercent"`
	Tiers       string     `gorm:"type:text" json:"tiers"`
	AppliesTo   string     `gorm:"type:varchar(100);not null;default:'all_products'" json:"appliesTo"`
	IsActive    bool       `gorm:"not null;default:true" json:"isActive"`
}

func (CommissionRule) TableName() string {
	return "commission_rules"
}

// CommissionTier represents one bracket of a tiered commission rule
type CommissionTier struct {
	MinAmount   float64 `json:"minAmount"`
	MaxAmount   float64 `json:"maxAmount"`
	RatePercent float64 `json:"ratePercent"`
}

// CommissionRecord represents a calculated commission earned on a sales order
type CommissionRecord struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this record to a branch/subsidiary Tenant within the
	// Company's database (nil for companies with only their default Tenant).
	TenantID                   *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	SalespersonName            string     `gorm:"type:varchar(255);not null;index" json:"salespersonName"`
	SalesOrderID               *uuid.UUID `gorm:"type:uuid;index" json:"salesOrderId,omitempty"`
	SalesOrderNumber           string     `gorm:"type:varchar(50)" json:"salesOrderNumber"`
	SalesOrderAmount           float64    `gorm:"type:decimal(15,2);not null" json:"salesOrderAmount"`
	CommissionRuleID           *uuid.UUID `gorm:"type:uuid;index" json:"commissionRuleId,omitempty"`
	CalculatedCommissionAmount float64    `gorm:"type:decimal(15,2);not null" json:"calculatedCommissionAmount"`
	Status                     string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Period                     string     `gorm:"type:varchar(7);not null;index" json:"period"`
}

func (CommissionRecord) TableName() string {
	return "commission_records"
}

type CommissionRepository interface {
	CreateRule(ctx context.Context, r *CommissionRule) error
	GetRuleByID(ctx context.Context, id uuid.UUID) (*CommissionRule, error)
	ListRules(ctx context.Context, query types.PaginationQuery) ([]CommissionRule, int64, error)
	CountRules(ctx context.Context) (int64, error)

	CreateRecord(ctx context.Context, r *CommissionRecord) error
	GetRecordByID(ctx context.Context, id uuid.UUID) (*CommissionRecord, error)
	ListRecords(ctx context.Context, query types.PaginationQuery) ([]CommissionRecord, int64, error)
	ListRecordsByPeriod(ctx context.Context, period string) ([]CommissionRecord, error)
	UpdateRecord(ctx context.Context, r *CommissionRecord) error
	CountRecords(ctx context.Context) (int64, error)
}
