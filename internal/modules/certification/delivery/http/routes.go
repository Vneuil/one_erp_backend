package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	certs := router.Group("/certifications", protected, tenantCtx)
	certs.Get("/schemes", h.Schemes)
	certs.Get("/eligibility", h.Eligibility)
	certs.Post("/", h.Create)
	certs.Get("/", h.List)
	certs.Get("/:id", h.Get)
	certs.Put("/:id", h.Update)
	// Suspending / resuming / revoking changes what can be procured: Manager or Admin.
	certs.Post("/:id/suspend", middleware.RequireRoles("admin", "manager"), h.Suspend)
	certs.Post("/:id/resume", middleware.RequireRoles("admin", "manager"), h.Resume)
	certs.Post("/:id/revoke", middleware.RequireRoles("admin", "manager"), h.Revoke)

	router.Get("/traceability/lot/:lot", protected, tenantCtx, h.TraceLot)
}
