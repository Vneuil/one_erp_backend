package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/commission/application"
	"github.com/divinecoid/one-backend/internal/modules/commission/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.CommissionUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewCommissionRepository(tenantDB)
	return application.NewCommissionUseCase(repo), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateRule(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateCommissionRuleDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	rule, err := uc.CreateRule(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Commission rule created successfully", rule)
}

func (h *Handler) GetRuleByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	rule, err := uc.GetRuleByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Commission rule retrieved successfully", rule)
}

func (h *Handler) ListRules(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListRules(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Commission rules retrieved successfully", items, meta)
}

func (h *Handler) CreateRecord(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateCommissionRecordDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	rec, err := uc.CalculateAndCreateRecord(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Commission record calculated and created successfully", rec)
}

func (h *Handler) GetRecordByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	rec, err := uc.GetRecordByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Commission record retrieved successfully", rec)
}

func (h *Handler) ListRecords(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListRecords(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Commission records retrieved successfully", items, meta)
}

func (h *Handler) ApproveRecord(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	rec, err := uc.ApproveRecord(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Commission record approved successfully", rec)
}

func (h *Handler) MarkRecordPaid(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	rec, err := uc.MarkRecordPaid(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Commission record marked as paid successfully", rec)
}

func (h *Handler) GetSummary(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	period := c.Query("period")
	summary, err := uc.GetSummary(h.ctx(c), period)
	if err != nil {
		return err
	}
	return response.OK(c, "Commission summary retrieved successfully", summary)
}
