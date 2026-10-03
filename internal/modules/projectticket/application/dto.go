package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/projectticket/domain"
	"github.com/google/uuid"
)

type CreateProjectTicketDTO struct {
	ProjectCode    string `json:"projectCode"`
	Subject        string `json:"subject"`
	CustomerName   string `json:"customerName"`
	Category       string `json:"category"`
	Priority       string `json:"priority"`
	SLATargetHours int    `json:"slaTargetHours"`
	AssignedTo     string `json:"assignedTo"`
}

type UpdateStatusDTO struct {
	Status string `json:"status"`
}

type ProjectTicketResponseDTO struct {
	ID             uuid.UUID `json:"id"`
	TicketNo       string    `json:"ticketNo"`
	ProjectCode    string    `json:"projectCode"`
	Subject        string    `json:"subject"`
	CustomerName   string    `json:"customerName"`
	Category       string    `json:"category"`
	Priority       string    `json:"priority"`
	SLATargetHours int       `json:"slaTargetHours"`
	Status         string    `json:"status"`
	AssignedTo     string    `json:"assignedTo"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func ToProjectTicketResponse(t *domain.ProjectTicket) *ProjectTicketResponseDTO {
	if t == nil {
		return nil
	}
	return &ProjectTicketResponseDTO{
		ID:             t.ID,
		TicketNo:       t.TicketNo,
		ProjectCode:    t.ProjectCode,
		Subject:        t.Subject,
		CustomerName:   t.CustomerName,
		Category:       t.Category,
		Priority:       t.Priority,
		SLATargetHours: t.SLATargetHours,
		Status:         t.Status,
		AssignedTo:     t.AssignedTo,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

func ToProjectTicketResponseList(tickets []domain.ProjectTicket) []ProjectTicketResponseDTO {
	result := make([]ProjectTicketResponseDTO, len(tickets))
	for i, t := range tickets {
		result[i] = *ToProjectTicketResponse(&t)
	}
	return result
}
