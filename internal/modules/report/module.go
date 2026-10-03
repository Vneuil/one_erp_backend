package report

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/report/delivery/http"
	"github.com/gofiber/fiber/v2"
)

// Module is tenant-scoped: report has no schema of its own, it just runs
// cross-module read queries against the caller's tenant database, so there
// is nothing to register with foundation/tenant.RegisterSchema - each
// request simply resolves a fresh *application.ReportQuery bound to
// middleware.TenantDB(c).
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("report module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
