package collaboration

import (
	"context"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/application"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	ChatRepo    domain.ChatRepository
	DriveRepo   domain.DriveRepository
	ExportsRepo domain.ExportsRepository
	Handler     *http.Handler
}

func NewModule(db *gorm.DB, router fiber.Router, store storage.Storage, jwtSecret string) *Module {
	if err := db.AutoMigrate(
		&domain.Conversation{},
		&domain.Message{},
		&domain.Folder{},
		&domain.File{},
		&domain.ExportJob{},
	); err != nil {
		slog.Error("failed to migrate collaboration schema", "error", err)
	}

	chatRepo := infrastructure.NewChatRepository(db)
	driveRepo := infrastructure.NewDriveRepository(db)
	exportsRepo := infrastructure.NewExportsRepository(db)

	chatUC := application.NewChatUseCase(chatRepo)
	driveUC := application.NewDriveUseCase(driveRepo, store)
	exportsUC := application.NewExportsUseCase(exportsRepo)

	handler := http.NewHandler(chatUC, driveUC, exportsUC)
	handler.RegisterRoutes(router, jwtSecret)

	// Sample chat channels, drive files, forms and meetings are demo data.
	if tenantMgr.DemoSeedingEnabled() {
		go func() {
			if err := chatUC.SeedInitialData(context.Background()); err != nil {
				slog.Warn("failed to seed initial collaboration chat data", "error", err)
			}
			if err := driveUC.SeedInitialData(context.Background()); err != nil {
				slog.Warn("failed to seed initial collaboration drive data", "error", err)
			}
			if err := exportsUC.SeedInitialData(context.Background()); err != nil {
				slog.Warn("failed to seed initial collaboration exports data", "error", err)
			}
		}()
	}

	slog.Info("collaboration module initialized")

	return &Module{
		ChatRepo:    chatRepo,
		DriveRepo:   driveRepo,
		ExportsRepo: exportsRepo,
		Handler:     handler,
	}
}
