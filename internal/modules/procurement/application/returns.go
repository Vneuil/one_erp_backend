package application

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// PurchaseReturnSourcePrefix keys the debit-note entry of a purchase return.
const PurchaseReturnSourcePrefix = "purchase-return:"

func round2(v float64) float64 { return math.Round(v*100) / 100 }

type debitAllocation struct {
	Invoice *domain.PurchaseInvoice
	Amount  float64 // debited incl. PPN
	VAT     float64
}

// allocateDebit spreads the pre-PPN value of returned goods over the PO's
// invoices, oldest first, up to what each still carries; PPN is added at each
// invoice's own ratio. Value no invoice can absorb (goods returned before
// being billed) is not debited.
func allocateDebit(invoices []domain.PurchaseInvoice, value float64) []debitAllocation {
	var out []debitAllocation
	remaining := round2(value)
	for i := range invoices {
		inv := &invoices[i]
		if remaining <= 0 {
			break
		}
		room := round2(inv.TotalAmount - inv.CreditedAmount)
		if room <= 0 {
			continue
		}
		ratio := 0.0
		if inv.TaxBase > 0 {
			ratio = inv.VATAmount / inv.TaxBase
		}
		gross := round2(remaining * (1 + ratio))
		take := math.Min(gross, room)
		vat := 0.0
		if ratio > 0 {
			vat = round2(take * ratio / (1 + ratio))
		}
		out = append(out, debitAllocation{Invoice: inv, Amount: take, VAT: vat})
		remaining = round2(remaining - take/(1+ratio))
	}
	return out
}

// PurchaseReturnLedgerEntry is the debit note: Dr Accounts Payable / Cr
// Inventory, with the creditable PPN Masukan credited back separately.
func PurchaseReturnLedgerEntry(r *domain.PurchaseReturn, creditableVAT float64) financeApp.LedgerEntry {
	lines := []financeApp.LedgerLine{
		{AccountCode: financeApp.AccountPayable, Debit: r.CreditedAmount, Description: r.SupplierName},
		{AccountCode: financeApp.AccountInventory, Credit: round2(r.CreditedAmount - creditableVAT), Description: r.ReturnNo},
	}
	if creditableVAT > 0 {
		lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountInputVAT, Credit: creditableVAT, Description: "PPN Masukan retur " + r.ReturnNo})
	}
	return financeApp.LedgerEntry{
		SourceDoc: PurchaseReturnSourcePrefix + r.ID.String(),
		Memo:      "Purchase return " + r.ReturnNo + " - " + r.SupplierName,
		Lines:     lines,
	}
}

