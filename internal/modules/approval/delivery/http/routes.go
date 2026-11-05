package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	approval := router.Group("/approval", protected, tenantCtx)

	workflows := approval.Group("/workflows")
	workflows.Post("/", h.CreateWorkflow)
	workflows.Get("/", h.ListWorkflows)
	workflows.Put("/:id", h.UpdateWorkflow)
	workflows.Delete("/:id", h.DeleteWorkflow)

	requests := approval.Group("/requests")
	requests.Get("/", h.ListRequests)
	requests.Get("/pending", h.ListMyPendingApprovals)
	requests.Get("/document/:type/:id", h.GetActiveRequestForDocument)
	requests.Post("/:id/approve", h.ApproveStep)
	requests.Post("/:id/reject", h.RejectStep)
}
