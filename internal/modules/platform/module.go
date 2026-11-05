package platform

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/license"
	"github.com/divinecoid/one-backend/internal/modules/platform/delivery/http"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(db *gorm.DB, router fiber.Router) *Module {
	if err := license.AutoMigrate(db); err != nil {
		slog.Error("failed to migrate platform subscription schema", "error", err)
	}

	handler := http.NewHandler(db)
	handler.RegisterRoutes(router)

	slog.Info("platform module initialized")

	return &Module{
		Handler: handler,
	}
}

