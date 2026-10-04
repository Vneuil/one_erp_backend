package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) domain.InventoryRepository {
	return &inventoryRepository{db: db}
}

// Warehouses

func (r *inventoryRepository) CreateWarehouse(ctx context.Context, w *domain.Warehouse) error {
	tenantctx.SetTenantID(ctx, &w.TenantID)
	return r.db.WithContext(ctx).Create(w).Error
}

func (r *inventoryRepository) GetWarehouseByID(ctx context.Context, id uuid.UUID) (*domain.Warehouse, error) {
	var w domain.Warehouse
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *inventoryRepository) ListWarehouses(ctx context.Context, query types.PaginationQuery) ([]domain.Warehouse, int64, error) {
	var warehouses []domain.Warehouse
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Warehouse{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("code ILIKE ? OR name ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("code asc").Offset(offset).Limit(query.PerPage).Find(&warehouses).Error
	return warehouses, total, err
}

func (r *inventoryRepository) ListAllWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	var warehouses []domain.Warehouse
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Order("code asc").Find(&warehouses).Error
	return warehouses, err
}

func (r *inventoryRepository) UpdateWarehouse(ctx context.Context, w *domain.Warehouse) error {
	return r.db.WithContext(ctx).Save(w).Error
}

func (r *inventoryRepository) DeleteWarehouse(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Warehouse{}, id).Error
}

func (r *inventoryRepository) CountWarehouses(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Warehouse{})).Count(&total).Error
	return total, err
}

// Stock levels

func (r *inventoryRepository) GetStockLevel(ctx context.Context, productID, warehouseID uuid.UUID) (*domain.StockLevel, error) {
	var s domain.StockLevel
	err := r.db.WithContext(ctx).Where("product_id = ? AND warehouse_id = ?", productID, warehouseID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// HasAnyStockLevel reports whether a product has been counted into ANY
// warehouse yet (a StockLevel row exists for at least one warehouse) -
// used to gate the one-time legacy Product.Stock fallback so it's only
// ever applied to the first warehouse that touches a product, never
// re-applied to a second warehouse (which would double-count the same
// legacy figure).
func (r *inventoryRepository) HasAnyStockLevel(ctx context.Context, productID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.StockLevel{}).Where("product_id = ?", productID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *inventoryRepository) ListStockLevels(ctx context.Context, query types.PaginationQuery) ([]domain.StockLevel, int64, error) {
	var levels []domain.StockLevel
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.StockLevel{}))

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&levels).Error
	return levels, total, err
}

func (r *inventoryRepository) ListStockLevelsByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]domain.StockLevel, error) {
	var levels []domain.StockLevel
	err := r.db.WithContext(ctx).Where("warehouse_id = ?", warehouseID).Find(&levels).Error
	return levels, err
}

func (r *inventoryRepository) UpsertStockLevel(ctx context.Context, s *domain.StockLevel) error {
	if s.ID == uuid.Nil {
		tenantctx.SetTenantID(ctx, &s.TenantID)
		return r.db.WithContext(ctx).Create(s).Error
	}
	return r.db.WithContext(ctx).Save(s).Error
}

// Movements

func (r *inventoryRepository) CreateMovement(ctx context.Context, m *domain.StockMovement) error {
	tenantctx.SetTenantID(ctx, &m.TenantID)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *inventoryRepository) ListMovements(ctx context.Context, query types.PaginationQuery) ([]domain.StockMovement, int64, error) {
	var movements []domain.StockMovement
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.StockMovement{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("reference ILIKE ? OR reason ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&movements).Error
	return movements, total, err
}

// MovementsBetween returns movements with from <= created_at < to, oldest first.
func (r *inventoryRepository) MovementsBetween(ctx context.Context, from, to time.Time, warehouseID *uuid.UUID) ([]domain.StockMovement, error) {
	var out []domain.StockMovement
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.StockMovement{})).Where("created_at >= ? AND created_at < ?", from, to)
	if warehouseID != nil {
		db = db.Where("warehouse_id = ?", *warehouseID)
	}
	err := db.Order("created_at asc").Limit(50000).Find(&out).Error
	return out, err
}

// Transfers

func (r *inventoryRepository) CreateTransfer(ctx context.Context, t *domain.StockTransfer) error {
	tenantctx.SetTenantID(ctx, &t.TenantID)
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *inventoryRepository) GetTransferByID(ctx context.Context, id uuid.UUID) (*domain.StockTransfer, error) {
	var t domain.StockTransfer
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *inventoryRepository) ListTransfers(ctx context.Context, query types.PaginationQuery) ([]domain.StockTransfer, int64, error) {
	var transfers []domain.StockTransfer
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.StockTransfer{}))

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&transfers).Error
	return transfers, total, err
}

func (r *inventoryRepository) UpdateTransfer(ctx context.Context, t *domain.StockTransfer) error {
	return r.db.WithContext(ctx).Save(t).Error
}

// Stock opname

func (r *inventoryRepository) CreateOpname(ctx context.Context, o *domain.StockOpname) error {
	tenantctx.SetTenantID(ctx, &o.TenantID)
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *inventoryRepository) GetOpnameByID(ctx context.Context, id uuid.UUID) (*domain.StockOpname, error) {
	var o domain.StockOpname
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *inventoryRepository) ListOpnames(ctx context.Context, query types.PaginationQuery) ([]domain.StockOpname, int64, error) {
	var opnames []domain.StockOpname
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.StockOpname{}))

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&opnames).Error
	return opnames, total, err
}

func (r *inventoryRepository) UpdateOpname(ctx context.Context, o *domain.StockOpname) error {
	return r.db.WithContext(ctx).Save(o).Error
}

