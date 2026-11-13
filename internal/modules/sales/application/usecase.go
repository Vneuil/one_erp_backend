package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"log/slog"
	"math"
	"strings"
	"time"

	approvalApp "github.com/divinecoid/one-backend/internal/modules/approval/application"
	currencyApp "github.com/divinecoid/one-backend/internal/modules/currency/application"
	customerDomain "github.com/divinecoid/one-backend/internal/modules/customer/domain"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// salesOrderDocumentType is the document type string this module submits to
// modules/approval when a Sales Order is created. Any workflow an admin
// configures for this type (see /approval/workflows) automatically gates SO
// approval - no code change needed to adjust thresholds/levels. A tenant
// with no workflow configured for this type sees no behavior change at all:
// SubmitDocument returns RequiresApproval=false and CreateOrder proceeds
// exactly as it did before this feature existed.
const salesOrderDocumentType = "sales_order"

type SalesUseCase interface {
	CreateOrder(ctx context.Context, dto CreateSalesOrderDTO) (*SalesOrderResponseDTO, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*SalesOrderResponseDTO, error)
	ListOrders(ctx context.Context, query types.PaginationQuery) ([]SalesOrderResponseDTO, types.PaginationMeta, error)
	SubmitOrder(ctx context.Context, id uuid.UUID) (*SalesOrderResponseDTO, error)
	ApproveOrder(ctx context.Context, id uuid.UUID, dto SalesApprovalActionDTO) (*SalesOrderResponseDTO, error)
	RejectOrder(ctx context.Context, id uuid.UUID, dto SalesApprovalActionDTO) (*SalesOrderResponseDTO, error)
	UpdateOrder(ctx context.Context, id uuid.UUID, dto UpdateSalesOrderDTO) (*SalesOrderResponseDTO, error)
	AttachInvoiceDocument(ctx context.Context, id uuid.UUID, dto AttachInvoiceDocDTO) (*SalesOrderResponseDTO, error)
	// ListOrderRevisions returns every earlier version of the order (oldest first).
	ListOrderRevisions(ctx context.Context, id uuid.UUID) ([]domain.SalesOrderRevision, error)

	// FinalizeOrderApproved/Rejected are the single place that applies the
	// effect of a Sales Order's approval workflow reaching a final state:
	// setting the order's status and, on approval, performing the deferred
	// stock deduction. Both ApproveOrder/RejectOrder (the module-specific
	// endpoints) and the approval module's DocumentStatusCallback (fired
	// when an SO is approved/rejected from the generic Approval Center)
	// call through here - never duplicated between the two paths, so stock
	// is never deducted twice.
	FinalizeOrderApproved(ctx context.Context, id uuid.UUID) error
	FinalizeOrderRejected(ctx context.Context, id uuid.UUID) error

	CreateQuotation(ctx context.Context, dto CreateQuotationDTO) (*QuotationResponseDTO, error)
	GetQuotationByID(ctx context.Context, id uuid.UUID) (*QuotationResponseDTO, error)
	UpdateQuotation(ctx context.Context, id uuid.UUID, dto UpdateQuotationDTO) (*QuotationResponseDTO, error)
	ListQuotations(ctx context.Context, query types.PaginationQuery) ([]QuotationResponseDTO, types.PaginationMeta, error)
	ConvertQuotationToOrder(ctx context.Context, id uuid.UUID) (*SalesOrderResponseDTO, error)

	CreateInvoice(ctx context.Context, dto CreateInvoiceDTO) (*InvoiceResponseDTO, error)
	SetBillingSchedule(ctx context.Context, orderID uuid.UUID, terms []BillingTermInput) ([]domain.BillingTerm, error)
	ListBillingSchedule(ctx context.Context, orderID uuid.UUID) ([]domain.BillingTerm, error)
	InvoiceBillingTerm(ctx context.Context, termID uuid.UUID) (*InvoiceResponseDTO, error)
	ListInvoices(ctx context.Context, query types.PaginationQuery) ([]InvoiceResponseDTO, types.PaginationMeta, error)
	RecordInvoicePayment(ctx context.Context, id uuid.UUID, dto RecordSalesPaymentDTO) (*InvoiceResponseDTO, error)

	CreateDelivery(ctx context.Context, dto CreateDeliveryDTO) (*DeliveryResponseDTO, error)
	RecordShippingCost(ctx context.Context, id uuid.UUID, dto ShippingCostDTO) (*DeliveryResponseDTO, error)
	ListDeliveries(ctx context.Context, query types.PaginationQuery) ([]DeliveryResponseDTO, types.PaginationMeta, error)

	// Sales down payments
	CreateDownPayment(ctx context.Context, dto CreateSalesDownPaymentDTO) (*SalesDownPaymentResponseDTO, error)
	ListDownPayments(ctx context.Context, query types.PaginationQuery) ([]SalesDownPaymentResponseDTO, types.PaginationMeta, error)
	ApplyDownPayment(ctx context.Context, id uuid.UUID, dto ApplySalesDownPaymentDTO) (*SalesDownPaymentResponseDTO, error)

	// Sales returns
	CreateSalesReturn(ctx context.Context, dto CreateSalesReturnDTO) (*SalesReturnResponseDTO, error)
	GetSalesReturnByID(ctx context.Context, id uuid.UUID) (*SalesReturnResponseDTO, error)
	ListSalesReturns(ctx context.Context, query types.PaginationQuery) ([]SalesReturnResponseDTO, types.PaginationMeta, error)

	SeedInitialData(ctx context.Context) error
}

type salesUseCase struct {
	repo        domain.SalesRepository
	currencyUC  currencyApp.CurrencyUseCase
	inventoryUC inventoryApp.InventoryUseCase
	products    productDomain.ProductRepository // optional: unit costs for cost of goods sold
	customers   customerDomain.CustomerRepository // optional: block inactive customers on new orders
	approvalUC  approvalApp.ApprovalUseCase
	ledger      financeApp.LedgerPoster // optional; nil disables auto-posting to the general ledger
}

// Option customises optional collaborators of the sales use case.
type Option func(*salesUseCase)

// WithProducts supplies product unit costs so stock leaving for a sale books cost of goods sold.
func WithProducts(p productDomain.ProductRepository) Option {
	return func(uc *salesUseCase) { uc.products = p }
}

// WithCustomers supplies customer master data to reject inactive customers on new orders.
func WithCustomers(c customerDomain.CustomerRepository) Option {
	return func(uc *salesUseCase) { uc.customers = c }
}

// WithLedger enables automatic journal posting for invoices and payments.
func WithLedger(l financeApp.LedgerPoster) Option {
	return func(uc *salesUseCase) { uc.ledger = l }
}

