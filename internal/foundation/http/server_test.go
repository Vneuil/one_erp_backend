package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	foundationHttp "github.com/divinecoid/one-backend/internal/foundation/http"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestHealthCheck(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping db test: postgres not reachable: %v", err)
	}

	server := foundationHttp.NewServer(cfg, db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := server.App.Test(req, 2000)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var res response.SuccessResponse
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected success to be true, got false")
	}
}
