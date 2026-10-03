package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	kpi := router.Group("/kpi", protected, tenantCtx)

	kpi.Post("/", h.CreateKpiReview)
	kpi.Get("/", h.ListKpiReviews)
	kpi.Put("/:id/status", h.UpdateKpiStatus)
}