func NewSalesUseCase(repo domain.SalesRepository, currencyUC currencyApp.CurrencyUseCase, inventoryUC inventoryApp.InventoryUseCase, approvalUC approvalApp.ApprovalUseCase, opts ...Option) SalesUseCase {
	uc := &salesUseCase{repo: repo, currencyUC: currencyUC, inventoryUC: inventoryUC, approvalUC: approvalUC}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

func (uc *salesUseCase) CreateOrder(ctx context.Context, dto CreateSalesOrderDTO) (*SalesOrderResponseDTO, error) {
	if dto.CustomerName == "" {
		return nil, apperrors.NewBadRequest("Customer name is required")
	}
	if uc.customers != nil {
		cust, err := uc.customers.GetByName(ctx, dto.CustomerName)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to verify customer status")
		}
		if cust != nil && strings.EqualFold(cust.Status, "Inactive") {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Pelanggan %s berstatus Nonaktif dan tidak dapat dipilih untuk transaksi baru", cust.Name))
		}
	}

	orderNum := dto.OrderNumber
	if orderNum == "" {
		orderNum = fmt.Sprintf("SO-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)
	}

	status := dto.Status
	if status == "" {
		status = "Confirmed"
	}
	paymentStatus := dto.PaymentStatus
	if paymentStatus == "" {
		paymentStatus = "Unpaid"
	}
	channel := dto.Channel
	if channel == "" {
		channel = "Direct B2B"
	}

	orderDate := time.Now().Format("2006-01-02")

	packaging, err := normalizeState("packagingAvailability", dto.PackagingAvailability, "pending", packagingStates)
	if err != nil {
		return nil, err
	}
	coa, err := normalizeState("coaStatus", dto.COAStatus, "pending", coaStates)
	if err != nil {
		return nil, err
	}
	production, err := normalizeState("productionStatus", dto.ProductionStatus, "not_started", productionStates)
	if err != nil {
		return nil, err
	}

	currency := dto.Currency
	baseAmount := dto.TotalAmount
	if uc.currencyUC != nil {
		result, err := uc.currencyUC.Convert(ctx, dto.TotalAmount, dto.Currency, orderDate)
		if err != nil {
			return nil, err
		}
		currency = result.OriginalCurrency
		baseAmount = result.BaseAmount
	}

	// Sales Orders capture customer demand, which triggers PR/PO procurement
	// when stock is insufficient. We do not block order creation on low/zero stock.
	var lines []domain.SalesOrderLine
	for _, l := range dto.Lines {
		if l.ProductID == uuid.Nil || l.Quantity <= 0 {
			return nil, apperrors.NewBadRequest("Each line requires a productId and a positive quantity")
		}
		lines = append(lines, domain.SalesOrderLine{
			ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice, Subtotal: l.Quantity * l.UnitPrice,
		})
	}

	order := &domain.SalesOrder{
		OrderNumber:          orderNum,
		CustomerName:         dto.CustomerName,
		TotalAmount:          dto.TotalAmount,
		Currency:             currency,
		BaseAmount:           baseAmount,
		Status:               status,
		CreatedByEmail:       actor.EmailFrom(ctx),
		PaymentStatus:        paymentStatus,
		Channel:              channel,
		OrderDate:            orderDate,
		WarehouseID:          dto.WarehouseID,
		Lines:                lines,
		MarketplacePlatform:  dto.MarketplacePlatform,
		MarketplaceOrderID:   dto.MarketplaceOrderID,
		CustomerPONumber:     dto.CustomerPONumber,
		CustomerPOAttachment: dto.CustomerPOAttachment,
		DPPaid:               dto.DPPaid,
		DPPercent:            dto.DPPercent,
		DPAmount:             dto.DPAmount,
		InvoiceDocType:       dto.InvoiceDocType,
		InvoiceDocAttachment: dto.InvoiceDocAttachment,
		InvoiceDocNumber:     dto.InvoiceDocNumber,
		ShippingAddress:      dto.ShippingAddress,
		ShippingCourier:      dto.ShippingCourier,
		DeliveryDate:         dto.DeliveryDate,

		PackagingAvailability: packaging,
		COAStatus:             coa,
		ProductionStatus:      production,
		VATExempt:             dto.VATExempt,
	}

	if err := uc.repo.CreateOrder(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create sales order")
	}

	// If an admin has configured an approval workflow matching this document
	// type/amount, route it through the workflow instead of deducting stock
	// immediately - see modules/approval. If no workflow rule matched, but
	// DP was not paid, the order still enters pending_approval awaiting approval.
	requiresApproval := false
	if uc.approvalUC != nil {
		result, err := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
			DocumentType:   salesOrderDocumentType,
			DocumentID:     order.ID,
			DocumentNumber: order.OrderNumber,
			Amount:         order.TotalAmount,
			RequesterName:  order.CustomerName,
		})
		if err == nil && result != nil && result.RequiresApproval {
			requiresApproval = true
			order.Status = "pending_approval"
			if err := uc.repo.UpdateOrder(ctx, order); err != nil {
				return nil, apperrors.NewInternal(err, "Failed to update sales order approval status")
			}
		}
	}
	if !requiresApproval {
		if !dto.DPPaid {
			requiresApproval = true
			order.Status = "pending_approval"
			if err := uc.repo.UpdateOrder(ctx, order); err != nil {
				return nil, apperrors.NewInternal(err, "Failed to update sales order approval status")
			}
		} else {
			order.Status = "Confirmed"
			if err := uc.repo.UpdateOrder(ctx, order); err != nil {
				return nil, apperrors.NewInternal(err, "Failed to update sales order status")
			}
		}
	}

	// Availability was already confirmed above; deduct now that the order
	// document itself exists - unless the order is still awaiting approval,
	// in which case committing stock against a not-yet-approved order would
	// be wrong; deduction happens instead when the order is approved (see
	// ApproveOrder). A failure here is logged on the returned error but the
	// order is not rolled back - matching how POS/loyalty treat inventory as
	// a best-effort side effect of a sale, not a precondition for the sale
	// existing.
	if dto.WarehouseID != nil && !requiresApproval && uc.inventoryUC != nil {
		for _, l := range lines {
			available, err := uc.inventoryUC.GetAvailability(ctx, l.ProductID, *dto.WarehouseID)
			if err == nil && float64(available) >= l.Quantity {
				_, _ = uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
					ProductID: l.ProductID, WarehouseID: *dto.WarehouseID, Quantity: -int(l.Quantity),
					Reason: "Sales Order", Reference: order.OrderNumber,
				})
			}
		}
		uc.postCOGS(ctx, order.ID, order.OrderNumber, lines)
	}

	return ToSalesOrderResponse(order), nil
}

