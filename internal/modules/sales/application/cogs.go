package application

import (
	"context"
	"log/slog"
	"math"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
)

// COGSSourcePrefix keys the cost-of-goods entry booked when a sales order's stock leaves.
const COGSSourcePrefix = "sales-cogs:"

// COGSLedgerEntry is Dr Cost of Goods Sold / Cr Inventory for cost (> 0).
func COGSLedgerEntry(orderID uuid.UUID, orderNumber string, cost float64) financeApp.LedgerEntry {
	cost = math.Round(cost*100) / 100
	return financeApp.LedgerEntry{
		SourceDoc: COGSSourcePrefix + orderID.String(),
		Memo:      "Cost of goods " + orderNumber,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountCOGS, Debit: cost, Description: orderNumber},
			{AccountCode: financeApp.AccountInventory, Credit: cost, Description: orderNumber},
		},
	}
}

// postCOGS books the cost of the goods that just left stock for an order. It is
// skipped when there is no ledger, no product costs, or no cost to book, and a
// failure is logged rather than failing the sale. The entry is keyed by order,
// so calling it twice never double-counts.
func (uc *salesUseCase) postCOGS(ctx context.Context, orderID uuid.UUID, orderNumber string, lines []domain.SalesOrderLine) {
	if uc.ledger == nil || uc.products == nil {
		return
	}
	var cost float64
	for _, l := range lines {
		p, err := uc.products.GetByID(ctx, l.ProductID)
		if err != nil || p == nil {
			continue
		}
		cost += p.CostPrice * l.Quantity
	}
	if cost <= 0 {
		return
	}
	e := COGSLedgerEntry(orderID, orderNumber, cost)
	if err := uc.ledger.PostEntry(ctx, e.SourceDoc, e.Memo, e.Lines); err != nil {
		slog.Error("sales: failed to post cost of goods", "order", orderNumber, "error", err)
	}
}
