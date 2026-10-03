package application

import (
	"context"
	"sort"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// MovementSummaryRow is one product in one warehouse over a period: what came
// in, what went out, and the balance at either end. Only products that moved in
// the period appear.
type MovementSummaryRow struct {
	ProductID     uuid.UUID `json:"productId"`
	ProductSKU    string    `json:"productSku"`
	ProductName   string    `json:"productName"`
	WarehouseID   uuid.UUID `json:"warehouseId"`
	WarehouseName string    `json:"warehouseName"`
	Opening       int       `json:"opening"`
	Received      int       `json:"received"`    // purchases, transfers in, positive adjustments
	Issued        int       `json:"issued"`      // sales, transfers out, negative adjustments (positive number)
	Adjustments   int       `json:"adjustments"` // net of adjustment movements, included in the two above
	Closing       int       `json:"closing"`
}

type MovementSummary struct {
	From string               `json:"from"`
	To   string               `json:"to"`
	Rows []MovementSummaryRow `json:"rows"`
	// TotalReceived and TotalIssued are sums of the rows.
	TotalReceived int `json:"totalReceived"`
	TotalIssued   int `json:"totalIssued"`
}

type summaryKey struct{ product, warehouse uuid.UUID }

// SummarizeMovements folds movements (any order) into per product/warehouse
// rows. Opening is derived from the first movement's running balance, closing
// is the last one's, so opening + received - issued = closing whenever the
// ledger is consistent.
func SummarizeMovements(moves []domain.StockMovement) []MovementSummaryRow {
	sorted := append([]domain.StockMovement(nil), moves...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].CreatedAt.Before(sorted[j].CreatedAt) })
	rows := map[summaryKey]*MovementSummaryRow{}
	var order []summaryKey
	for _, m := range sorted {
		k := summaryKey{m.ProductID, m.WarehouseID}
		r, ok := rows[k]
		if !ok {
			r = &MovementSummaryRow{ProductID: m.ProductID, WarehouseID: m.WarehouseID, Opening: m.Balance - m.Quantity}
			rows[k] = r
			order = append(order, k)
		}
		if m.Quantity >= 0 {
			r.Received += m.Quantity
		} else {
			r.Issued += -m.Quantity
		}
		if m.Type == domain.MovementAdjustment {
			r.Adjustments += m.Quantity
		}
		r.Closing = m.Balance
	}
	out := make([]MovementSummaryRow, 0, len(order))
	for _, k := range order {
		out = append(out, *rows[k])
	}
	return out
}

func (uc *inventoryUseCase) MovementSummary(ctx context.Context, from, to string, warehouseID *uuid.UUID) (*MovementSummary, error) {
	start, err := time.ParseInLocation("2006-01-02", from, batchZone)
	if err != nil {
		return nil, apperrors.NewBadRequest("from must be a date (YYYY-MM-DD)")
	}
	end, err := time.ParseInLocation("2006-01-02", to, batchZone)
	if err != nil {
		return nil, apperrors.NewBadRequest("to must be a date (YYYY-MM-DD)")
	}
	if end.Before(start) {
		return nil, apperrors.NewBadRequest("to must not be before from")
	}
	if end.Sub(start) > 366*24*time.Hour {
		return nil, apperrors.NewBadRequest("Period is limited to one year")
	}
	moves, err := uc.repo.MovementsBetween(ctx, start, end.AddDate(0, 0, 1), warehouseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load stock movements")
	}
	rows := SummarizeMovements(moves)
	res := &MovementSummary{From: from, To: to, Rows: rows}
	for i := range rows {
		rows[i].ProductSKU, rows[i].ProductName = uc.productInfo(ctx, rows[i].ProductID)
		rows[i].WarehouseName = uc.warehouseName(ctx, rows[i].WarehouseID)
		res.TotalReceived += rows[i].Received
		res.TotalIssued += rows[i].Issued
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ProductName < rows[j].ProductName })
	return res, nil
}
