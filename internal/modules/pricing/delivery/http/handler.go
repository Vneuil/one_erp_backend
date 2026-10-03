package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/pricing/application"
	"github.com/divinecoid/one-backend/internal/modules/pricing/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) resolve(c *fiber.Ctx) (application.PricingUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewPricingRepository(tenantDB)
	return application.NewPricingUseCase(repo), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// Payment terms

func (h *Handler) CreatePaymentTerm(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreatePaymentTermDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	p, err := uc.CreatePaymentTerm(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Payment term created successfully", p)
}

func (h *Handler) ListPaymentTerms(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	items, err := uc.ListPaymentTerms(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Payment terms retrieved successfully", items)
}

func (h *Handler) UpdatePaymentTerm(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid payment term ID format")
	}
	var dto application.UpdatePaymentTermDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	p, err := uc.UpdatePaymentTerm(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Payment term updated successfully", p)
}

func (h *Handler) DeletePaymentTerm(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid payment term ID format")
	}
	if err := uc.DeletePaymentTerm(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Payment term deleted successfully", nil)
}

// Customer types

func (h *Handler) CreateCustomerType(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateCustomerTypeDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.CreateCustomerType(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Customer type created successfully", item)
}

func (h *Handler) ListCustomerTypes(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	items, err := uc.ListCustomerTypes(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Customer types retrieved successfully", items)
}

func (h *Handler) UpdateCustomerType(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid customer type ID format")
	}
	var dto application.UpdateCustomerTypeDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.UpdateCustomerType(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Customer type updated successfully", item)
}

func (h *Handler) DeleteCustomerType(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid customer type ID format")
	}
	if err := uc.DeleteCustomerType(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Customer type deleted successfully", nil)
}

// Tax rates

func (h *Handler) CreateTaxRate(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateTaxRateDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.CreateTaxRate(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Tax rate created successfully", item)
}

func (h *Handler) ListTaxRates(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	items, err := uc.ListTaxRates(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Tax rates retrieved successfully", items)
}

func (h *Handler) UpdateTaxRate(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid tax rate ID format")
	}
	var dto application.UpdateTaxRateDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.UpdateTaxRate(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Tax rate updated successfully", item)
}

func (h *Handler) DeleteTaxRate(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid tax rate ID format")
	}
	if err := uc.DeleteTaxRate(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Tax rate deleted successfully", nil)
}
