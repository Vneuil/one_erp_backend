package hrops

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/hrops/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

// NewModule is tenant-scoped: its tables live in each company's own database.
func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager, store storage.Storage) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "hrops",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.AttendanceCorrection{}, &domain.Shift{}, &domain.ShiftAssignment{}, &domain.ShiftChangeRequest{},
				&domain.OvertimeRecord{}, &domain.EmployeeDocument{}, &domain.Feedback{},
				&domain.Announcement{}, &domain.Notification{},
				&domain.ReimbursementEvidence{}, &domain.CanteenItem{}, &domain.CanteenOrder{}, &domain.VisitStop{}, &domain.CashAdvance{},
			)
		},
	})

	handler := http.NewHandler(store)
	handler.RegisterRoutes(router, jwtSecret, manager)
	slog.Info("hrops module initialized (tenant-scoped)")
	return &Module{Handler: handler}
}
