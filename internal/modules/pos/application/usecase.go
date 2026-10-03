package application

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	loyaltyApp "github.com/divinecoid/one-backend/internal/modules/loyalty/application"
	"github.com/divinecoid/one-backend/internal/modules/pos/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type POSUseCase interface {
	Checkout(ctx context.Context, dto CheckoutDTO) (*POSTransactionResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*POSTransactionResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]POSTransactionResponseDTO, types.PaginationMeta, error)
	Void(ctx context.Context, id uuid.UUID, reason string) (*POSTransactionResponseDTO, error)
	Refund(ctx context.Context, id uuid.UUID, dto RefundDTO) (*POSTransactionResponseDTO, error)
	ListRefunds(ctx context.Context, id uuid.UUID) ([]domain.POSRefund, error)
	GetSettings(ctx context.Context) (*domain.POSSettings, error)
	UpdateSettings(ctx context.Context, dto UpdateSettingsDTO) (*domain.POSSettings, error)
	SalesReport(ctx context.Context, from, to, outlet string) (*SalesReport, error)
}

type posUseCase struct {
	repo        domain.POSRepository
	salesRepo   salesDomain.SalesRepository
	loyaltyUC   loyaltyApp.LoyaltyUseCase
	inventoryUC inventoryApp.InventoryUseCase
	products    productDomain.ProductRepository // optional: names, SKUs and costs for receipts and COGS
	ledger      financeApp.LedgerPoster         // optional: nil disables posting to the general ledger
}

type Option func(*posUseCase)

// WithLedger posts sales, cost of goods and refunds to the general ledger.
func WithLedger(l financeApp.LedgerPoster) Option { return func(uc *posUseCase) { uc.ledger = l } }

// WithProducts looks up product names, SKUs and costs.
func WithProducts(p productDomain.ProductRepository) Option {
	return func(uc *posUseCase) { uc.products = p }
}

func NewPOSUseCase(repo domain.POSRepository, salesRepo salesDomain.SalesRepository, loyaltyUC loyaltyApp.LoyaltyUseCase, inventoryUC inventoryApp.InventoryUseCase, opts ...Option) POSUseCase {
	uc := &posUseCase{repo: repo, salesRepo: salesRepo, loyaltyUC: loyaltyUC, inventoryUC: inventoryUC}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

var paymentMethods = map[string]bool{"cash": true, "card": true, "qris": true, "transfer": true, "ewallet": true, "debit": true, "credit": true}

func (uc *posUseCase) postLedger(ctx context.Context, e financeApp.LedgerEntry) {
	if uc.ledger == nil {
		return
	}
	if err := uc.ledger.PostEntry(ctx, e.SourceDoc, e.Memo, e.Lines); err != nil {
		slog.Error("pos: failed to post to the ledger", "source", e.SourceDoc, "error", err)
	}
}

// compensate puts back stock already taken for a sale that could not be finished.
func (uc *posUseCase) compensate(ctx context.Context, warehouse uuid.UUID, taken []salesDomain.SalesOrderLine, ref string) {
	for _, l := range taken {
		if _, err := uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
			ProductID: l.ProductID, WarehouseID: warehouse, Quantity: int(l.Quantity), Reason: "POS Sale rolled back", Reference: ref,
		}); err != nil {
			slog.Error("pos: could not restore stock after a failed sale", "product", l.ProductID, "reference", ref, "error", err)
		}
	}
}

