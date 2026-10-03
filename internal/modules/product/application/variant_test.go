package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/google/uuid"
)

type variantRepo struct {
	domain.ProductRepository
	products map[uuid.UUID]*domain.Product
}

func (r *variantRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	return r.products[id], nil
}
func (r *variantRepo) GetBySKU(context.Context, string) (*domain.Product, error) { return nil, nil }
func (r *variantRepo) Create(_ context.Context, p *domain.Product) error {
	p.ID = uuid.New()
	r.products[p.ID] = p
	return nil
}
func (r *variantRepo) Update(_ context.Context, p *domain.Product) error {
	r.products[p.ID] = p
	return nil
}
func (r *variantRepo) CountVariants(_ context.Context, base uuid.UUID) (int64, error) {
	var n int64
	for _, p := range r.products {
		if p.VariantOf != nil && *p.VariantOf == base {
			n++
		}
	}
	return n, nil
}

func TestVariantRules(t *testing.T) {
	repo := &variantRepo{products: map[uuid.UUID]*domain.Product{}}
	uc := NewProductUseCase(repo)
	ctx := context.Background()

	base, err := uc.Create(ctx, CreateProductDTO{SKU: "KAOS", Name: "Kaos Polos", SellingPrice: 80_000})
	if err != nil {
		t.Fatal(err)
	}
	v, err := uc.Create(ctx, CreateProductDTO{SKU: "KAOS-M", Name: "Kaos Polos", VariantOf: &base.ID, VariantLabel: " Merah / L "})
	if err != nil || v.VariantOf == nil || *v.VariantOf != base.ID || v.VariantLabel != "Merah / L" {
		t.Fatalf("variant: %+v %v", v, err)
	}
	// A variant needs a label, a real base, and the base cannot itself be a variant.
	unknown := uuid.New()
	for name, dto := range map[string]CreateProductDTO{
		"no label":           {SKU: "X1", Name: "x", VariantOf: &base.ID},
		"unknown base":       {SKU: "X2", Name: "x", VariantOf: &unknown, VariantLabel: "L"},
		"variant of variant": {SKU: "X3", Name: "x", VariantOf: &v.ID, VariantLabel: "L"},
	} {
		if _, err := uc.Create(ctx, dto); err == nil {
			t.Errorf("%s should be refused", name)
		}
	}
	// A product with variants cannot become a variant; nothing may be a variant of itself.
	other, _ := uc.Create(ctx, CreateProductDTO{SKU: "CELANA", Name: "Celana"})
	label := "S"
	if _, err := uc.Update(ctx, base.ID, UpdateProductDTO{VariantOf: &other.ID, VariantLabel: &label}); err == nil {
		t.Fatal("a product that has variants must not become a variant")
	}
	if _, err := uc.Update(ctx, other.ID, UpdateProductDTO{VariantOf: &other.ID, VariantLabel: &label}); err == nil {
		t.Fatal("a product cannot be a variant of itself")
	}
	// Detach with the nil uuid.
	nilID := uuid.Nil
	got, err := uc.Update(ctx, v.ID, UpdateProductDTO{VariantOf: &nilID})
	if err != nil || got.VariantOf != nil || got.VariantLabel != "" {
		t.Fatalf("detach: %+v %v", got, err)
	}
	// A base with variants cannot be deleted.
	if _, err := uc.Update(ctx, v.ID, UpdateProductDTO{VariantOf: &base.ID, VariantLabel: &label}); err != nil {
		t.Fatal(err)
	}
	if err := uc.Delete(ctx, base.ID); err == nil {
		t.Fatal("deleting a base product with variants must be refused")
	}
}
