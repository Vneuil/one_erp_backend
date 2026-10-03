package http

import (
	"context"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement/application"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.ReimbursementUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewReimbursementRepository(tenantDB)
	ledger := financeApp.NewLedgerPoster(financeInfra.NewFinanceRepository(tenantDB))
	return application.NewReimbursementUseCase(repo, application.WithLedger(ledger)), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateClaim(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateClaimDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	claim, err := uc.CreateClaim(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Reimbursement claim created successfully", claim)
}

func (h *Handler) ListClaims(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	claims, meta, err := uc.ListClaims(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Reimbursement claims retrieved successfully", claims, meta)
}

func (h *Handler) parseID(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid claim ID format")
	}
	return id, nil
}

func (h *Handler) ApproveClaim(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := h.parseID(c)
	if err != nil {
		return err
	}

	claim, err := uc.ApproveClaim(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Claim approved successfully", claim)
}

func (h *Handler) RejectClaim(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := h.parseID(c)
	if err != nil {
		return err
	}

	claim, err := uc.RejectClaim(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Claim rejected successfully", claim)
}

func (h *Handler) MarkPaid(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := h.parseID(c)
	if err != nil {
		return err
	}

	claim, err := uc.MarkPaid(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Claim marked as paid successfully", claim)
}
