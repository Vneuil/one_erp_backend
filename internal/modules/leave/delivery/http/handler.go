package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	hropsinfra "github.com/divinecoid/one-backend/internal/modules/hrops/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/leave/application"
	"github.com/divinecoid/one-backend/internal/modules/leave/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.LeaveUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewLeaveRepository(tenantDB)
	hrmRepo := hrminfra.NewHRMRepository(tenantDB)
	return application.NewLeaveUseCase(repo, hrmRepo, application.WithInbox(hropsinfra.NewInboxSender(tenantDB))), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateLeaveRequest(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateLeaveRequestDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	req, err := uc.CreateLeaveRequest(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Leave request created successfully", req)
}

func (h *Handler) ListLeaveRequests(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	items, meta, err := uc.ListLeaveRequests(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Leave requests retrieved successfully", items, meta)
}

func (h *Handler) ApproveLeaveRequest(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid leave request ID format")
	}

	var dto application.ApproveLeaveRequestDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	req, err := uc.ApproveLeaveRequest(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Leave request approved successfully", req)
}

func (h *Handler) RejectLeaveRequest(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid leave request ID format")
	}

	var dto application.RejectLeaveRequestDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	req, err := uc.RejectLeaveRequest(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Leave request rejected successfully", req)
}
