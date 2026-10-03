package http

import (
	"errors"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/product/application"
	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	workspace "github.com/divinecoid/one-backend/internal/modules/workspace/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TenantDB is resolved from the authenticated company; neither endpoint accepts a company ID.
func copyTarget(c *fiber.Ctx) (*gorm.DB, *workspace.Tenant, error) {
	db := middleware.TenantDB(c)
	if db == nil {
		return nil, nil, apperrors.NewBadRequest("Select a company first")
	}
	var target workspace.Tenant
	query := db.WithContext(c.UserContext())
	if id := middleware.CurrentTenantID(c); id != nil {
		query = query.Where("id = ?", *id)
	} else {
		query = query.Where("is_default = ?", true)
	}
	if err := query.Where("is_active = ?", true).First(&target).Error; err != nil {
		return nil, nil, apperrors.NewBadRequest("Active tenant not found")
	}
	return db.WithContext(c.UserContext()), &target, nil
}

func (h *Handler) CopySources(c *fiber.Ctx) error {
	db, target, err := copyTarget(c)
	if err != nil {
		return err
	}
	var tenants []workspace.Tenant
	if err := db.Where("id <> ? AND is_active = ?", target.ID, true).Order("name").Find(&tenants).Error; err != nil {
		return apperrors.NewInternal(err, "Failed to list source tenants")
	}
	return response.OK(c, "Source tenants", tenants)
}

func sourceTenant(db *gorm.DB, target uuid.UUID, source uuid.UUID) error {
	if source == uuid.Nil || source == target {
		return apperrors.NewBadRequest("Select another tenant in this company")
	}
	var tenant workspace.Tenant
	err := db.Where("id = ? AND is_active = ?", source, true).First(&tenant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.NewBadRequest("Source tenant is not available in this company")
	}
	return err
}

func (h *Handler) CopySourceProducts(c *fiber.Ctx) error {
	db, target, err := copyTarget(c)
	if err != nil {
		return err
	}
	source, err := uuid.Parse(c.Params("tenantId"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid source tenant")
	}
	if err := sourceTenant(db, target.ID, source); err != nil {
		return err
	}
	var products []domain.Product
	if err := db.Where("tenant_id = ?", source).Order("sku").Find(&products).Error; err != nil {
		return apperrors.NewInternal(err, "Failed to list source products")
	}
	return response.OK(c, "Source products", application.ToProductResponseList(products))
}

func (h *Handler) CopyProducts(c *fiber.Ctx) error {
	db, target, err := copyTarget(c)
	if err != nil {
		return err
	}
	var input struct {
		SourceTenantID uuid.UUID   `json:"sourceTenantId"`
		ProductIDs     []uuid.UUID `json:"productIds"`
	}
	if err := c.BodyParser(&input); err != nil || len(input.ProductIDs) == 0 || len(input.ProductIDs) > 500 {
		return apperrors.NewBadRequest("Select between 1 and 500 products")
	}
	if err := sourceTenant(db, target.ID, input.SourceTenantID); err != nil {
		return err
	}
	copied := []domain.Product{}
	skipped := []string{}
	err = db.Transaction(func(tx *gorm.DB) error {
		// Serialize imports into this destination, including duplicate submissions.
		var locked workspace.Tenant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "id = ?", target.ID).Error; err != nil {
			return err
		}
		ids := map[uuid.UUID]bool{}
		for _, id := range input.ProductIDs {
			ids[id] = true
		}
		var source []domain.Product
		if err := tx.Where("tenant_id = ? AND id IN ?", input.SourceTenantID, input.ProductIDs).Order("sku").Find(&source).Error; err != nil {
			return err
		}
		if len(source) != len(ids) {
			return apperrors.NewBadRequest("Some selected products are no longer available in the source tenant")
		}
		for _, original := range source {
			sku := strings.ToUpper(strings.TrimSpace(original.SKU))
			var count int64
			if err := tx.Model(&domain.Product{}).Where("tenant_id = ? AND UPPER(TRIM(sku)) = ?", target.ID, sku).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				skipped = append(skipped, sku)
				continue
			}
			product := copiedProduct(original, target.ID)
			if err := tx.Create(&product).Error; err != nil {
				return err
			}
			copied = append(copied, product)
		}
		return nil
	})
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			return appErr
		}
		return apperrors.NewInternal(err, "Failed to copy products")
	}
	return response.OK(c, "Products copied", fiber.Map{"products": application.ToProductResponseList(copied), "skippedSkus": skipped})
}

func copiedProduct(source domain.Product, tenantID uuid.UUID) domain.Product {
	return domain.Product{TenantID: &tenantID, SKU: strings.ToUpper(strings.TrimSpace(source.SKU)), Barcode: source.Barcode, Name: source.Name, Category: source.Category, Unit: source.Unit, CostPrice: source.CostPrice, SellingPrice: source.SellingPrice, Stock: 0, Status: "out_of_stock"}
}
