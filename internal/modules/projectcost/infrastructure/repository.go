package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/projectcost/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) domain.Repository { return &repository{db: db} }

func (r *repository) q(ctx context.Context, model any) *gorm.DB {
	return tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(model))
}

func (r *repository) ListBudgetItems(ctx context.Context, projectID uuid.UUID, kind string) ([]domain.BudgetItem, error) {
	var out []domain.BudgetItem
	db := r.q(ctx, &domain.BudgetItem{}).Where("project_id = ?", projectID)
	if kind != "" {
		db = db.Where("kind = ?", kind)
	}
	return out, db.Order("kind asc, category asc, created_at asc").Find(&out).Error
}

func (r *repository) AddBudgetItems(ctx context.Context, projectID uuid.UUID, replaceKinds []string, items []domain.BudgetItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(replaceKinds) > 0 {
			if err := tx.Where("project_id = ? AND kind IN ?", projectID, replaceKinds).Delete(&domain.BudgetItem{}).Error; err != nil {
				return err
			}
		}
		if len(items) == 0 {
			return nil
		}
		for i := range items {
			tenantctx.SetTenantID(ctx, &items[i].TenantID)
		}
		return tx.CreateInBatches(&items, 200).Error
	})
}

func (r *repository) DeleteBudgetItem(ctx context.Context, projectID, id uuid.UUID) (bool, error) {
	res := r.q(ctx, &domain.BudgetItem{}).Where("project_id = ? AND id = ?", projectID, id).Delete(&domain.BudgetItem{})
	return res.RowsAffected > 0, res.Error
}

func (r *repository) CreateCost(ctx context.Context, c *domain.CostEntry) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *repository) ListCosts(ctx context.Context, projectID uuid.UUID) ([]domain.CostEntry, error) {
	var out []domain.CostEntry
	return out, r.q(ctx, &domain.CostEntry{}).Where("project_id = ?", projectID).Order("date desc, created_at desc").Find(&out).Error
}

func (r *repository) DeleteCost(ctx context.Context, projectID, id uuid.UUID) (bool, error) {
	res := r.q(ctx, &domain.CostEntry{}).Where("project_id = ? AND id = ?", projectID, id).Delete(&domain.CostEntry{})
	return res.RowsAffected > 0, res.Error
}

func (r *repository) CreateWorkOrder(ctx context.Context, w *domain.WorkOrder) error {
	tenantctx.SetTenantID(ctx, &w.TenantID)
	return r.db.WithContext(ctx).Create(w).Error
}

func (r *repository) GetWorkOrder(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	var w domain.WorkOrder
	if err := r.q(ctx, &domain.WorkOrder{}).Where("id = ?", id).First(&w).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *repository) UpdateWorkOrder(ctx context.Context, w *domain.WorkOrder) error {
	return r.db.WithContext(ctx).Save(w).Error
}

func (r *repository) ListWorkOrders(ctx context.Context, status string, projectID *uuid.UUID) ([]domain.WorkOrder, error) {
	var out []domain.WorkOrder
	db := r.q(ctx, &domain.WorkOrder{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if projectID != nil {
		db = db.Where("project_id = ?", *projectID)
	}
	return out, db.Order("created_at desc").Find(&out).Error
}

// CountWorkOrdersWithPrefix counts numbers beginning with prefix, including
// soft-deleted rows, so a number is never handed out twice.
func (r *repository) CountWorkOrdersWithPrefix(ctx context.Context, prefix string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Unscoped().Model(&domain.WorkOrder{}).Where("number LIKE ?", prefix+"%").Count(&n).Error
	return n, err
}
