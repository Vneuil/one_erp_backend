package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	support := router.Group("/support", protected, tenantCtx)
	support.Get("/summary", h.GetSummary)

	tickets := support.Group("/tickets")
	tickets.Post("/", h.CreateTicket)
	tickets.Get("/", h.ListTickets)
	tickets.Get("/:id", h.GetTicketByID)
	tickets.Put("/:id", h.UpdateTicket)
	tickets.Put("/:id/status", h.UpdateStatus)
	tickets.Post("/:id/replies", h.AddReply)
	tickets.Get("/:id/replies", h.ListReplies)
}
