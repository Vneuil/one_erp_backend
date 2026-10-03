package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type ProductUseCase interface {
	Create(ctx context.Context, dto CreateProductDTO) (*ProductResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ProductResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]ProductResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateProductDTO) (*ProductResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SeedInitialData(ctx context.Context) error

	CreateCategory(ctx context.Context, dto CreateCategoryDTO) (*CategoryResponseDTO, error)
	ListCategories(ctx context.Context) ([]CategoryResponseDTO, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, dto UpdateCategoryDTO) (*CategoryResponseDTO, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error

	CreateUnit(ctx context.Context, dto CreateUnitDTO) (*UnitResponseDTO, error)
	ListUnits(ctx context.Context) ([]UnitResponseDTO, error)
	UpdateUnit(ctx context.Context, id uuid.UUID, dto UpdateUnitDTO) (*UnitResponseDTO, error)
	DeleteUnit(ctx context.Context, id uuid.UUID) error
}

type productUseCase struct {
	repo domain.ProductRepository
}

func NewProductUseCase(repo domain.ProductRepository) ProductUseCase {
	return &productUseCase{repo: repo}
}

func (uc *productUseCase) Create(ctx context.Context, dto CreateProductDTO) (*ProductResponseDTO, error) {
	sku := strings.ToUpper(strings.TrimSpace(dto.SKU))
	name := strings.TrimSpace(dto.Name)

	if sku == "" || name == "" {
		return nil, apperrors.NewBadRequest("SKU and Name are required")
	}

	existing, err := uc.repo.GetBySKU(ctx, sku)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing SKU")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("Product with this SKU already exists")
	}

	status := dto.Status
	if status == "" {
		if dto.Stock > 10 {
			status = "in_stock"
		} else if dto.Stock > 0 {
			status = "low_stock"
		} else {
			status = "out_of_stock"
		}
	}

	variantOf, variantLabel, err := uc.checkVariant(ctx, uuid.Nil, dto.VariantOf, dto.VariantLabel)
	if err != nil {
		return nil, err
	}

	product := &domain.Product{
		VariantOf:    variantOf,
		VariantLabel: variantLabel,
		SKU:          sku,
		Barcode:      strings.TrimSpace(dto.Barcode),
		Name:         name,
		Category:     dto.Category,
		Unit:         dto.Unit,
		Stock:        dto.Stock,
		CostPrice:    dto.CostPrice,
		SellingPrice: dto.SellingPrice,
		Status:       status,
	}

	if err := uc.repo.Create(ctx, product); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create product")
	}

	return ToProductResponse(product), nil
}

func (uc *productUseCase) GetByID(ctx context.Context, id uuid.UUID) (*ProductResponseDTO, error) {
	product, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve product")
	}
	if product == nil {
		return nil, apperrors.NewNotFound("Product not found")
	}
	return ToProductResponse(product), nil
}

func (uc *productUseCase) List(ctx context.Context, query types.PaginationQuery) ([]ProductResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	products, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list products")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToProductResponseList(products), meta, nil
}