func (uc *salesUseCase) SubmitOrder(ctx context.Context, id uuid.UUID) (*SalesOrderResponseDTO, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}
	if !strings.EqualFold(order.Status, "draft") {
		return nil, apperrors.NewConflict(fmt.Sprintf("Sales order in status %s cannot be submitted", order.Status))
	}
	if err := checkSubmitDocuments(order); err != nil {
		return nil, err
	}
	requiresApproval := false
	if uc.approvalUC != nil {
		result, err := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
			DocumentType:   salesOrderDocumentType,
			DocumentID:     order.ID,
			DocumentNumber: order.OrderNumber,
			Amount:         order.TotalAmount,
			RequesterName:  order.CustomerName,
		})
		if err != nil {
			return nil, err
		}
		if result != nil && result.RequiresApproval {
			requiresApproval = true
		}
	}
	if requiresApproval || !order.DPPaid {
		order.Status = "pending_approval"
	} else {
		order.Status = "Confirmed"
		if order.WarehouseID != nil && uc.inventoryUC != nil {
			for _, l := range order.Lines {
				available, err := uc.inventoryUC.GetAvailability(ctx, l.ProductID, *order.WarehouseID)
				if err == nil && float64(available) >= l.Quantity {
					_, _ = uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
						ProductID: l.ProductID, WarehouseID: *order.WarehouseID, Quantity: -int(l.Quantity),
						Reason: "Sales Order", Reference: order.OrderNumber,
					})
				}
			}
			uc.postCOGS(ctx, order.ID, order.OrderNumber, order.Lines)
		}
	}
	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update sales order")
	}
	return ToSalesOrderResponse(order), nil
}

// ApproveOrder advances a Sales Order routed through modules/approval
// (Status == "pending_approval") to its next approval step, mirroring
// procurement's ApprovePurchaseOrder. Once the workflow's final step
// approves, the order's original status effect (stock deduction, if a
// warehouse/lines were supplied) is applied - deferred from CreateOrder
// exactly because the order was not yet approved at that time.
func (uc *salesUseCase) ApproveOrder(ctx context.Context, id uuid.UUID, dto SalesApprovalActionDTO) (*SalesOrderResponseDTO, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}
	if err := sod.ForbidSelfApproval(ctx, order.CreatedByEmail); err != nil {
		return nil, err
	}

	if uc.approvalUC != nil {
		activeReq, err := uc.approvalUC.GetActiveRequestForDocument(ctx, salesOrderDocumentType, order.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check approval status")
		}
		if activeReq == nil && (order.Status == "pending_approval" || order.Status == "submitted" || strings.EqualFold(order.Status, "draft")) {
			submitRes, submitErr := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
				DocumentType:   salesOrderDocumentType,
				DocumentID:     order.ID,
				DocumentNumber: order.OrderNumber,
				Amount:         order.TotalAmount,
				RequesterName:  order.CustomerName,
			})
			if submitErr == nil && submitRes != nil && submitRes.RequiresApproval && submitRes.Request != nil {
				activeReq = submitRes.Request
				order.Status = "pending_approval"
				_ = uc.repo.UpdateOrder(ctx, order)
			}
		}

		if activeReq != nil && activeReq.Status == "pending" {
			if _, err := uc.approvalUC.ApproveStep(ctx, activeReq.ID, approvalApp.ActOnStepDTO{
				ActorName: dto.ActorName, ActorRole: dto.ActorRole, Comments: dto.Comments,
			}); err != nil {
				return nil, err
			}

			order, err = uc.repo.GetOrderByID(ctx, id)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to get sales order")
			}
			// Keep who rejected and why on the order itself, for later analysis.
			stampRejection(order, dto)
			_ = uc.repo.UpdateOrder(ctx, order)
			return ToSalesOrderResponse(order), nil
		}
	}

	if !strings.EqualFold(order.Status, "draft") && order.Status != "submitted" && order.Status != "pending_approval" {
		return nil, apperrors.NewConflict("This sales order is not awaiting approval")
	}

	order.Status = "Confirmed"
	if order.WarehouseID != nil && uc.inventoryUC != nil {
		for _, l := range order.Lines {
			available, err := uc.inventoryUC.GetAvailability(ctx, l.ProductID, *order.WarehouseID)
			if err == nil && float64(available) >= l.Quantity {
				_, _ = uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
					ProductID: l.ProductID, WarehouseID: *order.WarehouseID, Quantity: -int(l.Quantity),
					Reason: "Sales Order", Reference: order.OrderNumber,
				})
			}
		}
		uc.postCOGS(ctx, order.ID, order.OrderNumber, order.Lines)
	}
	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update sales order")
	}
	return ToSalesOrderResponse(order), nil
}

// FinalizeOrderApproved sets the order's status to "Confirmed" and performs
// the deferred stock deduction. It is the single place this happens,
// invoked either from ApproveOrder's call to approvalUC.ApproveStep (via
// the registered callback) or from the generic Approval Center's
// ApproveStep call on the same request - never both, and the
// pending_approval guard makes it a no-op if somehow invoked twice, so
// stock can never be deducted twice for the same order.
func (uc *salesUseCase) FinalizeOrderApproved(ctx context.Context, id uuid.UUID) error {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return apperrors.NewNotFound("Sales order not found")
	}
	if order.Status != "pending_approval" {
		return nil
	}

	order.Status = "Confirmed"

	if order.WarehouseID != nil && uc.inventoryUC != nil {
		for _, l := range order.Lines {
			available, err := uc.inventoryUC.GetAvailability(ctx, l.ProductID, *order.WarehouseID)
			if err == nil && float64(available) >= l.Quantity {
				_, _ = uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
					ProductID: l.ProductID, WarehouseID: *order.WarehouseID, Quantity: -int(l.Quantity),
					Reason: "Sales Order", Reference: order.OrderNumber,
				})
			}
		}
		uc.postCOGS(ctx, order.ID, order.OrderNumber, order.Lines)
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return apperrors.NewInternal(err, "Failed to update sales order")
	}
	return nil
}

