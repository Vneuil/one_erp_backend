package application

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// Source-document prefixes of the entries a sales return books.
const (
	SalesReturnSourcePrefix     = "sales-return:"
	SalesReturnCOGSSourcePrefix = "sales-return-cogs:"
)

// creditAllocation is the share of a return credited to one invoice.
type creditAllocation struct {
	Invoice *domain.Invoice
	Amount  float64 // credited incl. PPN
	VAT     float64
}

// allocateCredit spreads the pre-PPN value of returned goods over the order's
// invoices, oldest first, up to what each still has outstanding of its total
// (an invoice already paid is credited anyway: the customer then holds a
// credit). PPN is added at each invoice's own ratio. Value that no invoice
// can absorb (goods returned before being billed) is not credited.
func allocateCredit(invoices []domain.Invoice, value float64) []creditAllocation {
	var out []creditAllocation
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
		out = append(out, creditAllocation{Invoice: inv, Amount: take, VAT: vat})
		remaining = round2(remaining - take/(1+ratio))
	}
	return out
}

// SalesReturnLedgerEntry is the credit note: Dr Sales Returns (revenue) and
// Dr PPN Keluaran / Cr Accounts Receivable.
func SalesReturnLedgerEntry(r *domain.SalesReturn) financeApp.LedgerEntry {
	revenue := round2(r.CreditedAmount - r.VATAmount)
	return financeApp.LedgerEntry{
		SourceDoc: SalesReturnSourcePrefix + r.ID.String(),
		Memo:      "Sales return " + r.ReturnNo + " - " + r.CustomerName,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountSalesReturns, Debit: revenue, Description: r.ReturnNo},
			{AccountCode: financeApp.AccountSalesTaxPayable, Debit: r.VATAmount, Description: "PPN retur " + r.ReturnNo},
			{AccountCode: financeApp.AccountReceivble, Credit: r.CreditedAmount, Description: r.CustomerName},
		},
	}
}

// CreateSalesReturn takes goods back from a delivery: it checks the quantities
// against what was shipped and not yet returned, puts the goods back in stock,
// credits the order's invoices (revenue and PPN reversed against receivables)
// and reverses the cost of goods sold.
func (uc *salesUseCase) CreateSalesReturn(ctx context.Context, dto CreateSalesReturnDTO) (*SalesReturnResponseDTO, error) {
	if dto.DeliveryID == uuid.Nil || len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("deliveryId and at least one line are required")
	}
	delivery, err := uc.repo.GetDeliveryByID(ctx, dto.DeliveryID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get delivery")
	}
	if delivery == nil {
		return nil, apperrors.NewNotFound("Delivery not found")
	}
	returnDate := dto.ReturnDate
	if returnDate == "" {
		returnDate = time.Now().Format("2006-01-02")
	}

	shipped := map[uuid.UUID]float64{}
	for _, l := range delivery.Lines {
		shipped[l.ProductID] += l.Quantity
	}
	prior, err := uc.repo.ListSalesReturnsByDelivery(ctx, dto.DeliveryID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load earlier returns")
	}
	for _, r := range prior {
		for _, l := range r.Lines {
			shipped[l.ProductID] -= l.Quantity
		}
	}

	var order *domain.SalesOrder
	prices := map[uuid.UUID]float64{}
	if delivery.SalesOrderID != nil {
		if order, err = uc.repo.GetOrderByID(ctx, *delivery.SalesOrderID); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get sales order")
		}
		if order != nil {
			for _, l := range order.Lines {
				prices[l.ProductID] = l.UnitPrice
			}
		}
	}

	var total float64
	asked := map[uuid.UUID]float64{}
	lines := make([]domain.SalesReturnLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.ProductID == uuid.Nil || l.Quantity <= 0 {
			return nil, apperrors.NewBadRequest("Each return line needs a productId and a positive quantity")
		}
		if l.UnitPrice < 0 {
			return nil, apperrors.NewBadRequest("Unit price cannot be negative")
		}
		asked[l.ProductID] += l.Quantity
		if asked[l.ProductID] > shipped[l.ProductID] {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Product %s: returning %.0f but only %.0f of the delivery can still be returned", l.ProductID, asked[l.ProductID], math.Max(shipped[l.ProductID], 0)))
		}
		price := l.UnitPrice
		if price == 0 {
			price = prices[l.ProductID]
		}
		subtotal := round2(l.Quantity * price)
		total += subtotal
		lines[i] = domain.SalesReturnLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: price, Subtotal: subtotal}
	}
	total = round2(total)

	var allocs []creditAllocation
	if order != nil {
		invoices, err := uc.repo.ListInvoicesByOrder(ctx, order.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to load the order's invoices")
		}
		allocs = allocateCredit(invoices, total)
	}
	var credited, vat float64
	for _, a := range allocs {
		credited += a.Amount
		vat += a.VAT
	}

	// Cost of the goods put back (only when the sale itself booked cost).
	restock := order != nil && order.WarehouseID != nil && uc.inventoryUC != nil
	var cost float64
	if restock && uc.products != nil {
		for _, l := range lines {
			if p, err := uc.products.GetByID(ctx, l.ProductID); err == nil && p != nil {
				cost += p.CostPrice * l.Quantity
			}
		}
	}

	ret := &domain.SalesReturn{
		ReturnNo:       fmt.Sprintf("SRET-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000),
		DeliveryID:     dto.DeliveryID,
		CustomerName:   delivery.CustomerName,
		ReturnDate:     returnDate,
		Reason:         dto.Reason,
		Status:         "completed",
		TotalAmount:    total,
		CreditedAmount: round2(credited), VATAmount: round2(vat), CostAmount: round2(cost),
		Lines: lines,
	}
	if err := uc.repo.CreateSalesReturn(ctx, ret); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create sales return")
	}

	if restock {
		for _, l := range lines {
			if _, err := uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
				ProductID: l.ProductID, WarehouseID: *order.WarehouseID, Quantity: int(l.Quantity),
				Reason: "Sales Return", Reference: ret.ReturnNo,
			}); err != nil {
				return nil, apperrors.NewInternal(err, "Return saved but failed to restock product "+l.ProductID.String())
			}
		}
	}
	if ret.CostAmount > 0 {
		uc.postLedger(ctx, SalesReturnCOGSSourcePrefix+ret.ID.String(), "Cost reversal "+ret.ReturnNo, []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountInventory, Debit: ret.CostAmount, Description: ret.ReturnNo},
			{AccountCode: financeApp.AccountCOGS, Credit: ret.CostAmount, Description: ret.ReturnNo},
		})
	}
	if ret.CreditedAmount > 0 {
		e := SalesReturnLedgerEntry(ret)
		uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
		for _, a := range allocs {
			inv := a.Invoice
			inv.CreditedAmount = round2(inv.CreditedAmount + a.Amount)
			inv.Status = salesInvoiceStatus(inv.TotalAmount-inv.CreditedAmount, inv.PaidAmount, inv.DueDate)
			if err := uc.repo.UpdateInvoice(ctx, inv); err != nil {
				slog.Error("sales: failed to apply credit to invoice", "invoice", inv.ID, "error", err)
				continue
			}
			uc.syncReceivable(ctx, inv)
		}
	}
	return ToSalesReturnResponse(ret), nil
}
