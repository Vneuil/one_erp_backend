package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	"github.com/google/uuid"
)

type CreateSupplierDTO struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	ContactPerson string `json:"contactPerson"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Address       string `json:"address"`
	NPWP          string `json:"npwp"`
	NIK           string `json:"nik"`
	Category      string `json:"category"`
	Status        string `json:"status"`
}

type UpdateSupplierDTO struct {
	Name          *string `json:"name,omitempty"`
	ContactPerson *string `json:"contactPerson,omitempty"`
	Email         *string `json:"email,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Address       *string `json:"address,omitempty"`
	NPWP          *string `json:"npwp,omitempty"`
	NIK           *string `json:"nik,omitempty"`
	Category      *string `json:"category,omitempty"`
	Status        *string `json:"status,omitempty"`
}

type SupplierResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	ContactPerson string    `json:"contactPerson"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	Address       string    `json:"address"`
	NPWP          string    `json:"npwp"`
	NIK           string    `json:"nik"`
	Category      string    `json:"category"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func ToSupplierResponse(s *domain.Supplier) *SupplierResponseDTO {
	if s == nil {
		return nil
	}
	return &SupplierResponseDTO{
		ID:            s.ID,
		Code:          s.Code,
		Name:          s.Name,
		ContactPerson: s.ContactPerson,
		Email:         s.Email,
		Phone:         s.Phone,
		Address:       s.Address,
		NPWP:          s.NPWP,
		NIK:           s.NIK,
		Category:      s.Category,
		Status:        s.Status,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func ToSupplierResponseList(suppliers []domain.Supplier) []SupplierResponseDTO {
	result := make([]SupplierResponseDTO, len(suppliers))
	for i, s := range suppliers {
		result[i] = *ToSupplierResponse(&s)
	}
	return result
}
