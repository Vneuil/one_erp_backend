package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	tasks := router.Group("/project-tasks", protected, tenantCtx)

	tasks.Post("/", h.Create)
	tasks.Get("/", h.List)
	tasks.Put("/:id/status", h.UpdateStatus)
	tasks.Post("/:id/checklist", h.AddChecklistItem)
	tasks.Put("/checklist/:itemId/toggle", h.ToggleChecklistItem)
	tasks.Get("/:id", h.GetByID)
}
