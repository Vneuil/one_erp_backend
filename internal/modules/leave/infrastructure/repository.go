package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/leave/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type leaveRepository struct {
	db *gorm.DB
}

func NewLeaveRepository(db *gorm.DB) domain.LeaveRepository {
	return &leaveRepository{db: db}
}

func (r *leaveRepository) Create(ctx context.Context, req *domain.LeaveRequest) error {
	tenantctx.SetTenantID(ctx, &req.TenantID)
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *leaveRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.LeaveRequest, error) {
	var req domain.LeaveRequest
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.LeaveRequest{})).Where("id = ?", id).First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *leaveRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.LeaveRequest, int64, error) {
	var items []domain.LeaveRequest
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.LeaveRequest{}))

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("employee_name ILIKE ? OR department ILIKE ? OR type ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *leaveRepository) Update(ctx context.Context, req *domain.LeaveRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *leaveRepository) FindOverlapping(ctx context.Context, employeeID uuid.UUID, start, end string) ([]domain.LeaveRequest, error) {
	var items []domain.LeaveRequest
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.LeaveRequest{})).
		Where(&domain.LeaveRequest{EmployeeID: employeeID}).
		Where("status IN ?", []string{"pending", "approved"}).
		Where("start_date <= ? AND end_date >= ?", end, start).
		Find(&items).Error
	return items, err
}

func (r *leaveRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.LeaveRequest{})).Count(&total).Error
	return total, err
}
