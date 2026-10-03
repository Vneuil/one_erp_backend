package http

import (
	"context"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	approvalApp "github.com/divinecoid/one-backend/internal/modules/approval/application"
	approvalInfra "github.com/divinecoid/one-backend/internal/modules/approval/infrastructure"
	currencyApp "github.com/divinecoid/one-backend/internal/modules/currency/application"
	currencyInfra "github.com/divinecoid/one-backend/internal/modules/currency/infrastructure"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	inventoryInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/procurement/application"
	"github.com/divinecoid/one-backend/internal/modules/procurement/infrastructure"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func parseID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid " + param + " format")
	}
	return id, nil
}

// resolve builds a use-case bound to the caller's own tenant database. The
// schema is migrated and seeded once, at tenant-provision time (see
// module.go's registration with foundation/tenant.RegisterSchema).
func (h *Handler) resolve(c *fiber.Ctx) (application.ProcurementUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewProcurementRepository(tenantDB)
	approvalRepo := approvalInfra.NewApprovalRepository(tenantDB)
	approvalUC := approvalApp.NewApprovalUseCase(approvalRepo)
	inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
	productRepo := productInfra.NewProductRepository(tenantDB)
	inventoryUC := inventoryApp.NewInventoryUseCase(inventoryRepo, productRepo)
	currencyRepo := currencyInfra.NewCurrencyRepository(tenantDB)
	currencyUC := currencyApp.NewCurrencyUseCase(currencyRepo)
	ledger := financeApp.NewLedgerPoster(financeInfra.NewFinanceRepository(tenantDB))
	return application.NewProcurementUseCase(repo, approvalUC, inventoryUC, currencyUC, application.WithLedger(ledger)), nil
}

// actorFromRequest builds an ApprovalActionDTO from the caller's JWT claims,
// so approve/reject handlers don't need the client to supply identity.
func actorFromRequest(c *fiber.Ctx, comments string) application.ApprovalActionDTO {
	claims := middleware.CurrentUser(c)
	dto := application.ApprovalActionDTO{Comments: comments}
	if claims != nil {
		dto.ActorName = claims.Email
		dto.ActorRole = claims.Role
	}
	return dto
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// Purchase requests

func (h *Handler) CreatePurchaseRequest(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePurchaseRequestDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	// RequestedBy is never trusted from the client - it's always the
	// authenticated caller, matching what "Requested By" shows in the UI.
	if claims := middleware.CurrentUser(c); claims != nil {
		dto.RequestedBy = claims.Email
	}
	pr, err := uc.CreatePurchaseRequest(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Purchase request created successfully", pr)
}

func (h *Handler) GetPurchaseRequestByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	pr, err := uc.GetPurchaseRequestByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase request retrieved successfully", pr)
}

func (h *Handler) ListPurchaseRequests(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListPurchaseRequests(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Purchase requests retrieved successfully", items, meta)
}

func (h *Handler) SubmitPurchaseRequest(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	pr, err := uc.SubmitPurchaseRequest(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase request submitted successfully", pr)
}

func (h *Handler) ApprovePurchaseRequest(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	pr, err := uc.ApprovePurchaseRequest(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase request approved successfully", pr)
}

func (h *Handler) RejectPurchaseRequest(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	pr, err := uc.RejectPurchaseRequest(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase request rejected successfully", pr)
}

// Purchase orders

func (h *Handler) CreatePurchaseOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePurchaseOrderDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	po, err := uc.CreatePurchaseOrder(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Purchase order created successfully", po)
}

func (h *Handler) GetPurchaseOrderByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	po, err := uc.GetPurchaseOrderByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase order retrieved successfully", po)
}

func (h *Handler) ListPurchaseOrders(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListPurchaseOrders(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Purchase orders retrieved successfully", items, meta)
}

func (h *Handler) ApprovePurchaseOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var body struct {
		Comments string `json:"comments"`
	}
	_ = c.BodyParser(&body)
	po, err := uc.ApprovePurchaseOrder(h.ctx(c), id, actorFromRequest(c, body.Comments))
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase order approved successfully", po)
}

func (h *Handler) RejectPurchaseOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var body struct {
		Comments string `json:"comments"`
	}
	_ = c.BodyParser(&body)
	po, err := uc.RejectPurchaseOrder(h.ctx(c), id, actorFromRequest(c, body.Comments))
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase order rejected successfully", po)
}

func (h *Handler) CancelPurchaseOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	po, err := uc.CancelPurchaseOrder(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase order cancelled successfully", po)
}

// Goods receipts

func (h *Handler) CreateGoodsReceipt(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateGoodsReceiptDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	// ReceivedBy is never trusted from the client - it's always the
	// authenticated caller, matching what "Received By" shows in the UI.
	if claims := middleware.CurrentUser(c); claims != nil {
		dto.ReceivedBy = claims.Email
	}
	gr, err := uc.CreateGoodsReceipt(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Goods receipt created successfully", gr)
}

func (h *Handler) GetGoodsReceiptByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	gr, err := uc.GetGoodsReceiptByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Goods receipt retrieved successfully", gr)
}

func (h *Handler) ListGoodsReceipts(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListGoodsReceipts(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Goods receipts retrieved successfully", items, meta)
}

// Purchase invoices

func (h *Handler) CreatePurchaseInvoice(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePurchaseInvoiceDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	inv, err := uc.CreatePurchaseInvoice(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Purchase invoice created successfully", inv)
}

func (h *Handler) GetPurchaseInvoiceByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	inv, err := uc.GetPurchaseInvoiceByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase invoice retrieved successfully", inv)
}

func (h *Handler) ListPurchaseInvoices(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListPurchaseInvoices(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Purchase invoices retrieved successfully", items, meta)
}

func (h *Handler) RecordInvoicePayment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.RecordInvoicePaymentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	inv, err := uc.RecordInvoicePayment(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Payment recorded successfully", inv)
}

// Purchase down payments

func (h *Handler) CreateDownPayment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateDownPaymentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	dp, err := uc.CreateDownPayment(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Down payment recorded successfully", dp)
}

func (h *Handler) ListDownPayments(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListDownPayments(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Down payments retrieved successfully", items, meta)
}

// Purchase returns

func (h *Handler) CreatePurchaseReturn(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePurchaseReturnDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	ret, err := uc.CreatePurchaseReturn(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Purchase return created successfully", ret)
}

func (h *Handler) GetPurchaseReturnByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	ret, err := uc.GetPurchaseReturnByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Purchase return retrieved successfully", ret)
}

func (h *Handler) ListPurchaseReturns(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListPurchaseReturns(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Purchase returns retrieved successfully", items, meta)
}
