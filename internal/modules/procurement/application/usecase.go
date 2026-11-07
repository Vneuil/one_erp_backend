package application

import (
	"context"
	"fmt"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"log/slog"
	"time"

	approvalApp "github.com/divinecoid/one-backend/internal/modules/approval/application"
	currencyApp "github.com/divinecoid/one-backend/internal/modules/currency/application"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// purchaseOrderDocumentType is the document type string this module submits
// to modules/approval when a Purchase Order is created. Any workflow an
// admin configures for this type (see /approval/workflows) automatically
// gates PO approval - no code change needed to adjust thresholds/levels.
const (
	purchaseOrderDocumentType   = "purchase_order"
	purchaseRequestDocumentType = "purchase_request"
)

type ProcurementUseCase interface {
	// Purchase requests
	CreatePurchaseRequest(ctx context.Context, dto CreatePurchaseRequestDTO) (*PurchaseRequestResponseDTO, error)
	GetPurchaseRequestByID(ctx context.Context, id uuid.UUID) (*PurchaseRequestResponseDTO, error)
	ListPurchaseRequests(ctx context.Context, query types.PaginationQuery) ([]PurchaseRequestResponseDTO, types.PaginationMeta, error)
	UpdatePurchaseRequest(ctx context.Context, id uuid.UUID, dto UpdatePurchaseRequestDTO) (*PurchaseRequestResponseDTO, error)
	SubmitPurchaseRequest(ctx context.Context, id uuid.UUID) (*PurchaseRequestResponseDTO, error)
	ApprovePurchaseRequest(ctx context.Context, id uuid.UUID, dto ApprovalActionDTO) (*PurchaseRequestResponseDTO, error)
	RejectPurchaseRequest(ctx context.Context, id uuid.UUID, dto ApprovalActionDTO) (*PurchaseRequestResponseDTO, error)
	FinalizePurchaseRequestApproved(ctx context.Context, id uuid.UUID) error
	FinalizePurchaseRequestRejected(ctx context.Context, id uuid.UUID) error

	// Purchase orders
	CreatePurchaseOrder(ctx context.Context, dto CreatePurchaseOrderDTO) (*PurchaseOrderResponseDTO, error)
	GetPurchaseOrderByID(ctx context.Context, id uuid.UUID) (*PurchaseOrderResponseDTO, error)
	ListPurchaseOrders(ctx context.Context, query types.PaginationQuery) ([]PurchaseOrderResponseDTO, types.PaginationMeta, error)
	UpdatePurchaseOrder(ctx context.Context, id uuid.UUID, dto UpdatePurchaseOrderDTO) (*PurchaseOrderResponseDTO, error)
	SubmitPurchaseOrder(ctx context.Context, id uuid.UUID) (*PurchaseOrderResponseDTO, error)
	ApprovePurchaseOrder(ctx context.Context, id uuid.UUID, dto ApprovalActionDTO) (*PurchaseOrderResponseDTO, error)
	RejectPurchaseOrder(ctx context.Context, id uuid.UUID, dto ApprovalActionDTO) (*PurchaseOrderResponseDTO, error)
	CancelPurchaseOrder(ctx context.Context, id uuid.UUID) (*PurchaseOrderResponseDTO, error)
	GetLastPOPrice(ctx context.Context, supplierID, productID uuid.UUID) (float64, error)

	// FinalizePurchaseOrderApproved/Rejected are the single place that
	// applies the effect of a Purchase Order's approval workflow reaching a
	// final state: they set the PO's own status field. Both
	// ApprovePurchaseOrder/RejectPurchaseOrder (the module-specific
	// endpoints) and the approval module's DocumentStatusCallback (fired
	// when a PO is approved/rejected from the generic Approval Center) call
	// through here - never duplicated between the two paths.
	FinalizePurchaseOrderApproved(ctx context.Context, id uuid.UUID) error
	FinalizePurchaseOrderRejected(ctx context.Context, id uuid.UUID) error

	// Goods receipts
	CreateGoodsReceipt(ctx context.Context, dto CreateGoodsReceiptDTO) (*GoodsReceiptResponseDTO, error)
	GetGoodsReceiptByID(ctx context.Context, id uuid.UUID) (*GoodsReceiptResponseDTO, error)
	ListGoodsReceipts(ctx context.Context, query types.PaginationQuery) ([]GoodsReceiptResponseDTO, types.PaginationMeta, error)

	// Purchase invoices
	CreatePurchaseInvoice(ctx context.Context, dto CreatePurchaseInvoiceDTO) (*PurchaseInvoiceResponseDTO, error)
	GetPurchaseInvoiceByID(ctx context.Context, id uuid.UUID) (*PurchaseInvoiceResponseDTO, error)
	ListPurchaseInvoices(ctx context.Context, query types.PaginationQuery) ([]PurchaseInvoiceResponseDTO, types.PaginationMeta, error)
	RecordInvoicePayment(ctx context.Context, id uuid.UUID, dto RecordInvoicePaymentDTO) (*PurchaseInvoiceResponseDTO, error)

	// Purchase down payments
	CreateDownPayment(ctx context.Context, dto CreateDownPaymentDTO) (*PurchaseDownPaymentResponseDTO, error)
	ListDownPayments(ctx context.Context, query types.PaginationQuery) ([]PurchaseDownPaymentResponseDTO, types.PaginationMeta, error)

	// Purchase returns
	CreatePurchaseReturn(ctx context.Context, dto CreatePurchaseReturnDTO) (*PurchaseReturnResponseDTO, error)
	GetPurchaseReturnByID(ctx context.Context, id uuid.UUID) (*PurchaseReturnResponseDTO, error)
	ListPurchaseReturns(ctx context.Context, query types.PaginationQuery) ([]PurchaseReturnResponseDTO, types.PaginationMeta, error)

	// Invoice receipts
	CreateInvoiceReceipt(ctx context.Context, dto CreateInvoiceReceiptDTO) (*InvoiceReceiptResponseDTO, error)
	GetInvoiceReceiptByID(ctx context.Context, id uuid.UUID) (*InvoiceReceiptResponseDTO, error)
	ListInvoiceReceipts(ctx context.Context, query types.PaginationQuery) ([]InvoiceReceiptResponseDTO, types.PaginationMeta, error)

	SeedInitialData(ctx context.Context) error
}

type procurementUseCase struct {
	repo        domain.ProcurementRepository
	approvalUC  approvalApp.ApprovalUseCase
	inventoryUC inventoryApp.InventoryUseCase
	currencyUC  currencyApp.CurrencyUseCase
	productRepo productDomain.ProductRepository
	ledger      financeApp.LedgerPoster // optional; nil disables auto-posting to the general ledger
}

// Option customises optional collaborators of the procurement use case.
type Option func(*procurementUseCase)

// WithLedger enables automatic journal posting for purchase invoices, payments and down payments.
func WithLedger(l financeApp.LedgerPoster) Option {
	return func(uc *procurementUseCase) { uc.ledger = l }
}

// WithProductRepo provides access to product catalog for pricing checks during approvals.
func WithProductRepo(pr productDomain.ProductRepository) Option {
	return func(uc *procurementUseCase) { uc.productRepo = pr }
}

func NewProcurementUseCase(repo domain.ProcurementRepository, approvalUC approvalApp.ApprovalUseCase, inventoryUC inventoryApp.InventoryUseCase, currencyUC currencyApp.CurrencyUseCase, opts ...Option) ProcurementUseCase {
	uc := &procurementUseCase{repo: repo, approvalUC: approvalUC, inventoryUC: inventoryUC, currencyUC: currencyUC}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

// Purchase requests

func (uc *procurementUseCase) CreatePurchaseRequest(ctx context.Context, dto CreatePurchaseRequestDTO) (*PurchaseRequestResponseDTO, error) {
	if dto.RequestedBy == "" {
		return nil, apperrors.NewBadRequest("Requested by is required")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line item is required")
	}
	lines := make([]domain.PurchaseRequestLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.ProductID == uuid.Nil || l.Quantity <= 0 {
			return nil, apperrors.NewBadRequest("Each line requires a productId and a positive quantity")
		}
		lines[i] = domain.PurchaseRequestLine{ProductID: l.ProductID, Quantity: l.Quantity, Notes: l.Notes}
	}

	category := dto.Category
	if category == "" {
		category = "inventory"
	}

	pr := &domain.PurchaseRequest{
		RequestNo:    fmt.Sprintf("PR-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000),
		RequestedBy:  dto.RequestedBy,
		Department:   dto.Department,
		NeededByDate: dto.NeededByDate,
		Category:     category,
		SalesOrderID: dto.SalesOrderID,
		SalesOrderNo: dto.SalesOrderNo,
		Status:       "draft",
		Lines:        lines,
	}
	if err := uc.repo.CreatePurchaseRequest(ctx, pr); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create purchase request")
	}

	var totalAmount float64
	if uc.productRepo != nil {
		for _, l := range lines {
			if prod, err := uc.productRepo.GetByID(ctx, l.ProductID); err == nil && prod != nil {
				totalAmount += float64(l.Quantity) * prod.CostPrice
			}
		}
	}

	requiresApproval := false
	if uc.approvalUC != nil {
		result, err := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
			DocumentType:   purchaseRequestDocumentType,
			DocumentID:     pr.ID,
			DocumentNumber: pr.RequestNo,
			Amount:         totalAmount,
			RequesterName:  pr.RequestedBy,
		})
		if err == nil && result != nil && result.RequiresApproval {
			requiresApproval = true
			pr.Status = "pending_approval"
			_ = uc.repo.UpdatePurchaseRequest(ctx, pr)
		}
	}
	if !requiresApproval {
		pr.Status = "approved"
		_ = uc.repo.UpdatePurchaseRequest(ctx, pr)
	}

	return ToPurchaseRequestResponse(pr), nil
}