// RejectOrder mirrors procurement's RejectPurchaseOrder: rejecting any level
// immediately rejects the whole order, and stock (never deducted while the
// order was pending) stays untouched.
func (uc *salesUseCase) RejectOrder(ctx context.Context, id uuid.UUID, dto SalesApprovalActionDTO) (*SalesOrderResponseDTO, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}

	if uc.approvalUC != nil {
		activeReq, err := uc.approvalUC.GetActiveRequestForDocument(ctx, salesOrderDocumentType, order.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check approval status")
		}
		if activeReq == nil && (order.Status == "pending_approval" || order.Status == "submitted" || strings.EqualFold(order.Status, "draft")) {
			submitRes, submitErr := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
				DocumentType:   salesOrderDocumentType,
				DocumentID:     order.ID,
				DocumentNumber: order.OrderNumber,
				Amount:         order.TotalAmount,
				RequesterName:  order.CustomerName,
			})
			if submitErr == nil && submitRes != nil && submitRes.RequiresApproval && submitRes.Request != nil {
				activeReq = submitRes.Request
				order.Status = "pending_approval"
				_ = uc.repo.UpdateOrder(ctx, order)
			}
		}

		if activeReq != nil && activeReq.Status == "pending" {
			if _, err := uc.approvalUC.RejectStep(ctx, activeReq.ID, approvalApp.ActOnStepDTO{
				ActorName: dto.ActorName, ActorRole: dto.ActorRole, Comments: dto.Comments,
			}); err != nil {
				return nil, err
			}

			order, err = uc.repo.GetOrderByID(ctx, id)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to get sales order")
			}
			return ToSalesOrderResponse(order), nil
		}
	}

	if !strings.EqualFold(order.Status, "draft") && order.Status != "submitted" && order.Status != "pending_approval" {
		return nil, apperrors.NewConflict("This sales order cannot be rejected")
	}
	order.Status = "rejected"
	stampRejection(order, dto)
	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to reject sales order")
	}
	return ToSalesOrderResponse(order), nil
}

// FinalizeOrderRejected mirrors FinalizeOrderApproved for the rejected
// path. Stock was never deducted while the order was pending, so there is
// nothing to reverse.
func (uc *salesUseCase) FinalizeOrderRejected(ctx context.Context, id uuid.UUID) error {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return apperrors.NewNotFound("Sales order not found")
	}
	if order.Status != "pending_approval" {
		return nil
	}
	order.Status = "rejected"
	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return apperrors.NewInternal(err, "Failed to update sales order")
	}
	return nil
}

func (uc *salesUseCase) GetOrderByID(ctx context.Context, id uuid.UUID) (*SalesOrderResponseDTO, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}
	return ToSalesOrderResponse(order), nil
}

// UpdateOrder lets the customer name and line items of a Sales Order be
// corrected after the fact. It is intentionally restricted to orders this
// module itself created without an external, authoritative source of truth:
// a marketplace-synced order (MarketplacePlatform set) must stay in sync
// with the marketplace, and a POS sale (Channel "POS") is a completed,
// already-paid transaction - both must be corrected at their own source
// instead. Quotation conversions and manually entered orders share neither
// trait, so both remain editable, matching CreateOrder's own line-item
// validation. Stock is deliberately left untouched here: CreateOrder only
// deducts stock for orders created with a warehouse, and reconciling stock
// against an edited quantity is a separate concern this endpoint does not
// attempt.
func (uc *salesUseCase) UpdateOrder(ctx context.Context, id uuid.UUID, dto UpdateSalesOrderDTO) (*SalesOrderResponseDTO, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}
	if order.MarketplacePlatform != "" {
		return nil, apperrors.NewBadRequest("Marketplace orders must be edited from the marketplace itself")
	}
	if order.Channel == "POS" {
		return nil, apperrors.NewBadRequest("POS transactions cannot be edited")
	}
	if order.Status == "pending_approval" {
		return nil, apperrors.NewBadRequest("Order is awaiting approval and cannot be edited")
	}
	if strings.TrimSpace(dto.CustomerName) == "" {
		return nil, apperrors.NewBadRequest("Customer name is required")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line item is required")
	}

	lines := make([]domain.SalesOrderLine, 0, len(dto.Lines))
	total := 0.0
	for _, l := range dto.Lines {
		if l.ProductID == uuid.Nil || l.Quantity <= 0 || l.UnitPrice < 0 {
			return nil, apperrors.NewBadRequest("Each line requires a productId, a positive quantity, and a non-negative unit price")
		}
		subtotal := l.Quantity * l.UnitPrice
		total += subtotal
		lines = append(lines, domain.SalesOrderLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice, Subtotal: subtotal})
	}

	baseAmount := total
	if uc.currencyUC != nil {
		result, err := uc.currencyUC.Convert(ctx, total, order.Currency, order.OrderDate)
		if err != nil {
			return nil, err
		}
		baseAmount = result.BaseAmount
	}

	// Keep the previous version when the line items actually change, so a revised
	// order can still be analysed later. A reason is mandatory once past draft.
	if linesChanged(order.Lines, lines) {
		if !strings.EqualFold(order.Status, "draft") && strings.TrimSpace(dto.RevisionReason) == "" {
			return nil, apperrors.NewBadRequest("Revision reason is required when changing items of a submitted order")
		}
		snapshot, _ := json.Marshal(order.Lines)
		order.Revision++
		if err := uc.repo.CreateOrderRevision(ctx, &domain.SalesOrderRevision{
			SalesOrderID: order.ID, Revision: order.Revision, ChangedBy: actor.EmailFrom(ctx),
			Reason: strings.TrimSpace(dto.RevisionReason), TotalAmount: order.TotalAmount, LinesJSON: string(snapshot),
		}); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to record order revision")
		}
	}

	order.CustomerName = strings.TrimSpace(dto.CustomerName)
	order.TotalAmount = total
	order.BaseAmount = baseAmount
	order.Lines = lines
	if dto.PackagingAvailability != nil {
		v, err := normalizeState("packagingAvailability", *dto.PackagingAvailability, "pending", packagingStates)
		if err != nil {
			return nil, err
		}
		order.PackagingAvailability = v
	}
	if dto.COAStatus != nil {
		v, err := normalizeState("coaStatus", *dto.COAStatus, "pending", coaStates)
		if err != nil {
			return nil, err
		}
		order.COAStatus = v
	}
	if dto.ProductionStatus != nil {
		v, err := normalizeState("productionStatus", *dto.ProductionStatus, "not_started", productionStates)
		if err != nil {
			return nil, err
		}
		order.ProductionStatus = v
	}
	if dto.VATExempt != nil {
		order.VATExempt = *dto.VATExempt
	}
	if dto.CustomerPONumber != nil {
		order.CustomerPONumber = *dto.CustomerPONumber
	}
	if dto.CustomerPOAttachment != nil {
		order.CustomerPOAttachment = *dto.CustomerPOAttachment
	}
	if dto.DPPaid != nil {
		order.DPPaid = *dto.DPPaid
	}
	if dto.DPPercent != nil {
		order.DPPercent = *dto.DPPercent
	}
	if dto.DPAmount != nil {
		order.DPAmount = *dto.DPAmount
	}
	if dto.ShippingAddress != nil {
		order.ShippingAddress = *dto.ShippingAddress
	}
	if dto.ShippingCourier != nil {
		order.ShippingCourier = *dto.ShippingCourier
	}
	if dto.DeliveryDate != nil {
		order.DeliveryDate = *dto.DeliveryDate
	}
	if dto.InvoiceDocType != nil {
		order.InvoiceDocType = *dto.InvoiceDocType
	}
	if dto.InvoiceDocAttachment != nil {
		order.InvoiceDocAttachment = *dto.InvoiceDocAttachment
	}
	if dto.InvoiceDocNumber != nil {
		order.InvoiceDocNumber = *dto.InvoiceDocNumber
	}

	if err := uc.repo.UpdateOrderWithLines(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update sales order")
	}

	return ToSalesOrderResponse(order), nil
}

