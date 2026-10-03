package application

import (
	"github.com/divinecoid/one-backend/internal/modules/pricing/domain"
	"github.com/google/uuid"
)

type CreatePaymentTermDTO struct {
	Name string `json:"name"`
	Days int    `json:"days"`
}

type UpdatePaymentTermDTO struct {
	Name string `json:"name"`
	Days int    `json:"days"`
}

type PaymentTermResponseDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Days int       `json:"days"`
}

func ToPaymentTermResponseList(items []domain.PaymentTerm) []PaymentTermResponseDTO {
	result := make([]PaymentTermResponseDTO, len(items))
	for i, p := range items {
		result[i] = PaymentTermResponseDTO{ID: p.ID, Name: p.Name, Days: p.Days}
	}
	return result
}

type CreateCustomerTypeDTO struct {
	Name string `json:"name"`
}

type UpdateCustomerTypeDTO struct {
	Name string `json:"name"`
}

type CustomerTypeResponseDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func ToCustomerTypeResponseList(items []domain.CustomerType) []CustomerTypeResponseDTO {
	result := make([]CustomerTypeResponseDTO, len(items))
	for i, c := range items {
		result[i] = CustomerTypeResponseDTO{ID: c.ID, Name: c.Name}
	}
	return result
}

type CreateTaxRateDTO struct {
	Name      string  `json:"name"`
	Rate      float64 `json:"rate"`
	IsDefault bool    `json:"isDefault"`
}

type UpdateTaxRateDTO struct {
	Name      string  `json:"name"`
	Rate      float64 `json:"rate"`
	IsDefault bool    `json:"isDefault"`
}

type TaxRateResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Rate      float64   `json:"rate"`
	IsDefault bool      `json:"isDefault"`
}

func ToTaxRateResponseList(items []domain.TaxRate) []TaxRateResponseDTO {
	result := make([]TaxRateResponseDTO, len(items))
	for i, t := range items {
		result[i] = TaxRateResponseDTO{ID: t.ID, Name: t.Name, Rate: t.Rate, IsDefault: t.IsDefault}
	}
	return result
}
