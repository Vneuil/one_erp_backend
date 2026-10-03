package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers user endpoints on the router
func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string) {
	users := router.Group("/users")

	// Apply authentication middleware
	users.Use(middleware.Protected(jwtSecret))

	users.Post("/", middleware.RequireRoles("admin", "manager"), h.Create)
	users.Get("/", h.List)
	users.Get("/:id", h.GetByID)
	users.Put("/:id", middleware.RequireRoles("admin", "manager"), h.Update)
	users.Put("/:id/role", middleware.RequireRoles("admin"), h.AssignRole)
	users.Delete("/:id", middleware.RequireRoles("admin"), h.Delete)
}
