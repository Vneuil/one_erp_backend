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
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/sales/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// resolve builds a use-case bound to the caller's own tenant database. The
// schema is migrated and seeded once, at tenant-provision time (see
// module.go's registration with foundation/tenant.RegisterSchema).
func (h *Handler) resolve(c *fiber.Ctx) (application.SalesUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewSalesRepository(tenantDB)
	currencyRepo := currencyInfra.NewCurrencyRepository(tenantDB)
	currencyUC := currencyApp.NewCurrencyUseCase(currencyRepo)
	inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
	productRepo := productInfra.NewProductRepository(tenantDB)
	inventoryUC := inventoryApp.NewInventoryUseCase(inventoryRepo, productRepo)
	approvalRepo := approvalInfra.NewApprovalRepository(tenantDB)
	approvalUC := approvalApp.NewApprovalUseCase(approvalRepo)
	ledger := financeApp.NewLedgerPoster(financeInfra.NewFinanceRepository(tenantDB))
	return application.NewSalesUseCase(repo, currencyUC, inventoryUC, approvalUC, application.WithLedger(ledger), application.WithProducts(productRepo)), nil
}

// actorFromRequest builds a SalesApprovalActionDTO from the caller's JWT
// claims, so approve/reject handlers don't need the client to supply
// identity - mirrors procurement's actorFromRequest.
func actorFromRequest(c *fiber.Ctx, comments string) application.SalesApprovalActionDTO {
	claims := middleware.CurrentUser(c)
	dto := application.SalesApprovalActionDTO{Comments: comments}
	if claims != nil {
		dto.ActorName = claims.Email
		dto.ActorRole = claims.Role
	}
	return dto
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateSalesOrderDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	order, err := uc.CreateOrder(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Sales order created successfully", order)
}

func (h *Handler) GetOrderByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid order ID format")
	}

	order, err := uc.GetOrderByID(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Sales order retrieved successfully", order)
}

func (h *Handler) ListOrders(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	orders, meta, err := uc.ListOrders(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Sales orders retrieved successfully", orders, meta)
}

func (h *Handler) UpdateOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid order ID format")
	}

	var dto application.UpdateSalesOrderDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	order, err := uc.UpdateOrder(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Sales order updated successfully", order)
}

func (h *Handler) ApproveOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid order ID format")
	}
	var body struct {
		Comments string `json:"comments"`
	}
	_ = c.BodyParser(&body)

	order, err := uc.ApproveOrder(h.ctx(c), id, actorFromRequest(c, body.Comments))
	if err != nil {
		return err
	}

	return response.OK(c, "Sales order approved successfully", order)
}

func (h *Handler) RejectOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid order ID format")
	}
	var body struct {
		Comments string `json:"comments"`
	}
	_ = c.BodyParser(&body)

	order, err := uc.RejectOrder(h.ctx(c), id, actorFromRequest(c, body.Comments))
	if err != nil {
		return err
	}

	return response.OK(c, "Sales order rejected successfully", order)
}

func (h *Handler) CreateQuotation(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateQuotationDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	quo, err := uc.CreateQuotation(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Quotation created successfully", quo)
}

func (h *Handler) ListQuotations(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	quotations, meta, err := uc.ListQuotations(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Quotations retrieved successfully", quotations, meta)
}

func (h *Handler) GetQuotationByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid quotation ID")
	}

	quo, err := uc.GetQuotationByID(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Quotation retrieved successfully", quo)
}

func (h *Handler) UpdateQuotation(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid quotation ID")
	}

	var dto application.UpdateQuotationDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	quo, err := uc.UpdateQuotation(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Quotation updated successfully", quo)
}

func (h *Handler) ConvertQuotationToOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid quotation ID format")
	}

	order, err := uc.ConvertQuotationToOrder(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.Created(c, "Quotation converted to sales order successfully", order)
}

func (h *Handler) CreateInvoice(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateInvoiceDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	inv, err := uc.CreateInvoice(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Invoice created successfully", inv)
}

func (h *Handler) ListInvoices(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	invoices, meta, err := uc.ListInvoices(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Invoices retrieved successfully", invoices, meta)
}

func (h *Handler) RecordInvoicePayment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid invoice ID format")
	}

	var dto application.RecordSalesPaymentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	inv, err := uc.RecordInvoicePayment(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Payment recorded successfully", inv)
}

func (h *Handler) CreateDelivery(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateDeliveryDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	d, err := uc.CreateDelivery(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Delivery created successfully", d)
}

func (h *Handler) ListDeliveries(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	deliveries, meta, err := uc.ListDeliveries(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Deliveries retrieved successfully", deliveries, meta)
}

// Sales down payments

func (h *Handler) CreateDownPayment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateSalesDownPaymentDTO
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

func (h *Handler) ApplyDownPayment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid down payment ID format")
	}

	var dto application.ApplySalesDownPaymentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	dp, err := uc.ApplyDownPayment(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Down payment applied to invoice successfully", dp)
}

// Sales returns

func (h *Handler) CreateSalesReturn(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateSalesReturnDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	ret, err := uc.CreateSalesReturn(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Sales return created successfully", ret)
}

func (h *Handler) GetSalesReturnByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid return ID format")
	}
	ret, err := uc.GetSalesReturnByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Sales return retrieved successfully", ret)
}

func (h *Handler) ListSalesReturns(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListSalesReturns(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Sales returns retrieved successfully", items, meta)
}

// ---- billing schedule (termin)

func (h *Handler) SetBillingSchedule(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		Terms []application.BillingTermInput `json:"terms"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	terms, err := uc.SetBillingSchedule(h.ctx(c), id, in.Terms)
	if err != nil {
		return err
	}
	return response.OK(c, "Billing schedule saved", terms)
}

func (h *Handler) ListBillingSchedule(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	terms, err := uc.ListBillingSchedule(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Billing schedule retrieved", terms)
}

func (h *Handler) InvoiceBillingTerm(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	inv, err := uc.InvoiceBillingTerm(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.Created(c, "Sub-invoice issued", inv)
}

func parseID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid " + param + " format")
	}
	return id, nil
}

func (h *Handler) RecordShippingCost(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var in application.ShippingCostDTO
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	d, err := uc.RecordShippingCost(h.ctx(c), id, in)
	if err != nil {
		return err
	}
	return response.OK(c, "Shipping details saved", d)
}
