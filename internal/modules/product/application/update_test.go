package application

import (
	"context"
	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/google/uuid"
	"testing"
)

type editRepository struct {
	domain.ProductRepository
	product   *domain.Product
	duplicate *domain.Product
	saved     bool
}

func (r *editRepository) GetByID(context.Context, uuid.UUID) (*domain.Product, error) {
	return r.product, nil
}
func (r *editRepository) GetBySKU(context.Context, string) (*domain.Product, error) {
	return r.duplicate, nil
}
func (r *editRepository) Update(context.Context, *domain.Product) error { r.saved = true; return nil }

func TestUpdateProductIdentifiers(t *testing.T) {
	p := &domain.Product{SKU: "OLD", Barcode: "old", Stock: 12}
	p.ID = uuid.New()
	repo := &editRepository{product: p}
	sku, barcode, category, unit := " new-sku ", " 0123456789 ", "Material", "KG"
	result, err := NewProductUseCase(repo).Update(context.Background(), p.ID, UpdateProductDTO{SKU: &sku, Barcode: &barcode, Category: &category, Unit: &unit})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.saved || result.SKU != "NEW-SKU" || result.Barcode != "0123456789" || result.Category != category || result.Unit != unit || result.Stock != 12 {
		t.Fatalf("unexpected result: %+v", result)
	}
	empty := ""
	result, err = NewProductUseCase(repo).Update(context.Background(), p.ID, UpdateProductDTO{Barcode: &empty})
	if err != nil || result.Barcode != "" {
		t.Fatal("barcode was not cleared", err)
	}
}

func TestUpdateRejectsDuplicateSKU(t *testing.T) {
	p, other := &domain.Product{}, &domain.Product{}
	p.ID, other.ID = uuid.New(), uuid.New()
	repo := &editRepository{product: p, duplicate: other}
	sku := "DUPLICATE"
	if _, err := NewProductUseCase(repo).Update(context.Background(), p.ID, UpdateProductDTO{SKU: &sku}); err == nil || repo.saved {
		t.Fatal("duplicate SKU must not be saved")
	}
}
