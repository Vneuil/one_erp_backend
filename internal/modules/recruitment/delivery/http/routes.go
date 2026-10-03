package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	recruitment := router.Group("/recruitment", protected, tenantCtx)

	recruitment.Post("/vacancies", h.CreateVacancy)
	recruitment.Get("/vacancies", h.ListVacancies)
	recruitment.Get("/vacancies/:id", h.GetVacancyByID)
	recruitment.Put("/vacancies/:id", h.UpdateVacancy)

	recruitment.Post("/candidates", h.CreateCandidate)
	recruitment.Get("/candidates", h.ListCandidates)
	recruitment.Get("/candidates/:id", h.GetCandidateByID)
	recruitment.Post("/candidates/:id/advance-stage", h.AdvanceStage)
	recruitment.Post("/candidates/:id/hire", h.HireCandidate)

	recruitment.Post("/interviews", h.CreateInterview)
	recruitment.Get("/interviews", h.ListInterviews)
	recruitment.Get("/interviews/:id", h.GetInterviewByID)
	recruitment.Put("/interviews/:id", h.UpdateInterview)
}
