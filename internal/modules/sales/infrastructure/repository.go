package infrastructure

import (
	"context"
	"errors"
	"fmt"
	crmDomain "github.com/divinecoid/one-backend/internal/modules/crm/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type salesRepository struct {
	db *gorm.DB
}

func NewSalesRepository(db *gorm.DB) domain.SalesRepository {
	return &salesRepository{db: db}
}

func (r *salesRepository) CreateOrder(ctx context.Context, order *domain.SalesOrder) error {
	tenantctx.SetTenantID(ctx, &order.TenantID)
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *salesRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*domain.SalesOrder, error) {
	var order domain.SalesOrder
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *salesRepository) UpdateOrder(ctx context.Context, order *domain.SalesOrder) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(order).Error
}

func (r *salesRepository) UpdateOrderWithLines(ctx context.Context, order *domain.SalesOrder) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range order.Lines {
			order.Lines[i].ID = uuid.Nil
			order.Lines[i].SalesOrderID = order.ID
		}
		if err := tx.Where("sales_order_id = ?", order.ID).Delete(&domain.SalesOrderLine{}).Error; err != nil {
			return err
		}
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(order).Error
	})
}

func (r *salesRepository) ListOrders(ctx context.Context, query types.PaginationQuery) ([]domain.SalesOrder, int64, error) {
	var orders []domain.SalesOrder
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.SalesOrder{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("order_number ILIKE ? OR customer_name ILIKE ?", searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&orders).Error
	return orders, total, err
}

func (r *salesRepository) CountOrders(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.SalesOrder{}).Count(&total).Error
	return total, err
}

func (r *salesRepository) CreateQuotation(ctx context.Context, quo *domain.Quotation) error {
	if quo.TenantID == nil {
		quo.TenantID = tenantctx.FromContext(ctx)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if quo.LeadID != nil {
			var lead crmDomain.Lead
			err := tenantctx.Scope(ctx, tx.Model(&crmDomain.Lead{})).Where("id = ?", *quo.LeadID).First(&lead).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NewBadRequest("Customer CRM tidak ditemukan")
			}
			if err != nil {
				return err
			}
			quo.CustomerName = strings.TrimSpace(lead.Company)
			if quo.CustomerName == "" {
				quo.CustomerName = lead.Name
			}
		}
		for i := range quo.Lines {
			line := &quo.Lines[i]
			if line.ProductID == nil {
				if quo.LeadID != nil {
					return apperrors.NewBadRequest("Pilih barang dari master data untuk setiap item")
				}
				continue
			}
			var product productDomain.Product
			err := tenantctx.Scope(ctx, tx.Model(&productDomain.Product{})).Where("id = ?", *line.ProductID).First(&product).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NewBadRequest("Barang tidak ditemukan di master data")
			}
			if err != nil {
				return err
			}
			line.Description = product.Name
			line.Unit = product.Unit
		}
		return tx.Create(quo).Error
	})
}

func (r *salesRepository) GetQuotationByID(ctx context.Context, id uuid.UUID) (*domain.Quotation, error) {
	var quo domain.Quotation
	err := r.db.WithContext(ctx).Preload("Lines", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).Where("id = ?", id).First(&quo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &quo, nil
}

func (r *salesRepository) UpdateQuotation(ctx context.Context, quo *domain.Quotation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range quo.Lines {
			line := &quo.Lines[i]
			line.ID = uuid.Nil
			line.QuotationID = quo.ID
			if line.ProductID == nil {
				continue
			}
			var product productDomain.Product
			err := tenantctx.Scope(ctx, tx.Model(&productDomain.Product{})).Where("id = ?", *line.ProductID).First(&product).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.NewBadRequest("Barang tidak ditemukan di master data")
			}
			if err != nil {
				return err
			}
			line.Description = product.Name
			line.Unit = product.Unit
		}
		if err := tx.Where("quotation_id = ?", quo.ID).Delete(&domain.QuotationLine{}).Error; err != nil {
			return err
		}
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(quo).Error
	})
}

func (r *salesRepository) ListQuotations(ctx context.Context, query types.PaginationQuery) ([]domain.Quotation, int64, error) {
	var quotations []domain.Quotation
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Quotation{}))
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&quotations).Error
	return quotations, total, err
}

func (r *salesRepository) CreateInvoice(ctx context.Context, inv *domain.Invoice) error {
	tenantctx.SetTenantID(ctx, &inv.TenantID)
	return r.db.WithContext(ctx).Create(inv).Error
}