func (uc *productUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateProductDTO) (*ProductResponseDTO, error) {
	product, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get product")
	}
	if product == nil {
		return nil, apperrors.NewNotFound("Product not found")
	}

	if dto.SKU != nil {
		sku := strings.ToUpper(strings.TrimSpace(*dto.SKU))
		if sku == "" || len(sku) > 50 {
			return nil, apperrors.NewBadRequest("SKU is required and must be at most 50 characters")
		}
		existing, err := uc.repo.GetBySKU(ctx, sku)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check SKU")
		}
		if existing != nil && existing.ID != id {
			return nil, apperrors.NewConflict("Product with this SKU already exists")
		}
		product.SKU = sku
	}
	if dto.Barcode != nil {
		barcode := strings.TrimSpace(*dto.Barcode)
		if len(barcode) > 100 {
			return nil, apperrors.NewBadRequest("Barcode must be at most 100 characters")
		}
		product.Barcode = barcode
	}
	if dto.Name != nil {
		product.Name = *dto.Name
	}
	if dto.Category != nil {
		product.Category = *dto.Category
	}
	if dto.Unit != nil {
		product.Unit = *dto.Unit
	}
	if dto.Stock != nil {
		product.Stock = *dto.Stock
	}
	if dto.CostPrice != nil {
		product.CostPrice = *dto.CostPrice
	}
	if dto.SellingPrice != nil {
		product.SellingPrice = *dto.SellingPrice
	}
	if dto.Status != nil {
		product.Status = *dto.Status
	}
	if dto.VariantOf != nil || dto.VariantLabel != nil {
		of, label := product.VariantOf, product.VariantLabel
		if dto.VariantOf != nil {
			of = dto.VariantOf
			if *of == uuid.Nil {
				of = nil
			}
		}
		if dto.VariantLabel != nil {
			label = *dto.VariantLabel
		}
		product.VariantOf, product.VariantLabel, err = uc.checkVariant(ctx, id, of, label)
		if err != nil {
			return nil, err
		}
	}

	if err := uc.repo.Update(ctx, product); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update product")
	}

	return ToProductResponse(product), nil
}

// Delete removes a product, but only if it is not still referenced by any
// stock level, stock movement, stock transfer, stock opname line, or BOM
// record, mirroring the "block delete if referenced" convention used for
// warehouse deletion in the inventory module.
func (uc *productUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get product")
	}
	if p == nil {
		return apperrors.NewNotFound("Product not found")
	}
	if n, err := uc.repo.CountVariants(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to check product variants")
	} else if n > 0 {
		return apperrors.NewConflict("Cannot delete a product that still has variants; delete or detach them first")
	}
	referenced, err := uc.repo.HasReferences(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check product references")
	}
	if referenced {
		return apperrors.NewConflict("Cannot delete product with existing stock level, movement, transfer, opname, or BOM records")
	}
	if err := uc.repo.Delete(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete product")
	}
	return nil
}

func (uc *productUseCase) CreateCategory(ctx context.Context, dto CreateCategoryDTO) (*CategoryResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Category name is required")
	}
	c := &domain.ProductCategory{Name: name}
	if err := uc.repo.CreateCategory(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create category")
	}
	return &CategoryResponseDTO{ID: c.ID, Name: c.Name}, nil
}

func (uc *productUseCase) ListCategories(ctx context.Context) ([]CategoryResponseDTO, error) {
	items, err := uc.repo.ListCategories(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list categories")
	}
	return ToCategoryResponseList(items), nil
}

func (uc *productUseCase) UpdateCategory(ctx context.Context, id uuid.UUID, dto UpdateCategoryDTO) (*CategoryResponseDTO, error) {
	c, err := uc.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get category")
	}
	if c == nil {
		return nil, apperrors.NewNotFound("Category not found")
	}
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Category name is required")
	}
	c.Name = name
	if err := uc.repo.UpdateCategory(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update category")
	}
	return &CategoryResponseDTO{ID: c.ID, Name: c.Name}, nil
}

func (uc *productUseCase) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteCategory(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete category")
	}
	return nil
}

func (uc *productUseCase) CreateUnit(ctx context.Context, dto CreateUnitDTO) (*UnitResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Unit name is required")
	}
	u := &domain.UnitOfMeasure{Name: name, Symbol: dto.Symbol}
	if err := uc.repo.CreateUnit(ctx, u); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create unit")
	}
	return &UnitResponseDTO{ID: u.ID, Name: u.Name, Symbol: u.Symbol}, nil
}

func (uc *productUseCase) ListUnits(ctx context.Context) ([]UnitResponseDTO, error) {
	items, err := uc.repo.ListUnits(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list units")
	}
	return ToUnitResponseList(items), nil
}

