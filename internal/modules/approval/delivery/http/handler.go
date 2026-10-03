package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/approval/application"
	"github.com/divinecoid/one-backend/internal/modules/approval/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) resolve(c *fiber.Ctx) (application.ApprovalUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewApprovalRepository(tenantDB)
	uc := application.NewApprovalUseCase(repo)
	// Wire in every module's registered document-status callback (e.g.
	// procurement for "purchase_order", sales for "sales_order") so that
	// approving/rejecting a request from this generic Approval Center
	// updates the underlying document exactly the same way the module's
	// own "/approve" endpoint would - see application/callback.go.
	for documentType, cb := range application.BuildCallbacks(tenantDB) {
		uc.RegisterCallback(documentType, cb)
	}
	return uc, nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateWorkflow(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateWorkflowDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	w, err := uc.CreateWorkflow(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Approval workflow created successfully", w)
}

func (h *Handler) ListWorkflows(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	items, err := uc.ListWorkflows(h.ctx(c), c.Query("documentType"))
	if err != nil {
		return err
	}
	return response.OK(c, "Approval workflows retrieved successfully", items)
}

func (h *Handler) DeleteWorkflow(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid workflow ID format")
	}
	if err := uc.DeleteWorkflow(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Approval workflow deleted successfully", nil)
}

func (h *Handler) ApproveStep(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid request ID format")
	}
	var dto application.ActOnStepDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	claims := middleware.CurrentUser(c)
	if claims != nil {
		if dto.ActorName == "" {
			dto.ActorName = claims.Email
		}
		dto.ActorRole = claims.Role
	}
	req, err := uc.ApproveStep(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Approval step approved successfully", req)
}

func (h *Handler) RejectStep(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid request ID format")
	}
	var dto application.ActOnStepDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	claims := middleware.CurrentUser(c)
	if claims != nil {
		if dto.ActorName == "" {
			dto.ActorName = claims.Email
		}
		dto.ActorRole = claims.Role
	}
	req, err := uc.RejectStep(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Approval step rejected successfully", req)
}

func (h *Handler) ListMyPendingApprovals(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	claims := middleware.CurrentUser(c)
	role := ""
	if claims != nil {
		role = claims.Role
	}
	items, err := uc.ListMyPendingApprovals(h.ctx(c), role)
	if err != nil {
		return err
	}
	return response.OK(c, "Pending approvals retrieved successfully", items)
}

func (h *Handler) ListRequests(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListRequests(h.ctx(c), query, c.Query("documentType"))
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Approval requests retrieved successfully", items, meta)
}
