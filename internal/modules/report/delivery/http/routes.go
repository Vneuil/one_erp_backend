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
}