func (uc *productUseCase) UpdateUnit(ctx context.Context, id uuid.UUID, dto UpdateUnitDTO) (*UnitResponseDTO, error) {
	u, err := uc.repo.GetUnitByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get unit")
	}
	if u == nil {
		return nil, apperrors.NewNotFound("Unit not found")
	}
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Unit name is required")
	}
	u.Name = name
	u.Symbol = dto.Symbol
	if err := uc.repo.UpdateUnit(ctx, u); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update unit")
	}
	return &UnitResponseDTO{ID: u.ID, Name: u.Name, Symbol: u.Symbol}, nil
}

func (uc *productUseCase) DeleteUnit(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteUnit(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete unit")
	}
	return nil
}

func (uc *productUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Already seeded
	}

	initial := []CreateProductDTO{
		{
			SKU:          "PRD-RAW-001",
			Name:         "Aluminum Ingot 99.7% A7",
			Category:     "Raw Material",
			Unit:         "KG",
			Stock:        4500,
			CostPrice:    34000,
			SellingPrice: 42000,
			Status:       "in_stock",
		},
		{
			SKU:          "PRD-RAW-004",
			Name:         "Stainless Steel Coil 304 2mm",
			Category:     "Raw Material",
			Unit:         "Roll",
			Stock:        4,
			CostPrice:    18500000,
			SellingPrice: 22000000,
			Status:       "out_of_stock",
		},
		{
			SKU:          "PRD-ELC-018",
			Name:         "Servo Drive Controller 750W",
			Category:     "Electronics",
			Unit:         "Unit",
			Stock:        12,
			CostPrice:    2400000,
			SellingPrice: 3250000,
			Status:       "low_stock",
		},
		{
			SKU:          "PRD-FGD-102",
			Name:         "Automatic Hydraulic Press Machine 50T",
			Category:     "Finished Goods",
			Unit:         "Unit",
			Stock:        8,
			CostPrice:    85000000,
			SellingPrice: 125000000,
			Status:       "in_stock",
		},
		{
			SKU:          "PRD-RAW-012",
			Name:         "Kertas Kraft Roll 125 GSM",
			Category:     "Packaging",
			Unit:         "Roll",
			Stock:        45,
			CostPrice:    650000,
			SellingPrice: 850000,
			Status:       "low_stock",
		},
	}

	for _, item := range initial {
		_, _ = uc.Create(ctx, item)
	}
	return nil
}

// checkVariant validates a variant link: the base product must exist and be an
// ordinary product itself (variants are one level deep), it cannot be the
// product itself, a product that already has variants cannot become one, and a
// variant needs a label. It returns the cleaned link.
func (uc *productUseCase) checkVariant(ctx context.Context, self uuid.UUID, of *uuid.UUID, label string) (*uuid.UUID, string, error) {
	label = strings.TrimSpace(label)
	if of == nil {
		return nil, "", nil
	}
	if *of == self && self != uuid.Nil {
		return nil, "", apperrors.NewBadRequest("A product cannot be a variant of itself")
	}
	if label == "" || len(label) > 100 {
		return nil, "", apperrors.NewBadRequest("A variant needs a label (e.g. \"Merah / L\"), at most 100 characters")
	}
	base, err := uc.repo.GetByID(ctx, *of)
	if err != nil {
		return nil, "", apperrors.NewInternal(err, "Failed to look up the base product")
	}
	if base == nil {
		return nil, "", apperrors.NewNotFound("Base product not found")
	}
	if base.VariantOf != nil {
		return nil, "", apperrors.NewBadRequest("The base product is itself a variant; pick the original product")
	}
	if self != uuid.Nil {
		n, err := uc.repo.CountVariants(ctx, self)
		if err != nil {
			return nil, "", apperrors.NewInternal(err, "Failed to check existing variants")
		}
		if n > 0 {
			return nil, "", apperrors.NewBadRequest("This product already has variants, so it cannot become a variant")
		}
	}
	return of, label, nil
}