func (uc *procurementUseCase) GetPurchaseRequestByID(ctx context.Context, id uuid.UUID) (*PurchaseRequestResponseDTO, error) {
	pr, err := uc.repo.GetPurchaseRequestByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase request")
	}
	if pr == nil {
		return nil, apperrors.NewNotFound("Purchase request not found")
	}
	return ToPurchaseRequestResponse(pr), nil
}

func (uc *procurementUseCase) ListPurchaseRequests(ctx context.Context, query types.PaginationQuery) ([]PurchaseRequestResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListPurchaseRequests(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list purchase requests")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToPurchaseRequestResponseList(items), meta, nil
}

func (uc *procurementUseCase) transitionPurchaseRequest(ctx context.Context, id uuid.UUID, from map[string]bool, to string) (*PurchaseRequestResponseDTO, error) {
	pr, err := uc.repo.GetPurchaseRequestByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase request")
	}
	if pr == nil {
		return nil, apperrors.NewNotFound("Purchase request not found")
	}
	if !from[pr.Status] {
		return nil, apperrors.NewConflict(fmt.Sprintf("Purchase request in status %s cannot transition to %s", pr.Status, to))
	}
	// RequestedBy is the requester's login email (set from the JWT on create).
	if to == "approved" {
		if err := sod.ForbidSelfApproval(ctx, pr.RequestedBy); err != nil {
			return nil, err
		}
	}
	pr.Status = to
	if err := uc.repo.UpdatePurchaseRequest(ctx, pr); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update purchase request")
	}
	return ToPurchaseRequestResponse(pr), nil
}

func (uc *procurementUseCase) UpdatePurchaseRequest(ctx context.Context, id uuid.UUID, dto UpdatePurchaseRequestDTO) (*PurchaseRequestResponseDTO, error) {
	pr, err := uc.repo.GetPurchaseRequestByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase request")
	}
	if pr == nil {
		return nil, apperrors.NewNotFound("Purchase request not found")
	}
	if pr.Status != "draft" {
		return nil, apperrors.NewConflict("Only draft purchase requests can be edited")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line item is required")
	}
	lines := make([]domain.PurchaseRequestLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.ProductID == uuid.Nil || l.Quantity <= 0 {
			return nil, apperrors.NewBadRequest("Each line requires a productId and a positive quantity")
		}
		lines[i] = domain.PurchaseRequestLine{ProductID: l.ProductID, Quantity: l.Quantity, Notes: l.Notes}
	}
	if dto.Department != "" {
		pr.Department = dto.Department
	}
	if dto.NeededByDate != "" {
		pr.NeededByDate = dto.NeededByDate
	}
	if dto.Category != "" {
		pr.Category = dto.Category
	}
	if dto.SalesOrderID != nil {
		pr.SalesOrderID = dto.SalesOrderID
		pr.SalesOrderNo = dto.SalesOrderNo
	}
	pr.Lines = lines
	if err := uc.repo.UpdatePurchaseRequest(ctx, pr); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update purchase request")
	}
	return ToPurchaseRequestResponse(pr), nil
}

