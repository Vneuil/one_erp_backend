package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	crm := router.Group("/crm", protected, tenantCtx)

	crm.Post("/leads", h.Create)
	crm.Get("/leads", h.List)
	crm.Get("/leads/:id", h.GetByID)
	crm.Put("/leads/:id", h.Update)
	crm.Patch("/leads/:id/status", h.UpdateStatus)
	crm.Delete("/leads/:id", h.Delete)

	crm.Post("/deals", h.CreateDeal)
	crm.Get("/deals", h.ListDeals)
	crm.Get("/deals/:id", h.GetDealByID)
	crm.Put("/deals/:id", h.UpdateDeal)
	crm.Patch("/deals/:id/stage", h.UpdateDealStage)
	crm.Delete("/deals/:id", h.DeleteDeal)
}
