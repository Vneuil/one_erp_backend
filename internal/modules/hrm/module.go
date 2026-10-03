package hrm

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/hrm/application"
	"github.com/divinecoid/one-backend/internal/modules/hrm/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "hrm",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Employee{}, &domain.Attendance{}, &domain.AttendanceLocation{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewHRMRepository(tenantDB)
			uc := application.NewHRMUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	// Give every company member an employee record the first time they use the
	// company, so leave, attendance and reimbursements work without manual setup.
	tenantMgr.RegisterUserOnboarding(func(ctx context.Context, tenantDB *gorm.DB, u tenantMgr.OnboardingUser) error {
		_, err := application.EnsureEmployeeForMember(ctx, infrastructure.NewHRMRepository(tenantDB), application.MemberIdentity{
			Name: u.Name, Email: u.Email, Role: u.Role, TenantID: u.TenantID,
		})
		return err
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("HRM module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
