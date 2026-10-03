package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/report/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// resolve builds a report query bound to the caller's own tenant database.
// Report has no schema of its own - it reads other modules' tables - so
// there is nothing to migrate/seed, just a *gorm.DB to bind to.
func (h *Handler) resolve(c *fiber.Ctx) (*application.ReportQuery, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	return application.NewReportQuery(tenantDB), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) SalesPerformance(c *fiber.Ctx) error {
	query, err := h.resolve(c)
	if err != nil {
		return err
	}
	from := c.Query("from")
	to := c.Query("to")
	data, err := query.SalesPerformance(h.ctx(c), from, to)
	if err != nil {
		return err
	}
	return response.OK(c, "Sales performance report retrieved successfully", data)
}

func (h *Handler) InventoryValuation(c *fiber.Ctx) error {
	query, err := h.resolve(c)
	if err != nil {
		return err
	}
	data, err := query.InventoryValuation(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Inventory valuation report retrieved successfully", data)
}

func (h *Handler) HRMSummary(c *fiber.Ctx) error {
	query, err := h.resolve(c)
	if err != nil {
		return err
	}
	data, err := query.HRMSummary(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "HRM summary report retrieved successfully", data)
}

func (h *Handler) ExecutiveSummary(c *fiber.Ctx) error {
	query, err := h.resolve(c)
	if err != nil {
		return err
	}
	data, err := query.ExecutiveSummary(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Executive summary retrieved successfully", data)
}
