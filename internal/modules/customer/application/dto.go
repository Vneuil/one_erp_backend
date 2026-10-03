package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	"github.com/google/uuid"
)

type CreateCustomerDTO struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Segment string `json:"segment"`
	Status  string `json:"status"`
}

type UpdateCustomerDTO struct {
	Name    *string `json:"name,omitempty"`
	Email   *string `json:"email,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Address *string `json:"address,omitempty"`
	Segment *string `json:"segment,omitempty"`
	Status  *string `json:"status,omitempty"`
}

type CustomerResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	Segment   string    `json:"segment"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func ToCustomerResponse(c *domain.Customer) *CustomerResponseDTO {
	if c == nil {
		return nil
	}
	return &CustomerResponseDTO{
		ID:        c.ID,
		Code:      c.Code,
		Name:      c.Name,
		Email:     c.Email,
		Phone:     c.Phone,
		Address:   c.Address,
		Segment:   c.Segment,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func ToCustomerResponseList(customers []domain.Customer) []CustomerResponseDTO {
	result := make([]CustomerResponseDTO, len(customers))
	for i, c := range customers {
		result[i] = *ToCustomerResponse(&c)
	}
	return result
}
