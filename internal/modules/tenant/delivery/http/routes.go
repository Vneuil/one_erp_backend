package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenant "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenant.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	tenantGroup := router.Group("/tenant")
	tenantGroup.Post("/provision", protected, h.Provision)
	tenantGroup.Post("/backfill-schema", protected, middleware.RequireRoles("admin"), h.BackfillSchema)

	demo := router.Group("/tenant-demo")
	demo.Post("/ping", protected, tenantCtx, h.Ping)
}