func (uc *procurementUseCase) SubmitPurchaseRequest(ctx context.Context, id uuid.UUID) (*PurchaseRequestResponseDTO, error) {
	pr, err := uc.repo.GetPurchaseRequestByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase request")
	}
	if pr == nil {
		return nil, apperrors.NewNotFound("Purchase request not found")
	}
	if pr.Status != "draft" && pr.Status != "pending_supervisor" && pr.Status != "pending" {
		return nil, apperrors.NewConflict(fmt.Sprintf("Purchase request in status %s cannot be submitted", pr.Status))
	}

	var totalAmount float64
	if uc.productRepo != nil {
		for _, l := range pr.Lines {
			if prod, err := uc.productRepo.GetByID(ctx, l.ProductID); err == nil && prod != nil {
				totalAmount += float64(l.Quantity) * prod.CostPrice
			}
		}
	}

	requiresApproval := false
	if uc.approvalUC != nil {
		result, err := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
			DocumentType:   purchaseRequestDocumentType,
			DocumentID:     pr.ID,
			DocumentNumber: pr.RequestNo,
			Amount:         totalAmount,
			RequesterName:  pr.RequestedBy,
		})
		if err != nil {
			return nil, err
		}
		if result != nil && result.RequiresApproval {
			requiresApproval = true
		}
	}
	if requiresApproval {
		pr.Status = "pending_approval"
	} else {
		pr.Status = "approved"
	}
	if err := uc.repo.UpdatePurchaseRequest(ctx, pr); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update purchase request")
	}
	return ToPurchaseRequestResponse(pr), nil
}

func (uc *procurementUseCase) ApprovePurchaseRequest(ctx context.Context, id uuid.UUID, dto ApprovalActionDTO) (*PurchaseRequestResponseDTO, error) {
	pr, err := uc.repo.GetPurchaseRequestByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase request")
	}
	if pr == nil {
		return nil, apperrors.NewNotFound("Purchase request not found")
	}
	if err := sod.ForbidSelfApproval(ctx, pr.RequestedBy); err != nil {
		return nil, err
	}

	if uc.approvalUC != nil {
		activeReq, err := uc.approvalUC.GetActiveRequestForDocument(ctx, purchaseRequestDocumentType, pr.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check approval status")
		}
		// If there is no active approval request yet, but the document is in a pending state,
		// dynamically submit to approval workflow if a workflow matches the nominal:
		if activeReq == nil && (pr.Status == "pending_approval" || pr.Status == "pending_supervisor" || pr.Status == "pending_owner" || pr.Status == "pending" || pr.Status == "submitted") {
			var totalAmount float64
			if uc.productRepo != nil {
				for _, l := range pr.Lines {
					if prod, err := uc.productRepo.GetByID(ctx, l.ProductID); err == nil && prod != nil {
						totalAmount += float64(l.Quantity) * prod.CostPrice
					}
				}
			}
			submitRes, submitErr := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
				DocumentType:   purchaseRequestDocumentType,
				DocumentID:     pr.ID,
				DocumentNumber: pr.RequestNo,
				Amount:         totalAmount,
				RequesterName:  pr.RequestedBy,
			})
			if submitErr == nil && submitRes != nil && submitRes.RequiresApproval && submitRes.Request != nil {
				activeReq = submitRes.Request
				pr.Status = "pending_approval"
				_ = uc.repo.UpdatePurchaseRequest(ctx, pr)
			}
		}

		if activeReq != nil && activeReq.Status == "pending" {
			if _, err := uc.approvalUC.ApproveStep(ctx, activeReq.ID, approvalApp.ActOnStepDTO{
				ActorName: dto.ActorName, ActorRole: dto.ActorRole, Comments: dto.Comments,
			}); err != nil {
				return nil, err
			}
			pr, err = uc.repo.GetPurchaseRequestByID(ctx, id)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to get purchase request")
			}
			return ToPurchaseRequestResponse(pr), nil
		}
	}

	if pr.Status != "draft" && pr.Status != "submitted" && pr.Status != "pending_supervisor" && pr.Status != "pending_owner" && pr.Status != "pending_approval" {
		return nil, apperrors.NewConflict(fmt.Sprintf("Purchase request in status %s cannot be approved", pr.Status))
	}
	actorUser := actor.EmailFrom(ctx)
	now := time.Now()
	if pr.Status == "pending_supervisor" {
		pr.SupervisorApprovedBy = actorUser
		pr.Status = "pending_owner"
	} else {
		pr.OwnerApprovedBy = actorUser
		pr.OwnerApprovedAt = &now
		pr.Status = "approved"
	}
	if err := uc.repo.UpdatePurchaseRequest(ctx, pr); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update purchase request")
	}
	return ToPurchaseRequestResponse(pr), nil
}

func (uc *procurementUseCase) RejectPurchaseRequest(ctx context.Context, id uuid.UUID, dto ApprovalActionDTO) (*PurchaseRequestResponseDTO, error) {
	pr, err := uc.repo.GetPurchaseRequestByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase request")
	}
	if pr == nil {
		return nil, apperrors.NewNotFound("Purchase request not found")
	}

	if uc.approvalUC != nil {
		activeReq, err := uc.approvalUC.GetActiveRequestForDocument(ctx, purchaseRequestDocumentType, pr.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check approval status")
		}
		if activeReq == nil && (pr.Status == "pending_approval" || pr.Status == "pending_supervisor" || pr.Status == "pending_owner" || pr.Status == "pending" || pr.Status == "submitted") {
			var totalAmount float64
			if uc.productRepo != nil {
				for _, l := range pr.Lines {
					if prod, err := uc.productRepo.GetByID(ctx, l.ProductID); err == nil && prod != nil {
						totalAmount += float64(l.Quantity) * prod.CostPrice
					}
				}
			}
			submitRes, submitErr := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
				DocumentType:   purchaseRequestDocumentType,
				DocumentID:     pr.ID,
				DocumentNumber: pr.RequestNo,
				Amount:         totalAmount,
				RequesterName:  pr.RequestedBy,
			})
			if submitErr == nil && submitRes != nil && submitRes.RequiresApproval && submitRes.Request != nil {
				activeReq = submitRes.Request
				pr.Status = "pending_approval"
				_ = uc.repo.UpdatePurchaseRequest(ctx, pr)
			}
		}

		if activeReq != nil && activeReq.Status == "pending" {
			if _, err := uc.approvalUC.RejectStep(ctx, activeReq.ID, approvalApp.ActOnStepDTO{
				ActorName: dto.ActorName, ActorRole: dto.ActorRole, Comments: dto.Comments,
			}); err != nil {
				return nil, err
			}
			pr, err = uc.repo.GetPurchaseRequestByID(ctx, id)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to get purchase request")
			}
			return ToPurchaseRequestResponse(pr), nil
		}
	}

	if pr.Status != "draft" && pr.Status != "submitted" && pr.Status != "pending_supervisor" && pr.Status != "pending_owner" && pr.Status != "pending_approval" {
		return nil, apperrors.NewConflict(fmt.Sprintf("Purchase request in status %s cannot be rejected", pr.Status))
	}
	pr.Status = "rejected"
	if dto.Comments != "" {
		pr.RejectionReason = dto.Comments
	}
	if err := uc.repo.UpdatePurchaseRequest(ctx, pr); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update purchase request")
	}
	return ToPurchaseRequestResponse(pr), nil
}

