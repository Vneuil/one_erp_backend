package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type procurementRepository struct {
	db *gorm.DB
}

func NewProcurementRepository(db *gorm.DB) domain.ProcurementRepository {
	return &procurementRepository{db: db}
}

// Purchase requests

func (r *procurementRepository) CreatePurchaseRequest(ctx context.Context, pr *domain.PurchaseRequest) error {
	tenantctx.SetTenantID(ctx, &pr.TenantID)
	return r.db.WithContext(ctx).Create(pr).Error
}

func (r *procurementRepository) GetPurchaseRequestByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseRequest, error) {
	var pr domain.PurchaseRequest
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&pr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pr, nil
}

func (r *procurementRepository) ListPurchaseRequests(ctx context.Context, query types.PaginationQuery) ([]domain.PurchaseRequest, int64, error) {
	var items []domain.PurchaseRequest
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PurchaseRequest{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("request_no ILIKE ? OR requested_by ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *procurementRepository) UpdatePurchaseRequest(ctx context.Context, pr *domain.PurchaseRequest) error {
	return r.db.WithContext(ctx).Save(pr).Error
}

func (r *procurementRepository) CountPurchaseRequests(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.PurchaseRequest{}).Count(&total).Error
	return total, err
}

// Purchase orders

func (r *procurementRepository) CreatePurchaseOrder(ctx context.Context, po *domain.PurchaseOrder) error {
	tenantctx.SetTenantID(ctx, &po.TenantID)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *procurementRepository) GetPurchaseOrderByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseOrder, error) {
	var po domain.PurchaseOrder
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &po, nil
}

func (r *procurementRepository) ListPurchaseOrders(ctx context.Context, query types.PaginationQuery) ([]domain.PurchaseOrder, int64, error) {
	var items []domain.PurchaseOrder
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PurchaseOrder{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("order_no ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *procurementRepository) UpdatePurchaseOrder(ctx context.Context, po *domain.PurchaseOrder) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(po).Error
}

func (r *procurementRepository) CountPurchaseOrders(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.PurchaseOrder{}).Count(&total).Error
	return total, err
}

// Goods receipts

func (r *procurementRepository) CreateGoodsReceipt(ctx context.Context, gr *domain.GoodsReceipt) error {
	tenantctx.SetTenantID(ctx, &gr.TenantID)
	return r.db.WithContext(ctx).Create(gr).Error
}

func (r *procurementRepository) GetGoodsReceiptByID(ctx context.Context, id uuid.UUID) (*domain.GoodsReceipt, error) {
	var gr domain.GoodsReceipt
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&gr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &gr, nil
}

func (r *procurementRepository) ListGoodsReceipts(ctx context.Context, query types.PaginationQuery) ([]domain.GoodsReceipt, int64, error) {
	var items []domain.GoodsReceipt
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.GoodsReceipt{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("receipt_no ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *procurementRepository) CountGoodsReceipts(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.GoodsReceipt{}).Count(&total).Error
	return total, err
}

// Purchase invoices

func (r *procurementRepository) CreatePurchaseInvoice(ctx context.Context, inv *domain.PurchaseInvoice) error {
	tenantctx.SetTenantID(ctx, &inv.TenantID)
	return r.db.WithContext(ctx).Create(inv).Error
}

func (r *procurementRepository) GetPurchaseInvoiceByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseInvoice, error) {
	var inv domain.PurchaseInvoice
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *procurementRepository) ListPurchaseInvoices(ctx context.Context, query types.PaginationQuery) ([]domain.PurchaseInvoice, int64, error) {
	var items []domain.PurchaseInvoice
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PurchaseInvoice{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("invoice_number ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("due_date asc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *procurementRepository) UpdatePurchaseInvoice(ctx context.Context, inv *domain.PurchaseInvoice) error {
	return r.db.WithContext(ctx).Save(inv).Error
}

func (r *procurementRepository) CountPurchaseInvoices(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.PurchaseInvoice{}).Count(&total).Error
	return total, err
}

// Purchase down payments

func (r *procurementRepository) CreateDownPayment(ctx context.Context, dp *domain.PurchaseDownPayment) error {
	tenantctx.SetTenantID(ctx, &dp.TenantID)
	return r.db.WithContext(ctx).Create(dp).Error
}

func (r *procurementRepository) GetDownPaymentByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseDownPayment, error) {
	var dp domain.PurchaseDownPayment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&dp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dp, nil
}

func (r *procurementRepository) ListDownPayments(ctx context.Context, query types.PaginationQuery) ([]domain.PurchaseDownPayment, int64, error) {
	var items []domain.PurchaseDownPayment
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PurchaseDownPayment{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("dp_number ILIKE ? OR supplier_name ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *procurementRepository) UpdateDownPayment(ctx context.Context, dp *domain.PurchaseDownPayment) error {
	return r.db.WithContext(ctx).Save(dp).Error
}

func (r *procurementRepository) CountDownPayments(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.PurchaseDownPayment{}).Count(&total).Error
	return total, err
}

// Purchase returns

func (r *procurementRepository) CreatePurchaseReturn(ctx context.Context, ret *domain.PurchaseReturn) error {
	tenantctx.SetTenantID(ctx, &ret.TenantID)
	return r.db.WithContext(ctx).Create(ret).Error
}

func (r *procurementRepository) GetPurchaseReturnByID(ctx context.Context, id uuid.UUID) (*domain.PurchaseReturn, error) {
	var ret domain.PurchaseReturn
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&ret).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ret, nil
}

func (r *procurementRepository) ListPurchaseReturns(ctx context.Context, query types.PaginationQuery) ([]domain.PurchaseReturn, int64, error) {
	var items []domain.PurchaseReturn
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PurchaseReturn{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("return_no ILIKE ? OR supplier_name ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *procurementRepository) CountPurchaseReturns(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.PurchaseReturn{}).Count(&total).Error
	return total, err
}

// Invoice receipts

func (r *procurementRepository) CreateInvoiceReceipt(ctx context.Context, ir *domain.InvoiceReceipt) error {
	tenantctx.SetTenantID(ctx, &ir.TenantID)
	return r.db.WithContext(ctx).Create(ir).Error
}

func (r *procurementRepository) GetInvoiceReceiptByID(ctx context.Context, id uuid.UUID) (*domain.InvoiceReceipt, error) {
	var ir domain.InvoiceReceipt
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&ir).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ir, nil
}

func (r *procurementRepository) ListInvoiceReceipts(ctx context.Context, query types.PaginationQuery) ([]domain.InvoiceReceipt, int64, error) {
	var items []domain.InvoiceReceipt
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.InvoiceReceipt{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("receipt_no ILIKE ? OR supplier_name ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *procurementRepository) UpdateInvoiceReceipt(ctx context.Context, ir *domain.InvoiceReceipt) error {
	return r.db.WithContext(ctx).Save(ir).Error
}

func (r *procurementRepository) CountInvoiceReceipts(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.InvoiceReceipt{}).Count(&total).Error
	return total, err
}

