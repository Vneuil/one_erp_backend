package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/goal/application"
	"github.com/divinecoid/one-backend/internal/modules/goal/infrastructure"
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

// resolve builds a use-case bound to the caller's own tenant database,
// resolved per-request from the JWT-derived tenant context. The goal schema
// itself is migrated and seeded once, at tenant-provision time (see
// module.go's registration with foundation/tenant.RegisterSchema) - not
// here, so this stays a cheap per-request lookup.
func (h *Handler) resolve(c *fiber.Ctx) (application.GoalsUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}

	repo := infrastructure.NewGoalsRepository(tenantDB)
	return application.NewGoalsUseCase(repo), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateGoal(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateGoalDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	goal, err := uc.CreateGoal(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Goal created successfully", goal)
}

func (h *Handler) GetGoalByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	goal, err := uc.GetGoalByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Goal retrieved successfully", goal)
}

func (h *Handler) ListGoals(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListGoals(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Goals retrieved successfully", items, meta)
}

func (h *Handler) UpdateGoal(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateGoalDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	goal, err := uc.UpdateGoal(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Goal updated successfully", goal)
}

func (h *Handler) CheckIn(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.CheckInDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	goal, err := uc.CheckIn(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Check-in recorded successfully", goal)
}

func (h *Handler) CompleteGoal(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	goal, err := uc.CompleteGoal(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Goal marked as completed", goal)
}

func (h *Handler) GetSummary(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	summary, err := uc.GetSummary(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Goal summary retrieved successfully", summary)
}
