package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/salesman/domain"
	"github.com/google/uuid"
)

type CreateSalesmanDTO struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone"`
	Territory      string  `json:"territory"`
	CommissionRate float64 `json:"commissionRate"`
	Status         string  `json:"status"`
}

type UpdateSalesmanDTO struct {
	Name           *string  `json:"name,omitempty"`
	Email          *string  `json:"email,omitempty"`
	Phone          *string  `json:"phone,omitempty"`
	Territory      *string  `json:"territory,omitempty"`
	CommissionRate *float64 `json:"commissionRate,omitempty"`
	Status         *string  `json:"status,omitempty"`
}

type SalesmanResponseDTO struct {
	ID             uuid.UUID `json:"id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	Territory      string    `json:"territory"`
	CommissionRate float64   `json:"commissionRate"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func ToSalesmanResponse(c *domain.Salesman) *SalesmanResponseDTO {
	if c == nil {
		return nil
	}
	return &SalesmanResponseDTO{
		ID:             c.ID,
		Code:           c.Code,
		Name:           c.Name,
		Email:          c.Email,
		Phone:          c.Phone,
		Territory:      c.Territory,
		CommissionRate: c.CommissionRate,
		Status:         c.Status,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

func ToSalesmanResponseList(salesmen []domain.Salesman) []SalesmanResponseDTO {
	result := make([]SalesmanResponseDTO, len(salesmen))
	for i, c := range salesmen {
		result[i] = *ToSalesmanResponse(&c)
	}
	return result
}
