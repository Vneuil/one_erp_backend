package http

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts every checkout route as public - no
// middleware.Protected. A prospective customer paying for a plan has no
// account/JWT yet, and the Xendit webhook authenticates itself via a
// shared token instead (see Handler.XenditWebhook).
func (h *Handler) RegisterRoutes(router fiber.Router) {
	checkout := router.Group("/checkout")
	checkout.Post("/orders", h.CreateOrder)
	checkout.Get("/orders/:id", h.GetOrderStatus)
	checkout.Post("/xendit-webhook", h.XenditWebhook)
}
