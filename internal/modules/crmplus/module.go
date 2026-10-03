package crmplus

import (
	"context"
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/application"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct{ Handler *http.Handler }

// NewModule is tenant-scoped, plus one control-plane table (the public form
// index) for the same reason as whatsappregistry: a public submission has no
// session, so the owning company must be found before its database can be opened.
func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager, controlPlaneDB *gorm.DB, store storage.Storage) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "crmplus",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Interaction{}, &domain.SalesTask{}, &domain.Document{}, &domain.Tag{}, &domain.TagAssignment{},
				&domain.Contact{}, &domain.PipelineStage{}, &domain.LeadForm{})
		},
		// The default pipeline is what makes deals work at all, so it is
		// reference data every company needs, not demo data.
		SeedKind: tenantMgr.SeedReference,
		Seed: func(tenantDB *gorm.DB) error {
			return application.NewService(infrastructure.NewRepository(tenantDB), nil, nil, nil).SeedDefaultStages(context.Background())
		},
	})

	if err := controlPlaneDB.AutoMigrate(&domain.LeadFormIndex{}); err != nil {
		slog.Error("crmplus: failed to migrate control-plane crm_lead_form_index", "error", err)
	}
	registry := infrastructure.NewFormRegistry(controlPlaneDB)

	handler := http.NewHandler(registry).WithStorage(store)
	handler.RegisterRoutes(router, jwtSecret, manager, http.NewPublicHandler(registry, manager))
	slog.Info("crmplus module initialized (tenant-scoped)")
	return &Module{Handler: handler}
}
