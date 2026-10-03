package http

import (
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes mounts the authenticated CRM extras under /crm (RBAC module
// "crm") and the public lead form under /public.
func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager, public *PublicHandler) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)
	adminOnly := middleware.RequireRoles("admin")

	crm := router.Group("/crm", protected, tenantCtx)

	crm.Post("/interactions", h.LogInteraction())
	crm.Get("/interactions", h.ListInteractions())

	crm.Post("/tasks", h.CreateTask())
	crm.Get("/tasks", h.ListTasks())
	crm.Get("/tasks/reminders", h.Reminders())
	crm.Post("/tasks/:id/done", h.SetTaskDone(true))
	crm.Post("/tasks/:id/reopen", h.SetTaskDone(false))

	crm.Post("/documents", h.AddDocument())
	crm.Get("/documents", h.ListDocuments())
	crm.Delete("/documents/:id", h.DeleteDocument())
	crm.Get("/documents/:id/file", h.DownloadDocument)

	crm.Get("/tags", h.ListTags())
	crm.Post("/tags", h.CreateTag())
	crm.Delete("/tags/:id", h.DeleteTag())
	crm.Put("/tags/:id/assign", h.SetTag(true))
	crm.Put("/tags/:id/unassign", h.SetTag(false))
	crm.Get("/tags/:id/records", h.ParentsWithTag())
	crm.Get("/tagged", h.TagsFor())

	crm.Get("/contacts", h.ListContacts())
	crm.Post("/contacts", h.CreateContact())
	crm.Put("/contacts/:id", h.UpdateContact())
	crm.Delete("/contacts/:id", h.DeleteContact())

	crm.Get("/stages", h.ListStages())
	crm.Post("/stages", adminOnly, h.CreateStage())
	crm.Put("/stages/:id", adminOnly, h.UpdateStage())
	crm.Delete("/stages/:id", adminOnly, h.DeleteStage())

	crm.Get("/lead-forms", h.ListLeadForms())
	crm.Post("/lead-forms", adminOnly, h.CreateLeadForm())
	crm.Put("/lead-forms/:id", adminOnly, h.UpdateLeadForm())
	crm.Delete("/lead-forms/:id", adminOnly, h.DeleteLeadForm())

	crm.Post("/deals/:id/start-project", h.StartProject())
	crm.Get("/analytics", h.Analytics())

	// Public, unauthenticated. Rate limited per client IP so a form cannot be
	// used to flood a company's leads or to probe form keys.
	pub := router.Group("/public/lead-forms", rateLimit(20, time.Minute))
	pub.Get("/:key", public.Info)
	pub.Post("/:key/submit", public.Submit)
}
