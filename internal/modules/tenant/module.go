package tenant

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/tenant/application"
	"github.com/divinecoid/one-backend/internal/modules/tenant/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	"github.com/divinecoid/one-backend/internal/modules/tenant/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Repo           domain.TenantDatabaseRepository
	MembershipRepo domain.CompanyMembershipRepository
	UseCase        application.TenantUseCase
	Handler        *http.Handler
}

// NewModule wires up the tenant registry, provisioning, and demo-isolation
// routes. db is the central (control-plane) database - the same shared
// database the other 20 ERP modules already use.
func NewModule(db *gorm.DB, router fiber.Router, jwtSecret string, dbCfg *config.DatabaseConfig, manager *tenantMgr.Manager) *Module {
	if err := db.AutoMigrate(
		&domain.TenantDatabase{},
		&domain.CompanyMembership{},
	); err != nil {
		slog.Error("failed to migrate tenant registry schema", "error", err)
	}

	repo := infrastructure.NewTenantDatabaseRepository(db)
	membershipRepo := infrastructure.NewCompanyMembershipRepository(db)
	uc := application.NewTenantUseCase(repo, dbCfg)
	handler := http.NewHandler(uc)

	handler.RegisterRoutes(router, jwtSecret, manager)

	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "tenant-demo",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.TenantDemoPing{})
		},
	})

	slog.Info("tenant module initialized")

	return &Module{
		Repo:           repo,
		MembershipRepo: membershipRepo,
		UseCase:        uc,
		Handler:        handler,
	}
}