func (uc *posUseCase) Checkout(ctx context.Context, dto CheckoutDTO) (*POSTransactionResponseDTO, error) {
	settings, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load POS settings")
	}

	caller := actor.EmailFrom(ctx)
	outlet := strings.TrimSpace(dto.Outlet)
	if outlet == "" {
		outlet = settings.DefaultOutlet
	}
	if outlet == "" {
		outlet = "Outlet Utama"
	}
	// The cashier is whoever is signed in; a client-supplied name is only a label.
	cashier := strings.TrimSpace(dto.Cashier)
	if cashier == "" {
		cashier = caller
	}
	if cashier == "" {
		cashier = "-"
	}
	customer := strings.TrimSpace(dto.Customer)
	if customer == "" {
		customer = "Pelanggan Umum"
	}
	paymentMethod := strings.ToLower(strings.TrimSpace(dto.PaymentMethod))
	if paymentMethod == "" {
		paymentMethod = "cash"
	}
	if !paymentMethods[paymentMethod] {
		return nil, apperrors.NewBadRequest("Unsupported payment method")
	}

	hasCart := len(dto.Lines) > 0
	var (
		totals    SaleTotals
		txLines   []domain.POSTransactionLine
		soLines   []salesDomain.SalesOrderLine
		totalQty  int
		warehouse uuid.UUID
	)
	if hasCart {
		if dto.WarehouseID == nil || uc.inventoryUC == nil {
			return nil, apperrors.NewBadRequest("A warehouse is required for a sale with products")
		}
		warehouse = *dto.WarehouseID
		var subtotal float64
		// Validate every line and check stock BEFORE creating anything.
		need := map[uuid.UUID]float64{}
		for _, l := range dto.Lines {
			if l.ProductID == uuid.Nil || l.Quantity <= 0 || math.IsNaN(l.Quantity) || l.UnitPrice < 0 {
				return nil, apperrors.NewBadRequest("Each cart line requires a productId, a positive quantity and a valid price")
			}
			need[l.ProductID] += l.Quantity
		}
		for pid, qty := range need {
			available, err := uc.inventoryUC.GetAvailability(ctx, pid, warehouse)
			if err != nil {
				return nil, err
			}
			if float64(available) < qty {
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Insufficient stock for product %s: available %d, requested %.0f", pid, available, qty))
			}
		}
		for _, l := range dto.Lines {
			line := domain.POSTransactionLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice}
			if uc.products != nil {
				if p, err := uc.products.GetByID(ctx, l.ProductID); err == nil && p != nil {
					line.SKU, line.Name, line.UnitCost = p.SKU, p.Name, p.CostPrice
					if line.UnitPrice == 0 {
						line.UnitPrice = p.SellingPrice
					}
				}
			}
			line.Subtotal = round2(line.UnitPrice * l.Quantity)
			subtotal += line.Subtotal
			totalQty += int(math.Ceil(l.Quantity))
			txLines = append(txLines, line)
			soLines = append(soLines, salesDomain.SalesOrderLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: line.UnitPrice, Subtotal: line.Subtotal})
		}
		totals, err = ComputeSale(subtotal, dto.DiscountAmount, *settings)
		if err != nil {
			return nil, err
		}
		// A client that also sent a total must agree with the server's arithmetic.
		if dto.TotalAmount > 0 && math.Abs(dto.TotalAmount-totals.Total) > 1 {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("The total does not match the cart: expected %.2f", totals.Total))
		}
	} else {
		// A quick sale with no product lines: the amount is taken as given, no tax breakdown.
		if dto.TotalAmount <= 0 {
			return nil, apperrors.NewBadRequest("Total amount must be greater than 0")
		}
		totals = SaleTotals{Subtotal: round2(dto.TotalAmount), Total: round2(dto.TotalAmount)}
		totalQty = dto.TotalItems
	}

	var tendered, change float64
	if paymentMethod == "cash" && hasCart {
		if dto.AmountTendered < totals.Total {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("The cash received (%.2f) is less than the total (%.2f)", dto.AmountTendered, totals.Total))
		}
		tendered, change = round2(dto.AmountTendered), round2(dto.AmountTendered-totals.Total)
	} else if paymentMethod != "cash" {
		tendered = totals.Total
	}

	orderNo := fmt.Sprintf("POS-%s-%04d", time.Now().Format("060102"), time.Now().Nanosecond()%10000)
	order := &salesDomain.SalesOrder{
		OrderNumber: orderNo, CustomerName: customer, TotalAmount: totals.Total, BaseAmount: totals.Total,
		Status: "Completed", PaymentStatus: "Paid", Channel: "POS", CreatedByEmail: caller,
		OrderDate: time.Now().Format("2006-01-02"), WarehouseID: dto.WarehouseID, Lines: soLines,
	}
	if err := uc.salesRepo.CreateOrder(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create sales order for POS transaction")
	}

	// Take stock; if any line fails, give back what was already taken.
	var taken []salesDomain.SalesOrderLine
	for _, l := range soLines {
		if _, err := uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
			ProductID: l.ProductID, WarehouseID: warehouse, Quantity: -int(l.Quantity), Reason: "POS Sale", Reference: orderNo,
		}); err != nil {
			uc.compensate(ctx, warehouse, taken, orderNo)
			order.Status, order.PaymentStatus = "Cancelled", "Unpaid"
			_ = uc.salesRepo.UpdateOrder(ctx, order)
			return nil, apperrors.NewBadRequest("The sale could not be completed and stock was restored: " + err.Error())
		}
		taken = append(taken, l)
	}

	tx := &domain.POSTransaction{
		OrderNo: orderNo, Outlet: outlet, Cashier: cashier, CashierEmail: caller, Customer: customer, CustomerPhone: dto.CustomerPhone,
		TotalItems: totalQty, TotalAmount: totals.Total, Subtotal: totals.Subtotal, DiscountAmount: totals.Discount, TaxAmount: totals.Tax,
		AmountTendered: tendered, ChangeAmount: change, PaymentMethod: paymentMethod, Status: "Completed",
		SalesOrderID: &order.ID, SalesOrderNumber: order.OrderNumber,
	}
	if hasCart {
		tx.WarehouseID = &warehouse
	}
	if dto.CustomerPhone != "" && uc.loyaltyUC != nil {
		// Loyalty points are a bonus on top of the sale, never a precondition for it.
		if result, err := uc.loyaltyUC.EarnPoints(ctx, loyaltyApp.EarnPointsDTO{
			PhoneNumber: dto.CustomerPhone, Name: customer, SpendAmount: totals.Total, SalesOrderID: &order.ID, SalesOrderNumber: order.OrderNumber,
		}); err == nil {
			tx.LoyaltyMemberCode, tx.LoyaltyPointsEarned, tx.LoyaltyPointsBalance = result.Member.MemberCode, result.PointsEarned, result.Member.PointsBalance
		}
	}
	if err := uc.repo.Create(ctx, tx); err != nil {
		uc.compensate(ctx, warehouse, taken, orderNo)
		order.Status, order.PaymentStatus = "Cancelled", "Unpaid"
		_ = uc.salesRepo.UpdateOrder(ctx, order)
		return nil, apperrors.NewInternal(err, "Failed to record POS transaction; stock was restored")
	}
	for i := range txLines {
		txLines[i].TransactionID = tx.ID
	}
	if err := uc.repo.CreateLines(ctx, txLines); err != nil {
		slog.Error("pos: sale saved but its lines could not be stored", "order", orderNo, "error", err)
	}
	tx.Lines = txLines

	uc.postLedger(ctx, SaleLedgerEntry(tx, *settings))
	if e, ok := SaleCOGSEntry(tx, txLines); ok {
		uc.postLedger(ctx, e)
	}
	return ToPOSTransactionResponse(tx), nil
}

