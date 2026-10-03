package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	leaves := router.Group("/leaves", protected, tenantCtx)

	leaves.Post("/", h.CreateLeaveRequest)
	leaves.Get("/", h.ListLeaveRequests)
	leaves.Post("/:id/approve", h.ApproveLeaveRequest)
	leaves.Post("/:id/reject", h.RejectLeaveRequest)
}
