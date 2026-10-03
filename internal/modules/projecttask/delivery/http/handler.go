package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/projecttask/application"
	"github.com/divinecoid/one-backend/internal/modules/projecttask/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.ProjectTaskUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewProjectTaskRepository(tenantDB)
	return application.NewProjectTaskUseCase(repo), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) Create(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateProjectTaskDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	task, err := uc.Create(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Project task created successfully", task)
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
	projectCode := c.Query("projectCode")
	status := c.Query("status")
	tasks, meta, err := uc.List(h.ctx(c), query, projectCode, status)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Project tasks retrieved successfully", tasks, meta)
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	task, err := uc.GetByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Project task retrieved successfully", task)
}

func (h *Handler) UpdateStatus(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateStatusDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	task, err := uc.UpdateStatus(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Project task status updated successfully", task)
}

func (h *Handler) AddChecklistItem(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.CreateChecklistItemDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.AddChecklistItem(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Checklist item added successfully", item)
}

func (h *Handler) ToggleChecklistItem(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	itemID, err := parseID(c, "itemId")
	if err != nil {
		return err
	}
	item, err := uc.ToggleChecklistItem(h.ctx(c), itemID)
	if err != nil {
		return err
	}
	return response.OK(c, "Checklist item toggled successfully", item)
}
