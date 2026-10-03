package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	tickets := router.Group("/project-tickets", protected, tenantCtx)

	tickets.Post("/", h.Create)
	tickets.Get("/", h.List)
	tickets.Put("/:id/status", h.UpdateStatus)
	tickets.Get("/:id", h.GetByID)
}