func (uc *procurementUseCase) FinalizePurchaseRequestApproved(ctx context.Context, id uuid.UUID) error {
	pr, err := uc.repo.GetPurchaseRequestByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get purchase request")
	}
	if pr == nil {
		return apperrors.NewNotFound("Purchase request not found")
	}
	if pr.Status != "pending_approval" {
		return nil
	}
	actorUser := actor.EmailFrom(ctx)
	now := time.Now()
	pr.OwnerApprovedBy = actorUser
	pr.OwnerApprovedAt = &now
	pr.Status = "approved"
	if err := uc.repo.UpdatePurchaseRequest(ctx, pr); err != nil {
		return apperrors.NewInternal(err, "Failed to approve purchase request")
	}
	return nil
}

func (uc *procurementUseCase) FinalizePurchaseRequestRejected(ctx context.Context, id uuid.UUID) error {
	pr, err := uc.repo.GetPurchaseRequestByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get purchase request")
	}
	if pr == nil {
		return apperrors.NewNotFound("Purchase request not found")
	}
	if pr.Status != "pending_approval" {
		return nil
	}
	pr.Status = "rejected"
	if err := uc.repo.UpdatePurchaseRequest(ctx, pr); err != nil {
		return apperrors.NewInternal(err, "Failed to reject purchase request")
	}
	return nil
}

// Purchase orders

func (uc *procurementUseCase) CreatePurchaseOrder(ctx context.Context, dto CreatePurchaseOrderDTO) (*PurchaseOrderResponseDTO, error) {
	if dto.SupplierID == uuid.Nil {
		return nil, apperrors.NewBadRequest("supplierId is required")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line item is required")
	}

	var total float64
	lines := make([]domain.PurchaseOrderLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.ProductID == uuid.Nil {
			return nil, apperrors.NewBadRequest("Each line requires a productId")
		}
		if l.Quantity <= 0 || l.UnitPrice <= 0 {
			return nil, apperrors.NewBadRequest("Each line requires a positive quantity and unit price")
		}
		subtotal := l.Quantity * l.UnitPrice
		total += subtotal
		lines[i] = domain.PurchaseOrderLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice, Subtotal: subtotal}
	}

	var inheritedSalesOrderID *uuid.UUID = dto.SalesOrderID
	var inheritedSalesOrderNo string = dto.SalesOrderNo

	if dto.PurchaseRequestID != nil {
		pr, err := uc.repo.GetPurchaseRequestByID(ctx, *dto.PurchaseRequestID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get purchase request")
		}
		if pr == nil {
			return nil, apperrors.NewNotFound("Purchase request not found")
		}
		if pr.Status != "approved" {
			return nil, apperrors.NewBadRequest("Purchase order can only be created from an approved purchase request")
		}
		pr.Status = "converted"
		if err := uc.repo.UpdatePurchaseRequest(ctx, pr); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update purchase request")
		}
		if inheritedSalesOrderID == nil && pr.SalesOrderID != nil {
			inheritedSalesOrderID = pr.SalesOrderID
			inheritedSalesOrderNo = pr.SalesOrderNo
		}
	}

	orderDate := dto.OrderDate
	if orderDate == "" {
		orderDate = time.Now().Format("2006-01-02")
	}

	orderNo := fmt.Sprintf("PO-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)

	currency := dto.Currency
	baseAmount := total
	if uc.currencyUC != nil {
		result, err := uc.currencyUC.Convert(ctx, total, dto.Currency, orderDate)
		if err != nil {
			return nil, err
		}
		currency = result.OriginalCurrency
		baseAmount = result.BaseAmount
	}

	freightScheme := dto.FreightScheme
	if freightScheme == "" {
		freightScheme = "franco"
	}

	po := &domain.PurchaseOrder{
		OrderNo:           orderNo,
		SupplierID:        dto.SupplierID,
		SupplierName:      dto.SupplierName,
		PurchaseRequestID: dto.PurchaseRequestID,
		SalesOrderID:      inheritedSalesOrderID,
		SalesOrderNo:      inheritedSalesOrderNo,
		OrderDate:         orderDate,
		ExpectedDate:      dto.ExpectedDate,
		Status:            "draft",
		CreatedByEmail:    actor.EmailFrom(ctx),
		TotalAmount:       total,
		FreightScheme:     freightScheme,
		FreightCost:       dto.FreightCost,
		QualityNotes:      dto.QualityNotes,
		Currency:          currency,
		BaseAmount:        baseAmount,
		Lines:             lines,
	}
	if err := uc.repo.CreatePurchaseOrder(ctx, po); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create purchase order")
	}

	requiresApproval := false
	if uc.approvalUC != nil {
		result, err := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
			DocumentType:   purchaseOrderDocumentType,
			DocumentID:     po.ID,
			DocumentNumber: po.OrderNo,
			Amount:         po.TotalAmount,
			RequesterName:  po.CreatedByEmail,
		})
		if err == nil && result != nil && result.RequiresApproval {
			requiresApproval = true
			po.Status = "pending_approval"
			_ = uc.repo.UpdatePurchaseOrder(ctx, po)
		}
	}
	if !requiresApproval {
		po.Status = "approved"
		_ = uc.repo.UpdatePurchaseOrder(ctx, po)
	}

	return ToPurchaseOrderResponse(po), nil
}

func (uc *procurementUseCase) UpdatePurchaseOrder(ctx context.Context, id uuid.UUID, dto UpdatePurchaseOrderDTO) (*PurchaseOrderResponseDTO, error) {
	po, err := uc.repo.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order not found")
	}
	if po.Status != "draft" {
		return nil, apperrors.NewConflict("Only draft purchase orders can be edited")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line item is required")
	}
	var total float64
	lines := make([]domain.PurchaseOrderLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.ProductID == uuid.Nil || l.Quantity <= 0 || l.UnitPrice <= 0 {
			return nil, apperrors.NewBadRequest("Each line requires a productId, positive quantity and unit price")
		}
		subtotal := l.Quantity * l.UnitPrice
		total += subtotal
		lines[i] = domain.PurchaseOrderLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice, Subtotal: subtotal}
	}
	if dto.SupplierID != uuid.Nil {
		po.SupplierID = dto.SupplierID
	}
	if dto.SupplierName != "" {
		po.SupplierName = dto.SupplierName
	}
	if dto.PurchaseRequestID != nil {
		po.PurchaseRequestID = dto.PurchaseRequestID
	}
	if dto.SalesOrderID != nil {
		po.SalesOrderID = dto.SalesOrderID
		po.SalesOrderNo = dto.SalesOrderNo
	}
	if dto.ExpectedDate != "" {
		po.ExpectedDate = dto.ExpectedDate
	}
	if dto.FreightScheme != "" {
		po.FreightScheme = dto.FreightScheme
	}
	po.FreightCost = dto.FreightCost
	po.QualityNotes = dto.QualityNotes
	po.TotalAmount = total
	po.BaseAmount = total
	po.Lines = lines
	if err := uc.repo.UpdatePurchaseOrder(ctx, po); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update purchase order")
	}
	return ToPurchaseOrderResponse(po), nil
}

