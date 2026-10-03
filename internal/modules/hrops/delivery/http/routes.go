package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes mounts two groups:
//
//   - /hr-self: things every employee does for themselves (file a correction,
//     claim overtime, request a shift change, review a colleague, read
//     announcements and notifications). Exempt from module RBAC enforcement so a
//     view-only role can still use them; each handler only touches the caller's
//     own records.
//   - /hrm: HR administration and approvals, under the "hrm" permission module.
func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	self := router.Group("/hr-self", protected, tenantCtx)
	self.Post("/corrections", h.RequestCorrection)
	self.Get("/corrections", h.ListOwnCorrections)
	self.Post("/overtime", h.RequestOvertime)
	self.Get("/overtime", h.ListOwnOvertime)
	self.Get("/shifts", h.ListShifts)
	self.Get("/schedule", h.ListOwnSchedule)
	self.Post("/shift-changes", h.RequestShiftChange)
	self.Get("/shift-changes", h.ListOwnShiftChanges)
	self.Post("/feedback", h.SubmitFeedback)
	self.Get("/announcements", h.ListAnnouncements)
	self.Get("/notifications", h.ListNotifications)
	self.Post("/notifications/read-all", h.MarkAllNotificationsRead)
	self.Post("/notifications/:id/read", h.MarkNotificationRead)

	// A line manager decides their own reports' requests; no HR approval right needed.
	self.Get("/team/requests", h.TeamRequests)
	self.Post("/team/:kind/:id/approve", h.decideAsManager(true))
	self.Post("/team/:kind/:id/reject", h.decideAsManager(false))

	self.Post("/cash-advances", h.RequestAdvance)
	self.Get("/cash-advances", h.ListOwnAdvances)

	self.Post("/reimbursement-evidence", h.AddEvidence())
	self.Get("/reimbursement-evidence", h.ListEvidence())
	self.Get("/canteen/menu", h.ListCanteenItems())
	self.Post("/canteen/orders", h.OrderMeal())
	self.Get("/canteen/orders", h.MyMeals())
	self.Post("/canteen/orders/:id/cancel", h.CancelMeal())
	self.Post("/visits", h.PlanVisits())
	self.Get("/visits", h.MyPlan())
	self.Post("/visits/optimize", h.OptimizeRoute())
	self.Post("/visits/:id/check-in", h.CheckIn())
	self.Post("/visits/:id/skip", h.SkipStop())

	hrm := router.Group("/hrm", protected, tenantCtx)
	hrm.Get("/reimbursement-evidence", h.ListEvidence())
	hrm.Get("/canteen/items", h.ListCanteenItems())
	hrm.Post("/canteen/items", h.CreateCanteenItem())
	hrm.Put("/canteen/items/:id", h.UpdateCanteenItem())
	hrm.Get("/canteen/summary", h.CanteenSummary())
	hrm.Get("/visit-plans", h.TeamPlans())
	hrm.Get("/corrections", h.ListCorrections)
	hrm.Post("/corrections/:id/approve", h.decideCorrection(true))
	hrm.Post("/corrections/:id/reject", h.decideCorrection(false))
	hrm.Get("/attendance-summary", h.AttendanceSummary)

	hrm.Get("/shifts", h.ListShifts)
	hrm.Post("/shifts", h.CreateShift)
	hrm.Post("/shifts/assign", h.AssignShifts)
	hrm.Get("/schedule", h.ListSchedule)
	hrm.Get("/shift-changes", h.ListShiftChanges)
	hrm.Post("/shift-changes/:id/approve", h.decideShiftChange(true))
	hrm.Post("/shift-changes/:id/reject", h.decideShiftChange(false))

	hrm.Get("/dashboard", h.Dashboard)
	hrm.Get("/cash-advances", h.ListAdvances)
	hrm.Post("/cash-advances", h.RequestAdvance)
	hrm.Post("/cash-advances/:id/approve", h.decideAdvance(true))
	hrm.Post("/cash-advances/:id/reject", h.decideAdvance(false))

	hrm.Get("/overtime", h.ListOvertime)
	hrm.Post("/overtime", h.RequestOvertime)
	hrm.Post("/overtime/detect", h.DetectOvertime)
	hrm.Post("/overtime/:id/approve", h.decideOvertime(true))
	hrm.Post("/overtime/:id/reject", h.decideOvertime(false))

	hrm.Get("/employees/:id/documents", h.ListDocuments)
	hrm.Post("/employees/:id/documents", h.AddDocument)
	hrm.Delete("/documents/:id", h.DeleteDocument)
	hrm.Get("/documents/:id/file", h.DownloadDocument)
	hrm.Get("/employees/:id/feedback-summary", h.FeedbackSummary)

	hrm.Post("/announcements", h.PublishAnnouncement)
	hrm.Delete("/announcements/:id", h.DeleteAnnouncement)
}
