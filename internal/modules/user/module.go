package user

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/modules/user/application"
	"github.com/divinecoid/one-backend/internal/modules/user/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/user/domain"
	"github.com/divinecoid/one-backend/internal/modules/user/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Repo    domain.UserRepository
	UseCase application.UserUseCase
	Handler *http.Handler
}

// NewModule initializes the user module, auto-migrates domain entities, and registers routes
func NewModule(db *gorm.DB, router fiber.Router, jwtSecret string) *Module {
	// Auto-migrate user table
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		slog.Error("failed to migrate user schema", "error", err)
	}

	repo := infrastructure.NewUserRepository(db)
	uc := application.NewUserUseCase(repo)
	handler := http.NewHandler(uc)

	handler.RegisterRoutes(router, jwtSecret)

	slog.Info("user module initialized")

	return &Module{
		Repo:    repo,
		UseCase: uc,
		Handler: handler,
	}
}
