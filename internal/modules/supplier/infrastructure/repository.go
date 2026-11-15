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
	err := r.db.WithContext(ctx).Preload("Codes").Where("id = ?", id).First(&supplier).Error
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
	err := r.db.WithContext(ctx).Preload("Codes").Where("code = ?", code).First(&supplier).Error
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
		Preload("Codes").
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
	tables := []string{
		"procurement_purchase_orders",
		"procurement_purchase_invoices",
		"procurement_purchase_down_payments",
		"procurement_purchase_returns",
		"procurement_invoice_receipts",
		"qc_certificates",
	}
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

func (r *supplierRepository) CodesWithPrefix(ctx context.Context, prefix string) ([]string, error) {
	var codes []string
	_ = r.db.WithContext(ctx).Model(&domain.Supplier{}).
		Where("code LIKE ?", prefix+"%").Pluck("code", &codes).Error
	var scCodes []string
	_ = r.db.WithContext(ctx).Model(&domain.SupplierCode{}).
		Where("code LIKE ?", prefix+"%").Pluck("code", &scCodes).Error
	codes = append(codes, scCodes...)
	return codes, nil
}

func (r *supplierRepository) CreateCode(ctx context.Context, sc *domain.SupplierCode) error {
	tenantctx.SetTenantID(ctx, &sc.TenantID)
	return r.db.WithContext(ctx).Create(sc).Error
}

func (r *supplierRepository) GetCodeByID(ctx context.Context, id uuid.UUID) (*domain.SupplierCode, error) {
	var sc domain.SupplierCode
	err := r.db.WithContext(ctx).Table("supplier_codes sc").
		Select("sc.*, s.name as supplier_name").
		Joins("LEFT JOIN suppliers s ON s.id = sc.supplier_id").
		Where("sc.id = ?", id).First(&sc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sc, nil
}

func (r *supplierRepository) GetCodeByCode(ctx context.Context, code string) (*domain.SupplierCode, error) {
	var sc domain.SupplierCode
	err := r.db.WithContext(ctx).Table("supplier_codes sc").
		Select("sc.*, s.name as supplier_name").
		Joins("LEFT JOIN suppliers s ON s.id = sc.supplier_id").
		Where("sc.code = ?", code).First(&sc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sc, nil
}

func (r *supplierRepository) ListCodes(ctx context.Context, supplierID *uuid.UUID, vendorType string) ([]domain.SupplierCode, error) {
	var items []domain.SupplierCode
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Table("supplier_codes sc").
		Select("sc.*, s.name as supplier_name").
		Joins("LEFT JOIN suppliers s ON s.id = sc.supplier_id"))

	if supplierID != nil && *supplierID != uuid.Nil {
		db = db.Where("sc.supplier_id = ?", *supplierID)
	}
	if vendorType != "" {
		db = db.Where("sc.vendor_type = ?", vendorType)
	}

	err := db.Order("sc.code asc").Find(&items).Error
	return items, err
}

func (r *supplierRepository) UpdateCode(ctx context.Context, sc *domain.SupplierCode) error {
	return r.db.WithContext(ctx).Save(sc).Error
}

func (r *supplierRepository) DeleteCode(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.SupplierCode{}, id).Error
}