func (uc *procurementUseCase) SubmitPurchaseOrder(ctx context.Context, id uuid.UUID) (*PurchaseOrderResponseDTO, error) {
	po, err := uc.repo.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order not found")
	}
	if po.Status != "draft" {
		return nil, apperrors.NewConflict(fmt.Sprintf("Purchase order in status %s cannot be submitted", po.Status))
	}
	requiresApproval := false
	if uc.approvalUC != nil {
		result, err := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
			DocumentType:   purchaseOrderDocumentType,
			DocumentID:     po.ID,
			DocumentNumber: po.OrderNo,
			Amount:         po.TotalAmount,
			RequesterName:  po.CreatedByEmail,
		})
		if err != nil {
			return nil, err
		}
		if result != nil && result.RequiresApproval {
			requiresApproval = true
		}
	}
	if requiresApproval {
		po.Status = "pending_approval"
	} else {
		po.Status = "approved"
	}
	if err := uc.repo.UpdatePurchaseOrder(ctx, po); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update purchase order")
	}
	return ToPurchaseOrderResponse(po), nil
}

func (uc *procurementUseCase) GetPurchaseOrderByID(ctx context.Context, id uuid.UUID) (*PurchaseOrderResponseDTO, error) {
	po, err := uc.repo.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order not found")
	}
	return ToPurchaseOrderResponse(po), nil
}

func (uc *procurementUseCase) ListPurchaseOrders(ctx context.Context, query types.PaginationQuery) ([]PurchaseOrderResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListPurchaseOrders(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list purchase orders")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToPurchaseOrderResponseList(items), meta, nil
}

// ApprovePurchaseOrder is the module-specific approve endpoint. When the PO
// is routed through an approval workflow, it drives the workflow's current
// step via approvalUC.ApproveStep - the PO's own status field is NOT
// updated here directly. Instead, ApproveStep synchronously invokes the
// DocumentStatusCallback this module registers with the approval module
// (see module.go / callback.go), which calls FinalizePurchaseOrderApproved.
// That is the ONLY place the PO's status is set, so this endpoint and the
// generic Approval Center ("/approval/requests/:id/approve") can never
// double-fire the update: whichever one calls ApproveStep triggers the
// callback exactly once, and the PO is re-read afterward to reflect it.
func (uc *procurementUseCase) ApprovePurchaseOrder(ctx context.Context, id uuid.UUID, dto ApprovalActionDTO) (*PurchaseOrderResponseDTO, error) {
	po, err := uc.repo.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order not found")
	}
	if err := sod.ForbidSelfApproval(ctx, po.CreatedByEmail); err != nil {
		return nil, err
	}

	if uc.approvalUC != nil {
		activeReq, err := uc.approvalUC.GetActiveRequestForDocument(ctx, purchaseOrderDocumentType, po.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check approval status")
		}
		if activeReq == nil && (po.Status == "pending_approval" || po.Status == "submitted" || po.Status == "draft") {
			submitRes, submitErr := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
				DocumentType:   purchaseOrderDocumentType,
				DocumentID:     po.ID,
				DocumentNumber: po.OrderNo,
				Amount:         po.TotalAmount,
				RequesterName:  po.CreatedByEmail,
			})
			if submitErr == nil && submitRes != nil && submitRes.RequiresApproval && submitRes.Request != nil {
				activeReq = submitRes.Request
				po.Status = "pending_approval"
				_ = uc.repo.UpdatePurchaseOrder(ctx, po)
			}
		}
		if activeReq != nil && activeReq.Status == "pending" {
			if _, err := uc.approvalUC.ApproveStep(ctx, activeReq.ID, approvalApp.ActOnStepDTO{
				ActorName: dto.ActorName, ActorRole: dto.ActorRole, Comments: dto.Comments,
			}); err != nil {
				return nil, err
			}
			// Re-read: if that was the final approval level, the callback
			// above has already set po.Status = "approved". If it was an
			// intermediate level, the PO correctly remains pending_approval.
			po, err = uc.repo.GetPurchaseOrderByID(ctx, id)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to get purchase order")
			}
			return ToPurchaseOrderResponse(po), nil
		}
	}

	if po.Status != "draft" && po.Status != "submitted" && po.Status != "pending_approval" {
		return nil, apperrors.NewConflict("Only draft, submitted, or pending purchase orders can be approved")
	}
	po.Status = "approved"
	if err := uc.repo.UpdatePurchaseOrder(ctx, po); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to approve purchase order")
	}
	return ToPurchaseOrderResponse(po), nil
}

// RejectPurchaseOrder mirrors ApprovePurchaseOrder: it drives the workflow
// via approvalUC.RejectStep, which synchronously invokes
// FinalizePurchaseOrderRejected through the registered callback - the only
// place the PO's status is set to "rejected".
func (uc *procurementUseCase) RejectPurchaseOrder(ctx context.Context, id uuid.UUID, dto ApprovalActionDTO) (*PurchaseOrderResponseDTO, error) {
	po, err := uc.repo.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order not found")
	}

	if uc.approvalUC != nil {
		activeReq, err := uc.approvalUC.GetActiveRequestForDocument(ctx, purchaseOrderDocumentType, po.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check approval status")
		}
		if activeReq == nil && (po.Status == "pending_approval" || po.Status == "submitted" || po.Status == "draft") {
			submitRes, submitErr := uc.approvalUC.SubmitDocument(ctx, approvalApp.SubmitDocumentDTO{
				DocumentType:   purchaseOrderDocumentType,
				DocumentID:     po.ID,
				DocumentNumber: po.OrderNo,
				Amount:         po.TotalAmount,
				RequesterName:  po.CreatedByEmail,
			})
			if submitErr == nil && submitRes != nil && submitRes.RequiresApproval && submitRes.Request != nil {
				activeReq = submitRes.Request
				po.Status = "pending_approval"
				_ = uc.repo.UpdatePurchaseOrder(ctx, po)
			}
		}

		if activeReq != nil && activeReq.Status == "pending" {
			if _, err := uc.approvalUC.RejectStep(ctx, activeReq.ID, approvalApp.ActOnStepDTO{
				ActorName: dto.ActorName, ActorRole: dto.ActorRole, Comments: dto.Comments,
			}); err != nil {
				return nil, err
			}

			po, err = uc.repo.GetPurchaseOrderByID(ctx, id)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to get purchase order")
			}
			return ToPurchaseOrderResponse(po), nil
		}
	}

	if po.Status != "draft" && po.Status != "submitted" && po.Status != "pending_approval" {
		return nil, apperrors.NewConflict("This purchase order cannot be rejected")
	}
	po.Status = "rejected"
	if err := uc.repo.UpdatePurchaseOrder(ctx, po); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to reject purchase order")
	}
	return ToPurchaseOrderResponse(po), nil
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	return ToPurchaseOrderResponse(po), nil
}