// CreatePurchaseReturn sends goods from a goods receipt back to the supplier:
// quantities are checked against what was received and not yet returned, the
// stock leaves the receiving warehouse, and the PO's invoices are debited
// (payables reduced, inventory and creditable PPN Masukan reversed).
func (uc *procurementUseCase) CreatePurchaseReturn(ctx context.Context, dto CreatePurchaseReturnDTO) (*PurchaseReturnResponseDTO, error) {
	if dto.GoodsReceiptID == uuid.Nil || len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("goodsReceiptId and at least one line are required")
	}
	gr, err := uc.repo.GetGoodsReceiptByID(ctx, dto.GoodsReceiptID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get goods receipt")
	}
	if gr == nil {
		return nil, apperrors.NewNotFound("Goods receipt not found")
	}
	po, err := uc.repo.GetPurchaseOrderByID(ctx, gr.PurchaseOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order for this goods receipt not found")
	}
	returnDate := dto.ReturnDate
	if returnDate == "" {
		returnDate = time.Now().Format("2006-01-02")
	}

	available := map[uuid.UUID]float64{}
	for _, l := range gr.Lines {
		available[l.ProductID] += l.QuantityReceived
	}
	prior, err := uc.repo.ListPurchaseReturnsByReceipt(ctx, gr.ID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load earlier returns")
	}
	for _, r := range prior {
		for _, l := range r.Lines {
			available[l.ProductID] -= l.Quantity
		}
	}
	prices := map[uuid.UUID]float64{}
	for _, l := range po.Lines {
		prices[l.ProductID] = l.UnitPrice
	}

	var total float64
	asked := map[uuid.UUID]float64{}
	lines := make([]domain.PurchaseReturnLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.ProductID == uuid.Nil || l.Quantity <= 0 {
			return nil, apperrors.NewBadRequest("Each return line needs a productId and a positive quantity")
		}
		if l.UnitPrice < 0 {
			return nil, apperrors.NewBadRequest("Unit price cannot be negative")
		}
		asked[l.ProductID] += l.Quantity
		if asked[l.ProductID] > available[l.ProductID] {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Product %s: returning %.0f but only %.0f of the receipt can still be returned", l.ProductID, asked[l.ProductID], math.Max(available[l.ProductID], 0)))
		}
		price := l.UnitPrice
		if price == 0 {
			price = prices[l.ProductID]
		}
		subtotal := round2(l.Quantity * price)
		total += subtotal
		lines[i] = domain.PurchaseReturnLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: price, Subtotal: subtotal}
	}
	total = round2(total)

	invoices, err := uc.repo.ListPurchaseInvoicesByOrder(ctx, po.ID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the order's invoices")
	}
	allocs := allocateDebit(invoices, total)
	var debited, vat, creditableVAT float64
	for _, a := range allocs {
		debited += a.Amount
		vat += a.VAT
		if a.Invoice.VATCreditable {
			creditableVAT += a.VAT
		}
	}

	// Goods leave first: an insufficient-stock failure then leaves nothing behind.
	moved := 0
	stockOut := uc.inventoryUC != nil && gr.WarehouseID != nil
	if stockOut {
		for _, l := range lines {
			if _, err := uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
				ProductID: l.ProductID, WarehouseID: *gr.WarehouseID, Quantity: -int(l.Quantity),
				Reason: "Purchase Return", Reference: gr.ReceiptNo,
			}); err != nil {
				uc.restock(ctx, gr, lines[:moved])
				return nil, apperrors.NewBadRequest("Cannot return product " + l.ProductID.String() + ": " + err.Error())
			}
			moved++
		}
	}

	ret := &domain.PurchaseReturn{
		ReturnNo:            fmt.Sprintf("PRET-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000),
		GoodsReceiptID:      dto.GoodsReceiptID,
		SupplierID:          po.SupplierID,
		SupplierName:        po.SupplierName,
		ReturnDate:          returnDate,
		Reason:              dto.Reason,
		Status:              "completed",
		TotalAmount:         total,
		CreditedAmount:      round2(debited),
		VATAmount:           round2(vat),
		CreditableVATAmount: round2(creditableVAT),
		Lines:               lines,
	}
	if err := uc.repo.CreatePurchaseReturn(ctx, ret); err != nil {
		if stockOut {
			uc.restock(ctx, gr, lines)
		}
		return nil, apperrors.NewInternal(err, "Failed to create purchase return")
	}
	if ret.CreditedAmount > 0 {
		e := PurchaseReturnLedgerEntry(ret, round2(creditableVAT))
		uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
		for _, a := range allocs {
			inv := a.Invoice
			inv.CreditedAmount = round2(inv.CreditedAmount + a.Amount)
			inv.Status = purchaseInvoiceStatus(inv.TotalAmount-inv.CreditedAmount, inv.PaidAmount, inv.DueDate)
			if err := uc.repo.UpdatePurchaseInvoice(ctx, inv); err != nil {
				slog.Error("procurement: failed to apply debit to invoice", "invoice", inv.ID, "error", err)
				continue
			}
			uc.syncPayable(ctx, inv)
		}
	}
	return ToPurchaseReturnResponse(ret), nil
}

// restock puts returned lines back, undoing a partly completed return.
func (uc *procurementUseCase) restock(ctx context.Context, gr *domain.GoodsReceipt, lines []domain.PurchaseReturnLine) {
	for _, l := range lines {
		if _, err := uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
			ProductID: l.ProductID, WarehouseID: *gr.WarehouseID, Quantity: int(l.Quantity),
			Reason: "Purchase Return rollback", Reference: gr.ReceiptNo,
		}); err != nil {
			slog.Error("procurement: failed to undo stock removal", "product", l.ProductID, "error", err)
		}
	}
}
