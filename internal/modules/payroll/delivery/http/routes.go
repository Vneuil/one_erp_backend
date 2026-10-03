package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	self := router.Group("/hr-self/payslips", protected, tenantCtx)
	self.Get("/", h.MyPayslips)
	self.Get("/:id", h.Payslip(false))

	payroll := router.Group("/payroll", protected, tenantCtx)

	payroll.Post("/entries", h.CreateEntry)
	payroll.Get("/entries", h.ListEntries)
	payroll.Post("/entries/calculate", h.CalculatePeriod)
	payroll.Put("/entries/:id/status", h.UpdateStatus)
	// HR can open any payslip; employees reach their own through /hr-self/payslips.
	payroll.Get("/entries/:id/payslip", h.Payslip(true))
	payroll.Get("/policy", h.GetPolicy)
	payroll.Put("/policy", middleware.RequireRoles("admin"), h.UpdatePolicy)
}