func (r *salesRepository) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *salesRepository) ListInvoices(ctx context.Context, query types.PaginationQuery) ([]domain.Invoice, int64, error) {
	var invoices []domain.Invoice
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Invoice{}))
	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("invoice_number ILIKE ? OR customer_name ILIKE ?", searchPattern, searchPattern)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&invoices).Error
	return invoices, total, err
}

func (r *salesRepository) UpdateInvoice(ctx context.Context, inv *domain.Invoice) error {
	return r.db.WithContext(ctx).Save(inv).Error
}

func (r *salesRepository) CreateDelivery(ctx context.Context, d *domain.Delivery) error {
	tenantctx.SetTenantID(ctx, &d.TenantID)
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *salesRepository) GetDeliveryByID(ctx context.Context, id uuid.UUID) (*domain.Delivery, error) {
	var d domain.Delivery
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *salesRepository) ListDeliveries(ctx context.Context, query types.PaginationQuery) ([]domain.Delivery, int64, error) {
	var deliveries []domain.Delivery
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Delivery{}))
	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("delivery_number ILIKE ? OR customer_name ILIKE ?", searchPattern, searchPattern)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&deliveries).Error
	return deliveries, total, err
}

func (r *salesRepository) ListDeliveriesByOrderID(ctx context.Context, orderID uuid.UUID) ([]domain.Delivery, error) {
	var deliveries []domain.Delivery
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Delivery{})).
		Preload("Lines").Where("sales_order_id = ?", orderID).Find(&deliveries).Error
	return deliveries, err
}

// Sales down payments

func (r *salesRepository) CreateDownPayment(ctx context.Context, dp *domain.SalesDownPayment) error {
	tenantctx.SetTenantID(ctx, &dp.TenantID)
	return r.db.WithContext(ctx).Create(dp).Error
}

func (r *salesRepository) GetDownPaymentByID(ctx context.Context, id uuid.UUID) (*domain.SalesDownPayment, error) {
	var dp domain.SalesDownPayment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&dp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dp, nil
}

func (r *salesRepository) UpdateDownPayment(ctx context.Context, dp *domain.SalesDownPayment) error {
	return r.db.WithContext(ctx).Save(dp).Error
}

func (r *salesRepository) ListDownPayments(ctx context.Context, query types.PaginationQuery) ([]domain.SalesDownPayment, int64, error) {
	var items []domain.SalesDownPayment
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.SalesDownPayment{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("dp_number ILIKE ? OR customer_name ILIKE ?", pattern, pattern)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

// Sales returns

func (r *salesRepository) CreateSalesReturn(ctx context.Context, ret *domain.SalesReturn) error {
	tenantctx.SetTenantID(ctx, &ret.TenantID)
	return r.db.WithContext(ctx).Create(ret).Error
}

func (r *salesRepository) GetSalesReturnByID(ctx context.Context, id uuid.UUID) (*domain.SalesReturn, error) {
	var ret domain.SalesReturn
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&ret).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ret, nil
}

func (r *salesRepository) ListSalesReturns(ctx context.Context, query types.PaginationQuery) ([]domain.SalesReturn, int64, error) {
	var items []domain.SalesReturn
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.SalesReturn{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("return_no ILIKE ? OR customer_name ILIKE ?", pattern, pattern)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

// Billing schedule (termin)

// ReplaceBillingTerms swaps an order's whole schedule in one transaction, so a
// failure never leaves a half-written schedule that no longer sums to 100%.
func (r *salesRepository) ReplaceBillingTerms(ctx context.Context, orderID uuid.UUID, terms []domain.BillingTerm) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("sales_order_id = ?", orderID).Delete(&domain.BillingTerm{}).Error; err != nil {
			return err
		}
		for i := range terms {
			tenantctx.SetTenantID(ctx, &terms[i].TenantID)
			if err := tx.Create(&terms[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *salesRepository) ListBillingTerms(ctx context.Context, orderID uuid.UUID) ([]domain.BillingTerm, error) {
	var out []domain.BillingTerm
	return out, r.db.WithContext(ctx).Where("sales_order_id = ?", orderID).Order("seq asc").Find(&out).Error
}

func (r *salesRepository) GetBillingTerm(ctx context.Context, id uuid.UUID) (*domain.BillingTerm, error) {
	var t domain.BillingTerm
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *salesRepository) UpdateBillingTerm(ctx context.Context, t *domain.BillingTerm) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *salesRepository) UpdateDelivery(ctx context.Context, d *domain.Delivery) error {
	return r.db.WithContext(ctx).Omit("Lines").Save(d).Error
}
