package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string) {
	auth := router.Group("/auth")

	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
	auth.Post("/forgot-password", h.ForgotPassword)
	auth.Post("/reset-password", h.ResetPassword)
	auth.Get("/me", middleware.Protected(jwtSecret), h.GetMe)
	auth.Get("/companies", middleware.Protected(jwtSecret), h.ListCompanies)
	auth.Post("/switch-company", middleware.Protected(jwtSecret), h.SwitchCompany)
}
