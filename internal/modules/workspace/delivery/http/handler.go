package http

import (
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/workspace/application"
	"github.com/divinecoid/one-backend/internal/modules/workspace/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	jwtSecret string
	jwtExpiry time.Duration
}

func NewHandler(jwtSecret string, jwtExpiry time.Duration) *Handler {
	return &Handler{jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
}

func (h *Handler) useCaseForDB(tenantDB *gorm.DB) application.TenantUseCase {
	repo := infrastructure.NewTenantRepository(tenantDB)
	return application.NewTenantUseCase(repo, h.jwtSecret, h.jwtExpiry)
}

func (h *Handler) resolve(c *fiber.Ctx) (*gorm.DB, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	return tenantDB, nil
}

func (h *Handler) List(c *fiber.Ctx) error {
	db, err := h.resolve(c)
	if err != nil {
		return err
	}
	uc := h.useCaseForDB(db)
	tenants, err := uc.ListTenants(c.UserContext())
	if err != nil {
		return err
	}
	return response.OK(c, "Tenants retrieved successfully", tenants)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	db, err := h.resolve(c)
	if err != nil {
		return err
	}
	uc := h.useCaseForDB(db)
	var dto application.CreateTenantDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	t, err := uc.CreateTenant(c.UserContext(), dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Tenant created successfully", t)
}

func (h *Handler) Switch(c *fiber.Ctx) error {
	db, err := h.resolve(c)
	if err != nil {
		return err
	}
	uc := h.useCaseForDB(db)
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return apperrors.NewUnauthorized("Missing session")
	}
	var body struct {
		TenantID uuid.UUID `json:"tenantId"`
	}
	if err := c.BodyParser(&body); err != nil || body.TenantID == uuid.Nil {
		return apperrors.NewBadRequest("tenantId is required")
	}
	result, err := uc.SwitchTenant(c.UserContext(), claims, body.TenantID)
	if err != nil {
		return err
	}
	return response.OK(c, "Tenant switched successfully", result)
}
