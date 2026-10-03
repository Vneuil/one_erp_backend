package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	loyalty := router.Group("/loyalty", protected, tenantCtx)

	loyalty.Get("/config", h.GetConfig)
	loyalty.Put("/config", h.UpdateConfig)

	loyalty.Get("/members", h.ListMembers)
	loyalty.Post("/members", h.EnrollMember)
	loyalty.Get("/members/lookup", h.LookupMember)
}
