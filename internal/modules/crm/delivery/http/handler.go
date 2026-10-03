package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/crm/application"
	"github.com/divinecoid/one-backend/internal/modules/crm/infrastructure"
	crmplusapp "github.com/divinecoid/one-backend/internal/modules/crmplus/application"
	crmplusinfra "github.com/divinecoid/one-backend/internal/modules/crmplus/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.CRMUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewCRMRepository(tenantDB)
	// Deal stages come from the company's configured pipeline.
	stages := crmplusapp.NewService(crmplusinfra.NewRepository(tenantDB), nil, nil, nil)
	return application.NewCRMUseCase(repo, application.WithStages(stages)), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) Create(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateLeadDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	lead, err := uc.Create(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Lead created successfully", lead)
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid lead ID format")
	}

	lead, err := uc.GetByID(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Lead retrieved successfully", lead)
}

func (h *Handler) List(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	leads, meta, err := uc.List(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Leads retrieved successfully", leads, meta)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid lead ID format")
	}

	var dto application.UpdateLeadDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	lead, err := uc.Update(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Lead updated successfully", lead)
}

func (h *Handler) UpdateStatus(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid lead ID format")
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return apperrors.NewBadRequest("Status is required")
	}

	lead, err := uc.UpdateStatus(h.ctx(c), id, body.Status)
	if err != nil {
		return err
	}

	return response.OK(c, "Lead status updated successfully", lead)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid lead ID format")
	}

	if err := uc.Delete(h.ctx(c), id); err != nil {
		return err
	}

	return response.OK(c, "Lead deleted successfully", nil)
}

func (h *Handler) CreateDeal(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateDealDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	deal, err := uc.CreateDeal(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Deal created successfully", deal)
}

func (h *Handler) GetDealByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid deal ID format")
	}

	deal, err := uc.GetDealByID(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Deal retrieved successfully", deal)
}

func (h *Handler) ListDeals(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	deals, meta, err := uc.ListDeals(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Deals retrieved successfully", deals, meta)
}

func (h *Handler) UpdateDeal(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid deal ID format")
	}

	var dto application.UpdateDealDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	deal, err := uc.UpdateDeal(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Deal updated successfully", deal)
}

func (h *Handler) UpdateDealStage(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid deal ID format")
	}

	var dto application.UpdateDealStageDTO
	if err := c.BodyParser(&dto); err != nil || dto.Stage == "" {
		return apperrors.NewBadRequest("Stage is required")
	}

	deal, err := uc.UpdateDealStage(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Deal stage updated successfully", deal)
}

func (h *Handler) DeleteDeal(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid deal ID format")
	}

	if err := uc.DeleteDeal(h.ctx(c), id); err != nil {
		return err
	}

	return response.OK(c, "Deal deleted successfully", nil)
}
