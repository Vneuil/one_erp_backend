package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/currency/domain"
	"github.com/google/uuid"
)

type CreateCurrencyDTO struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type CurrencyResponseDTO struct {
	ID       uuid.UUID `json:"id"`
	Code     string    `json:"code"`
	Name     string    `json:"name"`
	Symbol   string    `json:"symbol"`
	IsBase   bool      `json:"isBase"`
	IsActive bool      `json:"isActive"`
}

func ToCurrencyResponse(c *domain.Currency) CurrencyResponseDTO {
	return CurrencyResponseDTO{ID: c.ID, Code: c.Code, Name: c.Name, Symbol: c.Symbol, IsBase: c.IsBase, IsActive: c.IsActive}
}

func ToCurrencyResponseList(items []domain.Currency) []CurrencyResponseDTO {
	result := make([]CurrencyResponseDTO, len(items))
	for i, c := range items {
		result[i] = ToCurrencyResponse(&c)
	}
	return result
}

type SetExchangeRateDTO struct {
	CurrencyCode string  `json:"currencyCode"`
	RateDate     string  `json:"rateDate"`
	RateToBase   float64 `json:"rateToBase"`
}

type ExchangeRateResponseDTO struct {
	ID           uuid.UUID `json:"id"`
	CurrencyCode string    `json:"currencyCode"`
	RateDate     string    `json:"rateDate"`
	RateToBase   float64   `json:"rateToBase"`
	CreatedAt    time.Time `json:"createdAt"`
}

func ToExchangeRateResponse(r *domain.ExchangeRate) ExchangeRateResponseDTO {
	return ExchangeRateResponseDTO{ID: r.ID, CurrencyCode: r.CurrencyCode, RateDate: r.RateDate, RateToBase: r.RateToBase, CreatedAt: r.CreatedAt}
}

func ToExchangeRateResponseList(items []domain.ExchangeRate) []ExchangeRateResponseDTO {
	result := make([]ExchangeRateResponseDTO, len(items))
	for i, r := range items {
		result[i] = ToExchangeRateResponse(&r)
	}
	return result
}

type ConvertResultDTO struct {
	OriginalAmount   float64 `json:"originalAmount"`
	OriginalCurrency string  `json:"originalCurrency"`
	BaseAmount       float64 `json:"baseAmount"`
	BaseCurrency     string  `json:"baseCurrency"`
	RateToBase       float64 `json:"rateToBase"`
	RateDate         string  `json:"rateDate"`
}
