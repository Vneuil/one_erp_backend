package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	pos := router.Group("/pos", protected, tenantCtx)

	pos.Post("/checkout", h.Checkout)
	pos.Get("/transactions", h.List)
	pos.Get("/transactions/:id", h.GetByID)
	pos.Post("/transactions/:id/void", h.Void)
	pos.Post("/transactions/:id/refund", h.Refund)
	pos.Get("/transactions/:id/refunds", h.ListRefunds)
	pos.Get("/settings", h.GetSettings)
	pos.Put("/settings", middleware.RequireRoles("admin"), h.UpdateSettings)
	pos.Get("/reports/sales", h.SalesReport)
}
