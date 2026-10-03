package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// RegisterRoutes mounts /omnichannel. The two webhook routes
// (GET/POST /omnichannel/webhooks/whatsapp) are public, unauthenticated
// endpoints called directly by Meta - they are attached per-route (NOT via a
// nested empty-prefix Group) to avoid the exact bug fixed in marketplace's
// routes.go ("Fix marketplace OAuth callbacks being blocked by auth
// middleware"): a `router.Group("", protected, tenantCtx)` mounts those
// middlewares as Use() at the parent prefix, which would also apply them to
// sibling public routes registered on the same parent group.
func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	oc := router.Group("/omnichannel")

	oc.Get("/conversations", protected, tenantCtx, h.ListConversations)
	oc.Get("/conversations/:id/messages", protected, tenantCtx, h.GetThread)
	oc.Post("/conversations/:id/messages", protected, tenantCtx, h.SendMessage)

	oc.Get("/connections", protected, tenantCtx, h.ListConnections)
	oc.Post("/connections/whatsapp", protected, tenantCtx, h.ConnectWhatsApp)
	oc.Post("/connections/:channel/disconnect", protected, tenantCtx, h.DisconnectChannel)

	// Public webhook routes - no protected/tenantCtx middleware.
	oc.Get("/webhooks/whatsapp", h.WhatsAppVerify)
	oc.Post("/webhooks/whatsapp", h.WhatsAppWebhook)
}
