package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	reports := router.Group("/reports", protected, tenantCtx)

	reports.Get("/sales-performance", h.SalesPerformance)
	reports.Get("/inventory-valuation", h.InventoryValuation)
	reports.Get("/hrm-summary", h.HRMSummary)
	reports.Get("/executive-summary", h.ExecutiveSummary)

	reports.Get("/sales/summary", h.SalesSummary)
	reports.Get("/sales/by-product", h.SalesByProduct)
	reports.Get("/sales/by-customer", h.SalesByCustomer)
	reports.Get("/sales/daily-product", h.DailyProductSales)
	reports.Get("/sales/gross-margin", h.GrossMargin)
	reports.Get("/purchases", h.PurchaseReport)

	reports.Get("/ar/balances", h.partyBalances(true))
	reports.Get("/ar/card", h.partyCard(true))
	reports.Get("/ap/balances", h.partyBalances(false))
	reports.Get("/ap/card", h.partyCard(false))

	reports.Get("/stock/card", h.StockCard)
	reports.Get("/stock/transactions", h.StockTransactions)
	reports.Get("/stock/aging", h.StockAging)
}