func (uc *posUseCase) GetByID(ctx context.Context, id uuid.UUID) (*POSTransactionResponseDTO, error) {
	tx, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get transaction")
	}
	if tx == nil {
		return nil, apperrors.NewNotFound("Transaction not found")
	}
	return ToPOSTransactionResponse(tx), nil
}

func (uc *posUseCase) List(ctx context.Context, query types.PaginationQuery) ([]POSTransactionResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	transactions, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list transactions")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToPOSTransactionResponseList(transactions), meta, nil
}

func (uc *posUseCase) ListRefunds(ctx context.Context, id uuid.UUID) ([]domain.POSRefund, error) {
	out, err := uc.repo.ListRefunds(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list refunds")
	}
	return out, nil
}

// Void cancels a whole sale that has had no refunds: everything is reversed and
// stock goes back. The person who rang the sale cannot void it.
func (uc *posUseCase) Void(ctx context.Context, id uuid.UUID, reason string) (*POSTransactionResponseDTO, error) {
	tx, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get transaction")
	}
	if tx == nil {
		return nil, apperrors.NewNotFound("Transaction not found")
	}
	if tx.Status != "Completed" || tx.RefundedAmount > 0 {
		return nil, apperrors.NewConflict("Only a completed sale with no refunds can be voided; use a refund instead")
	}
	all := map[uuid.UUID]float64{}
	for _, l := range tx.Lines {
		all[l.ID] = l.Quantity
	}
	return uc.reverse(ctx, tx, all, "void", reason, true)
}

