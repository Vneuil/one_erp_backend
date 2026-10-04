package main

import (
	"github.com/google/uuid"
	"log"
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/ai"
	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/divinecoid/one-backend/internal/foundation/database"
	"github.com/divinecoid/one-backend/internal/foundation/http"
	"github.com/divinecoid/one-backend/internal/foundation/logger"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/notify"
	"github.com/divinecoid/one-backend/internal/foundation/storage"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/activitylog"
	"github.com/divinecoid/one-backend/internal/modules/approval"
	"github.com/divinecoid/one-backend/internal/modules/assets"
	"github.com/divinecoid/one-backend/internal/modules/auth"
	"github.com/divinecoid/one-backend/internal/modules/banking"
	"github.com/divinecoid/one-backend/internal/modules/checkout"
	"github.com/divinecoid/one-backend/internal/modules/collaboration"
	"github.com/divinecoid/one-backend/internal/modules/commission"
	"github.com/divinecoid/one-backend/internal/modules/company"
	"github.com/divinecoid/one-backend/internal/modules/contracts"
	"github.com/divinecoid/one-backend/internal/modules/cooperative"
	"github.com/divinecoid/one-backend/internal/modules/crm"
	"github.com/divinecoid/one-backend/internal/modules/crmplus"
	"github.com/divinecoid/one-backend/internal/modules/currency"
	"github.com/divinecoid/one-backend/internal/modules/customer"
	"github.com/divinecoid/one-backend/internal/modules/device"
	"github.com/divinecoid/one-backend/internal/modules/docflow"
	"github.com/divinecoid/one-backend/internal/modules/finance"
	"github.com/divinecoid/one-backend/internal/modules/goal"
	"github.com/divinecoid/one-backend/internal/modules/hrletters"
	"github.com/divinecoid/one-backend/internal/modules/hrm"
	"github.com/divinecoid/one-backend/internal/modules/hrops"
	"github.com/divinecoid/one-backend/internal/modules/integration"
	"github.com/divinecoid/one-backend/internal/modules/inventory"
	"github.com/divinecoid/one-backend/internal/modules/kpi"
	"github.com/divinecoid/one-backend/internal/modules/leave"
	"github.com/divinecoid/one-backend/internal/modules/lms"
	"github.com/divinecoid/one-backend/internal/modules/loyalty"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing"
	"github.com/divinecoid/one-backend/internal/modules/marketplace"
	"github.com/divinecoid/one-backend/internal/modules/omnichannel"
	"github.com/divinecoid/one-backend/internal/modules/payroll"
	"github.com/divinecoid/one-backend/internal/modules/pos"
	"github.com/divinecoid/one-backend/internal/modules/pricing"
	"github.com/divinecoid/one-backend/internal/modules/procurement"
	"github.com/divinecoid/one-backend/internal/modules/product"
	"github.com/divinecoid/one-backend/internal/modules/project"
	"github.com/divinecoid/one-backend/internal/modules/projectcost"
	"github.com/divinecoid/one-backend/internal/modules/projecttask"
	"github.com/divinecoid/one-backend/internal/modules/projectticket"
	"github.com/divinecoid/one-backend/internal/modules/provisioning"
	"github.com/divinecoid/one-backend/internal/modules/rbac"
	"github.com/divinecoid/one-backend/internal/modules/recruitment"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement"
	"github.com/divinecoid/one-backend/internal/modules/report"
	"github.com/divinecoid/one-backend/internal/modules/sales"
	"github.com/divinecoid/one-backend/internal/modules/salesman"
	"github.com/divinecoid/one-backend/internal/modules/shipping"
	"github.com/divinecoid/one-backend/internal/modules/supplier"
	"github.com/divinecoid/one-backend/internal/modules/support"
	"github.com/divinecoid/one-backend/internal/modules/systemlogs"
	"github.com/divinecoid/one-backend/internal/modules/tax"
	"github.com/divinecoid/one-backend/internal/modules/tenant"
	tenantInfra "github.com/divinecoid/one-backend/internal/modules/tenant/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/user"
	"github.com/divinecoid/one-backend/internal/modules/warehouse"
	"github.com/divinecoid/one-backend/internal/modules/workspace"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	tenantMgr.SetDemoSeeding(cfg.App.SeedDemoData)

	// 2. Initialize Structured Logger
	logger.Init(cfg.App.Env)
	slog.Info("initializing application", "name", cfg.App.Name, "env", cfg.App.Env)

	// 3. Connect to PostgreSQL
	db, err := database.Connect(&cfg.Database, cfg.App.IsDevelopment())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		log.Fatalf("database connection error: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			slog.Error("failed to close database connection", "error", err)
		}
	}()

	// 4. Create HTTP Server
	server := http.NewServer(cfg, db)

	// Tenant registry (control-plane) + per-tenant connection pool manager. Created
	// before the middleware below because member onboarding needs it.
	tenantRegistryRepo := tenantInfra.NewTenantDatabaseRepository(db)
	tenantManager := tenantMgr.NewManager(&cfg.Database, tenantRegistryRepo)

	// License enforcement: blocks tenant business-data requests once a
	// company's trial/paid subscription has actually lapsed. Mounted before
	// any module registers routes; exempts /auth, /tenant, /companies and
	// /platform (billing) so an expired customer can still log in and pay.
	server.V1.Use(middleware.RequireActiveLicense(db, cfg.JWT.Secret))
	// Deactivated-user enforcement: blocks API access for a user as soon as
	// an admin deactivates their account, instead of waiting for their
	// existing JWT to expire naturally. See RequireActiveUser's doc comment.
	server.V1.Use(middleware.RequireActiveUser(db, cfg.JWT.Secret))
	server.V1.Use(rbac.Enforce(db))
	// Approval-type actions (approve/reject/pay/post) additionally need approval
	// rights: admin, a manager account, or a custom role with CanApprove.
	server.V1.Use(rbac.ApprovalGate(db))
	// Make the caller's identity available to use cases (self-approval checks).
	server.V1.Use(middleware.ActorContext())
	// First use of a company by a member creates their HR employee record.
	server.V1.Use(middleware.EnsureUserOnboarded(cfg.App.AutoOnboardEmployees, tenantManager, func(id uuid.UUID) string {
		var name string
		db.Table("users").Select("name").Where("id = ?", id).Scan(&name)
		return name
	}, tenantMgr.RunUserOnboarding))
	server.V1.Use(activitylog.Logger())

	// 5. Initialize Modules (Modular Monolith Dependency Injection)
	companyMod := company.NewModule(db, server.V1)
	checkout.NewModule(db, server.V1, cfg.Xendit)
	userMod := user.NewModule(db, server.V1, cfg.JWT.Secret)
	rbac.NewModule(db, server.V1, cfg.JWT.Secret)

	tenantMod := tenant.NewModule(db, server.V1, cfg.JWT.Secret, &cfg.Database, tenantManager)

	notifier := notify.New(cfg.Email, cfg.WhatsApp)

	auth.NewModule(db, userMod.Repo, companyMod.Repo, tenantMod.MembershipRepo, notifier, cfg.Frontend.BaseURL, server.V1, cfg.JWT.Secret, cfg.JWT.ExpiresIn)

	// Internal, service-to-service endpoints for the standalone abc-backend
	// (Associate Business Consultant program) to provision real companies
	// and tenant databases here - mounted on the raw app, outside /api/v1,
	// so it never passes through the license/RBAC/activity-log middlewares
	// above, which expect a normal user JWT this caller will never have.
	internalGroup := server.App.Group("/internal")
	provisioning.NewModule(
		internalGroup,
		companyMod.Repo, userMod.Repo, tenantMod.MembershipRepo, tenantMod.UseCase,
		cfg.JWT.Secret, cfg.JWT.ExpiresIn, cfg.Internal.ProvisioningToken,
	)

	product.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	customer.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	salesman.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	pricing.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	activitylog.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	approval.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	currency.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	shipping.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	supplier.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	sales.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	marketplace.NewModule(server.V1, cfg.JWT.Secret, tenantManager, cfg.Frontend.BaseURL, cfg.TikTokShop, cfg.Shopee, cfg.Blibli, cfg.Lazada)
	omnichannel.NewModule(server.V1, cfg.JWT.Secret, tenantManager, notifier, cfg.WhatsApp, db)
	workspace.NewModule(server.V1, cfg.JWT.Secret, cfg.JWT.ExpiresIn, tenantManager)
	loyalty.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	pos.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	crm.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	objectStore := storage.NewWithLocalFallback(cfg.R2, cfg.Storage.LocalDir)
	crmplus.NewModule(server.V1, cfg.JWT.Secret, tenantManager, db, objectStore)
	project.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	projectcost.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	hrm.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	hrops.NewModule(server.V1, cfg.JWT.Secret, tenantManager, objectStore)
	leave.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	hrletters.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	kpi.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	cooperative.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	reimbursement.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	payroll.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	projecttask.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	projectticket.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	finance.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	tax.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	inventory.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	procurement.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	banking.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	manufacturing.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	warehouse.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	assets.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	contracts.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	commission.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	recruitment.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	lms.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	device.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	goal.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	support.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	systemlogs.NewModule(server.V1, cfg.JWT.Secret)
	integration.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	report.NewModule(server.V1, cfg.JWT.Secret, tenantManager)
	collaboration.NewModule(db, server.V1, objectStore, cfg.JWT.Secret)
	aiClient := ai.NewClient(cfg.OpenAI)
	docflow.NewModule(db, server.V1, aiClient, objectStore, cfg.JWT.Secret)

	// 6. Start HTTP Server with Graceful Shutdown
	if err := server.Start(); err != nil {
		slog.Error("server stopped with error", "error", err)
	}
	tenantManager.Close()
}
