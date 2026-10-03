package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/project/application"
	"github.com/divinecoid/one-backend/internal/modules/project/infrastructure"
	taskinfra "github.com/divinecoid/one-backend/internal/modules/projecttask/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// resolve builds a use-case bound to the caller's own tenant database. The
// schema is migrated and seeded once, at tenant-provision time (see
// module.go's registration with foundation/tenant.RegisterSchema).
func (h *Handler) resolve(c *fiber.Ctx) (application.ProjectUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewProjectRepository(tenantDB)
	timeEntryRepo := infrastructure.NewTimeEntryRepository(tenantDB)
	taskRepo := taskinfra.NewProjectTaskRepository(tenantDB)
	return application.NewProjectUseCase(repo, timeEntryRepo, taskRepo), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) Create(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateProjectDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	project, err := uc.Create(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Project created successfully", project)
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid project ID format")
	}

	project, err := uc.GetByID(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Project retrieved successfully", project)
}

func (h *Handler) List(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	projects, meta, err := uc.List(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Projects retrieved successfully", projects, meta)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid project ID format")
	}

	var dto application.UpdateProjectDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	project, err := uc.Update(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Project updated successfully", project)
}

func (h *Handler) RecalculateProgress(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid project ID format")
	}

	project, err := uc.RecalculateProgress(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Project progress recalculated successfully", project)
}

func (h *Handler) resolveTimeEntry(c *fiber.Ctx) (application.TimeEntryUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewTimeEntryRepository(tenantDB)
	return application.NewTimeEntryUseCase(repo), nil
}

func (h *Handler) CreateTimeEntry(c *fiber.Ctx) error {
	uc, err := h.resolveTimeEntry(c)
	if err != nil {
		return err
	}

	var dto application.CreateTimeEntryDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	entry, err := uc.Create(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Time entry created successfully", entry)
}

func (h *Handler) ListTimeEntries(c *fiber.Ctx) error {
	uc, err := h.resolveTimeEntry(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	projectCode := c.Query("projectCode")

	entries, meta, err := uc.List(h.ctx(c), query, projectCode)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Time entries retrieved successfully", entries, meta)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid project ID format")
	}

	if err := uc.Delete(h.ctx(c), id); err != nil {
		return err
	}

	return response.OK(c, "Project deleted successfully", nil)
}
