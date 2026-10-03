package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	foundationHttp "github.com/divinecoid/one-backend/internal/foundation/http"
	"github.com/divinecoid/one-backend/internal/foundation/notify"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/auth"
	authApp "github.com/divinecoid/one-backend/internal/modules/auth/application"
	"github.com/divinecoid/one-backend/internal/modules/company"
	"github.com/divinecoid/one-backend/internal/modules/tenant"
	tenantInfra "github.com/divinecoid/one-backend/internal/modules/tenant/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/user"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupAuthTestApp(t *testing.T) (*foundationHttp.Server, *config.Config) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	server := foundationHttp.NewServer(cfg, db)
	companyMod := company.NewModule(db, server.V1)
	userMod := user.NewModule(db, server.V1, cfg.JWT.Secret)
	tenantRegistryRepo := tenantInfra.NewTenantDatabaseRepository(db)
	tenantManager := tenantMgr.NewManager(&cfg.Database, tenantRegistryRepo)
	tenantMod := tenant.NewModule(db, server.V1, cfg.JWT.Secret, &cfg.Database, tenantManager)
	notifier := notify.New(cfg.Email, cfg.WhatsApp)
	auth.NewModule(db, userMod.Repo, companyMod.Repo, tenantMod.MembershipRepo, notifier, cfg.Frontend.BaseURL, server.V1, cfg.JWT.Secret, cfg.JWT.ExpiresIn)

	return server, cfg
}

func TestAuthFlow(t *testing.T) {
	server, _ := setupAuthTestApp(t)

	uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	email := fmt.Sprintf("owner_%s@erpcorp.com", uniqueSuffix)
	password := "Secret123!"
	compCode := fmt.Sprintf("CORP%s", uniqueSuffix)

	// 1. Register
	regPayload := authApp.RegisterDTO{
		CompanyName: "ERP Corp " + uniqueSuffix,
		CompanyCode: compCode,
		Name:        "Owner Person",
		Email:       email,
		Password:    password,
	}
	regBytes, _ := json.Marshal(regPayload)

	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBytes))
	regReq.Header.Set("Content-Type", "application/json")
	regResp, err := server.App.Test(regReq, 3000)
	if err != nil || regResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(regResp.Body)
		t.Fatalf("register failed, status: %d, body: %s", regResp.StatusCode, string(body))
	}

	var regResult struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	_ = json.NewDecoder(regResp.Body).Decode(&regResult)
	if !regResult.Success || regResult.Data.AccessToken == "" {
		t.Fatalf("register response missing token: %+v", regResult)
	}
	token := regResult.Data.AccessToken

	// 2. Login
	loginPayload := authApp.LoginDTO{
		Email:    email,
		Password: password,
	}
	loginBytes, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBytes))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := server.App.Test(loginReq, 3000)
	if err != nil || loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed, status: %d", loginResp.StatusCode)
	}

	// 3. Get /me without token (should fail 401)
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	unauthResp, err := server.App.Test(unauthReq, 3000)
	if err != nil || unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauth request, got: %d", unauthResp.StatusCode)
	}

	// 4. Get /me with valid Bearer token (should succeed 200)
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meResp, err := server.App.Test(meReq, 3000)
	if err != nil || meResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for me request, got: %d", meResp.StatusCode)
	}

	var meResult struct {
		Success bool `json:"success"`
		Data    struct {
			User struct {
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"user"`
			Company struct {
				Code string `json:"code"`
			} `json:"company"`
		} `json:"data"`
	}
	_ = json.NewDecoder(meResp.Body).Decode(&meResult)
	if meResult.Data.User.Email != email || meResult.Data.Company.Code != compCode {
		t.Fatalf("unexpected me response: %+v", meResult)
	}

	// 5. Access Protected /users with admin token
	usersReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	usersReq.Header.Set("Authorization", "Bearer "+token)
	usersResp, err := server.App.Test(usersReq, 3000)
	if err != nil || usersResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for users list, got: %d", usersResp.StatusCode)
	}
}
