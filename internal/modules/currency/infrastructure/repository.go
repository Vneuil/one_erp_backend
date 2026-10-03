package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/currency/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type currencyRepository struct {
	db *gorm.DB
}

func NewCurrencyRepository(db *gorm.DB) domain.CurrencyRepository {
	return &currencyRepository{db: db}
}

func (r *currencyRepository) CreateCurrency(ctx context.Context, c *domain.Currency) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *currencyRepository) ListCurrencies(ctx context.Context) ([]domain.Currency, error) {
	var items []domain.Currency
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Currency{}))
	err := db.Order("is_base desc, code asc").Find(&items).Error
	return items, err
}

func (r *currencyRepository) GetCurrencyByCode(ctx context.Context, code string) (*domain.Currency, error) {
	var c domain.Currency
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("code = ?", code).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *currencyRepository) GetBaseCurrency(ctx context.Context) (*domain.Currency, error) {
	var c domain.Currency
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("is_base = ?", true).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *currencyRepository) UpdateCurrency(ctx context.Context, c *domain.Currency) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *currencyRepository) DeleteCurrency(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Currency{}, id).Error
}

func (r *currencyRepository) UpsertExchangeRate(ctx context.Context, rate *domain.ExchangeRate) error {
	tenantctx.SetTenantID(ctx, &rate.TenantID)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "currency_code"}, {Name: "rate_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate_to_base", "updated_at"}),
	}).Create(rate).Error
}

func (r *currencyRepository) ListExchangeRates(ctx context.Context, currencyCode string) ([]domain.ExchangeRate, error) {
	var items []domain.ExchangeRate
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ExchangeRate{}))
	if currencyCode != "" {
		db = db.Where("currency_code = ?", currencyCode)
	}
	err := db.Order("rate_date desc").Limit(365).Find(&items).Error
	return items, err
}

func (r *currencyRepository) GetLatestRate(ctx context.Context, currencyCode string, asOfDate string) (*domain.ExchangeRate, error) {
	var rate domain.ExchangeRate
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("currency_code = ? AND rate_date <= ?", currencyCode, asOfDate).
		Order("rate_date desc").First(&rate).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rate, nil
}
