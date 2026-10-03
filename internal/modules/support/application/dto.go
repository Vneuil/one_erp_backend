package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/support/domain"
	"github.com/google/uuid"
)

type CreateTicketDTO struct {
	Subject      string     `json:"subject"`
	Description  string     `json:"description"`
	CustomerName string     `json:"customerName"`
	Priority     string     `json:"priority"`
	Category     string     `json:"category"`
	AssignedTo   string     `json:"assignedTo"`
	CompanyID    *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateTicketDTO struct {
	Subject      string `json:"subject"`
	Description  string `json:"description"`
	CustomerName string `json:"customerName"`
	Priority     string `json:"priority"`
	Category     string `json:"category"`
	AssignedTo   string `json:"assignedTo"`
}

type UpdateStatusDTO struct {
	Status string `json:"status"`
}

type ReplyDTO struct {
	AuthorName string `json:"authorName"`
	AuthorType string `json:"authorType"`
	Message    string `json:"message"`
}

type TicketResponseDTO struct {
	ID           uuid.UUID  `json:"id"`
	CompanyID    *uuid.UUID `json:"companyId,omitempty"`
	TicketNumber string     `json:"ticketNumber"`
	Subject      string     `json:"subject"`
	Description  string     `json:"description"`
	CustomerName string     `json:"customerName"`
	Priority     string     `json:"priority"`
	Status       string     `json:"status"`
	Category     string     `json:"category"`
	AssignedTo   string     `json:"assignedTo"`
	SLADueAt     time.Time  `json:"slaDueAt"`
	ResolvedAt   *time.Time `json:"resolvedAt,omitempty"`
	IsOverdue    bool       `json:"isOverdue"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type ReplyResponseDTO struct {
	ID         uuid.UUID `json:"id"`
	TicketID   uuid.UUID `json:"ticketId"`
	AuthorName string    `json:"authorName"`
	AuthorType string    `json:"authorType"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"createdAt"`
}

func isOverdue(t *domain.Ticket) bool {
	if t.Status == "resolved" || t.Status == "closed" {
		return false
	}
	return time.Now().After(t.SLADueAt)
}

func ToTicketResponse(t *domain.Ticket) *TicketResponseDTO {
	if t == nil {
		return nil
	}
	return &TicketResponseDTO{
		ID:           t.ID,
		CompanyID:    t.CompanyID,
		TicketNumber: t.TicketNumber,
		Subject:      t.Subject,
		Description:  t.Description,
		CustomerName: t.CustomerName,
		Priority:     t.Priority,
		Status:       t.Status,
		Category:     t.Category,
		AssignedTo:   t.AssignedTo,
		SLADueAt:     t.SLADueAt,
		ResolvedAt:   t.ResolvedAt,
		IsOverdue:    isOverdue(t),
		CreatedAt:    t.CreatedAt,
	}
}

func ToTicketResponseList(tickets []domain.Ticket) []TicketResponseDTO {
	result := make([]TicketResponseDTO, len(tickets))
	for i, t := range tickets {
		result[i] = *ToTicketResponse(&t)
	}
	return result
}

func ToReplyResponse(r *domain.TicketReply) *ReplyResponseDTO {
	if r == nil {
		return nil
	}
	return &ReplyResponseDTO{
		ID:         r.ID,
		TicketID:   r.TicketID,
		AuthorName: r.AuthorName,
		AuthorType: r.AuthorType,
		Message:    r.Message,
		CreatedAt:  r.CreatedAt,
	}
}

func ToReplyResponseList(replies []domain.TicketReply) []ReplyResponseDTO {
	result := make([]ReplyResponseDTO, len(replies))
	for i, r := range replies {
		result[i] = *ToReplyResponse(&r)
	}
	return result
}
