package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	inventoryInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/application"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/infrastructure"
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
// Manufacturing depends on Inventory's and Product's repositories. Since all
// three repositories are stateless wrappers around a *gorm.DB, they are
// simply built fresh against the same tenant database here rather than
// injected once at startup.
func (h *Handler) resolve(c *fiber.Ctx) (application.ManufacturingUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewManufacturingRepository(tenantDB)
	inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
	productRepo := productInfra.NewProductRepository(tenantDB)
	return application.NewManufacturingUseCase(repo, inventoryRepo, productRepo), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// BOM

func (h *Handler) CreateBOM(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateBOMDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	b, err := uc.CreateBOM(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "BOM created successfully", b)
}

func (h *Handler) GetBOMByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	b, err := uc.GetBOMByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "BOM retrieved successfully", b)
}

func (h *Handler) ListBOMs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListBOMs(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "BOMs retrieved successfully", items, meta)
}

// Production orders

func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateProductionOrderDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	o, err := uc.CreateOrder(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Production order created successfully", o)
}

func (h *Handler) GetOrderByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	o, err := uc.GetOrderByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Production order retrieved successfully", o)
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
	items, meta, err := uc.ListOrders(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Production orders retrieved successfully", items, meta)
}

func (h *Handler) ReleaseOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	o, err := uc.ReleaseOrder(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Production order released successfully", o)
}

func (h *Handler) PauseOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	o, err := uc.PauseOrder(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Production order paused successfully", o)
}

func (h *Handler) ResumeOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	o, err := uc.ResumeOrder(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Production order resumed successfully", o)
}

func (h *Handler) CancelOrder(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	o, err := uc.CancelOrder(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Production order cancelled successfully", o)
}

func (h *Handler) CompleteBatch(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.CompleteBatchDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	o, err := uc.CompleteBatch(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Batch completion logged successfully", o)
}

func (h *Handler) GetDashboardSummary(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	summary, err := uc.GetDashboardSummary(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Manufacturing dashboard summary retrieved successfully", summary)
}
