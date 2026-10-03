package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) domain.CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) Create(ctx context.Context, customer *domain.Customer) error {
	tenantctx.SetTenantID(ctx, &customer.TenantID)
	return r.db.WithContext(ctx).Create(customer).Error
}

func (r *customerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	var customer domain.Customer
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *customerRepository) GetByCode(ctx context.Context, code string) (*domain.Customer, error) {
	var customer domain.Customer
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *customerRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.Customer, int64, error) {
	var customers []domain.Customer
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Customer{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ? OR code ILIKE ? OR email ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortField := "created_at"
	sortDir := "desc"
	if query.SortBy != "" {
		sortField = query.SortBy
	}
	if query.SortDir != "" {
		sortDir = query.SortDir
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order(fmt.Sprintf("%s %s", sortField, sortDir)).
		Offset(offset).
		Limit(query.PerPage).
		Find(&customers).Error

	return customers, total, err
}

func (r *customerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	return r.db.WithContext(ctx).Save(customer).Error
}

func (r *customerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Customer{}, id).Error
}

func (r *customerRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Customer{}).Count(&total).Error
	return total, err
}

// HasReferences checks, by customer_name, whether any Sales Order, Invoice,
// Quotation, or CRM Deal still references this customer. Raw table queries
// are used (rather than importing the sales/crm domain packages) to avoid a
// cross-module dependency cycle risk - these table/column names are stable,
// GORM-derived (see each entity's TableName()).
func (r *customerRepository) HasReferences(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	db := r.db.WithContext(ctx)
	checks := []struct {
		table  string
		column string
	}{
		{"sales_orders", "customer_name"},
		{"invoices", "customer_name"},
		{"quotations", "customer_name"},
		{"crm_deals", "customer"},
	}
	for _, c := range checks {
		var count int64
		if err := db.Table(c.table).Where(c.column+" = ?", name).Count(&count).Error; err != nil {
			// Tolerate a missing table (e.g. tenant schema not yet migrated
			// with that module) rather than failing the whole check.
			if strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}