func (uc *salesUseCase) AttachInvoiceDocument(ctx context.Context, id uuid.UUID, dto AttachInvoiceDocDTO) (*SalesOrderResponseDTO, error) {
	if strings.TrimSpace(dto.InvoiceDocAttachment) == "" {
		return nil, apperrors.NewBadRequest("Invoice document attachment is required")
	}
	docType := strings.ToUpper(strings.TrimSpace(dto.InvoiceDocType))
	if docType != "CI" && docType != "PI" && !strings.Contains(strings.ToLower(dto.InvoiceDocType), "invoice") {
		return nil, apperrors.NewBadRequest("Invoice document type must be CI (Commercial Invoice) or PI (Proforma Invoice)")
	}

	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}

	order.InvoiceDocType = docType
	order.InvoiceDocNumber = strings.TrimSpace(dto.InvoiceDocNumber)
	order.InvoiceDocAttachment = strings.TrimSpace(dto.InvoiceDocAttachment)

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to attach invoice document to sales order")
	}

	return ToSalesOrderResponse(order), nil
}

func (uc *salesUseCase) ListOrders(ctx context.Context, query types.PaginationQuery) ([]SalesOrderResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	orders, total, err := uc.repo.ListOrders(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list orders")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToSalesOrderResponseList(orders), meta, nil
}

func quotationLines(inputs []QuotationLineDTO) ([]domain.QuotationLine, float64, error) {
	if len(inputs) == 0 {
		return nil, 0, apperrors.NewBadRequest("At least one quotation item is required")
	}
	lines := make([]domain.QuotationLine, 0, len(inputs))
	total := 0.0
	for i, item := range inputs {
		if strings.TrimSpace(item.Description) == "" || len(strings.TrimSpace(item.Unit)) > 50 || math.IsNaN(item.Quantity) || math.IsInf(item.Quantity, 0) || item.Quantity <= 0 || math.IsNaN(item.UnitPrice) || math.IsInf(item.UnitPrice, 0) || item.UnitPrice < 0 || item.Quantity >= 1e13 || item.UnitPrice >= 1e13 {
			return nil, 0, apperrors.NewBadRequest(fmt.Sprintf("Invalid quotation item %d", i+1))
		}
		quantity := math.Round(item.Quantity*100) / 100
		price := math.Round(item.UnitPrice*100) / 100
		subtotal := math.Round(quantity*price*100) / 100
		total = math.Round((total+subtotal)*100) / 100
		if quantity <= 0 || total >= 1e13 {
			return nil, 0, apperrors.NewBadRequest("Quotation amount or quantity is out of range")
		}
		unit := strings.TrimSpace(item.Unit)
		if unit == "" {
			unit = "pcs"
		}
		lines = append(lines, domain.QuotationLine{ProductID: item.ProductID, Position: i, Description: strings.TrimSpace(item.Description), Unit: unit, Quantity: quantity, UnitPrice: price, Subtotal: subtotal})
	}
	return lines, total, nil
}

func (uc *salesUseCase) CreateQuotation(ctx context.Context, dto CreateQuotationDTO) (*QuotationResponseDTO, error) {
	if strings.TrimSpace(dto.CustomerName) == "" {
		return nil, apperrors.NewBadRequest("Customer name is required")
	}

	lines, total, err := quotationLines(dto.Lines)
	if err != nil {
		return nil, err
	}

	quoNum := dto.QuotationNumber
	if quoNum == "" {
		quoNum = fmt.Sprintf("QUO-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)
	}

	validUntil := dto.ValidUntil
	if validUntil == "" {
		validUntil = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	}

	quo := &domain.Quotation{
		LeadID:          dto.LeadID,
		QuotationNumber: quoNum,
		CustomerName:    strings.TrimSpace(dto.CustomerName),
		TotalAmount:     total,
		Lines:           lines,
		Status:          "Sent",
		ValidUntil:      validUntil,
	}

	if err := uc.repo.CreateQuotation(ctx, quo); err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, apperrors.NewInternal(err, "Failed to create quotation")
	}

	return ToQuotationResponse(quo), nil
}

func (uc *salesUseCase) GetQuotationByID(ctx context.Context, id uuid.UUID) (*QuotationResponseDTO, error) {
	quo, err := uc.repo.GetQuotationByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get quotation")
	}
	if quo == nil {
		return nil, apperrors.NewNotFound("Quotation not found")
	}
	return ToQuotationResponse(quo), nil
}

// UpdateQuotation replaces a quotation's customer name, validity, and line
// items. Only quotations that have not yet been converted to a sales order
// may be edited, since a converted quotation's totals are already reflected
// in a real order.
func (uc *salesUseCase) UpdateQuotation(ctx context.Context, id uuid.UUID, dto UpdateQuotationDTO) (*QuotationResponseDTO, error) {
	quo, err := uc.repo.GetQuotationByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get quotation")
	}
	if quo == nil {
		return nil, apperrors.NewNotFound("Quotation not found")
	}
	if quo.Status == "Converted" {
		return nil, apperrors.NewBadRequest("Quotation has already been converted to a sales order and can no longer be edited")
	}
	if strings.TrimSpace(dto.CustomerName) == "" {
		return nil, apperrors.NewBadRequest("Customer name is required")
	}

	lines, total, err := quotationLines(dto.Lines)
	if err != nil {
		return nil, err
	}

	validUntil := dto.ValidUntil
	if validUntil == "" {
		validUntil = quo.ValidUntil
	}

	quo.CustomerName = strings.TrimSpace(dto.CustomerName)
	quo.ValidUntil = validUntil
	quo.TotalAmount = total
	quo.Lines = lines

	if err := uc.repo.UpdateQuotation(ctx, quo); err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, apperrors.NewInternal(err, "Failed to update quotation")
	}

	return ToQuotationResponse(quo), nil
}

