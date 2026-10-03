package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	shippingMethods := router.Group("/shipping-methods", protected, tenantCtx)

	shippingMethods.Post("/", h.Create)
	shippingMethods.Get("/", h.List)
	shippingMethods.Get("/:id", h.GetByID)
	shippingMethods.Put("/:id", h.Update)
	shippingMethods.Delete("/:id", h.Delete)
}
