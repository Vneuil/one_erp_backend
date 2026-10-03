package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string) {
	protected := middleware.Protected(jwtSecret)
	adminOnly := middleware.RequireRoles("admin")

	roles := router.Group("/rbac/roles", protected)
	roles.Get("/", h.ListRoles)
	roles.Post("/", adminOnly, h.CreateRole)
	roles.Delete("/:id", adminOnly, h.DeleteRole)
	roles.Get("/:id/permissions", h.GetRolePermissions)
	roles.Put("/:id/permissions", adminOnly, h.SetRolePermissions)

	router.Get("/rbac/modules", protected, h.ListModules)
}
