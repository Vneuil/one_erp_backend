package rbac

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/modules/rbac/application"
	"github.com/divinecoid/one-backend/internal/modules/rbac/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/rbac/domain"
	"github.com/divinecoid/one-backend/internal/modules/rbac/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Repo    domain.RBACRepository
	UseCase application.RBACUseCase
	Handler *http.Handler
}

// NewModule initializes the RBAC module against the control-plane database
// (like modules/user and modules/company - Roles/Permissions govern Users,
// which are control-plane data, not per-tenant).
func NewModule(db *gorm.DB, router fiber.Router, jwtSecret string) *Module {
	// The can_approve column is new. On the first migration that adds it, keep
	// existing built-in Admin/Manager roles able to approve (they could before
	// this grant existed); custom roles start without it. The check runs once,
	// only when the column is missing, so later edits by an admin are never
	// overwritten.
	needsApproveBackfill := db.Migrator().HasTable(&domain.Permission{}) && !db.Migrator().HasColumn(&domain.Permission{}, "can_approve")
	needsPricesBackfill := db.Migrator().HasTable(&domain.Permission{}) && !db.Migrator().HasColumn(&domain.Permission{}, "can_view_prices")
	needsGranularBackfill := db.Migrator().HasTable(&domain.Permission{}) && !db.Migrator().HasColumn(&domain.Permission{}, "can_create")

	if err := db.AutoMigrate(&domain.Role{}, &domain.Permission{}); err != nil {
		slog.Error("failed to migrate rbac schema", "error", err)
	}
	if needsApproveBackfill {
		if err := db.Exec(`UPDATE role_permissions SET can_approve = can_manage
			WHERE role_id IN (SELECT id FROM roles WHERE is_system = true AND name IN ('Admin', 'Manager'))`).Error; err != nil {
			slog.Error("failed to backfill rbac approve permissions", "error", err)
		}
	}
	if needsPricesBackfill {
		if err := db.Exec(`UPDATE role_permissions SET can_view_prices = true
			WHERE role_id IN (SELECT id FROM roles WHERE is_system = true AND name IN ('Admin', 'Manager'))`).Error; err != nil {
			slog.Error("failed to backfill rbac view prices permissions", "error", err)
		}
	}
	if needsGranularBackfill {
		// Backfill existing permissions:
		// Existing permissions with can_manage = true receive can_create, can_edit, and can_submit.
		if err := db.Exec(`UPDATE role_permissions SET 
			can_create = can_manage, 
			can_edit = can_manage, 
			can_submit = can_manage 
			WHERE can_manage = true`).Error; err != nil {
			slog.Error("failed to backfill granular permissions from can_manage", "error", err)
		}

		// Ensure Admin system role gets delete and edit prices everywhere
		if err := db.Exec(`UPDATE role_permissions SET 
			can_delete = true,
			can_edit_prices = true
			WHERE role_id IN (SELECT id FROM roles WHERE is_system = true AND name = 'Admin')`).Error; err != nil {
			slog.Error("failed to backfill admin delete and price permissions", "error", err)
		}

		// Manager gets delete and edit prices where they have can_manage
		if err := db.Exec(`UPDATE role_permissions SET 
			can_delete = can_manage,
			can_edit_prices = can_manage
			WHERE role_id IN (SELECT id FROM roles WHERE is_system = true AND name = 'Manager')`).Error; err != nil {
			slog.Error("failed to backfill manager delete and price permissions", "error", err)
		}
	}

	repo := infrastructure.NewRBACRepository(db)
	uc := application.NewRBACUseCase(repo)
	handler := http.NewHandler(uc)

	handler.RegisterRoutes(router, jwtSecret)

	slog.Info("rbac module initialized")

	return &Module{
		Repo:    repo,
		UseCase: uc,
		Handler: handler,
	}
}
