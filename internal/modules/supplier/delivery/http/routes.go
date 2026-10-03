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

	suppliers.Post("/", h.Create)
	suppliers.Get("/", h.List)
	suppliers.Get("/:id", h.GetByID)
	suppliers.Put("/:id", h.Update)
	suppliers.Delete("/:id", h.Delete)
}
