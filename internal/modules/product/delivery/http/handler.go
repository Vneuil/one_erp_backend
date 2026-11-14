package http

import (
	"context"
	"sync"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/product/application"
	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
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
