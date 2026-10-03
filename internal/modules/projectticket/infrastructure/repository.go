package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/projectticket/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type projectTicketRepository struct {
	db *gorm.DB
}

func NewProjectTicketRepository(db *gorm.DB) domain.ProjectTicketRepository {
	return &projectTicketRepository{db: db}
}

func (r *projectTicketRepository) Create(ctx context.Context, ticket *domain.ProjectTicket) error {
	tenantctx.SetTenantID(ctx, &ticket.TenantID)
	return r.db.WithContext(ctx).Create(ticket).Error
}

func (r *projectTicketRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTicket, error) {
	var ticket domain.ProjectTicket
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ticket, nil
}

func (r *projectTicketRepository) List(ctx context.Context, query types.PaginationQuery, status string) ([]domain.ProjectTicket, int64, error) {
	var tickets []domain.ProjectTicket
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ProjectTicket{}))

	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("ticket_no ILIKE ? OR subject ILIKE ? OR customer_name ILIKE ?", pattern, pattern, pattern)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&tickets).Error
	return tickets, total, err
}

func (r *projectTicketRepository) Update(ctx context.Context, ticket *domain.ProjectTicket) error {
	return r.db.WithContext(ctx).Save(ticket).Error
}

func (r *projectTicketRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.ProjectTicket{}).Count(&total).Error
	return total, err
}
