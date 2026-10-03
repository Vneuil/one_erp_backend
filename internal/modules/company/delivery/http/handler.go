package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/company/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	uc application.CompanyUseCase
}

func NewHandler(uc application.CompanyUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var dto application.CreateCompanyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	company, err := h.uc.Create(c.UserContext(), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Company created successfully", company)
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid company ID format")
	}

	company, err := h.uc.GetByID(c.UserContext(), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Company retrieved successfully", company)
}

func (h *Handler) List(c *fiber.Ctx) error {
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	companies, meta, err := h.uc.List(c.UserContext(), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Companies retrieved successfully", companies, meta)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid company ID format")
	}

	var dto application.UpdateCompanyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	company, err := h.uc.Update(c.UserContext(), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Company updated successfully", company)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid company ID format")
	}

	if err := h.uc.Delete(c.UserContext(), id); err != nil {
		return err
	}

	return response.OK(c, "Company deleted successfully", nil)
}
