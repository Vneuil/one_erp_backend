package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/projectticket/application"
	"github.com/divinecoid/one-backend/internal/modules/projectticket/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.ProjectTicketUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewProjectTicketRepository(tenantDB)
	return application.NewProjectTicketUseCase(repo), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) Create(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateProjectTicketDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	ticket, err := uc.Create(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Project ticket created successfully", ticket)
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
	ticket, err := uc.GetByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Project ticket retrieved successfully", ticket)
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
	status := c.Query("status")
	tickets, meta, err := uc.List(h.ctx(c), query, status)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Project tickets retrieved successfully", tickets, meta)
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
	ticket, err := uc.UpdateStatus(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Project ticket status updated successfully", ticket)
}