func (uc *salesUseCase) ListQuotations(ctx context.Context, query types.PaginationQuery) ([]QuotationResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	quotations, total, err := uc.repo.ListQuotations(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list quotations")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToQuotationResponseList(quotations), meta, nil
}

// ConvertQuotationToOrder turns a still-valid, not-yet-converted Quotation
// into a real Sales Order, reusing CreateOrder's stock-check-then-commit
// logic so a converted order behaves identically to a manually entered one.
// The quotation's line items and pricing carry over untouched.
func (uc *salesUseCase) ConvertQuotationToOrder(ctx context.Context, id uuid.UUID) (*SalesOrderResponseDTO, error) {
	quo, err := uc.repo.GetQuotationByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get quotation")
	}
	if quo == nil {
		return nil, apperrors.NewNotFound("Quotation not found")
	}
	if quo.Status == "Converted" {
		return nil, apperrors.NewBadRequest("Quotation has already been converted to a sales order")
	}
	if quo.ValidUntil != "" {
		// ValidUntil parses to midnight of that day, so compare against the
		// end of that day (23:59:59) rather than its start - otherwise a
		// quotation reads as "expired" for the rest of its own valid day.
		if validUntil, err := time.Parse("2006-01-02", quo.ValidUntil); err == nil && time.Now().After(validUntil.Add(24*time.Hour-time.Nanosecond)) {
			return nil, apperrors.NewBadRequest("Quotation has expired and can no longer be converted")
		}
	}

	lines := make([]SalesOrderLineDTO, 0, len(quo.Lines))
	for _, l := range quo.Lines {
		if l.ProductID == nil {
			continue
		}
		lines = append(lines, SalesOrderLineDTO{ProductID: *l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
	}

	order, err := uc.CreateOrder(ctx, CreateSalesOrderDTO{
		CustomerName: quo.CustomerName,
		TotalAmount:  quo.TotalAmount,
		Status:       "Confirmed",
		Lines:        lines,
	})
	if err != nil {
		return nil, err
	}

	quo.Status = "Converted"
	if err := uc.repo.UpdateQuotation(ctx, quo); err != nil {
		return nil, apperrors.NewInternal(err, "Sales order created but failed to mark quotation as converted")
	}

	return order, nil
}

func (uc *salesUseCase) CreateInvoice(ctx context.Context, dto CreateInvoiceDTO) (*InvoiceResponseDTO, error) {
	if dto.CustomerName == "" {
		return nil, apperrors.NewBadRequest("Customer name is required")
	}

	invNum := dto.InvoiceNumber
	if invNum == "" {
		invNum = fmt.Sprintf("INV-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)
	}

	dueDate := dto.DueDate
	if dueDate == "" {
		dueDate = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	}

	if dto.SalesOrderID != nil {
		order, err := uc.repo.GetOrderByID(ctx, *dto.SalesOrderID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get sales order")
		}
		if order == nil {
			return nil, apperrors.NewNotFound("Sales order not found")
		}
		// PPN switched off on the order (e.g. export): never add VAT to its invoices.
		if order.VATExempt {
			dto.ApplyVAT = false
		}
	}

	amounts, err := financeApp.ComputeInvoiceAmounts(financeApp.InvoiceAdjustmentInput{
		Subtotal: dto.TotalAmount, DiscountPercent: dto.DiscountPercent, DiscountAmount: dto.DiscountAmount,
		AdditionalCost: dto.AdditionalCost, RoundTo: dto.RoundTo,
		VAT: financeApp.VATInput{Apply: dto.ApplyVAT, Rate: dto.VATRate, OtherValueBase: dto.VATOtherValueBase == nil || *dto.VATOtherValueBase},
	})
	if err != nil {
		return nil, err
	}

	inv := &domain.Invoice{
		TaxBase:           amounts.VAT.TaxBase,
		DPPOtherValue:     amounts.VAT.DPPOtherValue,
		VATRate:           amounts.VAT.Rate,
		VATOtherValueBase: amounts.VAT.OtherValueBase,
		VATAmount:         amounts.VAT.Amount,
		InvoiceNumber:     invNum,
		CustomerName:      dto.CustomerName,
		TotalAmount:       amounts.Total,
		Subtotal:          amounts.Subtotal,
		DiscountAmount:    amounts.Discount,
		AdditionalCost:    amounts.AdditionalCost,
		RoundingAmount:    amounts.Rounding,
		InvoiceDate:       time.Now().Format("2006-01-02"),
		DueDate:           dueDate,
		PaidAmount:        0,
		Status:            "pending",
		SalesOrderID:      dto.SalesOrderID,
	}

	if err := uc.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create invoice")
	}

	e := InvoiceLedgerEntry(inv)
	uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
	uc.syncReceivable(ctx, inv)

	return ToInvoiceResponse(inv), nil
}

func (uc *salesUseCase) ListInvoices(ctx context.Context, query types.PaginationQuery) ([]InvoiceResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	invoices, total, err := uc.repo.ListInvoices(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list invoices")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToInvoiceResponseList(invoices), meta, nil
}

func (uc *salesUseCase) RecordInvoicePayment(ctx context.Context, id uuid.UUID, dto RecordSalesPaymentDTO) (*InvoiceResponseDTO, error) {
	if dto.Amount <= 0 {
		return nil, apperrors.NewBadRequest("Payment amount must be positive")
	}
	inv, err := uc.repo.GetInvoiceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get invoice")
	}
	if inv == nil {
		return nil, apperrors.NewNotFound("Sales invoice not found")
	}
	if inv.PaidAmount+dto.Amount > inv.TotalAmount-inv.CreditedAmount {
		return nil, apperrors.NewBadRequest("Payment amount would exceed the amount still owed on the invoice")
	}
	inv.PaidAmount += dto.Amount
	inv.Status = salesInvoiceStatus(inv.TotalAmount-inv.CreditedAmount, inv.PaidAmount, inv.DueDate)
	if err := uc.repo.UpdateInvoice(ctx, inv); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record payment")
	}

	e := InvoicePaymentLedgerEntry(inv, dto.Amount)
	uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
	uc.syncReceivable(ctx, inv)
	return ToInvoiceResponse(inv), nil
}

// syncReceivable mirrors the invoice into finance's AR sub-ledger. Like
// postLedger, a failure is logged rather than failing the invoice.
func (uc *salesUseCase) syncReceivable(ctx context.Context, inv *domain.Invoice) {
	if uc.ledger == nil {
		return
	}
	if err := uc.ledger.UpsertReceivable(ctx, InvoiceSubledgerDoc(inv)); err != nil {
		slog.Warn("sales: failed to sync receivable", "invoice", inv.ID, "error", err)
	}
}

// postLedger records an automatic journal entry. A ledger failure must not
// roll back the business document, so it is logged for later reconciliation.
func (uc *salesUseCase) postLedger(ctx context.Context, sourceDoc, memo string, lines []financeApp.LedgerLine) {
	if uc.ledger == nil {
		return
	}
	if err := uc.ledger.PostEntry(ctx, sourceDoc, memo, lines); err != nil {
		slog.Warn("sales: failed to post ledger entry", "sourceDoc", sourceDoc, "error", err)
	}
}

// salesInvoiceStatus derives pending/partial/paid/overdue from amounts and due date,
// mirroring procurement's purchaseInvoiceStatus. Kept as a local copy rather than a
// cross-module call so sales and procurement can evolve independently.
func salesInvoiceStatus(total, paid float64, dueDate string) string {
	if paid >= total {
		return "paid"
	}
	if paid > 0 {
		if due, err := time.Parse("2006-01-02", dueDate); err == nil && time.Now().After(due) {
			return "overdue"
		}
		return "partial"
	}
	if due, err := time.Parse("2006-01-02", dueDate); err == nil && time.Now().After(due) {
		return "overdue"
	}
	return "pending"
}

func (uc *salesUseCase) CreateDelivery(ctx context.Context, dto CreateDeliveryDTO) (*DeliveryResponseDTO, error) {
	if dto.CustomerName == "" {
		return nil, apperrors.NewBadRequest("Customer name is required")
	}
	if dto.SalesOrderID == uuid.Nil {
		return nil, apperrors.NewBadRequest("A valid sales order must be selected")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one delivery line item is required")
	}
	order, err := uc.repo.GetOrderByID(ctx, dto.SalesOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}

	// orderedQty/shippedQty are keyed by ProductID so a Sales Order that
	// happens to repeat the same product across two lines is still tracked
	// as one combined total to ship, matching how much of it the order
	// actually holds.
	orderedQty := map[uuid.UUID]float64{}
	for _, l := range order.Lines {
		orderedQty[l.ProductID] += l.Quantity
	}

	priorDeliveries, err := uc.repo.ListDeliveriesByOrderID(ctx, dto.SalesOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing deliveries for this sales order")
	}
	shippedQty := map[uuid.UUID]float64{}
	for _, prior := range priorDeliveries {
		for _, l := range prior.Lines {
			shippedQty[l.ProductID] += l.Quantity
		}
	}

	lines := make([]domain.DeliveryLine, 0, len(dto.Lines))
	for _, l := range dto.Lines {
		if l.ProductID == uuid.Nil || !(l.Quantity > 0) {
			return nil, apperrors.NewBadRequest("Each delivery line requires a productId and a positive quantity")
		}
		ordered, onOrder := orderedQty[l.ProductID]
		if !onOrder {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Product %s is not part of sales order %s", l.ProductID, order.OrderNumber))
		}
		remaining := ordered - shippedQty[l.ProductID]
		if l.Quantity > remaining {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Delivery quantity for product %s (%.2f) exceeds what's left to ship on this order (%.2f)", l.ProductID, l.Quantity, remaining))
		}
		shippedQty[l.ProductID] += l.Quantity
		lines = append(lines, domain.DeliveryLine{ProductID: l.ProductID, Quantity: l.Quantity})
	}

	deliveryNum := fmt.Sprintf("DO-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)
	deliveryDate := dto.DeliveryDate
	if deliveryDate == "" {
		deliveryDate = time.Now().Format("2006-01-02")
	}

	soID := dto.SalesOrderID
	d := &domain.Delivery{
		SalesOrderID:   &soID,
		DeliveryNumber: deliveryNum,
		SONumber:       order.OrderNumber,
		CustomerName:   dto.CustomerName,
		DeliveryDate:   deliveryDate,
		Carrier:        dto.Carrier,
		Status:         "processing",
		Lines:          lines,
	}

	if err := uc.repo.CreateDelivery(ctx, d); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create delivery")
	}

	return ToDeliveryResponse(d), nil
}

func (uc *salesUseCase) ListDeliveries(ctx context.Context, query types.PaginationQuery) ([]DeliveryResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	deliveries, total, err := uc.repo.ListDeliveries(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list deliveries")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToDeliveryResponseList(deliveries), meta, nil
}

func (uc *salesUseCase) soNumberFor(ctx context.Context, soID uuid.UUID) string {
	order, err := uc.repo.GetOrderByID(ctx, soID)
	if err != nil || order == nil {
		return ""
	}
	return order.OrderNumber
}

// Sales down payments

func (uc *salesUseCase) CreateDownPayment(ctx context.Context, dto CreateSalesDownPaymentDTO) (*SalesDownPaymentResponseDTO, error) {
	if dto.SalesOrderID == uuid.Nil || dto.Amount <= 0 {
		return nil, apperrors.NewBadRequest("salesOrderId and a positive amount are required")
	}
	order, err := uc.repo.GetOrderByID(ctx, dto.SalesOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}

	customerName := dto.CustomerName
	if customerName == "" {
		customerName = order.CustomerName
	}

	paymentDate := dto.PaymentDate
	if paymentDate == "" {
		paymentDate = time.Now().Format("2006-01-02")
	}

	dp := &domain.SalesDownPayment{
		DPNumber:      fmt.Sprintf("SDP-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000),
		SalesOrderID:  dto.SalesOrderID,
		CustomerName:  customerName,
		Amount:        dto.Amount,
		PaymentDate:   paymentDate,
		PaymentMethod: dto.PaymentMethod,
		Status:        "unapplied",
		Notes:         dto.Notes,
	}
	if err := uc.repo.CreateDownPayment(ctx, dp); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record down payment")
	}
	return ToSalesDownPaymentResponse(dp, order.OrderNumber), nil
}

// ApplyDownPayment offsets a target Invoice's outstanding balance by this
// down payment's remaining amount (same customer), then flips the down
// payment's status to applied/partially_applied so it can never be applied
// twice for more than it actually holds.
func (uc *salesUseCase) ApplyDownPayment(ctx context.Context, id uuid.UUID, dto ApplySalesDownPaymentDTO) (*SalesDownPaymentResponseDTO, error) {
	dp, err := uc.repo.GetDownPaymentByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get down payment")
	}
	if dp == nil {
		return nil, apperrors.NewNotFound("Down payment not found")
	}
	remaining := dp.Amount - dp.AppliedAmount
	if remaining <= 0 {
		return nil, apperrors.NewBadRequest("Down payment has already been fully applied")
	}
	if dto.InvoiceID == uuid.Nil {
		return nil, apperrors.NewBadRequest("invoiceId is required")
	}

	inv, err := uc.repo.GetInvoiceByID(ctx, dto.InvoiceID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get invoice")
	}
	if inv == nil {
		return nil, apperrors.NewNotFound("Invoice not found")
	}
	if !strings.EqualFold(strings.TrimSpace(inv.CustomerName), strings.TrimSpace(dp.CustomerName)) {
		return nil, apperrors.NewBadRequest("Down payment customer does not match the selected invoice's customer")
	}

	outstanding := inv.TotalAmount - inv.PaidAmount
	if outstanding <= 0 {
		return nil, apperrors.NewBadRequest("Selected invoice has no outstanding balance to apply against")
	}

	applyAmount := remaining
	if applyAmount > outstanding {
		applyAmount = outstanding
	}

	inv.PaidAmount += applyAmount
	inv.Status = salesInvoiceStatus(inv.TotalAmount, inv.PaidAmount, inv.DueDate)
	if err := uc.repo.UpdateInvoice(ctx, inv); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to apply down payment to invoice")
	}

	dp.AppliedAmount += applyAmount
	if dp.AppliedAmount >= dp.Amount {
		dp.Status = "applied"
	} else {
		dp.Status = "partially_applied"
	}
	if err := uc.repo.UpdateDownPayment(ctx, dp); err != nil {
		return nil, apperrors.NewInternal(err, "Down payment applied to invoice but failed to update down payment record")
	}

	return ToSalesDownPaymentResponse(dp, uc.soNumberFor(ctx, dp.SalesOrderID)), nil
}

