package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/user/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	uc application.UserUseCase
}

func NewHandler(uc application.UserUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var dto application.CreateUserDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	// If caller is authenticated, default companyId to caller's companyId if not set
	claims := middleware.CurrentUser(c)
	if claims != nil && dto.CompanyID == nil {
		dto.CompanyID = claims.CompanyID
	}

	user, err := h.uc.Create(c.UserContext(), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "User created successfully", user)
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid user ID format")
	}

	user, err := h.uc.GetByID(c.UserContext(), id)
	if err != nil {
		return err
	}

	return response.OK(c, "User retrieved successfully", user)
}

func (h *Handler) List(c *fiber.Ctx) error {
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	var companyID *uuid.UUID
	claims := middleware.CurrentUser(c)
	if claims != nil {
		companyID = claims.CompanyID
	}

	users, meta, err := h.uc.List(c.UserContext(), query, companyID)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Users retrieved successfully", users, meta)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid user ID format")
	}

	var dto application.UpdateUserDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	user, err := h.uc.Update(c.UserContext(), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "User updated successfully", user)
}

func (h *Handler) AssignRole(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid user ID format")
	}

	var dto application.AssignRoleDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	user, err := h.uc.AssignRole(c.UserContext(), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Role assigned successfully", user)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid user ID format")
	}

	if err := h.uc.Delete(c.UserContext(), id); err != nil {
		return err
	}

	return response.OK(c, "User deleted successfully", nil)
}
