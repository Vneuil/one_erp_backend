package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/support/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ticketsRepository struct {
	db *gorm.DB
}

func NewTicketsRepository(db *gorm.DB) domain.TicketsRepository {
	return &ticketsRepository{db: db}
}

func (r *ticketsRepository) CreateTicket(ctx context.Context, t *domain.Ticket) error {
	tenantctx.SetTenantID(ctx, &t.TenantID)
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *ticketsRepository) GetTicketByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	var t domain.Ticket
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *ticketsRepository) ListTickets(ctx context.Context, query types.PaginationQuery) ([]domain.Ticket, int64, error) {
	var tickets []domain.Ticket
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Ticket{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("ticket_number ILIKE ? OR subject ILIKE ? OR customer_name ILIKE ?", pattern, pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&tickets).Error
	return tickets, total, err
}

func (r *ticketsRepository) ListAllTickets(ctx context.Context) ([]domain.Ticket, error) {
	var tickets []domain.Ticket
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Order("created_at desc").Find(&tickets).Error
	return tickets, err
}

func (r *ticketsRepository) UpdateTicket(ctx context.Context, t *domain.Ticket) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *ticketsRepository) CountTickets(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Ticket{}).Count(&total).Error
	return total, err
}

func (r *ticketsRepository) CreateReply(ctx context.Context, rp *domain.TicketReply) error {
	return r.db.WithContext(ctx).Create(rp).Error
}

func (r *ticketsRepository) ListRepliesByTicket(ctx context.Context, ticketID uuid.UUID) ([]domain.TicketReply, error) {
	var replies []domain.TicketReply
	err := r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Order("created_at asc").Find(&replies).Error
	return replies, err
}
