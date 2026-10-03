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
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/company"
	"github.com/divinecoid/one-backend/internal/modules/company/application"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestApp(t *testing.T) (*foundationHttp.Server, *gorm.DB) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	server := foundationHttp.NewServer(cfg, db)
	company.NewModule(db, server.V1)

	return server, db
}

func TestCompanyCRUD(t *testing.T) {
	server, _ := setupTestApp(t)

	code := fmt.Sprintf("CMP_%d", time.Now().UnixNano()%1000000)

	// 1. Create Company
	createPayload := application.CreateCompanyDTO{
		Code:     code,
		Name:     "PT Test Mandiri Perkasa",
		Email:    "test@mandiri.com",
		Phone:    "08123456789",
		Address:  "Jl. Sudirman No. 1, Jakarta",
		TaxID:    "01.234.567.8-901.000",
		Currency: "IDR",
	}
	bodyBytes, _ := json.Marshal(createPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := server.App.Test(req, 3000)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 201, got %d. Body: %s", resp.StatusCode, string(respBody))
	}

	var createResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createResp)
	if !createResp.Success || createResp.Data.ID == "" {
		t.Fatalf("unexpected create response: %+v", createResp)
	}
	companyID := createResp.Data.ID

	// 2. Get Company by ID
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/companies/"+companyID, nil)
	getResp, err := server.App.Test(getReq, 3000)
	if err != nil || getResp.StatusCode != http.StatusOK {
		t.Fatalf("get by id failed, status: %d", getResp.StatusCode)
	}

	// 3. List Companies
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/companies?page=1&perPage=10&search="+code, nil)
	listResp, err := server.App.Test(listReq, 3000)
	if err != nil || listResp.StatusCode != http.StatusOK {
		t.Fatalf("list companies failed, status: %d", listResp.StatusCode)
	}

	var listData response.SuccessResponse
	_ = json.NewDecoder(listResp.Body).Decode(&listData)
	if !listData.Success {
		t.Fatalf("expected list success to be true")
	}

	// 4. Update Company
	newName := "PT Test Mandiri Perkasa Tbk"
	updatePayload := application.UpdateCompanyDTO{
		Name: &newName,
	}
	updateBytes, _ := json.Marshal(updatePayload)

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyID, bytes.NewReader(updateBytes))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := server.App.Test(updateReq, 3000)
	if err != nil || updateResp.StatusCode != http.StatusOK {
		t.Fatalf("update company failed, status: %d", updateResp.StatusCode)
	}

	// 5. Delete Company
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/companies/"+companyID, nil)
	delResp, err := server.App.Test(delReq, 3000)
	if err != nil || delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete company failed, status: %d", delResp.StatusCode)
	}

	// 6. Verify Delete (Should return 404)
	verifyReq := httptest.NewRequest(http.MethodGet, "/api/v1/companies/"+companyID, nil)
	verifyResp, err := server.App.Test(verifyReq, 3000)
	if err != nil || verifyResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after deletion, got status: %d", verifyResp.StatusCode)
	}
}
