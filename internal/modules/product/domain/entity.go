package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Product represents an item in inventory/catalog
type Product struct {
	Barcode string `gorm:"type:varchar(100)" json:"barcode"`
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this product to one business unit within the company
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	SKU          string     `gorm:"type:varchar(50);not null;index" json:"sku"`
	Name         string     `gorm:"type:varchar(255);not null" json:"name"`
	Category     string     `gorm:"type:varchar(100);not null" json:"category"`
	Unit         string     `gorm:"type:varchar(20);not null" json:"unit"`
	Stock        int        `gorm:"default:0" json:"stock"`
	CostPrice    float64    `gorm:"type:decimal(15,2);default:0" json:"costPrice"`
	SellingPrice float64    `gorm:"type:decimal(15,2);default:0" json:"sellingPrice"`
	Status       string     `gorm:"type:varchar(50);default:'in_stock'" json:"status"`
	// VariantOf points a variant (e.g. "Merah / L") at its base product; nil for
	// ordinary products. Variants are one level deep and are sold, priced and
	// stocked as products in their own right.
	VariantOf    *uuid.UUID `gorm:"type:uuid;index" json:"variantOf,omitempty"`
	VariantLabel string     `gorm:"type:varchar(100)" json:"variantLabel,omitempty"`
	// MarketplacePlatform/MarketplaceProductID identify the marketplace
	// listing this product was synced from (e.g. tiktok_shop + TikTok's own
	// product id), set by the marketplace module's product sync.
	MarketplacePlatform  string `gorm:"type:varchar(30);index" json:"marketplacePlatform,omitempty"`
	MarketplaceProductID string `gorm:"type:varchar(150);index" json:"marketplaceProductId,omitempty"`
}

func (Product) TableName() string {
	return "products"
}

// ProductCategory and UnitOfMeasure are simple named master-data lists.
// Product.Category/Unit stay free-text (no FK) to avoid a breaking schema
// change - these give admins a canonical list to pick from in the UI.
type ProductCategory struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name      string     `gorm:"type:varchar(100);not null;index" json:"name"`
}

func (ProductCategory) TableName() string {
	return "product_categories"
}

type UnitOfMeasure struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name      string     `gorm:"type:varchar(50);not null;index" json:"name"`
	Symbol    string     `gorm:"type:varchar(10)" json:"symbol"`
}

func (UnitOfMeasure) TableName() string {
	return "units_of_measure"
}

type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	GetBySKU(ctx context.Context, sku string) (*Product, error)
	List(ctx context.Context, query types.PaginationQuery) ([]Product, int64, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
	// CountVariants counts the variants of a base product.
	CountVariants(ctx context.Context, baseID uuid.UUID) (int64, error)
	// HasReferences reports whether the product is still referenced by any
	// stock level, stock movement, stock transfer, stock opname line, or BOM
	// record, so Delete can be blocked the same way warehouse delete is
	// blocked when stock level rows still reference the warehouse.
	HasReferences(ctx context.Context, id uuid.UUID) (bool, error)

	CreateCategory(ctx context.Context, c *ProductCategory) error
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*ProductCategory, error)
	ListCategories(ctx context.Context) ([]ProductCategory, error)
	UpdateCategory(ctx context.Context, c *ProductCategory) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error

	CreateUnit(ctx context.Context, u *UnitOfMeasure) error
	GetUnitByID(ctx context.Context, id uuid.UUID) (*UnitOfMeasure, error)
	ListUnits(ctx context.Context) ([]UnitOfMeasure, error)
	UpdateUnit(ctx context.Context, u *UnitOfMeasure) error
	DeleteUnit(ctx context.Context, id uuid.UUID) error
}
