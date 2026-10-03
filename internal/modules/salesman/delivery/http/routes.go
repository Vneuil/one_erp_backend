package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	salesmen := router.Group("/salesmen", protected, tenantCtx)

	salesmen.Post("/", h.Create)
	salesmen.Get("/", h.List)
	salesmen.Get("/:id", h.GetByID)
	salesmen.Put("/:id", h.Update)
	salesmen.Delete("/:id", h.Delete)
}
