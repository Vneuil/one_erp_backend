package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	tenantctx.SetTenantID(ctx, &product.TenantID)
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	var product domain.Product
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	var product domain.Product
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("sku = ?", sku).First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.Product, int64, error) {
	var products []domain.Product
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Product{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ? OR sku ILIKE ? OR category ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortField := "created_at"
	sortDir := "desc"
	if query.SortBy != "" {
		sortField = query.SortBy
	}
	if query.SortDir != "" {
		sortDir = query.SortDir
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order(fmt.Sprintf("%s %s", sortField, sortDir)).
		Offset(offset).
		Limit(query.PerPage).
		Find(&products).Error

	return products, total, err
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Product{}, id).Error
}

// HasReferences checks, via raw table queries (to avoid importing other
// modules' domain packages from here), whether any stock level, stock
// movement, stock transfer, stock opname line, or BOM record still points at
// this product. It mirrors the "block delete if referenced" check used for
// warehouse deletion in the inventory module.
func (r *productRepository) HasReferences(ctx context.Context, id uuid.UUID) (bool, error) {
	type refCheck struct {
		table  string
		column string
	}
	checks := []refCheck{
		{"stock_levels", "product_id"},
		{"stock_movements", "product_id"},
		{"stock_transfers", "product_id"},
		{"stock_opname_lines", "product_id"},
		{"manufacturing_boms", "product_id"},
		{"manufacturing_bom_lines", "component_product_id"},
	}
	db := r.db.WithContext(ctx)
	for _, chk := range checks {
		var count int64
		if err := db.Table(chk.table).Where(fmt.Sprintf("%s = ?", chk.column), id).Count(&count).Error; err != nil {
			// If the table doesn't exist yet (module not migrated for this
			// tenant), treat it as "no references" rather than failing the
			// delete outright.
			if isMissingTableError(err) {
				continue
			}
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

// isMissingTableError reports whether err looks like a "relation does not
// exist" error (Postgres), which can happen for tenants that haven't
// migrated a given module's schema yet.
func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "does not exist")
}

func (r *productRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Product{}).Count(&total).Error
	return total, err
}

func (r *productRepository) CreateCategory(ctx context.Context, c *domain.ProductCategory) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *productRepository) ListCategories(ctx context.Context) ([]domain.ProductCategory, error) {
	var items []domain.ProductCategory
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ProductCategory{}))
	err := db.Order("name asc").Find(&items).Error
	return items, err
}

func (r *productRepository) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.ProductCategory, error) {
	var c domain.ProductCategory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *productRepository) UpdateCategory(ctx context.Context, c *domain.ProductCategory) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *productRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.ProductCategory{}, id).Error
}

func (r *productRepository) CreateUnit(ctx context.Context, u *domain.UnitOfMeasure) error {
	tenantctx.SetTenantID(ctx, &u.TenantID)
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *productRepository) ListUnits(ctx context.Context) ([]domain.UnitOfMeasure, error) {
	var items []domain.UnitOfMeasure
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.UnitOfMeasure{}))
	err := db.Order("name asc").Find(&items).Error
	return items, err
}

func (r *productRepository) GetUnitByID(ctx context.Context, id uuid.UUID) (*domain.UnitOfMeasure, error) {
	var u domain.UnitOfMeasure
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *productRepository) UpdateUnit(ctx context.Context, u *domain.UnitOfMeasure) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *productRepository) DeleteUnit(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.UnitOfMeasure{}, id).Error
}

func (r *productRepository) CountVariants(ctx context.Context, baseID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Product{}).Where("variant_of = ?", baseID).Count(&n).Error
	return n, err
}
