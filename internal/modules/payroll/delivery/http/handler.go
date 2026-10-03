package http

import (
	"context"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	coopinfra "github.com/divinecoid/one-backend/internal/modules/cooperative/infrastructure"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	hropsinfra "github.com/divinecoid/one-backend/internal/modules/hrops/infrastructure"
	leaveinfra "github.com/divinecoid/one-backend/internal/modules/leave/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/payroll/application"
	"github.com/divinecoid/one-backend/internal/modules/payroll/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.PayrollUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewPayrollRepository(tenantDB)
	hrmRepo := hrminfra.NewHRMRepository(tenantDB)
	coopRepo := coopinfra.NewCooperativeRepository(tenantDB)
	ledger := financeApp.NewLedgerPoster(financeInfra.NewFinanceRepository(tenantDB))
	return application.NewPayrollUseCase(repo, hrmRepo, coopRepo, application.WithLedger(ledger), application.WithLeave(leaveinfra.NewLeaveRepository(tenantDB)), application.WithOvertime(hropsinfra.NewOvertimeSource(tenantDB)), application.WithCanteen(hropsinfra.NewOvertimeSource(tenantDB)), application.WithAdvances(hropsinfra.NewOvertimeSource(tenantDB)), application.WithOvertimeDetail(hropsinfra.NewOvertimeSource(tenantDB)), application.WithInbox(hropsinfra.NewInboxSender(tenantDB))), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePayrollEntryDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	entry, err := uc.CreateEntry(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Payroll entry created successfully", entry)
}

func (h *Handler) ListEntries(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	period := c.Query("period")

	entries, meta, err := uc.ListEntries(h.ctx(c), query, period)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Payroll entries retrieved successfully", entries, meta)
}

func (h *Handler) CalculatePeriod(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CalculatePeriodDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	entries, err := uc.CalculatePeriod(h.ctx(c), dto.Period)
	if err != nil {
		return err
	}

	return response.OK(c, "Payroll period calculated successfully", entries)
}

func (h *Handler) UpdateStatus(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid payroll entry ID format")
	}

	var dto application.UpdateStatusDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	entry, err := uc.UpdateStatus(h.ctx(c), id, dto.Status)
	if err != nil {
		return err
	}

	return response.OK(c, "Payroll entry status updated successfully", entry)
}

func (h *Handler) GetPolicy(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	p, err := uc.GetPolicy(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Payroll policy retrieved successfully", p)
}

func (h *Handler) UpdatePolicy(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.UpdatePolicyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	p, err := uc.UpdatePolicy(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Payroll policy updated successfully", p)
}
