package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/shipping/domain"
	"github.com/google/uuid"
)

type CreateShippingMethodDTO struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Carrier       string  `json:"carrier"`
	EstimatedDays int     `json:"estimatedDays"`
	Cost          float64 `json:"cost"`
	Status        string  `json:"status"`
}

type UpdateShippingMethodDTO struct {
	Name          *string  `json:"name,omitempty"`
	Carrier       *string  `json:"carrier,omitempty"`
	EstimatedDays *int     `json:"estimatedDays,omitempty"`
	Cost          *float64 `json:"cost,omitempty"`
	Status        *string  `json:"status,omitempty"`
}

type ShippingMethodResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Carrier       string    `json:"carrier"`
	EstimatedDays int       `json:"estimatedDays"`
	Cost          float64   `json:"cost"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func ToShippingMethodResponse(c *domain.ShippingMethod) *ShippingMethodResponseDTO {
	if c == nil {
		return nil
	}
	return &ShippingMethodResponseDTO{
		ID:            c.ID,
		Code:          c.Code,
		Name:          c.Name,
		Carrier:       c.Carrier,
		EstimatedDays: c.EstimatedDays,
		Cost:          c.Cost,
		Status:        c.Status,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func ToShippingMethodResponseList(items []domain.ShippingMethod) []ShippingMethodResponseDTO {
	result := make([]ShippingMethodResponseDTO, len(items))
	for i, c := range items {
		result[i] = *ToShippingMethodResponse(&c)
	}
	return result
}