// Refund hands back some units of a sale. Legacy sales made without product
// lines can only be voided as a whole.
func (uc *posUseCase) Refund(ctx context.Context, id uuid.UUID, dto RefundDTO) (*POSTransactionResponseDTO, error) {
	tx, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get transaction")
	}
	if tx == nil {
		return nil, apperrors.NewNotFound("Transaction not found")
	}
	if tx.Status != "Completed" && tx.Status != "Partially Refunded" {
		return nil, apperrors.NewConflict("This sale can no longer be refunded (" + tx.Status + ")")
	}
	if len(tx.Lines) == 0 {
		return nil, apperrors.NewBadRequest("This sale has no product lines; void it instead")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("Choose at least one line to refund")
	}
	qtys := map[uuid.UUID]float64{}
	for _, l := range dto.Lines {
		if l.Quantity <= 0 {
			return nil, apperrors.NewBadRequest("Refund quantities must be positive")
		}
		qtys[l.LineID] += l.Quantity
	}
	restock := dto.Restock == nil || *dto.Restock
	return uc.reverse(ctx, tx, qtys, "refund", dto.Reason, restock)
}

// reverse is the shared engine behind void and refund: it validates quantities,
// prices the return, writes the refund record, updates the sale, returns stock
// and posts the reversing journal entry.
func (uc *posUseCase) reverse(ctx context.Context, tx *domain.POSTransaction, qtys map[uuid.UUID]float64, kind, reason string, restock bool) (*POSTransactionResponseDTO, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required")
	}
	if err := sod.ForbidSelfApproval(ctx, tx.CashierEmail); err != nil {
		return nil, err
	}
	settings, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load POS settings")
	}
	prior, err := uc.repo.ListRefunds(ctx, tx.ID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load earlier refunds")
	}

	byID := map[uuid.UUID]*domain.POSTransactionLine{}
	for i := range tx.Lines {
		byID[tx.Lines[i].ID] = &tx.Lines[i]
	}
	var share RefundShare
	type change struct {
		line *domain.POSTransactionLine
		qty  float64
	}
	var changes []change
	for lineID, qty := range qtys {
		l := byID[lineID]
		if l == nil {
			return nil, apperrors.NewNotFound("A refund line does not belong to this sale")
		}
		if qty > l.Quantity-l.RefundedQty+1e-9 {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Only %.2f of %s can still be refunded", l.Quantity-l.RefundedQty, l.Name))
		}
		s := RefundFor(tx, *l, qty)
		share.Amount += s.Amount
		share.Tax += s.Tax
		share.Cost += s.Cost
		changes = append(changes, change{l, qty})
	}
	if len(tx.Lines) == 0 { // legacy sale: void the whole amount
		share = RefundShare{Amount: tx.TotalAmount, Tax: tx.TaxAmount}
	}

	// Does this settle every unit? Then the last refund takes exactly what is left,
	// so rounding on earlier partial refunds never leaves a rupiah stranded.
	fullyRefunded := true
	for i := range tx.Lines {
		remaining := tx.Lines[i].Quantity - tx.Lines[i].RefundedQty
		if qtys[tx.Lines[i].ID] < remaining-1e-9 {
			fullyRefunded = false
		}
	}
	if len(tx.Lines) == 0 {
		fullyRefunded = true
	}
	if fullyRefunded {
		var priorTax float64
		for _, p := range prior {
			priorTax += p.TaxAmount
		}
		share.Amount = round2(tx.TotalAmount - tx.RefundedAmount)
		share.Tax = round2(tx.TaxAmount - priorTax)
	}
	share.Amount, share.Tax, share.Cost = round2(share.Amount), round2(share.Tax), round2(share.Cost)
	if share.Amount <= 0 {
		return nil, apperrors.NewBadRequest("Nothing to refund")
	}

	rf := &domain.POSRefund{TransactionID: tx.ID, Kind: kind, Amount: share.Amount, TaxAmount: share.Tax, CostReturned: share.Cost,
		Reason: reason, Restocked: restock && len(tx.Lines) > 0, ProcessedBy: actor.EmailFrom(ctx)}
	if err := uc.repo.CreateRefund(ctx, rf); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record the refund")
	}

	// Stock first: if it cannot go back we say so rather than pretend.
	if rf.Restocked && tx.WarehouseID != nil && uc.inventoryUC != nil {
		for _, c := range changes {
			if _, err := uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
				ProductID: c.line.ProductID, WarehouseID: *tx.WarehouseID, Quantity: int(math.Round(c.qty)),
				Reason: "POS " + kind, Reference: tx.OrderNo,
			}); err != nil {
				slog.Error("pos: refund recorded but stock could not be restored", "order", tx.OrderNo, "product", c.line.ProductID, "error", err)
			}
		}
	}
	for _, c := range changes {
		c.line.RefundedQty += c.qty
		if err := uc.repo.UpdateLine(ctx, c.line); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update the sale lines")
		}
	}
	tx.RefundedAmount = round2(tx.RefundedAmount + share.Amount)
	switch {
	case kind == "void":
		tx.Status, tx.VoidReason, tx.VoidedBy = "Voided", reason, actor.EmailFrom(ctx)
	case fullyRefunded:
		tx.Status = "Refunded"
	default:
		tx.Status = "Partially Refunded"
	}
	if err := uc.repo.Update(ctx, tx); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update the sale")
	}
	if tx.SalesOrderID != nil && (kind == "void" || fullyRefunded) {
		if order, err := uc.salesRepo.GetOrderByID(ctx, *tx.SalesOrderID); err == nil && order != nil {
			order.Status, order.PaymentStatus = "Cancelled", "Refunded"
			_ = uc.salesRepo.UpdateOrder(ctx, order)
		}
	}
	uc.postLedger(ctx, RefundLedgerEntry(tx, rf, *settings))
	return ToPOSTransactionResponse(tx), nil
}

