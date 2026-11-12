package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	suppliers := router.Group("/suppliers", protected, tenantCtx)

	suppliers.Get("/types", h.Types)
	suppliers.Post("/", h.Create)
	suppliers.Get("/", h.List)
	suppliers.Get("/:id", h.GetByID)
	suppliers.Put("/:id", h.Update)
	// Deactivate/reactivate: Manager or Admin only. Staff cannot change vendor status.
	suppliers.Patch("/:id/status", middleware.RequireRoles("admin", "manager"), h.SetStatus)
	// Hard delete is reserved for Manager/Admin and is blocked once the vendor is referenced.
	suppliers.Delete("/:id", middleware.RequireRoles("admin", "manager"), h.Delete)
}
