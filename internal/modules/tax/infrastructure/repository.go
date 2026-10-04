package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type taxRepository struct {
	db *gorm.DB
}

func NewTaxRepository(db *gorm.DB) domain.TaxRepository {
	return &taxRepository{db: db}
}

func (r *taxRepository) GetSettings(ctx context.Context) (*domain.TaxSettings, error) {
	var found []domain.TaxSettings
	if err := r.db.WithContext(ctx).Order("created_at asc").Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, nil
	}
	return &found[0], nil
}

func (r *taxRepository) SaveSettings(ctx context.Context, s *domain.TaxSettings) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *taxRepository) CreateSerialRange(ctx context.Context, sr *domain.SerialRange) error {
	return r.db.WithContext(ctx).Create(sr).Error
}

func (r *taxRepository) GetSerialRange(ctx context.Context, id uuid.UUID) (*domain.SerialRange, error) {
	var sr domain.SerialRange
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&sr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sr, nil
}

func (r *taxRepository) UpdateSerialRange(ctx context.Context, sr *domain.SerialRange) error {
	return r.db.WithContext(ctx).Save(sr).Error
}

func (r *taxRepository) ListSerialRanges(ctx context.Context) ([]domain.SerialRange, error) {
	var out []domain.SerialRange
	err := r.db.WithContext(ctx).Order("created_at asc").Find(&out).Error
	return out, err
}

func (r *taxRepository) ClaimNextSerial(ctx context.Context) (string, error) {
	var number string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sr domain.SerialRange
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("is_active = ? AND next_number <= last_number", true).Order("created_at asc").First(&sr).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		number = fmt.Sprintf("%s%0*d", sr.Prefix, sr.Width, sr.Next)
		sr.Next++
		return tx.Save(&sr).Error
	})
	return number, err
}

func (r *taxRepository) CreateTaxInvoice(ctx context.Context, t *domain.TaxInvoice) error {
	tenantctx.SetTenantID(ctx, &t.TenantID)
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *taxRepository) GetTaxInvoice(ctx context.Context, id uuid.UUID) (*domain.TaxInvoice, error) {
	var t domain.TaxInvoice
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Lines", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Where("id = ?", id).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// UpdateTaxInvoice saves the header only; lines are immutable after creation.
func (r *taxRepository) UpdateTaxInvoice(ctx context.Context, t *domain.TaxInvoice) error {
	lines := t.Lines
	t.Lines = nil
	err := r.db.WithContext(ctx).Save(t).Error
	t.Lines = lines
	return err
}

func (r *taxRepository) ListTaxInvoices(ctx context.Context, f domain.InvoiceFilter) ([]domain.TaxInvoice, error) {
	var out []domain.TaxInvoice
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Lines", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") })
	if f.Direction != "" {
		db = db.Where("direction = ?", f.Direction)
	}
	if f.Status != "" {
		db = db.Where("status = ?", f.Status)
	}
	if f.From != "" {
		db = db.Where("date >= ?", f.From)
	}
	if f.To != "" {
		db = db.Where("date <= ?", f.To)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := "%" + s + "%"
		db = db.Where("number ILIKE ? OR tax_number ILIKE ? OR counterparty_name ILIKE ? OR source_reference ILIKE ?", p, p, p, p)
	}
	err := db.Order("date desc, created_at desc").Find(&out).Error
	return out, err
}

func (r *taxRepository) FindOpenBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) (*domain.TaxInvoice, error) {
	var found []domain.TaxInvoice
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).
		Where("source_type = ? AND source_id = ? AND status IN ?", sourceType, sourceID, []string{domain.StatusDraft, domain.StatusIssued}).
		Limit(1).Find(&found).Error
	if err != nil || len(found) == 0 {
		return nil, err
	}
	return &found[0], nil
}

func (r *taxRepository) ListBySources(ctx context.Context, sourceType string, ids []uuid.UUID) ([]domain.TaxInvoice, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []domain.TaxInvoice
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).
		Where("source_type = ? AND source_id IN ? AND status IN ?", sourceType, ids, []string{domain.StatusDraft, domain.StatusIssued}).
		Find(&out).Error
	return out, err
}

func (r *taxRepository) TaxNumberTaken(ctx context.Context, direction, taxNumber string, exceptID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.TaxInvoice{}).
		Where("direction = ? AND tax_number = ? AND id <> ? AND status <> ?", direction, taxNumber, exceptID, domain.StatusCancelled).
		Count(&n).Error
	return n > 0, err
}

func (r *taxRepository) CountTaxInvoices(ctx context.Context, direction, numberPrefix string) (int64, error) {
	var n int64
	// Unscoped: soft-deleted documents still hold their number.
	err := r.db.WithContext(ctx).Unscoped().Model(&domain.TaxInvoice{}).
		Where("direction = ? AND number LIKE ?", direction, numberPrefix+"%").Count(&n).Error
	return n, err
}
