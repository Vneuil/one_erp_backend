package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/loyalty/application"
	"github.com/divinecoid/one-backend/internal/modules/loyalty/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) resolve(c *fiber.Ctx) (application.LoyaltyUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewLoyaltyRepository(tenantDB)
	return application.NewLoyaltyUseCase(repo), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) GetConfig(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	cfg, err := uc.GetConfig(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Loyalty config retrieved successfully", cfg)
}

func (h *Handler) UpdateConfig(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.UpdateLoyaltyConfigDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	cfg, err := uc.UpdateConfig(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Loyalty config updated successfully", cfg)
}

func (h *Handler) LookupMember(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	phone := c.Query("phone")
	card, err := uc.LookupMemberByPhone(h.ctx(c), phone)
	if err != nil {
		return err
	}
	return response.OK(c, "Member card retrieved successfully", card)
}

func (h *Handler) EnrollMember(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.EnrollMemberDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	member, err := uc.EnrollMember(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Member enrolled successfully", member)
}

func (h *Handler) ListMembers(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	members, meta, err := uc.ListMembers(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Members retrieved successfully", members, meta)
}
