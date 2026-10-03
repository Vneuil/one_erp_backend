package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	cooperative := router.Group("/cooperative", protected, tenantCtx)

	cooperative.Post("/loans", h.CreateLoan)
	cooperative.Get("/loans", h.ListLoans)
	cooperative.Post("/loans/:id/record-payment", h.RecordPayment)
}
