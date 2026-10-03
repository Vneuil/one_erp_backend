package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	crmdomain "github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/application"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/infrastructure"
	tenantdomain "github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeTenantRepo struct {
	tenantdomain.TenantDatabaseRepository
	dbName string
}

func (f fakeTenantRepo) GetByCompanyID(_ context.Context, id uuid.UUID) (*tenantdomain.TenantDatabase, error) {
	return &tenantdomain.TenantDatabase{CompanyID: id, DatabaseName: f.dbName, Status: "active"}, nil
}

// End to end against real Postgres: the public form resolves the company through
// the control-plane index, opens that company's own database and creates a lead.
// Set CRMPLUS_HTTP_CONTROL_DB and CRMPLUS_HTTP_TENANT_DB to two scratch databases.
func TestPublicLeadFormEndToEnd(t *testing.T) {
	controlName, tenantName := os.Getenv("CRMPLUS_HTTP_CONTROL_DB"), os.Getenv("CRMPLUS_HTTP_TENANT_DB")
	if controlName == "" || tenantName == "" {
		t.Skip("CRMPLUS_HTTP_CONTROL_DB / CRMPLUS_HTTP_TENANT_DB not set")
	}
	dbCfg := &config.DatabaseConfig{Host: "localhost", Port: 5432, User: os.Getenv("USER"), SSLMode: "disable", MaxOpenConns: 5, MaxIdleConns: 2}
	open := func(name string) *gorm.DB {
		db, err := gorm.Open(postgres.Open(dbCfg.DSNForDatabase(name)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err != nil {
			t.Fatal(err)
		}
		return db
	}
	control, tenant := open(controlName), open(tenantName)
	if err := tenant.AutoMigrate(&crmdomain.Lead{}, &crmdomain.Deal{}, &domain.Interaction{}, &domain.PipelineStage{}, &domain.LeadForm{}, &domain.SalesTask{},
		&domain.Document{}, &domain.Tag{}, &domain.TagAssignment{}, &domain.Contact{}); err != nil {
		t.Fatal(err)
	}
	if err := control.AutoMigrate(&domain.LeadFormIndex{}); err != nil {
		t.Fatal(err)
	}

	manager := tenantMgr.NewManager(dbCfg, fakeTenantRepo{dbName: tenantName})
	defer manager.Close()
	registry := infrastructure.NewFormRegistry(control)
	h := NewHandler(registry)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	h.RegisterRoutes(app.Group("/api/v1"), "secret", manager, NewPublicHandler(registry, manager))

	company := uuid.New()
	svc := application.NewService(infrastructure.NewRepository(tenant), nil, nil, nil)
	form, err := svc.CreateLeadForm(context.Background(), registry, company, nil, application.LeadFormInput{Name: "Kontak Kami", Source: "web-form", DefaultPIC: "sales@x.com"})
	if err != nil {
		t.Fatal(err)
	}

	call := func(method, path string, body any) (int, map[string]any) {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.9")
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	base := "/api/v1/public/lead-forms/" + form.Key

	if code, out := call("GET", base, nil); code != 200 || out["data"].(map[string]any)["name"] != "Kontak Kami" {
		t.Fatalf("info: %d %v", code, out)
	}
	if code, _ := call("GET", "/api/v1/public/lead-forms/doesnotexist", nil); code != 404 {
		t.Fatalf("unknown key must be 404, got %d", code)
	}
	if code, _ := call("POST", base+"/submit", map[string]string{"name": "Sari", "email": "sari@x.com", "message": "Halo"}); code != 200 {
		t.Fatalf("submit: %d", code)
	}
	var lead crmdomain.Lead
	if err := tenant.Where("name = ?", "Sari").First(&lead).Error; err != nil || lead.Source != "web-form" || lead.PIC != "sales@x.com" {
		t.Fatalf("lead not created in the tenant database: %+v %v", lead, err)
	}
	if code, _ := call("POST", base+"/submit", map[string]string{"name": "Nobody"}); code != 400 {
		t.Fatalf("missing contact details must be 400, got %d", code)
	}
	// Honeypot: looks successful, stores nothing.
	if code, _ := call("POST", base+"/submit", map[string]string{"name": "Botty", "email": "b@x.com", "website": "spam.example"}); code != 200 {
		t.Fatalf("honeypot response: %d", code)
	}
	var n int64
	tenant.Model(&crmdomain.Lead{}).Where("name = ?", "Botty").Count(&n)
	if n != 0 {
		t.Fatal("honeypot submission must not create a lead")
	}
	// Deactivate: the form disappears from the public side.
	off := false
	if _, err := svc.UpdateLeadForm(context.Background(), form.ID, application.LeadFormInput{IsActive: &off}); err != nil {
		t.Fatal(err)
	}
	if code, _ := call("POST", base+"/submit", map[string]string{"name": "Late", "email": "l@x.com"}); code != 404 {
		t.Fatalf("inactive form must be 404, got %d", code)
	}
	// Rate limit per client IP: 20 requests per minute, all used above except the rest here.
	limited := false
	for i := 0; i < 25; i++ {
		if code, _ := call("GET", "/api/v1/public/lead-forms/doesnotexist", nil); code == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("the public endpoints must be rate limited")
	}
	// Delete: key unregistered.
	if err := svc.DeleteLeadForm(context.Background(), registry, form.ID); err != nil {
		t.Fatal(err)
	}
	if idx, _ := registry.Lookup(context.Background(), form.Key); idx != nil {
		t.Fatal("deleting a form must unregister its key")
	}
}
