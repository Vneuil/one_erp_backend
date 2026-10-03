package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/assets/domain"
	"github.com/google/uuid"
)

type CreateAssetDTO struct {
	AssetCode       string     `json:"assetCode"`
	Name            string     `json:"name"`
	Category        string     `json:"category"`
	Location        string     `json:"location"`
	PurchaseDate    string     `json:"purchaseDate"`
	PurchasePrice   float64    `json:"purchasePrice"`
	UsefulLifeYears int        `json:"usefulLifeYears"`
	CompanyID       *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateAssetDTO struct {
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	Location        string   `json:"location"`
	PurchaseDate    string   `json:"purchaseDate"`
	PurchasePrice   *float64 `json:"purchasePrice,omitempty"`
	UsefulLifeYears *int     `json:"usefulLifeYears,omitempty"`
}

type UpdateAssetStatusDTO struct {
	Status string `json:"status"`
}

type AssetResponseDTO struct {
	ID                      uuid.UUID  `json:"id"`
	CompanyID               *uuid.UUID `json:"companyId,omitempty"`
	AssetCode               string     `json:"assetCode"`
	Name                    string     `json:"name"`
	Category                string     `json:"category"`
	Location                string     `json:"location"`
	PurchaseDate            string     `json:"purchaseDate"`
	PurchasePrice           float64    `json:"purchasePrice"`
	UsefulLifeYears         int        `json:"usefulLifeYears"`
	AccumulatedDepreciation float64    `json:"accumulatedDepreciation"`
	CurrentBookValue        float64    `json:"currentBookValue"`
	Status                  string     `json:"status"`
	CreatedAt               time.Time  `json:"createdAt"`
}

func ToAssetResponse(a *domain.FixedAsset) *AssetResponseDTO {
	if a == nil {
		return nil
	}
	return &AssetResponseDTO{
		ID:                      a.ID,
		CompanyID:               a.CompanyID,
		AssetCode:               a.AssetCode,
		Name:                    a.Name,
		Category:                a.Category,
		Location:                a.Location,
		PurchaseDate:            a.PurchaseDate,
		PurchasePrice:           a.PurchasePrice,
		UsefulLifeYears:         a.UsefulLifeYears,
		AccumulatedDepreciation: a.AccumulatedDepreciation,
		CurrentBookValue:        a.CurrentBookValue,
		Status:                  a.Status,
		CreatedAt:               a.CreatedAt,
	}
}

func ToAssetResponseList(assets []domain.FixedAsset) []AssetResponseDTO {
	result := make([]AssetResponseDTO, len(assets))
	for i, a := range assets {
		result[i] = *ToAssetResponse(&a)
	}
	return result
}
