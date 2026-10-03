package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/hrm/application"
	"github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	lmsinfra "github.com/divinecoid/one-backend/internal/modules/lms/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.HRMUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewHRMRepository(tenantDB)
	return application.NewHRMUseCase(repo), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateEmployee(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateEmployeeDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	emp, err := uc.CreateEmployee(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Employee created successfully", emp)
}

func (h *Handler) Organization(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	org, err := uc.Organization(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Organization retrieved successfully", org)
}

func (h *Handler) UpdateEmployee(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid employee ID format")
	}
	var dto application.UpdateEmployeeDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	emp, err := uc.UpdateEmployee(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Employee updated successfully", emp)
}

func (h *Handler) DeleteEmployee(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid employee ID format")
	}
	if err := uc.DeleteEmployee(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Employee deleted successfully", nil)
}

func (h *Handler) GetEmployeeByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid employee ID format")
	}

	emp, err := uc.GetEmployeeByID(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Employee retrieved successfully", emp)
}

func (h *Handler) ListEmployees(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	employees, meta, err := uc.ListEmployees(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Employees retrieved successfully", employees, meta)
}

func (h *Handler) ClockIn(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.ClockInDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	att, err := uc.RecordClockIn(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Attendance clocked in successfully", att)
}

func (h *Handler) ListAttendance(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	records, meta, err := uc.ListAttendance(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Attendance records retrieved successfully", records, meta)
}

// GetTrainingHistory returns the employee's LMS enrollment/completion
// history (see modules/lms), so HRM can surface a read-only "Training
// History" section on the Employee detail page without LMS completions
// being siloed from the employee record.
func (h *Handler) GetTrainingHistory(c *fiber.Ctx) error {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid employee ID format")
	}

	lmsRepo := lmsinfra.NewLMSRepository(tenantDB)
	enrollments, err := lmsRepo.ListEnrollmentsByEmployeeID(h.ctx(c), id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get training history")
	}

	return response.OK(c, "Training history retrieved successfully", enrollments)
}

func (h *Handler) ClockOut(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.ClockOutDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	att, err := uc.RecordClockOut(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Attendance clocked out successfully", att)
}

func (h *Handler) ListAttendanceLocations(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	locs, err := uc.ListAttendanceLocations(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Attendance locations retrieved successfully", locs)
}

func (h *Handler) CreateAttendanceLocation(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateAttendanceLocationDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	loc, err := uc.CreateAttendanceLocation(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Attendance location created successfully", loc)
}

func (h *Handler) DeleteAttendanceLocation(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid id format")
	}
	if err := uc.DeleteAttendanceLocation(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Attendance location deleted successfully", nil)
}
