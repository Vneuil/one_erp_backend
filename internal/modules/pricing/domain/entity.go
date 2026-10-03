package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// PaymentTerm is a named payment-due rule (e.g. "Net 30") usable across
// Sales/Procurement documents.
type PaymentTerm struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name      string     `gorm:"type:varchar(100);not null;index" json:"name"`
	Days      int        `gorm:"not null;default:0" json:"days"`
	IsActive  bool       `gorm:"default:true" json:"isActive"`
}

func (PaymentTerm) TableName() string {
	return "payment_terms"
}

// CustomerType is a named customer segment/tier (e.g. "Enterprise B2B").
type CustomerType struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name      string     `gorm:"type:varchar(100);not null;index" json:"name"`
	IsActive  bool       `gorm:"default:true" json:"isActive"`
}

func (CustomerType) TableName() string {
	return "customer_types"
}

// TaxRate is a named tax percentage (e.g. "PPN 11%").
type TaxRate struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name      string     `gorm:"type:varchar(100);not null;index" json:"name"`
	Rate      float64    `gorm:"type:decimal(5,2);not null;default:0" json:"rate"`
	IsDefault bool       `gorm:"default:false" json:"isDefault"`
	IsActive  bool       `gorm:"default:true" json:"isActive"`
}

func (TaxRate) TableName() string {
	return "tax_rates"
}

type PricingRepository interface {
	CreatePaymentTerm(ctx context.Context, p *PaymentTerm) error
	GetPaymentTermByID(ctx context.Context, id uuid.UUID) (*PaymentTerm, error)
	ListPaymentTerms(ctx context.Context) ([]PaymentTerm, error)
	UpdatePaymentTerm(ctx context.Context, p *PaymentTerm) error
	DeletePaymentTerm(ctx context.Context, id uuid.UUID) error

	CreateCustomerType(ctx context.Context, c *CustomerType) error
	GetCustomerTypeByID(ctx context.Context, id uuid.UUID) (*CustomerType, error)
	ListCustomerTypes(ctx context.Context) ([]CustomerType, error)
	UpdateCustomerType(ctx context.Context, c *CustomerType) error
	DeleteCustomerType(ctx context.Context, id uuid.UUID) error

	CreateTaxRate(ctx context.Context, t *TaxRate) error
	GetTaxRateByID(ctx context.Context, id uuid.UUID) (*TaxRate, error)
	ListTaxRates(ctx context.Context) ([]TaxRate, error)
	UpdateTaxRate(ctx context.Context, t *TaxRate) error
	DeleteTaxRate(ctx context.Context, id uuid.UUID) error
}
