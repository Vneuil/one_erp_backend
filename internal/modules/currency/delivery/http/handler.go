package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/currency/application"
	"github.com/divinecoid/one-backend/internal/modules/currency/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) resolve(c *fiber.Ctx) (application.CurrencyUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewCurrencyRepository(tenantDB)
	return application.NewCurrencyUseCase(repo), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) ListCurrencies(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	items, err := uc.ListCurrencies(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Currencies retrieved successfully", items)
}

func (h *Handler) CreateCurrency(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateCurrencyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.CreateCurrency(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Currency added successfully", item)
}

func (h *Handler) DeleteCurrency(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid currency ID format")
	}
	if err := uc.DeleteCurrency(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Currency removed successfully", nil)
}

func (h *Handler) SetExchangeRate(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.SetExchangeRateDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	rate, err := uc.SetExchangeRate(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Exchange rate saved successfully", rate)
}

func (h *Handler) ListExchangeRates(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	items, err := uc.ListExchangeRates(h.ctx(c), c.Query("currencyCode"))
	if err != nil {
		return err
	}
	return response.OK(c, "Exchange rates retrieved successfully", items)
}

func (h *Handler) Convert(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var body struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
		AsOfDate string  `json:"asOfDate"`
	}
	if err := c.BodyParser(&body); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	result, err := uc.Convert(h.ctx(c), body.Amount, body.Currency, body.AsOfDate)
	if err != nil {
		return err
	}
	return response.OK(c, "Converted successfully", result)
}
