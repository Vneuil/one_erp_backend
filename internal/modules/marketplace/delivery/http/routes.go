package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes mounts /marketplace. Most routes run behind the normal
// Protected+TenantContext middleware (they're called by our own frontend's
// authenticated XHR client), except the OAuth provider callbacks
// (tiktok/callback, shopee/callback): those are plain browser redirects
// issued by TikTok Shop / Shopee themselves and carry no Authorization
// header, so they resolve the tenant from a JWT round-tripped through the
// OAuth state/redirect params instead (see handler.resolveFromToken).
func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	mp := router.Group("/marketplace")

	// Middleware is attached per-route (not via a nested empty-prefix
	// Group) because Fiber mounts a Group's middleware as Use() at that
	// path prefix - a `mp.Group("", protected, tenantCtx)` would apply
	// protected/tenantCtx to every request under /marketplace/*, including
	// the public OAuth callbacks registered below on mp directly, which is
	// exactly the bug that made Shopee/TikTok's redirect callbacks 401 with
	// "Missing Authorization header".
	mp.Get("/connections", protected, tenantCtx, h.ListConnections)
	mp.Get("/tiktok/connect", protected, tenantCtx, h.TikTokConnect)
	mp.Get("/shopee/connect", protected, tenantCtx, h.ShopeeConnect)
	// Blibli has no OAuth redirect flow, so unlike TikTok/Shopee this is a
	// plain authenticated POST (not a public callback) - it runs behind the
	// same Protected+TenantContext middleware as the other authed routes.
	mp.Post("/blibli/connect", protected, tenantCtx, h.BlibliConnect)
	mp.Get("/lazada/connect", protected, tenantCtx, h.LazadaConnect)
	mp.Post("/:platform/sync", protected, tenantCtx, h.Sync)
	mp.Post("/orders/:salesOrderId/ready-to-ship", protected, tenantCtx, h.MarkOrderReadyToShip)
	mp.Post("/:platform/disconnect", protected, tenantCtx, h.Disconnect)

	mp.Get("/tiktok/callback", h.TikTokCallback)
	mp.Get("/shopee/callback", h.ShopeeCallback)
	mp.Get("/lazada/callback", h.LazadaCallback)
}
