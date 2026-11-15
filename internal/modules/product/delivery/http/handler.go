package http

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/product/application"
	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/divinecoid/one-backend/internal/shared/attachment"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	store storage.Storage
}

func NewHandler(store ...storage.Storage) *Handler {
	var s storage.Storage
	if len(store) > 0 {
		s = store[0]
	}
	return &Handler{store: s}
}

// migrated remembers tenant DBs whose tables were verified, ensuring any newly
// added columns (e.g. grade, safety_stock, organic_type) are automatically created.
var migrated sync.Map

func ensureSchema(db *gorm.DB) error {
	if _, ok := migrated.Load(db); ok {
		return nil
	}
	if err := db.AutoMigrate(&domain.Product{}, &domain.ProductCategory{}, &domain.UnitOfMeasure{}, &domain.Certification{}); err != nil {
		return err
	}
	migrated.Store(db, true)
	return nil
}

// resolve builds a use-case bound to the caller's own tenant database.
func (h *Handler) resolve(c *fiber.Ctx) (application.ProductUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	if err := ensureSchema(tenantDB); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	repo := infrastructure.NewProductRepository(tenantDB)
	return application.NewProductUseCase(repo), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) Create(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateProductDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	product, err := uc.Create(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "Product created successfully", product)
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid product ID format")
	}

	product, err := uc.GetByID(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Product retrieved successfully", product)
}

func (h *Handler) List(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	products, meta, err := uc.List(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Products retrieved successfully", products, meta)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid product ID format")
	}

	var dto application.UpdateProductDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	product, err := uc.Update(h.ctx(c), id, dto)
	if err != nil {
		return err
	}

	return response.OK(c, "Product updated successfully", product)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid product ID format")
	}

	if err := uc.Delete(h.ctx(c), id); err != nil {
		return err
	}

	return response.OK(c, "Product deleted successfully", nil)
}

// Categories

func (h *Handler) CreateCategory(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateCategoryDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	category, err := uc.CreateCategory(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Category created successfully", category)
}

func (h *Handler) ListCategories(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	categories, err := uc.ListCategories(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Categories retrieved successfully", categories)
}

func (h *Handler) UpdateCategory(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid category ID format")
	}
	var dto application.UpdateCategoryDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	category, err := uc.UpdateCategory(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Category updated successfully", category)
}

func (h *Handler) DeleteCategory(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid category ID format")
	}
	if err := uc.DeleteCategory(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Category deleted successfully", nil)
}

// Units of measure

func (h *Handler) CreateUnit(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateUnitDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	unit, err := uc.CreateUnit(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Unit created successfully", unit)
}

func (h *Handler) ListUnits(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	units, err := uc.ListUnits(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Units retrieved successfully", units)
}

func (h *Handler) UpdateUnit(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid unit ID format")
	}
	var dto application.UpdateUnitDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	unit, err := uc.UpdateUnit(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Unit updated successfully", unit)
}

func (h *Handler) DeleteUnit(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid unit ID format")
	}
	if err := uc.DeleteUnit(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Unit deleted successfully", nil)
}

func (h *Handler) CreateCertification(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateCertificationDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	cert, err := uc.CreateCertification(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Certification created successfully", cert)
}

func (h *Handler) ListCertifications(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	certs, err := uc.ListCertifications(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Certifications retrieved successfully", certs)
}

func (h *Handler) UpdateCertification(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid certification ID format")
	}
	var dto application.UpdateCertificationDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	cert, err := uc.UpdateCertification(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Certification updated successfully", cert)
}

func (h *Handler) DeleteCertification(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid certification ID format")
	}
	if err := uc.DeleteCertification(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Certification deleted successfully", nil)
}

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

// UploadAttachment uploads a certification proof file to Cloudflare R2 or local storage.
func (h *Handler) UploadAttachment(c *fiber.Ctx) error {
	if h.store == nil || !h.store.Enabled() {
		return apperrors.NewServiceUnavailable("File storage is not configured. Please ensure R2 credentials or LOCAL_STORAGE_DIR are set.")
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
	prefix := "product-certifications"
	if tenantID != nil {
		prefix = fmt.Sprintf("tenants/%s/product-certifications", tenantID.String())
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

// DownloadAttachment streams the proof document attached to a specific certification.
func (h *Handler) DownloadAttachment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid certification ID format")
	}

	certs, err := uc.ListCertifications(h.ctx(c))
	if err != nil {
		return err
	}

	var target *application.CertificationResponseDTO
	for i := range certs {
		if certs[i].ID == id {
			target = &certs[i]
			break
		}
	}

	if target == nil {
		return apperrors.NewNotFound("Certification not found")
	}

	if target.Attachment == "" {
		return apperrors.NewNotFound("No proof document found for this certification")
	}

	if strings.HasPrefix(target.Attachment, "http://") || strings.HasPrefix(target.Attachment, "https://") {
		return c.Redirect(target.Attachment, fiber.StatusTemporaryRedirect)
	}

	if h.store == nil || !h.store.Enabled() {
		return apperrors.NewServiceUnavailable("File storage is not configured.")
	}

	data, err := attachment.Load(h.ctx(c), h.store, target.Attachment)
	if err != nil {
		return err
	}

	filename := target.AttachmentName
	if filename == "" {
		filename = filepath.Base(target.Attachment)
	}
	f, _ := attachment.Check(filename, []byte{0})

	contentType := f.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Set("Content-Type", contentType)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
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
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	return c.Send(data)
}
