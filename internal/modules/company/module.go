package company

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/modules/company/application"
	"github.com/divinecoid/one-backend/internal/modules/company/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/company/domain"
	"github.com/divinecoid/one-backend/internal/modules/company/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module encapsulates company dependencies
type Module struct {
	Repo    domain.CompanyRepository
	UseCase application.CompanyUseCase
	Handler *http.Handler
}

// NewModule initializes the company module, auto-migrates domain entities, and registers routes
func NewModule(db *gorm.DB, router fiber.Router) *Module {
	// Auto-migrate company table
	if err := db.AutoMigrate(&domain.Company{}); err != nil {
		slog.Error("failed to migrate company schema", "error", err)
	}

	repo := infrastructure.NewCompanyRepository(db)
	uc := application.NewCompanyUseCase(repo)
	handler := http.NewHandler(uc)

	handler.RegisterRoutes(router)

	slog.Info("company module initialized")

	return &Module{
		Repo:    repo,
		UseCase: uc,
		Handler: handler,
	}
}
