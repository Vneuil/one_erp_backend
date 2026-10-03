package application

import (
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/google/uuid"
)

func mv(p, w uuid.UUID, typ string, qty, bal int, at time.Time) domain.StockMovement {
	m := domain.StockMovement{ProductID: p, WarehouseID: w, Type: typ, Quantity: qty, Balance: bal}
	m.CreatedAt = at
	return m
}

func TestSummarizeMovements(t *testing.T) {
	p, q, w := uuid.New(), uuid.New(), uuid.New()
	t0 := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	// Deliberately unsorted: the balance chain, not the input order, decides opening/closing.
	moves := []domain.StockMovement{
		mv(p, w, domain.MovementOut, -3, 17, t0.Add(2*time.Hour)),
		mv(p, w, domain.MovementIn, 10, 20, t0),
		mv(p, w, domain.MovementAdjustment, -2, 15, t0.Add(3*time.Hour)),
		mv(q, w, domain.MovementIn, 5, 5, t0),
	}
	rows := SummarizeMovements(moves)
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	var r MovementSummaryRow
	for _, x := range rows {
		if x.ProductID == p {
			r = x
		}
	}
	if r.Opening != 10 || r.Received != 10 || r.Issued != 5 || r.Adjustments != -2 || r.Closing != 15 {
		t.Fatalf("got %+v", r)
	}
	if r.Opening+r.Received-r.Issued != r.Closing {
		t.Fatalf("does not reconcile: %+v", r)
	}
}