func (uc *salesUseCase) ListDownPayments(ctx context.Context, query types.PaginationQuery) ([]SalesDownPaymentResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListDownPayments(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list down payments")
	}
	soNumbers := make(map[uuid.UUID]string)
	for _, dp := range items {
		if _, ok := soNumbers[dp.SalesOrderID]; !ok {
			soNumbers[dp.SalesOrderID] = uc.soNumberFor(ctx, dp.SalesOrderID)
		}
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToSalesDownPaymentResponseList(items, soNumbers), meta, nil
}

// Sales returns

func (uc *salesUseCase) GetSalesReturnByID(ctx context.Context, id uuid.UUID) (*SalesReturnResponseDTO, error) {
	ret, err := uc.repo.GetSalesReturnByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales return")
	}
	if ret == nil {
		return nil, apperrors.NewNotFound("Sales return not found")
	}
	return ToSalesReturnResponse(ret), nil
}

func (uc *salesUseCase) ListSalesReturns(ctx context.Context, query types.PaginationQuery) ([]SalesReturnResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListSalesReturns(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list sales returns")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToSalesReturnResponseList(items), meta, nil
}

func (uc *salesUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountOrders(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	orders := []CreateSalesOrderDTO{
		{
			OrderNumber:   "SO-2026-0814",
			CustomerName:  "PT Graha Konstruksi Nusantara",
			TotalAmount:   450000000,
			Channel:       "Direct B2B",
			PaymentStatus: "Paid",
			Status:        "Delivered",
		},
		{
			OrderNumber:   "SO-2026-0820",
			CustomerName:  "CV Surya Makmur Logistik",
			TotalAmount:   185000000,
			Channel:       "Direct B2B",
			PaymentStatus: "Paid",
			Status:        "Processing",
		},
		{
			OrderNumber:   "SO-2026-0902",
			CustomerName:  "Marketplace TikTok Shop",
			TotalAmount:   18200000,
			Channel:       "TikTok Shop",
			PaymentStatus: "Paid",
			Status:        "Confirmed",
		},
	}

	for _, item := range orders {
		_, _ = uc.CreateOrder(ctx, item)
	}

	quotations := []CreateQuotationDTO{
		{
			QuotationNumber: "QUO-2026-0814",
			CustomerName:    "PT Graha Konstruksi Nusantara",
			TotalAmount:     450000000,
			Lines:           []QuotationLineDTO{{Description: "Paket material konstruksi", Unit: "paket", Quantity: 1, UnitPrice: 450000000}},
			ValidUntil:      "2026-09-30",
		},
		{
			QuotationNumber: "QUO-2026-0901",
			CustomerName:    "PT Sentosa Jaya Abadi",
			TotalAmount:     230000000,
			Lines:           []QuotationLineDTO{{Description: "Paket material", Unit: "paket", Quantity: 1, UnitPrice: 230000000}},
			ValidUntil:      "2026-10-15",
		},
	}

	for _, item := range quotations {
		_, _ = uc.CreateQuotation(ctx, item)
	}

	return nil
}

func (uc *salesUseCase) ListOrderRevisions(ctx context.Context, id uuid.UUID) ([]domain.SalesOrderRevision, error) {
	order, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}
	revs, err := uc.repo.ListOrderRevisions(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list order revisions")
	}
	return revs, nil
}

// linesChanged compares item sets by product, quantity and unit price.
func linesChanged(old, next []domain.SalesOrderLine) bool {
	if len(old) != len(next) {
		return true
	}
	type key struct {
		p    uuid.UUID
		q, u float64
	}
	seen := make(map[key]int, len(old))
	for _, l := range old {
		seen[key{l.ProductID, l.Quantity, l.UnitPrice}]++
	}
	for _, l := range next {
		k := key{l.ProductID, l.Quantity, l.UnitPrice}
		if seen[k] == 0 {
			return true
		}
		seen[k]--
	}
	return false
}

func stampRejection(order *domain.SalesOrder, dto SalesApprovalActionDTO) {
	now := time.Now()
	order.RejectionReason = strings.TrimSpace(dto.Comments)
	order.RejectedBy = dto.ActorName
	order.RejectedAt = &now
}

// checkSubmitDocuments enforces the B2B rule: before an order goes for approval the
// incoming PO / outgoing PI document must be uploaded (number or attachment) and/or the
// down payment received. Marketplace and POS orders are not B2B documents and are exempt.
func checkSubmitDocuments(o *domain.SalesOrder) error {
	if o.MarketplacePlatform != "" || o.Channel == "POS" {
		return nil
	}
	hasDoc := strings.TrimSpace(o.CustomerPONumber) != "" || strings.TrimSpace(o.CustomerPOAttachment) != ""
	if hasDoc || o.DPPaid {
		return nil
	}
	return apperrors.NewBadRequest("Upload the incoming PO / outgoing PI document, or mark the down payment as received, before submitting this order")
}
