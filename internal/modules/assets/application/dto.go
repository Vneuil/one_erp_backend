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
	// FundingSource (cash | payable | equity) posts the acquisition journal
	// right away; empty records the asset without a journal.
	FundingSource string `json:"fundingSource"`
}

// PostAcquisitionDTO posts the acquisition journal of an asset recorded
// without one. "equity" books it against owner's equity (opening balance).
type PostAcquisitionDTO struct {
	FundingSource string `json:"fundingSource"`
	Date          string `json:"date"`
}

// DisposeAssetDTO sells or writes off an asset. Proceeds are received in
// cash/bank by default; ReceivedIn "receivable" books them as an amount the
// buyer (BuyerName, required) still owes, due on DueDate.
type DisposeAssetDTO struct {
	Date       string  `json:"date"`
	Proceeds   float64 `json:"proceeds"`
	Notes      string  `json:"notes"`
	ReceivedIn string  `json:"receivedIn"`
	BuyerName  string  `json:"buyerName"`
	DueDate    string  `json:"dueDate"`
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
	AcquisitionPosted       bool       `json:"acquisitionPosted"`
	DisposedAt              string     `json:"disposedAt,omitempty"`
	DisposalProceeds        float64    `json:"disposalProceeds"`
	DisposalGainLoss        float64    `json:"disposalGainLoss"`
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
		AcquisitionPosted:       a.AcquisitionDoc != "",
		DisposedAt:              a.DisposedAt,
		DisposalProceeds:        a.DisposalProceeds,
		DisposalGainLoss:        a.DisposalGainLoss,
	}
}

func ToAssetResponseList(assets []domain.FixedAsset) []AssetResponseDTO {
	result := make([]AssetResponseDTO, len(assets))
	for i, a := range assets {
		result[i] = *ToAssetResponse(&a)
	}
	return result
}
