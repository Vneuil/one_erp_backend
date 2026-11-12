package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type tradeInvoiceRepository struct{ db *gorm.DB }

func NewTradeInvoiceRepository(db *gorm.DB) domain.TradeInvoiceRepository {
	return &tradeInvoiceRepository{db: db}
}

func (r *tradeInvoiceRepository) Create(ctx context.Context, t *domain.TradeInvoice) error {
	tenantctx.SetTenantID(ctx, &t.TenantID)
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *tradeInvoiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error) {
	var t domain.TradeInvoice
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *tradeInvoiceRepository) Update(ctx context.Context, t *domain.TradeInvoice) error {
	return r.db.WithContext(ctx).Omit("Lines").Save(t).Error
}

func (r *tradeInvoiceRepository) List(ctx context.Context, kind string, query types.PaginationQuery) ([]domain.TradeInvoice, int64, error) {
	var out []domain.TradeInvoice
	var total int64
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.TradeInvoice{}))
	if kind != "" {
		db = db.Where("kind = ?", kind)
	}
	if query.Search != "" {
		p := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("number ILIKE ? OR customer_name ILIKE ?", p, p)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Preload("Lines").Order("created_at desc").Offset((query.Page - 1) * query.PerPage).Limit(query.PerPage).Find(&out).Error
	return out, total, err
}

func (r *tradeInvoiceRepository) CountByKind(ctx context.Context, kind string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.TradeInvoice{}).Where("kind = ?", kind).Count(&n).Error
	return n, err
}
