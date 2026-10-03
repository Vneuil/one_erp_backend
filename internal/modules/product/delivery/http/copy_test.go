package http

import (
	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/google/uuid"
	"testing"
)

func TestCopiedProductHasNewIdentityAndNoStock(t *testing.T) {
	sourceTenant, targetTenant := uuid.New(), uuid.New()
	source := domain.Product{TenantID: &sourceTenant, SKU: " abc ", Barcode: "00123", Name: "Item", Category: "Material", Unit: "KG", Stock: 99, CostPrice: 10, SellingPrice: 20}
	source.ID = uuid.New()
	copied := copiedProduct(source, targetTenant)
	if copied.ID != uuid.Nil || copied.TenantID == nil || *copied.TenantID != targetTenant || copied.Stock != 0 || copied.Status != "out_of_stock" {
		t.Fatalf("bad destination: %+v", copied)
	}
	if copied.SKU != "ABC" || copied.Barcode != source.Barcode || copied.Name != source.Name || copied.Category != source.Category || copied.Unit != source.Unit || copied.CostPrice != 10 || copied.SellingPrice != 20 {
		t.Fatalf("missing catalog fields: %+v", copied)
	}
	if source.Stock != 99 || *source.TenantID != sourceTenant {
		t.Fatal("source changed")
	}
}
