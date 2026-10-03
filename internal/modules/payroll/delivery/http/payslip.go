package http

import (
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h *Handler) MyPayslips(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return apperrors.NewForbidden("Not authenticated")
	}
	list, err := uc.MyPayslips(h.ctx(c), claims.Email)
	if err != nil {
		return err
	}
	return response.OK(c, "Payslips retrieved", list)
}

// Payslip serves one payslip. asHR is true on the HR route, which sits behind
// the payroll module permission; the self-service route only ever returns the
// caller's own approved or paid slip.
func (h *Handler) Payslip(asHR bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		uc, err := h.resolve(c)
		if err != nil {
			return err
		}
		claims := middleware.CurrentUser(c)
		if claims == nil {
			return apperrors.NewForbidden("Not authenticated")
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return apperrors.NewBadRequest("Invalid payslip ID format")
		}
		role := strings.ToLower(claims.Role)
		privileged := asHR && (role == "admin" || role == "manager")
		slip, err := uc.Payslip(h.ctx(c), id, claims.Email, privileged)
		if err != nil {
			return err
		}
		return response.OK(c, "Payslip retrieved", slip)
	}
}