// FinalizePurchaseOrderApproved sets the PO's status to "approved". It is
// the single place this happens, called either synchronously from within
// ApprovePurchaseOrder's call to approvalUC.ApproveStep (via the registered
// callback) or from the generic Approval Center's ApproveStep call on the
// same request. The pending_approval guard makes it idempotent if it were
// ever invoked more than once for the same request.
func (uc *procurementUseCase) FinalizePurchaseOrderApproved(ctx context.Context, id uuid.UUID) error {
	po, err := uc.repo.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return apperrors.NewNotFound("Purchase order not found")
	}
	if po.Status != "pending_approval" {
		return nil
	}
	po.Status = "approved"
	if err := uc.repo.UpdatePurchaseOrder(ctx, po); err != nil {
		return apperrors.NewInternal(err, "Failed to approve purchase order")
	}
	return nil
}

// FinalizePurchaseOrderRejected mirrors FinalizePurchaseOrderApproved for
// the rejected path.
func (uc *procurementUseCase) FinalizePurchaseOrderRejected(ctx context.Context, id uuid.UUID) error {
	po, err := uc.repo.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return apperrors.NewNotFound("Purchase order not found")
	}
	if po.Status != "pending_approval" {
		return nil
	}
	po.Status = "rejected"
	if err := uc.repo.UpdatePurchaseOrder(ctx, po); err != nil {
		return apperrors.NewInternal(err, "Failed to update purchase order")
	}
	return nil
}

func (uc *procurementUseCase) GetLastPOPrice(ctx context.Context, supplierID, productID uuid.UUID) (float64, error) {
	return uc.repo.GetLastPOPrice(ctx, supplierID, productID)
}

func (uc *procurementUseCase) CancelPurchaseOrder(ctx context.Context, id uuid.UUID) (*PurchaseOrderResponseDTO, error) {
	po, err := uc.repo.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order not found")
	}
	if po.Status == "completed" || po.Status == "cancelled" {
		return nil, apperrors.NewConflict("Purchase order cannot be cancelled in its current status")
	}
	po.Status = "cancelled"
	if err := uc.repo.UpdatePurchaseOrder(ctx, po); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to cancel purchase order")
	}
	return ToPurchaseOrderResponse(po), nil
}

// Goods receipts

func (uc *procurementUseCase) CreateGoodsReceipt(ctx context.Context, dto CreateGoodsReceiptDTO) (*GoodsReceiptResponseDTO, error) {
	if dto.PurchaseOrderID == uuid.Nil {
		return nil, apperrors.NewBadRequest("purchaseOrderId is required")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line item is required")
	}

	po, err := uc.repo.GetPurchaseOrderByID(ctx, dto.PurchaseOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order not found")
	}
	if po.Status == "cancelled" || po.Status == "draft" {
		return nil, apperrors.NewBadRequest("Purchase order must be approved before receiving goods")
	}

	if dto.WarehouseID == nil {
		return nil, apperrors.NewBadRequest("A destination warehouse is required to receive goods")
	}

	poLineByProduct := make(map[uuid.UUID]*domain.PurchaseOrderLine, len(po.Lines))
	for i := range po.Lines {
		poLineByProduct[po.Lines[i].ProductID] = &po.Lines[i]
	}

	lines := make([]domain.GoodsReceiptLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.ProductID == uuid.Nil || l.QuantityReceived <= 0 {
			return nil, apperrors.NewBadRequest("Each line requires a productId and a positive quantity received")
		}
		poLine, ok := poLineByProduct[l.ProductID]
		if !ok {
			return nil, apperrors.NewBadRequest("Product is not part of the referenced purchase order")
		}
		remaining := poLine.Quantity - poLine.QuantityReceived
		if l.QuantityReceived > remaining {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Received quantity exceeds remaining ordered quantity (%.2f) for product", remaining))
		}
		net := l.QuantityReceived
		if l.WeightNet != nil && *l.WeightNet > 0 {
			net = *l.WeightNet
		} else if l.WeightGross != nil && l.WeightTare != nil {
			calcNet := *l.WeightGross - *l.WeightTare
			net = calcNet
			l.WeightNet = &calcNet
		}
		var diffPO *float64
		if poLine.Quantity > 0 {
			diff := net - poLine.Quantity
			diffPO = &diff
		}
		lines[i] = domain.GoodsReceiptLine{
			ProductID:          l.ProductID,
			QuantityReceived:   net,
			QuantityOrderedRef: poLine.Quantity,
			WeightGross:        l.WeightGross,
			WeightTare:         l.WeightTare,
			WeightNet:          l.WeightNet,
			WeightDiffPO:       diffPO,
			BagWeights:         l.BagWeights,
		}
		poLine.QuantityReceived += net
	}

	receivedDate := dto.ReceivedDate
	if receivedDate == "" {
		receivedDate = time.Now().Format("2006-01-02")
	}

	gr := &domain.GoodsReceipt{
		ReceiptNo:       fmt.Sprintf("GR-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000),
		DocumentCode:    "GD-007-LBM",
		PurchaseOrderID: dto.PurchaseOrderID,
		ReceivedDate:    receivedDate,
		WarehouseID:     dto.WarehouseID,
		ReceivedBy:      dto.ReceivedBy,
		Status:          "completed",
		DeliveryNoteNo:   dto.DeliveryNoteNo,
		SugarCondition:   dto.SugarCondition,
		PackageCondition: dto.PackageCondition,
		PlasticTare:      dto.PlasticTare,
		Notes:            dto.Notes,
		Lines:           lines,
	}
	if err := uc.repo.CreateGoodsReceipt(ctx, gr); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create goods receipt")
	}

	// Increase on-hand stock for each received line now that the receipt
	// document exists, mirroring how Sales Order deducts stock via the same
	// inventory use case (see modules/sales/application/usecase.go).
	// WarehouseID is validated as required above, so this always runs.
	if uc.inventoryUC != nil && gr.WarehouseID != nil {
		for _, l := range gr.Lines {
			if _, err := uc.inventoryUC.AdjustStock(ctx, inventoryApp.AdjustStockDTO{
				ProductID: l.ProductID, WarehouseID: *gr.WarehouseID, Quantity: int(l.QuantityReceived),
				Reason: "Goods Receipt", Reference: gr.ReceiptNo, CreatedBy: gr.ReceivedBy,
			}); err != nil {
				return nil, apperrors.NewInternal(err, "Goods receipt created but failed to increase stock for product "+l.ProductID.String())
			}
		}
	}

	allReceived := true
	anyReceived := false
	for _, l := range po.Lines {
		if l.QuantityReceived > 0 {
			anyReceived = true
		}
		if l.QuantityReceived < l.Quantity {
			allReceived = false
		}
	}
	if allReceived {
		po.Status = "completed"
	} else if anyReceived {
		po.Status = "partially_received"
	}
	if err := uc.repo.UpdatePurchaseOrder(ctx, po); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update purchase order status")
	}

	return ToGoodsReceiptResponse(gr), nil
}

