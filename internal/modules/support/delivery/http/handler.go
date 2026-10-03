package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/support/application"
	"github.com/divinecoid/one-backend/internal/modules/support/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.TicketsUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewTicketsRepository(tenantDB)
	return application.NewTicketsUseCase(repo), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateTicket(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateTicketDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	ticket, err := uc.CreateTicket(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Ticket created successfully", ticket)
}

func (h *Handler) GetTicketByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	ticket, err := uc.GetTicketByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Ticket retrieved successfully", ticket)
}

func (h *Handler) ListTickets(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListTickets(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Tickets retrieved successfully", items, meta)
}

func (h *Handler) UpdateTicket(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateTicketDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	ticket, err := uc.UpdateTicket(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Ticket updated successfully", ticket)
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
	return response.OK(c, "Ticket status updated successfully", ticket)
}

func (h *Handler) AddReply(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.ReplyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	reply, err := uc.AddReply(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Reply added successfully", reply)
}

func (h *Handler) ListReplies(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	replies, err := uc.ListReplies(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Replies retrieved successfully", replies)
}

func (h *Handler) GetSummary(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	summary, err := uc.GetSummary(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Support summary retrieved successfully", summary)
}