func (r *inventoryRepository) UpdateOpnameLine(ctx context.Context, l *domain.StockOpnameLine) error {
	return r.db.WithContext(ctx).Save(l).Error
}

// Stock documents

func (r *inventoryRepository) CreateStockDocument(ctx context.Context, d *domain.StockDocument) error {
	tenantctx.SetTenantID(ctx, &d.TenantID)
	return r.db.WithContext(ctx).Create(d).Error
}

// UpdateStockDocument saves the header only; lines are immutable once posted.
func (r *inventoryRepository) UpdateStockDocument(ctx context.Context, d *domain.StockDocument) error {
	return r.db.WithContext(ctx).Model(&domain.StockDocument{}).Where("id = ?", d.ID).
		Updates(map[string]any{"posted": d.Posted, "total_value": d.TotalValue}).Error
}

func (r *inventoryRepository) GetStockDocument(ctx context.Context, id uuid.UUID) (*domain.StockDocument, error) {
	var d domain.StockDocument
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Lines").Where("id = ?", id).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *inventoryRepository) ListStockDocuments(ctx context.Context, f domain.StockDocumentFilter) ([]domain.StockDocument, error) {
	var out []domain.StockDocument
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Lines")
	if f.Type != "" {
		db = db.Where("type = ?", f.Type)
	}
	if f.From != "" {
		db = db.Where("date >= ?", f.From)
	}
	if f.To != "" {
		db = db.Where("date <= ?", f.To)
	}
	err := db.Order("date desc, created_at desc").Limit(500).Find(&out).Error
	return out, err
}

func (r *inventoryRepository) CountStockDocuments(ctx context.Context, docType, numberPrefix string) (int64, error) {
	var n int64
	// Unscoped: soft-deleted documents still hold their number.
	err := r.db.WithContext(ctx).Unscoped().Model(&domain.StockDocument{}).
		Where("type = ? AND number LIKE ?", docType, numberPrefix+"%").Count(&n).Error
	return n, err
}

func (r *inventoryRepository) SumIssuedByOrder(ctx context.Context, orderID uuid.UUID) (map[uuid.UUID]int, error) {
	var rows []struct {
		ProductID uuid.UUID
		Qty       int
	}
	err := r.db.WithContext(ctx).Table("inventory_stock_document_lines l").
		Joins("JOIN inventory_stock_documents d ON d.id = l.document_id AND d.deleted_at IS NULL").
		Where("l.deleted_at IS NULL AND d.type = ? AND d.production_order_id = ?", domain.DocMaterialIssue, orderID).
		Select("l.product_id AS product_id, COALESCE(SUM(l.quantity),0) AS qty").Group("l.product_id").Scan(&rows).Error
	out := make(map[uuid.UUID]int, len(rows))
	for _, r := range rows {
		out[r.ProductID] = r.Qty
	}
	return out, err
}

func (r *inventoryRepository) WithTransaction(ctx context.Context, fn func(txRepo domain.InventoryRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&inventoryRepository{db: tx})
	})
}

// Batches

func (r *inventoryRepository) GetBatch(ctx context.Context, id uuid.UUID) (*domain.StockBatch, error) {
	var b domain.StockBatch
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.StockBatch{})).Where("id = ?", id).First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *inventoryRepository) GetBatchByNo(ctx context.Context, productID, warehouseID uuid.UUID, batchNo string) (*domain.StockBatch, error) {
	var found []domain.StockBatch
	err := r.db.WithContext(ctx).Where("product_id = ? AND warehouse_id = ? AND batch_no = ?", productID, warehouseID, batchNo).Limit(1).Find(&found).Error
	if err != nil || len(found) == 0 {
		return nil, err
	}
	return &found[0], nil
}

func (r *inventoryRepository) SaveBatch(ctx context.Context, b *domain.StockBatch) error {
	if b.CreatedAt.IsZero() {
		tenantctx.SetTenantID(ctx, &b.TenantID)
	}
	return r.db.WithContext(ctx).Save(b).Error
}

func (r *inventoryRepository) ListBatches(ctx context.Context, productID, warehouseID *uuid.UUID, includeEmpty bool) ([]domain.StockBatch, error) {
	var out []domain.StockBatch
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.StockBatch{}))
	if productID != nil {
		db = db.Where("product_id = ?", *productID)
	}
	if warehouseID != nil {
		db = db.Where("warehouse_id = ?", *warehouseID)
	}
	if !includeEmpty {
		db = db.Where("quantity > 0")
	}
	return out, db.Order("CASE WHEN expiry_date = '' OR expiry_date IS NULL THEN 1 ELSE 0 END, expiry_date asc, created_at asc").Find(&out).Error
}

func (r *inventoryRepository) ListBatchesFEFO(ctx context.Context, productID, warehouseID uuid.UUID) ([]domain.StockBatch, error) {
	var out []domain.StockBatch
	return out, r.db.WithContext(ctx).Where("product_id = ? AND warehouse_id = ? AND quantity > 0", productID, warehouseID).
		Order("CASE WHEN expiry_date = '' OR expiry_date IS NULL THEN 1 ELSE 0 END, expiry_date asc, created_at asc").Find(&out).Error
}

func (r *inventoryRepository) ListBatchesExpiringBy(ctx context.Context, date string) ([]domain.StockBatch, error) {
	var out []domain.StockBatch
	return out, tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.StockBatch{})).
		Where("quantity > 0 AND expiry_date <> '' AND expiry_date <= ?", date).Order("expiry_date asc").Find(&out).Error
}
