package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/tenant/application"
	"github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	uc application.TenantUseCase
}

func NewHandler(uc application.TenantUseCase) *Handler {
	return &Handler{uc: uc}
}

// Provision looks up (or provisions) the tenant database for the caller's
// active company and returns its registry record.
func (h *Handler) Provision(c *fiber.Ctx) error {
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return apperrors.NewUnauthorized("Authentication required")
	}
	if claims.CompanyID == nil {
		return apperrors.NewBadRequest("No active company")
	}

	result, err := h.uc.GetOrCreateTenantDatabase(c.UserContext(), *claims.CompanyID)
	if err != nil {
		return err
	}
	return response.OK(c, "Tenant database ready", result)
}

// BackfillSchema re-runs every registered module's tenant schema migrator
// (see foundation/tenant.ProvisionAll) against the caller's already-
// provisioned tenant database. New tenants get every module's schema for
// free at creation time via GetOrCreateTenantDatabase, but that only runs
// once - a module registering new tables/columns after a tenant already
// exists (the normal case as this codebase evolves) never reaches that
// tenant's database until this is called. Idempotent (AutoMigrate only
// creates what's missing), so safe to call repeatedly or after any deploy
// that added schema.
func (h *Handler) BackfillSchema(c *fiber.Ctx) error {
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return apperrors.NewUnauthorized("Authentication required")
	}
	if claims.CompanyID == nil {
		return apperrors.NewBadRequest("No active company")
	}

	if err := h.uc.ProvisionTenantSchema(c.UserContext(), *claims.CompanyID); err != nil {
		return err
	}
	return response.OK(c, "Tenant schema is up to date", nil)
}

// Ping demonstrates per-tenant data isolation: it inserts one row into the
// caller's tenant database and returns the resulting row count.
func (h *Handler) Ping(c *fiber.Ctx) error {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return apperrors.NewBadRequest("No active company")
	}

	if err := tenantDB.AutoMigrate(&domain.TenantDemoPing{}); err != nil {
		return apperrors.NewInternal(err, "Failed to migrate tenant demo schema")
	}

	if err := tenantDB.WithContext(c.UserContext()).Create(&domain.TenantDemoPing{}).Error; err != nil {
		return apperrors.NewInternal(err, "Failed to insert demo ping")
	}

	var count int64
	if err := tenantDB.WithContext(c.UserContext()).Model(&domain.TenantDemoPing{}).Count(&count).Error; err != nil {
		return apperrors.NewInternal(err, "Failed to count demo pings")
	}

	return response.OK(c, "Pong", fiber.Map{"pingCount": count})
}
