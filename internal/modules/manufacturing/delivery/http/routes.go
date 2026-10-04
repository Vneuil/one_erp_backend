package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	manufacturing := router.Group("/manufacturing", protected, tenantCtx)

	manufacturing.Get("/dashboard", h.GetDashboardSummary)

	bom := manufacturing.Group("/bom")
	bom.Post("/", h.CreateBOM)
	bom.Get("/", h.ListBOMs)
	bom.Get("/:id", h.GetBOMByID)

	orders := manufacturing.Group("/production-orders")
	orders.Post("/", h.CreateOrder)
	orders.Get("/", h.ListOrders)
	orders.Get("/:id", h.GetOrderByID)
	orders.Post("/:id/release", h.ReleaseOrder)
	orders.Post("/:id/pause", h.PauseOrder)
	orders.Post("/:id/resume", h.ResumeOrder)
	orders.Post("/:id/cancel", h.CancelOrder)
	orders.Post("/:id/complete-batch", h.CompleteBatch)
	orders.Post("/:id/steps/:stepId/log", h.LogStep)

	reports := manufacturing.Group("/reports")
	reports.Get("/daily", h.DailyReport)
	reports.Get("/order-summary", h.OrderSummary)
	reports.Get("/process-summary", h.ProcessSummary)
	reports.Get("/wip", h.WIPReport)
}
