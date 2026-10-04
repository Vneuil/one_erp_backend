package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/report/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

// Sub-ledger, sales, purchase and stock reports

func optionalUUID(c *fiber.Ctx, name string) (*uuid.UUID, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperrors.NewBadRequest("Invalid " + name + " format")
	}
	return &id, nil
}

// report resolves the query, runs fn and wraps the result as 200 OK.
func report[T any](h *Handler, c *fiber.Ctx, message string, fn func(*application.ReportQuery, context.Context) (T, error)) error {
	query, err := h.resolve(c)
	if err != nil {
		return err
	}
	data, err := fn(query, h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, message, data)
}

func (h *Handler) SalesSummary(c *fiber.Ctx) error {
	return report(h, c, "Sales summary retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.SalesSummaryDTO, error) {
		return q.SalesSummary(ctx, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) SalesByProduct(c *fiber.Ctx) error {
	return report(h, c, "Sales by product retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.SalesByProductDTO, error) {
		return q.SalesByProduct(ctx, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) DailyProductSales(c *fiber.Ctx) error {
	return report(h, c, "Daily product sales retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.DailyProductSalesDTO, error) {
		return q.DailyProductSales(ctx, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) SalesByCustomer(c *fiber.Ctx) error {
	return report(h, c, "Sales by customer retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.SalesByCustomerDTO, error) {
		return q.SalesByCustomer(ctx, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) GrossMargin(c *fiber.Ctx) error {
	return report(h, c, "Gross margin analysis retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.GrossMarginDTO, error) {
		return q.GrossMargin(ctx, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) PurchaseReport(c *fiber.Ctx) error {
	return report(h, c, "Purchase report retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.PurchaseReportDTO, error) {
		return q.PurchaseReport(ctx, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) partyBalances(receivable bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return report(h, c, "Closing balances retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.PartyBalancesDTO, error) {
			return q.PartyBalances(ctx, receivable, c.QueryBool("includeZero"), c.Query("asOf"))
		})
	}
}

func (h *Handler) partyCard(receivable bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return report(h, c, "Account card retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.PartyCardDTO, error) {
			return q.PartyCard(ctx, receivable, c.Query("party"), c.Query("from"), c.Query("to"))
		})
	}
}

func (h *Handler) StockCard(c *fiber.Ctx) error {
	productID, err := optionalUUID(c, "productId")
	if err != nil {
		return err
	}
	if productID == nil {
		return apperrors.NewBadRequest("productId is required")
	}
	warehouseID, err := optionalUUID(c, "warehouseId")
	if err != nil {
		return err
	}
	return report(h, c, "Stock card retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.StockCardDTO, error) {
		return q.StockCard(ctx, *productID, warehouseID, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) StockTransactions(c *fiber.Ctx) error {
	productID, err := optionalUUID(c, "productId")
	if err != nil {
		return err
	}
	warehouseID, err := optionalUUID(c, "warehouseId")
	if err != nil {
		return err
	}
	return report(h, c, "Stock transactions retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.StockTransactionsDTO, error) {
		return q.StockTransactions(ctx, c.Query("from"), c.Query("to"), c.Query("type"), productID, warehouseID)
	})
}

func (h *Handler) StockAging(c *fiber.Ctx) error {
	warehouseID, err := optionalUUID(c, "warehouseId")
	if err != nil {
		return err
	}
	return report(h, c, "Stock aging retrieved successfully", func(q *application.ReportQuery, ctx context.Context) (*application.StockAgingDTO, error) {
		return q.StockAging(ctx, c.Query("asOf"), warehouseID)
	})
}
