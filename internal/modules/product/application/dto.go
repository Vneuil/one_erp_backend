package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/google/uuid"
)

type CreateProductDTO struct {
	Barcode      string  `json:"barcode"`
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	Unit         string  `json:"unit"`
	Stock        int     `json:"stock"`
	CostPrice    float64 `json:"costPrice"`
	SellingPrice float64 `json:"sellingPrice"`
	Status       string  `json:"status"`
	// VariantOf/VariantLabel make this product a variant of another.
	VariantOf    *uuid.UUID `json:"variantOf,omitempty"`
	VariantLabel string     `json:"variantLabel,omitempty"`
}

type UpdateProductDTO struct {
	SKU          *string  `json:"sku,omitempty"`
	Barcode      *string  `json:"barcode,omitempty"`
	Name         *string  `json:"name,omitempty"`
	Category     *string  `json:"category,omitempty"`
	Unit         *string  `json:"unit,omitempty"`
	Stock        *int     `json:"stock,omitempty"`
	CostPrice    *float64 `json:"costPrice,omitempty"`
	SellingPrice *float64 `json:"sellingPrice,omitempty"`
	Status       *string  `json:"status,omitempty"`
	// VariantOf: send the base product id to make this a variant; send the nil
	// uuid (all zeros) to turn a variant back into an ordinary product.
	VariantOf    *uuid.UUID `json:"variantOf,omitempty"`
	VariantLabel *string    `json:"variantLabel,omitempty"`
}

type ProductResponseDTO struct {
	Barcode              string     `json:"barcode"`
	ID                   uuid.UUID  `json:"id"`
	SKU                  string     `json:"sku"`
	Name                 string     `json:"name"`
	Category             string     `json:"category"`
	Unit                 string     `json:"unit"`
	Stock                int        `json:"stock"`
	CostPrice            float64    `json:"costPrice"`
	SellingPrice         float64    `json:"sellingPrice"`
	Status               string     `json:"status"`
	VariantOf            *uuid.UUID `json:"variantOf,omitempty"`
	VariantLabel         string     `json:"variantLabel,omitempty"`
	MarketplacePlatform  string     `json:"marketplacePlatform,omitempty"`
	MarketplaceProductID string     `json:"marketplaceProductId,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func ToProductResponse(p *domain.Product) *ProductResponseDTO {
	if p == nil {
		return nil
	}
	return &ProductResponseDTO{
		ID:                   p.ID,
		SKU:                  p.SKU,
		Barcode:              p.Barcode,
		Name:                 p.Name,
		Category:             p.Category,
		Unit:                 p.Unit,
		Stock:                p.Stock,
		CostPrice:            p.CostPrice,
		SellingPrice:         p.SellingPrice,
		Status:               p.Status,
		VariantOf:            p.VariantOf,
		VariantLabel:         p.VariantLabel,
		MarketplacePlatform:  p.MarketplacePlatform,
		MarketplaceProductID: p.MarketplaceProductID,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

func ToProductResponseList(products []domain.Product) []ProductResponseDTO {
	result := make([]ProductResponseDTO, len(products))
	for i, p := range products {
		result[i] = *ToProductResponse(&p)
	}
	return result
}

type CreateCategoryDTO struct {
	Name string `json:"name"`
}

type UpdateCategoryDTO struct {
	Name string `json:"name"`
}

type CategoryResponseDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func ToCategoryResponseList(items []domain.ProductCategory) []CategoryResponseDTO {
	result := make([]CategoryResponseDTO, len(items))
	for i, c := range items {
		result[i] = CategoryResponseDTO{ID: c.ID, Name: c.Name}
	}
	return result
}

type CreateUnitDTO struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type UpdateUnitDTO struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type UnitResponseDTO struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Symbol string    `json:"symbol"`
}

func ToUnitResponseList(items []domain.UnitOfMeasure) []UnitResponseDTO {
	result := make([]UnitResponseDTO, len(items))
	for i, u := range items {
		result[i] = UnitResponseDTO{ID: u.ID, Name: u.Name, Symbol: u.Symbol}
	}
	return result
}