func (uc *posUseCase) GetSettings(ctx context.Context) (*domain.POSSettings, error) {
	s, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load POS settings")
	}
	return s, nil
}

func (uc *posUseCase) UpdateSettings(ctx context.Context, dto UpdateSettingsDTO) (*domain.POSSettings, error) {
	if dto.TaxPercent < 0 || dto.TaxPercent > 100 || dto.RoundTo < 0 {
		return nil, apperrors.NewBadRequest("taxPercent must be 0-100 and roundTo cannot be negative")
	}
	s, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load POS settings")
	}
	s.TaxPercent, s.TaxInclusive, s.RoundTo = round2(dto.TaxPercent), dto.TaxInclusive, dto.RoundTo
	s.DefaultOutlet, s.ReceiptHeader, s.ReceiptFooter = strings.TrimSpace(dto.DefaultOutlet), strings.TrimSpace(dto.ReceiptHeader), strings.TrimSpace(dto.ReceiptFooter)
	s.NonCashAccountCode = strings.TrimSpace(dto.NonCashAccountCode)
	if err := uc.repo.SaveSettings(ctx, s); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save POS settings")
	}
	return s, nil
}

func (uc *posUseCase) SalesReport(ctx context.Context, from, to, outlet string) (*SalesReport, error) {
	today := time.Now().In(zone).Format("2006-01-02")
	if to == "" {
		to = today
	}
	if from == "" {
		from = to
	}
	for _, d := range []string{from, to} {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			return nil, apperrors.NewBadRequest("from and to must be YYYY-MM-DD")
		}
	}
	if from > to {
		return nil, apperrors.NewBadRequest("from must not be after to")
	}
	txs, err := uc.repo.ListInRange(ctx, from, to, outlet)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load transactions")
	}
	refunds, err := uc.repo.ListRefundsInRange(ctx, from, to, outlet)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load refunds")
	}
	rep := BuildSalesReport(from, to, txs, refunds)
	return &rep, nil
}
