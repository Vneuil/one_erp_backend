package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	letters := router.Group("/hr-letters", protected, tenantCtx)
	letters.Get("/", h.List)
	letters.Post("/", h.Create)
	letters.Post("/preview", h.Preview)
	letters.Get("/contracts/expiring", h.ExpiringContracts)
	letters.Get("/employees/:id/summary", h.EmployeeSummary)
	letters.Get("/:id", h.Get)
	letters.Put("/:id", h.Update)
	letters.Post("/:id/issue", h.Issue)
	letters.Post("/:id/cancel", h.Cancel)
	letters.Post("/:id/acknowledge", h.Acknowledge)
	letters.Post("/:id/apply", h.Apply)
}
