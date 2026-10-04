package application

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/assets/domain"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var validCategories = map[string]bool{
	"Machinery & Equipment": true,
	"Vehicles":              true,
	"IT & Computers":        true,
	"Furniture & Office":    true,
}

var validStatuses = map[string]bool{
	"In Use":      true,
	"Maintenance": true,
	"Disposed":    true,
}

type AssetsUseCase interface {
	CreateAsset(ctx context.Context, dto CreateAssetDTO) (*AssetResponseDTO, error)
	GetAssetByID(ctx context.Context, id uuid.UUID) (*AssetResponseDTO, error)
	ListAssets(ctx context.Context, query types.PaginationQuery) ([]AssetResponseDTO, types.PaginationMeta, error)
	UpdateAsset(ctx context.Context, id uuid.UUID, dto UpdateAssetDTO) (*AssetResponseDTO, error)
	UpdateAssetStatus(ctx context.Context, id uuid.UUID, dto UpdateAssetStatusDTO) (*AssetResponseDTO, error)
	RecalculateDepreciation(ctx context.Context, id uuid.UUID) (*AssetResponseDTO, error)
	RecalculateAllDepreciation(ctx context.Context) ([]AssetResponseDTO, error)
	PostDepreciation(ctx context.Context, period string, catchUp bool) (*DepreciationPostResultDTO, error)
	DepreciationReport(ctx context.Context, period string) (*DepreciationReportDTO, error)

	SeedInitialData(ctx context.Context) error
}

type assetsUseCase struct {
	repo   domain.AssetsRepository
	ledger financeApp.LedgerPoster
}

// Option configures optional collaborators of the assets use case.
type Option func(*assetsUseCase)

// WithLedger enables posting depreciation to the general ledger.
func WithLedger(l financeApp.LedgerPoster) Option {
	return func(uc *assetsUseCase) { uc.ledger = l }
}

func NewAssetsUseCase(repo domain.AssetsRepository, opts ...Option) AssetsUseCase {
	uc := &assetsUseCase{repo: repo}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

// computeDepreciation applies straight-line depreciation as of now.
func computeDepreciation(purchasePrice float64, usefulLifeYears int, purchaseDate string) (accumulated, bookValue float64) {
	if usefulLifeYears <= 0 {
		return 0, purchasePrice
	}
	purchased, err := time.Parse("2006-01-02", purchaseDate)
	if err != nil {
		return 0, purchasePrice
	}
	now := time.Now()
	if now.Before(purchased) {
		return 0, purchasePrice
	}
	yearsElapsed := now.Sub(purchased).Hours() / 24 / 365.25
	annualDepreciation := purchasePrice / float64(usefulLifeYears)
	accumulated = annualDepreciation * yearsElapsed
	if accumulated > purchasePrice {
		accumulated = purchasePrice
	}
	if accumulated < 0 {
		accumulated = 0
	}
	bookValue = purchasePrice - accumulated
	return accumulated, bookValue
}

func (uc *assetsUseCase) validateAssetInput(price float64, usefulLifeYears int) error {
	if usefulLifeYears <= 0 {
		return apperrors.NewBadRequest("Useful life years must be positive")
	}
	if price < 0 {
		return apperrors.NewBadRequest("Purchase price cannot be negative")
	}
	return nil
}

func (uc *assetsUseCase) CreateAsset(ctx context.Context, dto CreateAssetDTO) (*AssetResponseDTO, error) {
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("Asset name is required")
	}
	if !validCategories[dto.Category] {
		return nil, apperrors.NewBadRequest("Category must be one of Machinery & Equipment, Vehicles, IT & Computers, Furniture & Office")
	}
	if err := uc.validateAssetInput(dto.PurchasePrice, dto.UsefulLifeYears); err != nil {
		return nil, err
	}

	purchaseDate := dto.PurchaseDate
	if purchaseDate == "" {
		purchaseDate = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", purchaseDate); err != nil {
		return nil, apperrors.NewBadRequest("Purchase date must be in YYYY-MM-DD format")
	}

	assetCode := dto.AssetCode
	if assetCode == "" {
		assetCode = "AST-" + time.Now().Format("200601") + "-" + uuid.New().String()[:6]
	}

	accumulated, bookValue := computeDepreciation(dto.PurchasePrice, dto.UsefulLifeYears, purchaseDate)

	a := &domain.FixedAsset{
		CompanyID:               dto.CompanyID,
		AssetCode:               assetCode,
		Name:                    dto.Name,
		Category:                dto.Category,
		Location:                dto.Location,
		PurchaseDate:            purchaseDate,
		PurchasePrice:           dto.PurchasePrice,
		UsefulLifeYears:         dto.UsefulLifeYears,
		AccumulatedDepreciation: accumulated,
		CurrentBookValue:        bookValue,
		Status:                  "In Use",
	}

	if err := uc.repo.CreateAsset(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create asset")
	}
	return ToAssetResponse(a), nil
}

