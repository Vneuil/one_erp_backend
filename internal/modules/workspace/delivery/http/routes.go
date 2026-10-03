package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	ws := router.Group("/tenants")
	ws.Get("/", protected, tenantCtx, h.List)
	ws.Post("/", protected, tenantCtx, h.Create)
	ws.Post("/switch", protected, tenantCtx, h.Switch)
}
