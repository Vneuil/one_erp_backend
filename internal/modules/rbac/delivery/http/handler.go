package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/rbac/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	uc application.RBACUseCase
}

func NewHandler(uc application.RBACUseCase) *Handler {
	return &Handler{uc: uc}
}

func currentCompanyID(c *fiber.Ctx) (uuid.UUID, error) {
	claims := middleware.CurrentUser(c)
	if claims == nil || claims.CompanyID == nil {
		return uuid.Nil, apperrors.NewBadRequest("No active company")
	}
	return *claims.CompanyID, nil
}

func (h *Handler) ListRoles(c *fiber.Ctx) error {
	companyID, err := currentCompanyID(c)
	if err != nil {
		return err
	}
	roles, err := h.uc.ListRoles(c.UserContext(), companyID)
	if err != nil {
		return err
	}
	return response.OK(c, "Roles retrieved successfully", roles)
}

func (h *Handler) CreateRole(c *fiber.Ctx) error {
	companyID, err := currentCompanyID(c)
	if err != nil {
		return err
	}
	var dto application.CreateRoleDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	role, err := h.uc.CreateRole(c.UserContext(), companyID, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Role created successfully", role)
}

func (h *Handler) DeleteRole(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid role ID format")
	}
	if err := h.uc.DeleteRole(c.UserContext(), id); err != nil {
		return err
	}
	return response.OK(c, "Role deleted successfully", nil)
}

func (h *Handler) GetRolePermissions(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid role ID format")
	}
	result, err := h.uc.GetRoleWithPermissions(c.UserContext(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Role permissions retrieved successfully", result)
}

func (h *Handler) SetRolePermissions(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid role ID format")
	}
	var dto application.SetPermissionsDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	perms, err := h.uc.SetPermissions(c.UserContext(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Permissions saved successfully", perms)
}

func (h *Handler) ListModules(c *fiber.Ctx) error {
	return response.OK(c, "Modules retrieved successfully", h.uc.AvailableModules())
}
