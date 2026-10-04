package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// FixedAsset represents a company fixed asset tracked for depreciation
type FixedAsset struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this asset to a branch/subsidiary Tenant within the
	// Company's database (nil for companies with only their default Tenant).
	TenantID                *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	AssetCode               string     `gorm:"type:varchar(50);not null;index" json:"assetCode"`
	Name                    string     `gorm:"type:varchar(255);not null" json:"name"`
	Category                string     `gorm:"type:varchar(50);not null" json:"category"`
	Location                string     `gorm:"type:varchar(255)" json:"location"`
	PurchaseDate            string     `gorm:"type:varchar(50);not null" json:"purchaseDate"`
	PurchasePrice           float64    `gorm:"type:decimal(15,2);not null" json:"purchasePrice"`
	UsefulLifeYears         int        `gorm:"not null" json:"usefulLifeYears"`
	AccumulatedDepreciation float64    `gorm:"type:decimal(15,2);default:0" json:"accumulatedDepreciation"`
	CurrentBookValue        float64    `gorm:"type:decimal(15,2);not null" json:"currentBookValue"`
	Status                  string     `gorm:"type:varchar(50);default:'In Use'" json:"status"`
}

func (FixedAsset) TableName() string {
	return "assets_fixed_assets"
}

// DepreciationPosting records that one asset's depreciation for one month
// (YYYY-MM) has been posted to the general ledger. (asset, period) is unique,
// so a month can never be posted twice for the same asset.
type DepreciationPosting struct {
	types.BaseEntity
	AssetID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_dep_asset_period" json:"assetId"`
	Period  string    `gorm:"type:varchar(7);not null;uniqueIndex:idx_dep_asset_period;index" json:"period"`
	Amount  float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	// JournalSourceDoc is the source-doc key of the ledger entry.
	JournalSourceDoc string `gorm:"type:varchar(120)" json:"journalSourceDoc"`
}

func (DepreciationPosting) TableName() string { return "assets_depreciation_postings" }

type AssetsRepository interface {
	CreateDepreciationPosting(ctx context.Context, p *DepreciationPosting) error
	// ListDepreciationPostings returns postings for one period, or all when period is empty.
	ListDepreciationPostings(ctx context.Context, period string) ([]DepreciationPosting, error)

	CreateAsset(ctx context.Context, a *FixedAsset) error
	GetAssetByID(ctx context.Context, id uuid.UUID) (*FixedAsset, error)
	ListAssets(ctx context.Context, query types.PaginationQuery) ([]FixedAsset, int64, error)
	ListAllAssets(ctx context.Context) ([]FixedAsset, error)
	UpdateAsset(ctx context.Context, a *FixedAsset) error
	CountAssets(ctx context.Context) (int64, error)
}
