package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	qc := router.Group("/qc", protected, tenantCtx)

	qc.Get("/stats", h.GetStats)

	params := qc.Group("/parameters")
	params.Get("/", h.ListParameters)
	params.Post("/", h.CreateParameter)
	params.Get("/:id", h.GetParameter)
	params.Put("/:id", h.UpdateParameter)
	params.Delete("/:id", h.DeleteParameter)

	inspections := qc.Group("/inspections")
	inspections.Get("/", h.ListInspections)
	inspections.Post("/", h.CreateInspection)
	inspections.Get("/:id", h.GetInspection)
	inspections.Post("/:id/void", h.VoidInspection)
}
