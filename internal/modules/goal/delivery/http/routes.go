package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenant "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenant.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	goals := router.Group("/goals", protected, tenantCtx)
	goals.Post("/", h.CreateGoal)
	goals.Get("/", h.ListGoals)
	goals.Get("/summary", h.GetSummary)
	goals.Get("/:id", h.GetGoalByID)
	goals.Put("/:id", h.UpdateGoal)
	goals.Post("/:id/check-in", h.CheckIn)
	goals.Post("/:id/complete", h.CompleteGoal)
}