func (uc *procurementUseCase) GetGoodsReceiptByID(ctx context.Context, id uuid.UUID) (*GoodsReceiptResponseDTO, error) {
	gr, err := uc.repo.GetGoodsReceiptByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get goods receipt")
	}
	if gr == nil {
		return nil, apperrors.NewNotFound("Goods receipt not found")
	}
	return ToGoodsReceiptResponse(gr), nil
}

func (uc *procurementUseCase) ListGoodsReceipts(ctx context.Context, query types.PaginationQuery) ([]GoodsReceiptResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListGoodsReceipts(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list goods receipts")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToGoodsReceiptResponseList(items), meta, nil
}

// Purchase invoices

func (uc *procurementUseCase) CreatePurchaseInvoice(ctx context.Context, dto CreatePurchaseInvoiceDTO) (*PurchaseInvoiceResponseDTO, error) {
	if dto.SupplierID == uuid.Nil || dto.TotalAmount <= 0 {
		return nil, apperrors.NewBadRequest("supplierId and a positive totalAmount are required")
	}
	amounts, err := financeApp.ComputeInvoiceAmounts(financeApp.InvoiceAdjustmentInput{
		Subtotal: dto.TotalAmount, DiscountPercent: dto.DiscountPercent, DiscountAmount: dto.DiscountAmount,
		AdditionalCost: dto.AdditionalCost, RoundTo: dto.RoundTo,
		VAT: financeApp.VATInput{Apply: dto.ApplyVAT, Rate: dto.VATRate, OtherValueBase: dto.VATOtherValueBase == nil || *dto.VATOtherValueBase},
	})
	if err != nil {
		return nil, err
	}
	invoiceNumber := dto.InvoiceNumber
	if invoiceNumber == "" {
		invoiceNumber = fmt.Sprintf("PINV-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)
	}
	invoiceDate := dto.InvoiceDate
	if invoiceDate == "" {
		invoiceDate = time.Now().Format("2006-01-02")
	}
	inv := &domain.PurchaseInvoice{
		SupplierID:        dto.SupplierID,
		SupplierName:      dto.SupplierName,
		PurchaseOrderID:   dto.PurchaseOrderID,
		InvoiceNumber:     invoiceNumber,
		InvoiceDate:       invoiceDate,
		DueDate:           dto.DueDate,
		TotalAmount:       amounts.Total,
		Subtotal:          amounts.Subtotal,
		DiscountAmount:    amounts.Discount,
		AdditionalCost:    amounts.AdditionalCost,
		RoundingAmount:    amounts.Rounding,
		TaxBase:           amounts.VAT.TaxBase,
		DPPOtherValue:     amounts.VAT.DPPOtherValue,
		VATRate:           amounts.VAT.Rate,
		VATOtherValueBase: amounts.VAT.OtherValueBase,
		VATAmount:         amounts.VAT.Amount,
		VATCreditable:     amounts.VAT.Amount > 0 && (dto.VATCreditable == nil || *dto.VATCreditable),
		Status:            "unpaid",
	}
	if err := uc.repo.CreatePurchaseInvoice(ctx, inv); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create purchase invoice")
	}

	e := PurchaseInvoiceLedgerEntry(inv)
	uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
	uc.syncPayable(ctx, inv)
	return ToPurchaseInvoiceResponse(inv, uc.poNumberFor(ctx, inv.PurchaseOrderID)), nil
}

func (uc *procurementUseCase) poNumberFor(ctx context.Context, poID *uuid.UUID) string {
	if poID == nil {
		return ""
	}
	po, err := uc.repo.GetPurchaseOrderByID(ctx, *poID)
	if err != nil || po == nil {
		return ""
	}
	return po.OrderNo
}

func (uc *procurementUseCase) GetPurchaseInvoiceByID(ctx context.Context, id uuid.UUID) (*PurchaseInvoiceResponseDTO, error) {
	inv, err := uc.repo.GetPurchaseInvoiceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase invoice")
	}
	if inv == nil {
		return nil, apperrors.NewNotFound("Purchase invoice not found")
	}
	return ToPurchaseInvoiceResponse(inv, uc.poNumberFor(ctx, inv.PurchaseOrderID)), nil
}

func (uc *procurementUseCase) ListPurchaseInvoices(ctx context.Context, query types.PaginationQuery) ([]PurchaseInvoiceResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListPurchaseInvoices(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list purchase invoices")
	}
	poNumbers := make(map[uuid.UUID]string)
	for _, inv := range items {
		if inv.PurchaseOrderID != nil {
			if _, ok := poNumbers[*inv.PurchaseOrderID]; !ok {
				poNumbers[*inv.PurchaseOrderID] = uc.poNumberFor(ctx, inv.PurchaseOrderID)
			}
		}
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToPurchaseInvoiceResponseList(items, poNumbers), meta, nil
}

func (uc *procurementUseCase) RecordInvoicePayment(ctx context.Context, id uuid.UUID, dto RecordInvoicePaymentDTO) (*PurchaseInvoiceResponseDTO, error) {
	if dto.Amount <= 0 {
		return nil, apperrors.NewBadRequest("Payment amount must be positive")
	}
	inv, err := uc.repo.GetPurchaseInvoiceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase invoice")
	}
	if inv == nil {
		return nil, apperrors.NewNotFound("Purchase invoice not found")
	}
	if inv.PaidAmount+dto.Amount > inv.TotalAmount-inv.CreditedAmount {
		return nil, apperrors.NewBadRequest("Payment amount would exceed the amount still owed on the invoice")
	}
	inv.PaidAmount += dto.Amount
	inv.Status = purchaseInvoiceStatus(inv.TotalAmount-inv.CreditedAmount, inv.PaidAmount, inv.DueDate)
	if err := uc.repo.UpdatePurchaseInvoice(ctx, inv); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record payment")
	}

	e := PurchaseInvoicePaymentLedgerEntry(inv, dto.Amount)
	uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
	uc.syncPayable(ctx, inv)
	return ToPurchaseInvoiceResponse(inv, uc.poNumberFor(ctx, inv.PurchaseOrderID)), nil
}

