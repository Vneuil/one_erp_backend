package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/projectcost/domain"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes mounts per-project budget routes under /projects/:id (an extra
// path segment, so they cannot clash with the project module's own routes) and
// the SPK routes under /project-work-orders, both under the "project" RBAC module.
func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	p := router.Group("/projects/:id", protected, tenantCtx)
	p.Get("/budget", h.Budget())
	p.Post("/budget/import", h.ImportBudget())
	p.Delete("/budget/:lineId", h.DeleteBudgetLine())
	p.Get("/costs", h.ListCosts())
	p.Post("/costs", h.RecordCost())
	p.Delete("/costs/:costId", h.DeleteCost())

	router.Get("/project-profitability", protected, tenantCtx, h.Profitability())

	wo := router.Group("/project-work-orders", protected, tenantCtx)
	wo.Get("/", h.ListWorkOrders())
	wo.Post("/", h.CreateWorkOrder())
	wo.Post("/:id/issue", h.Advance(domain.WOIssued))
	wo.Post("/:id/start", h.Advance(domain.WOInProgress))
	wo.Post("/:id/complete", h.Advance(domain.WOCompleted))
	wo.Post("/:id/cancel", h.Advance(domain.WOCancelled))
	wo.Put("/:id/progress", h.SetProgress())
}
