package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/auth/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	uc application.AuthUseCase
}

func NewHandler(uc application.AuthUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var dto application.RegisterDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	result, err := h.uc.Register(c.UserContext(), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Registration successful", result)
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var dto application.LoginDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	result, err := h.uc.Login(c.UserContext(), dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Login successful", result)
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return apperrors.NewUnauthorized("Authentication required")
	}

	result, err := h.uc.GetMe(c.UserContext(), claims.UserID)
	if err != nil {
		return err
	}

	return response.OK(c, "User profile retrieved successfully", result)
}

func (h *Handler) ListCompanies(c *fiber.Ctx) error {
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return apperrors.NewUnauthorized("Authentication required")
	}

	result, err := h.uc.ListCompanies(c.UserContext(), claims.UserID)
	if err != nil {
		return err
	}

	return response.OK(c, "Company memberships retrieved successfully", result)
}

func (h *Handler) ForgotPassword(c *fiber.Ctx) error {
	var dto application.ForgotPasswordDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	// Intentionally ignore the error: ForgotPassword never returns one for
	// enumeration-relevant cases, and the response is generic regardless of
	// whether the email exists.
	_ = h.uc.ForgotPassword(c.UserContext(), dto)

	return response.OK(c, "If an account with that email exists, a password reset link has been sent", nil)
}

func (h *Handler) ResetPassword(c *fiber.Ctx) error {
	var dto application.ResetPasswordDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	if err := h.uc.ResetPassword(c.UserContext(), dto); err != nil {
		return err
	}

	return response.OK(c, "Password has been reset successfully", nil)
}

func (h *Handler) SwitchCompany(c *fiber.Ctx) error {
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return apperrors.NewUnauthorized("Authentication required")
	}

	var dto application.SwitchCompanyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	result, err := h.uc.SwitchCompany(c.UserContext(), claims.UserID, dto.CompanyID)
	if err != nil {
		return err
	}

	return response.OK(c, "Company switched successfully", result)
}
