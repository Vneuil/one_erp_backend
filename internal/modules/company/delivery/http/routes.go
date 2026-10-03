package http

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers all company endpoints on the provided router
func (h *Handler) RegisterRoutes(router fiber.Router) {
	companies := router.Group("/companies")

	companies.Post("/", h.Create)
	companies.Get("/", h.List)
	companies.Get("/:id", h.GetByID)
	companies.Put("/:id", h.Update)
	companies.Delete("/:id", h.Delete)
}
