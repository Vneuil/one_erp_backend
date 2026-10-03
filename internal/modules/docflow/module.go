package docflow

import (
	"context"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/ai"
	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"github.com/divinecoid/one-backend/internal/modules/docflow/application"
	"github.com/divinecoid/one-backend/internal/modules/docflow/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/docflow/domain"
	"github.com/divinecoid/one-backend/internal/modules/docflow/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	FormsRepo         domain.FormsRepository
	SignaturesRepo    domain.SignaturesRepository
	MeetingsRepo      domain.MeetingsRepository
	FormsUseCase      application.FormsUseCase
	SignaturesUseCase application.SignaturesUseCase
	MeetingsUseCase   application.MeetingsUseCase
	Handler           *http.Handler
}

func NewModule(db *gorm.DB, router fiber.Router, aiClient ai.Client, store storage.Storage, jwtSecret string) *Module {
	if err := db.AutoMigrate(
		&domain.Form{},
		&domain.FormResponse{},
		&domain.SignatureDocument{},
		&domain.SignatureRequest{},
		&domain.Meeting{},
		&domain.MeetingNote{},
	); err != nil {
		slog.Error("failed to migrate docflow schema", "error", err)
	}

	formsRepo := infrastructure.NewFormsRepository(db)
	signaturesRepo := infrastructure.NewSignaturesRepository(db)
	meetingsRepo := infrastructure.NewMeetingsRepository(db)

	formsUC := application.NewFormsUseCase(formsRepo)
	signaturesUC := application.NewSignaturesUseCase(signaturesRepo, store)
	meetingsUC := application.NewMeetingsUseCase(meetingsRepo, aiClient)

	handler := http.NewHandler(formsUC, signaturesUC, meetingsUC)
	handler.RegisterRoutes(router, jwtSecret)

	// Sample chat channels, drive files, forms and meetings are demo data.
	if tenantMgr.DemoSeedingEnabled() {
		go func() {
			if err := formsUC.SeedInitialData(context.Background()); err != nil {
				slog.Warn("failed to seed initial docflow forms data", "error", err)
			}
			if err := signaturesUC.SeedInitialData(context.Background()); err != nil {
				slog.Warn("failed to seed initial docflow signatures data", "error", err)
			}
			if err := meetingsUC.SeedInitialData(context.Background()); err != nil {
				slog.Warn("failed to seed initial docflow meetings data", "error", err)
			}
		}()
	}

	slog.Info("docflow module initialized")

	return &Module{
		FormsRepo:         formsRepo,
		SignaturesRepo:    signaturesRepo,
		MeetingsRepo:      meetingsRepo,
		FormsUseCase:      formsUC,
		SignaturesUseCase: signaturesUC,
		MeetingsUseCase:   meetingsUC,
		Handler:           handler,
	}
}