// syncPayable mirrors the invoice into finance's AP sub-ledger. Like
// postLedger, a failure is logged rather than failing the invoice.
func (uc *procurementUseCase) syncPayable(ctx context.Context, inv *domain.PurchaseInvoice) {
	if uc.ledger == nil {
		return
	}
	if err := uc.ledger.UpsertPayable(ctx, PurchaseInvoiceSubledgerDoc(inv)); err != nil {
		slog.Warn("procurement: failed to sync payable", "invoice", inv.ID, "error", err)
	}
}

// postLedger records an automatic journal entry. A ledger failure must not
// roll back the business document, so it is logged for later reconciliation.
func (uc *procurementUseCase) postLedger(ctx context.Context, sourceDoc, memo string, lines []financeApp.LedgerLine) {
	if uc.ledger == nil {
		return
	}
	if err := uc.ledger.PostEntry(ctx, sourceDoc, memo, lines); err != nil {
		slog.Warn("procurement: failed to post ledger entry", "sourceDoc", sourceDoc, "error", err)
	}
}

// purchaseInvoiceStatus derives unpaid/partial/paid/overdue from amounts and due date,
// mirroring the finance module's Payable status derivation (internal/modules/finance/application/usecase.go).
// Kept as a local copy so status derivation does not need a finance call.
func purchaseInvoiceStatus(total, paid float64, dueDate string) string {
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
	return "unpaid"
}

// Purchase down payments

func (uc *procurementUseCase) CreateDownPayment(ctx context.Context, dto CreateDownPaymentDTO) (*PurchaseDownPaymentResponseDTO, error) {
	if dto.PurchaseOrderID == uuid.Nil || dto.SupplierID == uuid.Nil || dto.Amount <= 0 {
		return nil, apperrors.NewBadRequest("purchaseOrderId, supplierId and a positive amount are required")
	}
	po, err := uc.repo.GetPurchaseOrderByID(ctx, dto.PurchaseOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase order")
	}
	if po == nil {
		return nil, apperrors.NewNotFound("Purchase order not found")
	}

	paymentDate := dto.PaymentDate
	if paymentDate == "" {
		paymentDate = time.Now().Format("2006-01-02")
	}

	dp := &domain.PurchaseDownPayment{
		DPNumber:        fmt.Sprintf("DP-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000),
		PurchaseOrderID: dto.PurchaseOrderID,
		SupplierID:      dto.SupplierID,
		SupplierName:    dto.SupplierName,
		Amount:          dto.Amount,
		PaymentDate:     paymentDate,
		PaymentMethod:   dto.PaymentMethod,
		Status:          "unapplied",
		Notes:           dto.Notes,
	}
	if err := uc.repo.CreateDownPayment(ctx, dp); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record down payment")
	}

	e := DownPaymentLedgerEntry(dp)
	uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
	return ToDownPaymentResponse(dp, po.OrderNo), nil
}

func (uc *procurementUseCase) ListDownPayments(ctx context.Context, query types.PaginationQuery) ([]PurchaseDownPaymentResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListDownPayments(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list down payments")
	}
	poNumbers := make(map[uuid.UUID]string)
	for _, dp := range items {
		if _, ok := poNumbers[dp.PurchaseOrderID]; !ok {
			poNumbers[dp.PurchaseOrderID] = uc.poNumberFor(ctx, &dp.PurchaseOrderID)
		}
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToDownPaymentResponseList(items, poNumbers), meta, nil
}

// Purchase returns

func (uc *procurementUseCase) GetPurchaseReturnByID(ctx context.Context, id uuid.UUID) (*PurchaseReturnResponseDTO, error) {
	ret, err := uc.repo.GetPurchaseReturnByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get purchase return")
	}
	if ret == nil {
		return nil, apperrors.NewNotFound("Purchase return not found")
	}
	return ToPurchaseReturnResponse(ret), nil
}

func (uc *procurementUseCase) ListPurchaseReturns(ctx context.Context, query types.PaginationQuery) ([]PurchaseReturnResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListPurchaseReturns(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list purchase returns")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToPurchaseReturnResponseList(items), meta, nil
}

// Invoice receipts

func (uc *procurementUseCase) CreateInvoiceReceipt(ctx context.Context, dto CreateInvoiceReceiptDTO) (*InvoiceReceiptResponseDTO, error) {
	if dto.SupplierID == uuid.Nil {
		return nil, apperrors.NewBadRequest("supplierId is required")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line item is required")
	}

	date := dto.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	lines := make([]domain.InvoiceReceiptLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.InvoiceNo == "" {
			return nil, apperrors.NewBadRequest("Each line requires an invoiceNo")
		}
		lines[i] = domain.InvoiceReceiptLine{
			PurchaseInvoiceID: l.PurchaseInvoiceID,
			InvoiceNo:         l.InvoiceNo,
			Amount:            l.Amount,
			Remarks:           l.Remarks,
		}
	}

	ir := &domain.InvoiceReceipt{
		ReceiptNo:    fmt.Sprintf("TTN-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000),
		Date:         date,
		SupplierID:   dto.SupplierID,
		SupplierName: dto.SupplierName,
		Notes:        dto.Notes,
		Status:       "completed",
		Lines:        lines,
	}

	if err := uc.repo.CreateInvoiceReceipt(ctx, ir); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create invoice receipt")
	}
	return ToInvoiceReceiptResponse(ir), nil
}

func (uc *procurementUseCase) GetInvoiceReceiptByID(ctx context.Context, id uuid.UUID) (*InvoiceReceiptResponseDTO, error) {
	ir, err := uc.repo.GetInvoiceReceiptByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get invoice receipt")
	}
	if ir == nil {
		return nil, apperrors.NewNotFound("Invoice receipt not found")
	}
	return ToInvoiceReceiptResponse(ir), nil
}

func (uc *procurementUseCase) ListInvoiceReceipts(ctx context.Context, query types.PaginationQuery) ([]InvoiceReceiptResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListInvoiceReceipts(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list invoice receipts")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToInvoiceReceiptResponseList(items), meta, nil
}

// SeedInitialData populates a couple of sample procurement records on first boot
func (uc *procurementUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountPurchaseRequests(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	pr := &domain.PurchaseRequest{
		RequestNo:    "PR-SEED-0001",
		RequestedBy:  "System Seed",
		Department:   "Operations",
		NeededByDate: time.Now().AddDate(0, 0, 14).Format("2006-01-02"),
		Status:       "draft",
	}
	_ = uc.repo.CreatePurchaseRequest(ctx, pr)
	return nil
}
