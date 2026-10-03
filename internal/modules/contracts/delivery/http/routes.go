package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	contracts := router.Group("/contracts", protected, tenantCtx)
	contracts.Post("/", h.CreateContract)
	contracts.Get("/", h.ListContracts)
	contracts.Get("/expiring-soon", h.ListExpiringSoon)
	contracts.Get("/:id", h.GetContractByID)
	contracts.Put("/:id", h.UpdateContract)
	contracts.Put("/:id/status", h.UpdateContractStatus)
	contracts.Post("/:id/renew", h.RenewContract)
}
