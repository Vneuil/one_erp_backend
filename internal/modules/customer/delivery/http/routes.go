package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	customers := router.Group("/customers", protected, tenantCtx)

	customers.Post("/", h.Create)
	customers.Get("/", h.List)
	customers.Get("/:id", h.GetByID)
	customers.Put("/:id", h.Update)
	// Deactivate/reactivate and hard delete: Manager or Admin only (staff cannot remove customers).
	customers.Patch("/:id/status", middleware.RequireRoles("admin", "manager"), h.SetStatus)
	customers.Delete("/:id", middleware.RequireRoles("admin", "manager"), h.Delete)
}
