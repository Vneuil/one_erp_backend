package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	projects := router.Group("/projects", protected, tenantCtx)

	projects.Post("/", h.Create)
	projects.Get("/", h.List)
	projects.Post("/time-entries", h.CreateTimeEntry)
	projects.Get("/time-entries", h.ListTimeEntries)
	projects.Get("/:id", h.GetByID)
	projects.Put("/:id", h.Update)
	projects.Delete("/:id", h.Delete)
	projects.Post("/:id/recalculate", h.RecalculateProgress)
}
