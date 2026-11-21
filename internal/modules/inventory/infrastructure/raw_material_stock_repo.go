package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *inventoryRepository) CreateRawMaterialStockEntry(ctx context.Context, e *domain.RawMaterialStockEntry) error {
	tenantctx.SetTenantID(ctx, &e.TenantID)
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *inventoryRepository) GetRawMaterialStockEntry(ctx context.Context, id uuid.UUID) (*domain.RawMaterialStockEntry, error) {
	var e domain.RawMaterialStockEntry
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.RawMaterialStockEntry{})).
		Preload("Logs", func(db *gorm.DB) *gorm.DB {
			return db.Order("occurred_at asc, created_at asc")
		}).
		Where("id = ?", id).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *inventoryRepository) UpdateRawMaterialStockEntry(ctx context.Context, e *domain.RawMaterialStockEntry) error {
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *inventoryRepository) DeleteRawMaterialStockEntry(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.RawMaterialStockEntry{}, id).Error
}

func (r *inventoryRepository) ListRawMaterialStockEntries(ctx context.Context, f domain.RawMaterialStockFilter) ([]domain.RawMaterialStockEntry, error) {
	var out []domain.RawMaterialStockEntry
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.RawMaterialStockEntry{}))
	if f.Status != "" && f.Status != "all" {
		db = db.Where("status = ?", f.Status)
	}
	if f.WarehouseID != nil {
		db = db.Where("warehouse_id = ?", *f.WarehouseID)
	}
	if f.ProductID != nil {
		db = db.Where("product_id = ?", *f.ProductID)
	}
	if f.StorageZone != "" && f.StorageZone != "all" {
		db = db.Where("storage_zone = ?", f.StorageZone)
	}
	if f.OrganicStatus != "" && f.OrganicStatus != "all" {
		db = db.Where("organic_status = ?", f.OrganicStatus)
	}
	if f.LBMNo != "" {
		db = db.Where("lbm_no ILIKE ?", "%"+strings.TrimSpace(f.LBMNo)+"%")
	}
	if f.GoodsReceiptID != nil {
		db = db.Where("goods_receipt_id = ?", *f.GoodsReceiptID)
	}
	if f.FromDate != "" {
		db = db.Where("entry_date >= ?", f.FromDate)
	}
	if f.ToDate != "" {
		db = db.Where("entry_date <= ?", f.ToDate)
	}
	if q := strings.TrimSpace(f.Search); q != "" {
		pattern := fmt.Sprintf("%%%s%%", q)
		db = db.Where(
			"entry_no ILIKE ? OR batch_no ILIKE ? OR lbm_no ILIKE ? OR product_name ILIKE ? OR product_sku ILIKE ? OR supplier_name ILIKE ? OR rack_location ILIKE ? OR pallet_no ILIKE ?",
			pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern,
		)
	}
	err := db.Order("entry_date desc, created_at desc").Limit(500).Find(&out).Error
	return out, err
}

// ListAvailableRawMaterialLots returns lots that are available for production requisition (BPB/LBB),
// ordered FEFO (soonest expiry first, then earliest entry date).
func (r *inventoryRepository) ListAvailableRawMaterialLots(ctx context.Context, productID, warehouseID *uuid.UUID) ([]domain.RawMaterialStockEntry, error) {
	var out []domain.RawMaterialStockEntry
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.RawMaterialStockEntry{})).
		Where("status = ? AND qty_remaining > 0", domain.RawStockStatusAvailable)
	if productID != nil {
		db = db.Where("product_id = ?", *productID)
	}
	if warehouseID != nil {
		db = db.Where("warehouse_id = ?", *warehouseID)
	}
	err := db.Order("CASE WHEN expiry_date = '' OR expiry_date IS NULL THEN 1 ELSE 0 END, expiry_date asc, entry_date asc, created_at asc").
		Find(&out).Error
	return out, err
}

func (r *inventoryRepository) CountRawMaterialStockEntriesByPrefix(ctx context.Context, prefix string) (int64, error) {
	var n int64
	// Unscoped so soft-deleted entries still preserve numbering monotonicity.
	err := r.db.WithContext(ctx).Unscoped().Model(&domain.RawMaterialStockEntry{}).
		Where("entry_no LIKE ?", prefix+"%").Count(&n).Error
	return n, err
}

func (r *inventoryRepository) CreateRawMaterialStockLog(ctx context.Context, l *domain.RawMaterialStockLog) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *inventoryRepository) ListRawMaterialStockLogs(ctx context.Context, entryID uuid.UUID) ([]domain.RawMaterialStockLog, error) {
	var out []domain.RawMaterialStockLog
	err := r.db.WithContext(ctx).
		Where("entry_id = ?", entryID).
		Order("occurred_at asc, created_at asc").
		Find(&out).Error
	return out, err
}
