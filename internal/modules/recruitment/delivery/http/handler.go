package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/recruitment/application"
	"github.com/divinecoid/one-backend/internal/modules/recruitment/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func parseID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid " + param + " format")
	}
	return id, nil
}

// resolve builds a use-case bound to the caller's own tenant database. The
// schema is migrated and seeded once, at tenant-provision time (see
// module.go's registration with foundation/tenant.RegisterSchema).
func (h *Handler) resolve(c *fiber.Ctx) (application.RecruitmentUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewRecruitmentRepository(tenantDB)
	hrmRepo := hrminfra.NewHRMRepository(tenantDB)
	return application.NewRecruitmentUseCase(repo, hrmRepo), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateVacancy(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateJobVacancyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	vacancy, err := uc.CreateVacancy(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Job vacancy created successfully", vacancy)
}

func (h *Handler) GetVacancyByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	vacancy, err := uc.GetVacancyByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Job vacancy retrieved successfully", vacancy)
}

func (h *Handler) ListVacancies(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListVacancies(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Job vacancies retrieved successfully", items, meta)
}

func (h *Handler) UpdateVacancy(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateJobVacancyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	vacancy, err := uc.UpdateVacancy(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Job vacancy updated successfully", vacancy)
}

func (h *Handler) CreateCandidate(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateCandidateDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	candidate, err := uc.CreateCandidate(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Candidate created successfully", candidate)
}

func (h *Handler) GetCandidateByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	candidate, err := uc.GetCandidateByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Candidate retrieved successfully", candidate)
}

func (h *Handler) ListCandidates(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListCandidates(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Candidates retrieved successfully", items, meta)
}

func (h *Handler) AdvanceStage(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.AdvanceStageDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	candidate, err := uc.AdvanceStage(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Candidate stage updated successfully", candidate)
}

func (h *Handler) HireCandidate(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	candidate, err := uc.HireCandidate(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Candidate hired successfully", candidate)
}

func (h *Handler) CreateInterview(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateInterviewDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	interview, err := uc.CreateInterview(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Interview scheduled successfully", interview)
}

func (h *Handler) GetInterviewByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	interview, err := uc.GetInterviewByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Interview retrieved successfully", interview)
}

func (h *Handler) ListInterviews(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListInterviews(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Interviews retrieved successfully", items, meta)
}

func (h *Handler) UpdateInterview(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateInterviewDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	interview, err := uc.UpdateInterview(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Interview updated successfully", interview)
}
