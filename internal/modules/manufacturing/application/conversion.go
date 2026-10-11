package application

import (
	"context"
	"log/slog"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/google/uuid"
)

// Source-document prefix of the labour/overhead charged when units pass a step.
const conversionSourcePrefix = "production-conversion:"

// ConversionLedgerEntry charges the standard labour and overhead of units that
// finished a step to Work in Process: Dr WIP / Cr Direct Labor Applied and
// Cr Manufacturing Overhead Applied. The applied accounts are offset by the
// real wages and factory expenses booked elsewhere, so the difference between
// them is the labour and overhead variance.
func ConversionLedgerEntry(logID uuid.UUID, orderNumber, stepName string, labor, overhead float64) financeApp.LedgerEntry {
	labor, overhead = round2(labor), round2(overhead)
	desc := orderNumber + " - " + stepName
	return financeApp.LedgerEntry{
		SourceDoc: conversionSourcePrefix + logID.String(),
		Memo:      "Conversion cost " + desc,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountWIP, Debit: labor + overhead, Description: desc},
			{AccountCode: financeApp.AccountLaborApplied, Credit: labor, Description: desc},
			{AccountCode: financeApp.AccountOverheadApplied, Credit: overhead, Description: desc},
		},
	}
}

// postConversion books the labour and overhead of qty units finishing a step. A failure is logged, not fatal.
func (uc *manufacturingUseCase) postConversion(ctx context.Context, o *domain.ProductionOrder, step domain.ProductionStep, logID uuid.UUID, qty int, date string) {
	if uc.ledger == nil {
		return
	}
	labor, overhead := step.UnitLabor()*float64(qty), step.UnitOverhead()*float64(qty)
	if round2(labor)+round2(overhead) <= 0 {
		return
	}
	e := ConversionLedgerEntry(logID, OrderLabel(o), step.Name, labor, overhead)
	if err := uc.ledger.PostEntryOn(ctx, date, e.SourceDoc, e.Memo, e.Lines); err != nil {
		slog.Error("manufacturing: failed to post conversion cost", "order", o.ID, "step", step.ID, "error", err)
	}
}

// updateProductCost folds the full unit cost of a finished batch (material plus
// conversion) into the product's cost price as a moving average, so Inventory
// and later cost of goods sold carry the labour and overhead. It only runs for
// products whose routing has conversion cost, leaving other products' cost
// prices as they were set.
func (uc *manufacturingUseCase) updateProductCost(ctx context.Context, productID uuid.UUID, qty int, unitCost float64) bool {
	p, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil || p == nil || qty <= 0 {
		return false
	}
	total, err := uc.inventoryRepo.SumQuantityByProduct(ctx, productID)
	if err != nil {
		slog.Error("manufacturing: failed to read stock for costing", "product", productID, "error", err)
		return false
	}
	before := total - qty // the batch is already in stock
	if before < 0 {
		before = 0
	}
	p.CostPrice = round2((float64(before)*p.CostPrice + float64(qty)*unitCost) / float64(before+qty))
	if err := uc.productRepo.Update(ctx, p); err != nil {
		slog.Error("manufacturing: failed to update product cost", "product", productID, "error", err)
		return false
	}
	return true
}

// revertProductCost takes a voided batch back out of the product's moving
// average cost. total is the stock on hand with the batch still counted.
func (uc *manufacturingUseCase) revertProductCost(ctx context.Context, productID uuid.UUID, qty, total int, unitCost float64) {
	p, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil || p == nil || total-qty <= 0 {
		return
	}
	cost := (float64(total)*p.CostPrice - float64(qty)*unitCost) / float64(total-qty)
	if cost < 0 {
		return
	}
	p.CostPrice = round2(cost)
	if err := uc.productRepo.Update(ctx, p); err != nil {
		slog.Error("manufacturing: failed to revert product cost", "product", productID, "error", err)
	}
}
