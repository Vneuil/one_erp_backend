package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type supplierRepository struct {
	db *gorm.DB
}

func NewSupplierRepository(db *gorm.DB) domain.SupplierRepository {
	return &supplierRepository{db: db}
}

func (r *supplierRepository) Create(ctx context.Context, supplier *domain.Supplier) error {
	tenantctx.SetTenantID(ctx, &supplier.TenantID)
	return r.db.WithContext(ctx).Create(supplier).Error
}

func (r *supplierRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Supplier, error) {
	var supplier domain.Supplier
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&supplier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &supplier, nil
}

func (r *supplierRepository) GetByCode(ctx context.Context, code string) (*domain.Supplier, error) {
	var supplier domain.Supplier
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&supplier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &supplier, nil
}

func (r *supplierRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.Supplier, int64, error) {
	var suppliers []domain.Supplier
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Supplier{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ? OR code ILIKE ? OR contact_person ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortField := "created_at"
	sortDir := "desc"
	if query.SortBy != "" {
		sortField = query.SortBy
	}
	if query.SortDir != "" {
		sortDir = query.SortDir
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order(fmt.Sprintf("%s %s", sortField, sortDir)).
		Offset(offset).
		Limit(query.PerPage).
		Find(&suppliers).Error

	return suppliers, total, err
}

func (r *supplierRepository) Update(ctx context.Context, supplier *domain.Supplier) error {
	return r.db.WithContext(ctx).Save(supplier).Error
}

func (r *supplierRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Supplier{}, id).Error
}

func (r *supplierRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Supplier{}).Count(&total).Error
	return total, err
}

// HasReferences checks, by supplier_id, whether any Purchase Order or
// Purchase Invoice still references this supplier. Raw table queries are
// used (rather than importing the procurement domain package) to avoid a
// cross-module dependency - these table/column names are stable, GORM-
// derived (see each entity's TableName()).
func (r *supplierRepository) HasReferences(ctx context.Context, id uuid.UUID) (bool, error) {
	db := r.db.WithContext(ctx)
	tables := []string{"procurement_purchase_orders", "procurement_purchase_invoices"}
	for _, t := range tables {
		var count int64
		if err := db.Table(t).Where("supplier_id = ?", id).Count(&count).Error; err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}
