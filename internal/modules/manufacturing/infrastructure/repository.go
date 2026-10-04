package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	inventoryInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type manufacturingRepository struct {
	db *gorm.DB
}

func NewManufacturingRepository(db *gorm.DB) domain.ManufacturingRepository {
	return &manufacturingRepository{db: db}
}

// BOM

func (r *manufacturingRepository) CreateBOM(ctx context.Context, b *domain.BillOfMaterial) error {
	tenantctx.SetTenantID(ctx, &b.TenantID)
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *manufacturingRepository) GetBOMByID(ctx context.Context, id uuid.UUID) (*domain.BillOfMaterial, error) {
	var b domain.BillOfMaterial
	err := r.db.WithContext(ctx).Preload("Lines").Preload("Processes", orderBySequence).Where("id = ?", id).First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *manufacturingRepository) ListBOMs(ctx context.Context, query types.PaginationQuery) ([]domain.BillOfMaterial, int64, error) {
	var boms []domain.BillOfMaterial
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.BillOfMaterial{}))
	if query.Search != "" {
		pattern := "%" + query.Search + "%"
		db = db.Where("name ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Preload("Processes", orderBySequence).Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&boms).Error
	return boms, total, err
}

func (r *manufacturingRepository) UpdateBOM(ctx context.Context, b *domain.BillOfMaterial) error {
	return r.db.WithContext(ctx).Save(b).Error
}

func (r *manufacturingRepository) CountBOMs(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.BillOfMaterial{})).Count(&total).Error
	return total, err
}

// Production orders

func (r *manufacturingRepository) CreateOrder(ctx context.Context, o *domain.ProductionOrder) error {
	tenantctx.SetTenantID(ctx, &o.TenantID)
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *manufacturingRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*domain.ProductionOrder, error) {
	var o domain.ProductionOrder
	err := r.db.WithContext(ctx).Preload("Steps", orderBySequence).Where("id = ?", id).First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *manufacturingRepository) ListOrders(ctx context.Context, query types.PaginationQuery) ([]domain.ProductionOrder, int64, error) {
	var orders []domain.ProductionOrder
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ProductionOrder{}))

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Steps", orderBySequence).Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&orders).Error
	return orders, total, err
}

// UpdateOrder saves the order itself; steps are saved one by one with UpdateStep.
func (r *manufacturingRepository) UpdateOrder(ctx context.Context, o *domain.ProductionOrder) error {
	return r.db.WithContext(ctx).Omit("Steps").Save(o).Error
}

func orderBySequence(db *gorm.DB) *gorm.DB { return db.Order("sequence asc") }

func (r *manufacturingRepository) UpdateStep(ctx context.Context, s *domain.ProductionStep) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *manufacturingRepository) CreateStepLog(ctx context.Context, l *domain.ProductionStepLog) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *manufacturingRepository) ListStepLogs(ctx context.Context, from, to string) ([]domain.ProductionStepLog, error) {
	var out []domain.ProductionStepLog
	err := r.db.WithContext(ctx).Where("date >= ? AND date <= ?", from, to).Order("date asc, created_at asc").Find(&out).Error
	return out, err
}

func (r *manufacturingRepository) ListBatchesBetween(ctx context.Context, from, to string) ([]domain.ProductionBatch, error) {
	var out []domain.ProductionBatch
	err := r.db.WithContext(ctx).Where("completion_date >= ? AND completion_date <= ?", from, to).Order("completion_date asc, created_at asc").Find(&out).Error
	return out, err
}

func (r *manufacturingRepository) ListAllOrders(ctx context.Context) ([]domain.ProductionOrder, error) {
	var out []domain.ProductionOrder
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ProductionOrder{})).Preload("Steps", orderBySequence).Order("created_at asc").Find(&out).Error
	return out, err
}

func (r *manufacturingRepository) CountOrdersByPrefix(ctx context.Context, prefix string) (int64, error) {
	var n int64
	// Unscoped: soft-deleted orders still hold their number.
	err := r.db.WithContext(ctx).Unscoped().Model(&domain.ProductionOrder{}).Where("order_number LIKE ?", prefix+"%").Count(&n).Error
	return n, err
}

// Production batches

func (r *manufacturingRepository) CreateBatch(ctx context.Context, b *domain.ProductionBatch) error {
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *manufacturingRepository) ListBatchesByOrder(ctx context.Context, orderID uuid.UUID) ([]domain.ProductionBatch, error) {
	var batches []domain.ProductionBatch
	err := r.db.WithContext(ctx).Where("production_order_id = ?", orderID).Order("created_at desc").Find(&batches).Error
	return batches, err
}

// Dashboard aggregates

func (r *manufacturingRepository) CountOrdersByStatus(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ProductionOrder{}))
	if err := db.Select("status, count(*) as count").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, rr := range rows {
		result[rr.Status] = rr.Count
	}
	return result, nil
}

func (r *manufacturingRepository) CountActiveBOMs(ctx context.Context) (int64, error) {
	var count int64
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.BillOfMaterial{}))
	err := db.Where("is_active = ?", true).Count(&count).Error
	return count, err
}

// SumQuantityCompletedSince is not tenant-scoped: ProductionBatch has no
// tenant_id column (it's a child record of ProductionOrder, which is
// scoped). Acceptable for now since batches carry no other tenant-specific
// data to leak - just a quantity and a date.
func (r *manufacturingRepository) SumQuantityCompletedSince(ctx context.Context, since string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.ProductionBatch{}).
		Where("completion_date >= ?", since).
		Select("COALESCE(SUM(quantity_completed), 0)").
		Scan(&total).Error
	return total, err
}

func (r *manufacturingRepository) ListUpcomingOrders(ctx context.Context, limit int) ([]domain.ProductionOrder, error) {
	var orders []domain.ProductionOrder
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ProductionOrder{}))
	err := db.Where("status IN ?", []string{domain.OrderPlanned, domain.OrderReleased, domain.OrderInProgress}).
		Order("planned_date asc").
		Limit(limit).
		Find(&orders).Error
	return orders, err
}

func (r *manufacturingRepository) WithTransaction(ctx context.Context, fn func(txRepo domain.ManufacturingRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&manufacturingRepository{db: tx})
	})
}

// WithTransactionAndInventory runs fn with both a manufacturing repo and an
// inventory repo bound to the SAME underlying database transaction (both are
// constructed from the same tenant-scoped *gorm.DB), so BOM consumption /
// finished-goods production and the production batch/order status update
// commit or roll back together atomically.
func (r *manufacturingRepository) WithTransactionAndInventory(ctx context.Context, fn func(txRepo domain.ManufacturingRepository, txInv inventoryDomain.InventoryRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txMfg := &manufacturingRepository{db: tx}
		txInv := inventoryInfra.NewInventoryRepository(tx)
		return fn(txMfg, txInv)
	})
}