func (uc *assetsUseCase) GetAssetByID(ctx context.Context, id uuid.UUID) (*AssetResponseDTO, error) {
	a, err := uc.repo.GetAssetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get asset")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Asset not found")
	}
	return ToAssetResponse(a), nil
}

func (uc *assetsUseCase) ListAssets(ctx context.Context, query types.PaginationQuery) ([]AssetResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	assets, total, err := uc.repo.ListAssets(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list assets")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToAssetResponseList(assets), meta, nil
}

func (uc *assetsUseCase) UpdateAsset(ctx context.Context, id uuid.UUID, dto UpdateAssetDTO) (*AssetResponseDTO, error) {
	a, err := uc.repo.GetAssetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get asset")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Asset not found")
	}
	if dto.Name != "" {
		a.Name = dto.Name
	}
	if dto.Category != "" {
		if !validCategories[dto.Category] {
			return nil, apperrors.NewBadRequest("Category must be one of Machinery & Equipment, Vehicles, IT & Computers, Furniture & Office")
		}
		a.Category = dto.Category
	}
	if dto.Location != "" {
		a.Location = dto.Location
	}
	if dto.PurchaseDate != "" {
		if _, err := time.Parse("2006-01-02", dto.PurchaseDate); err != nil {
			return nil, apperrors.NewBadRequest("Purchase date must be in YYYY-MM-DD format")
		}
		a.PurchaseDate = dto.PurchaseDate
	}
	if dto.PurchasePrice != nil {
		a.PurchasePrice = *dto.PurchasePrice
	}
	if dto.UsefulLifeYears != nil {
		a.UsefulLifeYears = *dto.UsefulLifeYears
	}
	if err := uc.validateAssetInput(a.PurchasePrice, a.UsefulLifeYears); err != nil {
		return nil, err
	}

	a.AccumulatedDepreciation, a.CurrentBookValue = computeDepreciation(a.PurchasePrice, a.UsefulLifeYears, a.PurchaseDate)

	if err := uc.repo.UpdateAsset(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update asset")
	}
	return ToAssetResponse(a), nil
}

func (uc *assetsUseCase) UpdateAssetStatus(ctx context.Context, id uuid.UUID, dto UpdateAssetStatusDTO) (*AssetResponseDTO, error) {
	if !validStatuses[dto.Status] {
		return nil, apperrors.NewBadRequest("Status must be one of In Use, Maintenance, Disposed")
	}
	a, err := uc.repo.GetAssetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get asset")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Asset not found")
	}
	a.Status = dto.Status
	if err := uc.repo.UpdateAsset(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update asset status")
	}
	return ToAssetResponse(a), nil
}

func (uc *assetsUseCase) RecalculateDepreciation(ctx context.Context, id uuid.UUID) (*AssetResponseDTO, error) {
	a, err := uc.repo.GetAssetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get asset")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Asset not found")
	}
	a.AccumulatedDepreciation, a.CurrentBookValue = computeDepreciation(a.PurchasePrice, a.UsefulLifeYears, a.PurchaseDate)
	if err := uc.repo.UpdateAsset(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to recalculate depreciation")
	}
	return ToAssetResponse(a), nil
}

func (uc *assetsUseCase) RecalculateAllDepreciation(ctx context.Context) ([]AssetResponseDTO, error) {
	assets, err := uc.repo.ListAllAssets(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list assets")
	}
	for i := range assets {
		a := &assets[i]
		a.AccumulatedDepreciation, a.CurrentBookValue = computeDepreciation(a.PurchasePrice, a.UsefulLifeYears, a.PurchaseDate)
		if err := uc.repo.UpdateAsset(ctx, a); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to recalculate depreciation")
		}
	}
	return ToAssetResponseList(assets), nil
}

// SeedInitialData populates a few sample fixed assets on first boot
func (uc *assetsUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountAssets(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	assets := []CreateAssetDTO{
		{
			AssetCode:       "AST-MCH-001",
			Name:            "CNC Milling Machine",
			Category:        "Machinery & Equipment",
			Location:        "Main Factory - Line A",
			PurchaseDate:    "2022-03-15",
			PurchasePrice:   450000000,
			UsefulLifeYears: 10,
		},
		{
			AssetCode:       "AST-VHC-001",
			Name:            "Toyota Hilux Pickup",
			Category:        "Vehicles",
			Location:        "Logistics Depot",
			PurchaseDate:    "2023-07-01",
			PurchasePrice:   320000000,
			UsefulLifeYears: 8,
		},
		{
			AssetCode:       "AST-ITC-001",
			Name:            "Dell PowerEdge Server",
			Category:        "IT & Computers",
			Location:        "Head Office - Server Room",
			PurchaseDate:    "2024-01-10",
			PurchasePrice:   85000000,
			UsefulLifeYears: 5,
		},
	}
	for _, a := range assets {
		_, _ = uc.CreateAsset(ctx, a)
	}
	return nil
}
