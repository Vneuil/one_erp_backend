package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	reimbursements := router.Group("/reimbursements", protected, tenantCtx)

	reimbursements.Post("/", h.CreateClaim)
	reimbursements.Get("/", h.ListClaims)
	reimbursements.Post("/:id/approve", h.ApproveClaim)
	reimbursements.Post("/:id/reject", h.RejectClaim)
	reimbursements.Post("/:id/mark-paid", h.MarkPaid)
}
