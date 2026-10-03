package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// ProjectTicket is a service ticket scoped to a Project (loosely, via
// ProjectCode - this codebase denormalizes across module boundaries rather
// than using hard DB foreign keys between modules).
type ProjectTicket struct {
	types.BaseEntity
	CompanyID      *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	TicketNo       string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"ticketNo"`
	ProjectCode    string     `gorm:"type:varchar(50);not null;index" json:"projectCode"`
	Subject        string     `gorm:"type:varchar(255);not null" json:"subject"`
	CustomerName   string     `gorm:"type:varchar(255);not null" json:"customerName"`
	Category       string     `gorm:"type:varchar(100)" json:"category"`
	Priority       string     `gorm:"type:varchar(20);not null" json:"priority"`
	SLATargetHours int        `gorm:"default:0" json:"slaTargetHours"`
	Status         string     `gorm:"type:varchar(20);not null;default:'open'" json:"status"`
	AssignedTo     string     `gorm:"type:varchar(255)" json:"assignedTo"`
}

func (ProjectTicket) TableName() string {
	return "project_tickets"
}

type ProjectTicketRepository interface {
	Create(ctx context.Context, ticket *ProjectTicket) error
	GetByID(ctx context.Context, id uuid.UUID) (*ProjectTicket, error)
	List(ctx context.Context, query types.PaginationQuery, status string) ([]ProjectTicket, int64, error)
	Update(ctx context.Context, ticket *ProjectTicket) error
	Count(ctx context.Context) (int64, error)
}
