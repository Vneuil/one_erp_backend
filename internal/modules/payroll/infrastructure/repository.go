package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type payrollRepository struct {
	db *gorm.DB
}

func NewPayrollRepository(db *gorm.DB) domain.PayrollRepository {
	return &payrollRepository{db: db}
}

func (r *payrollRepository) CreateEntry(ctx context.Context, entry *domain.PayrollEntry) error {
	tenantctx.SetTenantID(ctx, &entry.TenantID)
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *payrollRepository) GetEntryByID(ctx context.Context, id uuid.UUID) (*domain.PayrollEntry, error) {
	var entry domain.PayrollEntry
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("id = ?", id).First(&entry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

func (r *payrollRepository) ListEntries(ctx context.Context, query types.PaginationQuery, period string) ([]domain.PayrollEntry, int64, error) {
	var entries []domain.PayrollEntry
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PayrollEntry{}))

	if period != "" {
		db = db.Where("period = ?", period)
	}

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("employee_name ILIKE ? OR n_ip ILIKE ? OR department ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&entries).Error
	return entries, total, err
}

func (r *payrollRepository) ListEntriesByEmployee(ctx context.Context, employeeID uuid.UUID) ([]domain.PayrollEntry, error) {
	var entries []domain.PayrollEntry
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PayrollEntry{})).Where("employee_id = ?", employeeID).Order("period desc").Find(&entries).Error
	return entries, err
}

func (r *payrollRepository) ListEntriesByPeriodAndStatus(ctx context.Context, period string, status string) ([]domain.PayrollEntry, error) {
	var entries []domain.PayrollEntry
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PayrollEntry{}))
	err := db.Where("period = ? AND status = ?", period, status).Find(&entries).Error
	return entries, err
}

func (r *payrollRepository) UpdateEntry(ctx context.Context, entry *domain.PayrollEntry) error {
	return r.db.WithContext(ctx).Save(entry).Error
}

func (r *payrollRepository) CountEntries(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PayrollEntry{})).Count(&total).Error
	return total, err
}

func (r *payrollRepository) GetPolicy(ctx context.Context) (*domain.PayrollPolicy, error) {
	var found []domain.PayrollPolicy
	if err := r.db.WithContext(ctx).Order("created_at asc").Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		p := domain.DefaultPayrollPolicy()
		return &p, nil
	}
	return &found[0], nil
}

func (r *payrollRepository) SavePolicy(ctx context.Context, p *domain.PayrollPolicy) error {
	return r.db.WithContext(ctx).Save(p).Error
}
