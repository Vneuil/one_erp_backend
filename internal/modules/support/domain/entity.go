package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Ticket represents a customer support ticket tracked against an SLA
type Ticket struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this ticket to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant -
	// the default state for companies that never created more than their
	// seeded default Tenant.
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	TicketNumber string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"ticketNumber"`
	Subject      string     `gorm:"type:varchar(255);not null" json:"subject"`
	Description  string     `gorm:"type:text" json:"description"`
	CustomerName string     `gorm:"type:varchar(255);not null" json:"customerName"`
	Priority     string     `gorm:"type:varchar(20);not null" json:"priority"`
	Status       string     `gorm:"type:varchar(20);not null;default:'open'" json:"status"`
	Category     string     `gorm:"type:varchar(100);not null" json:"category"`
	AssignedTo   string     `gorm:"type:varchar(255)" json:"assignedTo"`
	SLADueAt     time.Time  `gorm:"not null" json:"slaDueAt"`
	ResolvedAt   *time.Time `json:"resolvedAt,omitempty"`
}

func (Ticket) TableName() string {
	return "support_tickets"
}

// TicketReply represents a single message in a ticket's reply thread
type TicketReply struct {
	types.BaseEntity
	TicketID   uuid.UUID `gorm:"type:uuid;not null;index" json:"ticketId"`
	AuthorName string    `gorm:"type:varchar(255);not null" json:"authorName"`
	AuthorType string    `gorm:"type:varchar(20);not null" json:"authorType"`
	Message    string    `gorm:"type:text;not null" json:"message"`
}

func (TicketReply) TableName() string {
	return "support_ticket_replies"
}

type TicketSummary struct {
	TotalTickets       int64            `json:"totalTickets"`
	OpenTickets        int64            `json:"openTickets"`
	InProgressTickets  int64            `json:"inProgressTickets"`
	WaitingTickets     int64            `json:"waitingTickets"`
	ResolvedTickets    int64            `json:"resolvedTickets"`
	ClosedTickets      int64            `json:"closedTickets"`
	OverdueTickets     int64            `json:"overdueTickets"`
	ByPriority         map[string]int64 `json:"byPriority"`
	AvgResolutionHours float64          `json:"avgResolutionHours"`
}

type TicketsRepository interface {
	CreateTicket(ctx context.Context, t *Ticket) error
	GetTicketByID(ctx context.Context, id uuid.UUID) (*Ticket, error)
	ListTickets(ctx context.Context, query types.PaginationQuery) ([]Ticket, int64, error)
	ListAllTickets(ctx context.Context) ([]Ticket, error)
	UpdateTicket(ctx context.Context, t *Ticket) error
	CountTickets(ctx context.Context) (int64, error)

	CreateReply(ctx context.Context, r *TicketReply) error
	ListRepliesByTicket(ctx context.Context, ticketID uuid.UUID) ([]TicketReply, error)
}
