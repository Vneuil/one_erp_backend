package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	inventoryInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/warehouse/application"
	"github.com/divinecoid/one-backend/internal/modules/warehouse/infrastructure"
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
// Warehouse depends on Inventory's and Product's repositories. Since all
// three repositories are stateless wrappers around a *gorm.DB, they are
// simply built fresh against the same tenant database here rather than
// injected once at startup.
func (h *Handler) resolve(c *fiber.Ctx) (application.WarehouseUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewWarehouseRepository(tenantDB)
	inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
	productRepo := productInfra.NewProductRepository(tenantDB)
	return application.NewWarehouseUseCase(repo, inventoryRepo, productRepo), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// Pick waves

func (h *Handler) CreatePickWave(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePickWaveDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	w, err := uc.CreatePickWave(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Pick wave created successfully", w)
}

func (h *Handler) ListPickWaves(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListPickWaves(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Pick waves retrieved successfully", items, meta)
}

func (h *Handler) GetPickWaveByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	w, err := uc.GetPickWaveByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Pick wave retrieved successfully", w)
}

func (h *Handler) RecordPick(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.RecordPickDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	w, err := uc.RecordPick(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Pick recorded successfully", w)
}

func (h *Handler) CompletePickWave(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	w, err := uc.CompletePickWave(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Pick wave completed successfully", w)
}

func (h *Handler) CancelPickWave(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	w, err := uc.CancelPickWave(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Pick wave cancelled successfully", w)
}

// Packing sessions

func (h *Handler) CreatePackingSession(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePackingSessionDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	s, err := uc.CreatePackingSession(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Packing session started successfully", s)
}

func (h *Handler) ListPackingSessions(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListPackingSessions(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Packing sessions retrieved successfully", items, meta)
}

func (h *Handler) CompletePackingSession(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.CompletePackingSessionDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	s, err := uc.CompletePackingSession(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Packing session completed successfully", s)
}

// Shipments

func (h *Handler) CreateShipment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateShipmentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	s, err := uc.CreateShipment(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Shipment booked successfully", s)
}

func (h *Handler) ListShipments(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListShipments(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Shipments retrieved successfully", items, meta)
}

func (h *Handler) DispatchShipment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	s, err := uc.DispatchShipment(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Shipment dispatched successfully", s)
}
