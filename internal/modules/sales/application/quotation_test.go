package application

import (
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
	"math"
	"testing"
)

func TestQuotationLines(t *testing.T) {
	lines, total, err := quotationLines([]QuotationLineDTO{
		{Description: " Item A ", Quantity: 2, UnitPrice: 12500.25},
		{Description: "Item B", Unit: "box", Quantity: 1.5, UnitPrice: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 25150.5 || lines[0].Subtotal != 25000.5 || lines[0].Description != "Item A" || lines[0].Unit != "pcs" || lines[1].Position != 1 {
		t.Fatalf("unexpected lines: %+v, total %v", lines, total)
	}
}

func TestQuotationLinesRejectInvalid(t *testing.T) {
	cases := [][]QuotationLineDTO{
		nil,
		{{Description: " ", Quantity: 1}},
		{{Description: "A", Quantity: 0}},
		{{Description: "A", Quantity: 0.001}},
		{{Description: "A", Quantity: 1, UnitPrice: -1}},
		{{Description: "A", Quantity: math.NaN()}},
		{{Description: "A", Quantity: 1, UnitPrice: math.Inf(1)}},
		{{Description: "A", Quantity: 1e12, UnitPrice: 100}},
	}
	for i, input := range cases {
		if _, _, err := quotationLines(input); err == nil {
			t.Errorf("case %d should fail", i)
		}
	}
}

func TestQuotationReferencesSurviveMapping(t *testing.T) {
	productID, leadID := uuid.New(), uuid.New()
	lines, _, err := quotationLines([]QuotationLineDTO{{ProductID: &productID, Description: "Barang", Unit: "pcs", Quantity: 2, UnitPrice: 100}})
	if err != nil {
		t.Fatal(err)
	}
	response := ToQuotationResponse(&domain.Quotation{LeadID: &leadID, Lines: lines})
	if response.LeadID == nil || *response.LeadID != leadID || response.Lines[0].ProductID == nil || *response.Lines[0].ProductID != productID {
		t.Fatalf("missing references: %+v", response)
	}
}
