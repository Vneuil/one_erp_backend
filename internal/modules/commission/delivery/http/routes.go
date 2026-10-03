package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	commission := router.Group("/commission", protected, tenantCtx)

	commission.Post("/rules", h.CreateRule)
	commission.Get("/rules", h.ListRules)
	commission.Get("/rules/:id", h.GetRuleByID)

	commission.Post("/records", h.CreateRecord)
	commission.Get("/records", h.ListRecords)
	commission.Get("/records/:id", h.GetRecordByID)
	commission.Post("/records/:id/approve", h.ApproveRecord)
	commission.Post("/records/:id/mark-paid", h.MarkRecordPaid)

	commission.Get("/summary", h.GetSummary)
}
