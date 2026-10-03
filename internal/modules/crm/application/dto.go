package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/google/uuid"
)

type CreateLeadDTO struct {
	Name           string  `json:"name"`
	Company        string  `json:"company"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone"`
	Segment        string  `json:"segment"`
	Source         string  `json:"source"`
	EstimatedValue float64 `json:"estimatedValue"`
	Status         string  `json:"status"`
	PIC            string  `json:"pic"`
}

type UpdateLeadDTO struct {
	Name           *string  `json:"name,omitempty"`
	Company        *string  `json:"company,omitempty"`
	Email          *string  `json:"email,omitempty"`
	Phone          *string  `json:"phone,omitempty"`
	Segment        *string  `json:"segment,omitempty"`
	Source         *string  `json:"source,omitempty"`
	EstimatedValue *float64 `json:"estimatedValue,omitempty"`
	Status         *string  `json:"status,omitempty"`
	PIC            *string  `json:"pic,omitempty"`
}

type LeadResponseDTO struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Company        string    `json:"company"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	Segment        string    `json:"segment"`
	Source         string    `json:"source"`
	EstimatedValue float64   `json:"estimatedValue"`
	Status         string    `json:"status"`
	PIC            string    `json:"pic"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func ToLeadResponse(l *domain.Lead) *LeadResponseDTO {
	if l == nil {
		return nil
	}
	return &LeadResponseDTO{
		ID:             l.ID,
		Name:           l.Name,
		Company:        l.Company,
		Email:          l.Email,
		Phone:          l.Phone,
		Segment:        l.Segment,
		Source:         l.Source,
		EstimatedValue: l.EstimatedValue,
		Status:         l.Status,
		PIC:            l.PIC,
		CreatedAt:      l.CreatedAt,
		UpdatedAt:      l.UpdatedAt,
	}
}

func ToLeadResponseList(leads []domain.Lead) []LeadResponseDTO {
	result := make([]LeadResponseDTO, len(leads))
	for i, l := range leads {
		result[i] = *ToLeadResponse(&l)
	}
	return result
}

type CreateDealDTO struct {
	Title           string  `json:"title"`
	Customer        string  `json:"customer"`
	Value           float64 `json:"value"`
	Stage           string  `json:"stage"`
	PIC             string  `json:"pic"`
	ExpectedClosing string  `json:"expectedClosing"`
	// LeadID optionally records the lead this deal came from.
	LeadID *uuid.UUID `json:"leadId,omitempty"`
}

type UpdateDealDTO struct {
	Title           *string  `json:"title,omitempty"`
	Customer        *string  `json:"customer,omitempty"`
	Value           *float64 `json:"value,omitempty"`
	PIC             *string  `json:"pic,omitempty"`
	ExpectedClosing *string  `json:"expectedClosing,omitempty"`
}

type UpdateDealStageDTO struct {
	Stage      string `json:"stage"`
	LostReason string `json:"lostReason,omitempty"`
}

type DealResponseDTO struct {
	ID              uuid.UUID  `json:"id"`
	Title           string     `json:"title"`
	Customer        string     `json:"customer"`
	Value           float64    `json:"value"`
	Probability     int        `json:"probability"`
	Stage           string     `json:"stage"`
	PIC             string     `json:"pic"`
	ExpectedClosing string     `json:"expectedClosing"`
	LostReason      string     `json:"lostReason,omitempty"`
	LeadID          *uuid.UUID `json:"leadId,omitempty"`
	ProjectID       *uuid.UUID `json:"projectId,omitempty"`
	ClosedAt        *time.Time `json:"closedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func ToDealResponse(d *domain.Deal) *DealResponseDTO {
	if d == nil {
		return nil
	}
	return &DealResponseDTO{
		ID:              d.ID,
		Title:           d.Title,
		Customer:        d.Customer,
		Value:           d.Value,
		Probability:     d.Probability,
		Stage:           d.Stage,
		PIC:             d.PIC,
		ExpectedClosing: d.ExpectedClosing,
		LostReason:      d.LostReason,
		LeadID:          d.LeadID,
		ProjectID:       d.ProjectID,
		ClosedAt:        d.ClosedAt,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

func ToDealResponseList(deals []domain.Deal) []DealResponseDTO {
	result := make([]DealResponseDTO, len(deals))
	for i, d := range deals {
		result[i] = *ToDealResponse(&d)
	}
	return result
}
