package http

import (
	"context"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
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
//
// Inventory depends on Product's repository. Since both repositories are
// stateless wrappers around a *gorm.DB, the product repo is simply built
// fresh against the same tenant database here rather than injected once at
// startup - there is no cross-module singleton to break once product is
// also tenant-scoped.
func (h *Handler) resolve(c *fiber.Ctx) (application.InventoryUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewInventoryRepository(tenantDB)
	productRepo := productInfra.NewProductRepository(tenantDB)
	ledger := financeApp.NewLedgerPoster(financeInfra.NewFinanceRepository(tenantDB))
	return application.NewInventoryUseCase(repo, productRepo, application.WithLedger(ledger)), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// Warehouses

func (h *Handler) CreateWarehouse(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateWarehouseDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	w, err := uc.CreateWarehouse(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Warehouse created successfully", w)
}

func (h *Handler) ListWarehouses(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListWarehouses(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Warehouses retrieved successfully", items, meta)
}

func (h *Handler) UpdateWarehouse(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.CreateWarehouseDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	w, err := uc.UpdateWarehouse(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Warehouse updated successfully", w)
}

func (h *Handler) DeleteWarehouse(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteWarehouse(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Warehouse deleted successfully", nil)
}

// Stock levels

func (h *Handler) ListStockLevels(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListStockLevels(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Stock levels retrieved successfully", items, meta)
}

func (h *Handler) AdjustStock(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.AdjustStockDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	s, err := uc.AdjustStock(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Stock level adjusted successfully", s)
}

func (h *Handler) SetBufferStock(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.SetBufferStockDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	s, err := uc.SetBufferStock(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Buffer stock updated successfully", s)
}

// Movements

func (h *Handler) ListMovements(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListMovements(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Stock movements retrieved successfully", items, meta)
}

// MovementSummary reports received/issued/opening/closing per product for a period.
func (h *Handler) MovementSummary(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var wh *uuid.UUID
	if raw := c.Query("warehouseId"); raw != "" {
		id, perr := uuid.Parse(raw)
		if perr != nil {
			return apperrors.NewBadRequest("Invalid warehouseId")
		}
		wh = &id
	}
	res, err := uc.MovementSummary(h.ctx(c), c.Query("from"), c.Query("to"), wh)
	if err != nil {
		return err
	}
	return response.OK(c, "Movement summary retrieved", res)
}

// Transfers

func (h *Handler) CreateTransfer(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateTransferDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	t, err := uc.CreateTransfer(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Stock transfer requested successfully", t)
}

func (h *Handler) ListTransfers(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListTransfers(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Stock transfers retrieved successfully", items, meta)
}

func (h *Handler) CompleteTransfer(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	t, err := uc.CompleteTransfer(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Stock transfer completed successfully", t)
}

func (h *Handler) CancelTransfer(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	t, err := uc.CancelTransfer(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Stock transfer cancelled successfully", t)
}

// Stock opname

func (h *Handler) CreateOpname(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateOpnameDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	o, err := uc.CreateOpname(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Stock opname created successfully", o)
}

func (h *Handler) GetOpnameByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	o, err := uc.GetOpnameByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Stock opname retrieved successfully", o)
}

func (h *Handler) ListOpnames(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListOpnames(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Stock opnames retrieved successfully", items, meta)
}

func (h *Handler) CountOpnameLine(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.CountOpnameLineDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	o, err := uc.CountOpnameLine(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Stock opname line recorded successfully", o)
}

func (h *Handler) FinalizeOpname(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	o, err := uc.FinalizeOpname(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Stock opname finalized successfully", o)
}

// ---- batches

func (h *Handler) ReceiveBatch(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.ReceiveBatchDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	b, err := uc.ReceiveBatch(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Batch received", b)
}

func optionalID(c *fiber.Ctx, name string) (*uuid.UUID, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperrors.NewBadRequest("Invalid " + name)
	}
	return &id, nil
}

func (h *Handler) ListBatches(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	pid, err := optionalID(c, "productId")
	if err != nil {
		return err
	}
	wid, err := optionalID(c, "warehouseId")
	if err != nil {
		return err
	}
	out, err := uc.ListBatches(h.ctx(c), pid, wid, c.Query("all") == "true")
	if err != nil {
		return err
	}
	return response.OK(c, "Batches retrieved", out)
}

func (h *Handler) ExpiringBatches(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	days := c.QueryInt("days", 30)
	out, err := uc.ExpiringBatches(h.ctx(c), days)
	if err != nil {
		return err
	}
	return response.OK(c, "Expiring batches retrieved", out)
}

func (h *Handler) WriteOffBatch(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid batch id")
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	b, err := uc.WriteOffBatch(h.ctx(c), id, in.Reason)
	if err != nil {
		return err
	}
	return response.OK(c, "Batch written off", b)
}

// Stock documents

func (h *Handler) CreateStockDocument(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateStockDocumentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	d, err := uc.CreateStockDocument(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Stock document posted successfully", d)
}

func (h *Handler) GetStockDocument(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	d, err := uc.GetStockDocument(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Stock document retrieved successfully", d)
}

func (h *Handler) ListStockDocuments(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	items, err := uc.ListStockDocuments(h.ctx(c), c.Query("type"), c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	return response.OK(c, "Stock documents retrieved successfully", items)
}

func (h *Handler) ImportOpnameCounts(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var body struct {
		Rows []application.OpnameCountRow `json:"rows"`
	}
	if err := c.BodyParser(&body); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := uc.ImportOpnameCounts(h.ctx(c), id, body.Rows)
	if err != nil {
		return err
	}
	return response.OK(c, "Opname counts imported", res)
}
