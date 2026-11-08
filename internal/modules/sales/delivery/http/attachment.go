package http

import (
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/shared/attachment"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// readUpload reads a multipart file, refusing anything over the size limit (10MB).
func readUpload(fh *multipart.FileHeader) ([]byte, error) {
	if fh.Size > attachment.MaxBytes {
		return nil, apperrors.NewBadRequest("The file is larger than 10 MB")
	}
	f, err := fh.Open()
	if err != nil {
		return nil, apperrors.NewBadRequest("Failed to read the uploaded file")
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, attachment.MaxBytes+1))
}

// UploadAttachment uploads a document/file for a sales order to Cloudflare R2.
func (h *Handler) UploadAttachment(c *fiber.Ctx) error {
	if h.store == nil || !h.store.Enabled() {
		return apperrors.NewServiceUnavailable("File storage is not configured. Please ensure R2 credentials are set.")
	}

	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return apperrors.NewBadRequest("No file uploaded. Please include a 'file' field.")
	}

	content, err := readUpload(fh)
	if err != nil {
		return err
	}

	tenantID := middleware.CurrentTenantID(c)
	prefix := "sales/orders"
	if tenantID != nil {
		prefix = fmt.Sprintf("tenants/%s/sales/orders", tenantID.String())
	}

	saved, err := attachment.Save(h.ctx(c), h.store, prefix, fh.Filename, content)
	if err != nil {
		return err
	}

	return response.Created(c, "File uploaded successfully", fiber.Map{
		"key":         saved.Key,
		"filename":    saved.Name,
		"size":        saved.Size,
		"contentType": saved.ContentType,
	})
}

// DownloadAttachment streams the document attached to a specific Sales Order.
func (h *Handler) DownloadAttachment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid order ID format")
	}

	order, err := uc.GetOrderByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	if order == nil {
		return apperrors.NewNotFound("Sales order not found")
	}

	if order.CustomerPOAttachment == "" {
		return apperrors.NewNotFound("No attachment document found for this order")
	}

	// Backward compatibility: If it's an external URL, redirect directly
	if strings.HasPrefix(order.CustomerPOAttachment, "http://") || strings.HasPrefix(order.CustomerPOAttachment, "https://") {
		return c.Redirect(order.CustomerPOAttachment, fiber.StatusTemporaryRedirect)
	}

	if h.store == nil || !h.store.Enabled() {
		return apperrors.NewServiceUnavailable("File storage is not configured.")
	}

	data, err := attachment.Load(h.ctx(c), h.store, order.CustomerPOAttachment)
	if err != nil {
		return err
	}

	filename := filepath.Base(order.CustomerPOAttachment)
	f, _ := attachment.Check(filename, []byte{0})

	contentType := f.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Set("Content-Type", contentType)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	return c.Send(data)
}

// DownloadAttachmentByKey streams any file by storage key (used for preview or direct key access).
func (h *Handler) DownloadAttachmentByKey(c *fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return apperrors.NewBadRequest("Key parameter is required")
	}

	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		return c.Redirect(key, fiber.StatusTemporaryRedirect)
	}

	if h.store == nil || !h.store.Enabled() {
		return apperrors.NewServiceUnavailable("File storage is not configured.")
	}

	data, err := attachment.Load(h.ctx(c), h.store, key)
	if err != nil {
		return err
	}

	filename := filepath.Base(key)
	f, _ := attachment.Check(filename, []byte{0})

	contentType := f.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Set("Content-Type", contentType)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	return c.Send(data)
}
