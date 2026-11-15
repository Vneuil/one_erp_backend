package http

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"github.com/gofiber/fiber/v2"
)

func TestProductCertificationsRoutes(t *testing.T) {
	app := fiber.New()
	v1 := app.Group("/api/v1")

	store := storage.NewLocal(t.TempDir())
	h := NewHandler(store)

	// Register routes directly without auth middleware for route path verification
	certs := v1.Group("/product-certifications")
	certs.Post("/upload-attachment", h.UploadAttachment)
	certs.Get("/attachment/download", h.DownloadAttachmentByKey)
	certs.Get("/:id/attachment", h.DownloadAttachment)
	certs.Post("/", h.CreateCertification)
	certs.Get("/", h.ListCertifications)
	certs.Put("/:id", h.UpdateCertification)
	certs.Delete("/:id", h.DeleteCertification)

	// Create multipart body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.pdf")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("%PDF-1.4 test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/product-certifications/upload-attachment", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode == fiber.StatusMethodNotAllowed {
		t.Fatalf("got 405 Method Not Allowed - route matching failed!")
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}
}
