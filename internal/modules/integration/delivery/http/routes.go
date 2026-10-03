package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	integration := router.Group("/integration", protected, tenantCtx)

	apiKeys := integration.Group("/api-keys")
	apiKeys.Post("/", h.CreateAPIKey)
	apiKeys.Get("/", h.ListAPIKeys)
	apiKeys.Put("/:id/revoke", h.RevokeAPIKey)

	webhooks := integration.Group("/webhooks")
	webhooks.Post("/", h.CreateWebhook)
	webhooks.Get("/", h.ListWebhooks)
	webhooks.Put("/:id", h.UpdateWebhook)
	webhooks.Post("/:id/test", h.TestWebhook)
	webhooks.Get("/:id/deliveries", h.ListDeliveries)
}
