package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Currency is one currency a company has enabled for transactions. Exactly
// one row should have IsBase=true at a time - that's the company's
// reporting/base currency (defaults to the company's existing Currency
// string field on modules/company, kept in sync at seed time but not
// tightly coupled so a company can still change its base later).
type Currency struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Code      string     `gorm:"type:varchar(10);not null;index" json:"code"` // ISO 4217, e.g. "IDR", "USD"
	Name      string     `gorm:"type:varchar(100);not null" json:"name"`
	Symbol    string     `gorm:"type:varchar(10)" json:"symbol"`
	IsBase    bool       `gorm:"default:false" json:"isBase"`
	IsActive  bool       `gorm:"default:true" json:"isActive"`
}

func (Currency) TableName() string {
	return "currencies"
}

// ExchangeRate is one daily rate: how many units of the company's base
// currency one unit of CurrencyCode is worth on RateDate. Historical rows
// are never overwritten by a new day's rate - RateDate keeps the full
// history, which is what lets past transactions stay converted at the
// rate that was actually in effect when they were made.
type ExchangeRate struct {
	types.BaseEntity
	CompanyID    *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	CurrencyCode string     `gorm:"type:varchar(10);not null;index:idx_rate_code_date,unique" json:"currencyCode"`
	RateDate     string     `gorm:"type:varchar(20);not null;index:idx_rate_code_date,unique" json:"rateDate"`
	RateToBase   float64    `gorm:"type:decimal(18,6);not null" json:"rateToBase"`
}

func (ExchangeRate) TableName() string {
	return "exchange_rates"
}

type CurrencyRepository interface {
	CreateCurrency(ctx context.Context, c *Currency) error
	ListCurrencies(ctx context.Context) ([]Currency, error)
	GetCurrencyByCode(ctx context.Context, code string) (*Currency, error)
	GetBaseCurrency(ctx context.Context) (*Currency, error)
	UpdateCurrency(ctx context.Context, c *Currency) error
	DeleteCurrency(ctx context.Context, id uuid.UUID) error

	UpsertExchangeRate(ctx context.Context, r *ExchangeRate) error
	ListExchangeRates(ctx context.Context, currencyCode string) ([]ExchangeRate, error)
	// GetLatestRate returns the most recent rate for currencyCode on or
	// before asOfDate - the rate "in effect" for a transaction dated
	// asOfDate, even if today's rate hasn't been entered yet.
	GetLatestRate(ctx context.Context, currencyCode string, asOfDate string) (*ExchangeRate, error)
}
