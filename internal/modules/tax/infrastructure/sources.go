package infrastructure

import (
	"context"
	"errors"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	customerDomain "github.com/divinecoid/one-backend/internal/modules/customer/domain"
	procDomain "github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	supplierDomain "github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type sourceReader struct {
	db *gorm.DB
}

func NewSourceReader(db *gorm.DB) domain.SourceReader {
	return &sourceReader{db: db}
}

func salesRef(i salesDomain.Invoice) domain.SalesInvoiceRef {
	return domain.SalesInvoiceRef{
		ID: i.ID, Number: i.InvoiceNumber, CustomerName: i.CustomerName, Date: i.InvoiceDate,
		Subtotal: i.Subtotal, Discount: i.DiscountAmount, AdditionalCost: i.AdditionalCost, Rounding: i.RoundingAmount, Total: i.TotalAmount,
		TaxBase: i.TaxBase, DPPOtherValue: i.DPPOtherValue, VATRate: i.VATRate, VATAmt: i.VATAmount, VATOtherValueBase: i.VATOtherValueBase,
	}
}

func purchaseRef(i procDomain.PurchaseInvoice) domain.PurchaseInvoiceRef {
	return domain.PurchaseInvoiceRef{
		ID: i.ID, SupplierID: i.SupplierID, Number: i.InvoiceNumber, SupplierName: i.SupplierName, Date: i.InvoiceDate, Total: i.TotalAmount,
		TaxBase: i.TaxBase, DPPOtherValue: i.DPPOtherValue, VATRate: i.VATRate, VATAmt: i.VATAmount,
		VATOtherValueBase: i.VATOtherValueBase, VATCreditable: i.VATCreditable,
	}
}

func (r *sourceReader) SalesInvoice(ctx context.Context, id uuid.UUID) (*domain.SalesInvoiceRef, error) {
	var inv salesDomain.Invoice
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	ref := salesRef(inv)
	return &ref, nil
}

func (r *sourceReader) SalesInvoicesBetween(ctx context.Context, from, to string) ([]domain.SalesInvoiceRef, error) {
	var rows []salesDomain.Invoice
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).
		Where("invoice_date >= ? AND invoice_date <= ?", from, to).Order("invoice_date asc, created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SalesInvoiceRef, len(rows))
	for i, row := range rows {
		out[i] = salesRef(row)
	}
	return out, nil
}

func (r *sourceReader) SalesInvoiceLines(ctx context.Context, id uuid.UUID) ([]domain.SourceLine, error) {
	var inv salesDomain.Invoice
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&inv).Error; err != nil {
		return nil, err
	}
	if inv.SalesOrderID == nil {
		return nil, nil
	}
	var lines []salesDomain.SalesOrderLine
	if err := r.db.WithContext(ctx).Where("sales_order_id = ?", *inv.SalesOrderID).Order("created_at asc").Find(&lines).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SourceLine, 0, len(lines))
	for _, l := range lines {
		var p productDomain.Product
		name, unit := "Barang", ""
		if err := r.db.WithContext(ctx).Where("id = ?", l.ProductID).First(&p).Error; err == nil {
			name, unit = p.Name, p.Unit
		}
		out = append(out, domain.SourceLine{Description: name, Unit: unit, Quantity: l.Quantity, Subtotal: l.Subtotal})
	}
	return out, nil
}

func (r *sourceReader) PurchaseInvoice(ctx context.Context, id uuid.UUID) (*domain.PurchaseInvoiceRef, error) {
	var inv procDomain.PurchaseInvoice
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	ref := purchaseRef(inv)
	return &ref, nil
}

func (r *sourceReader) PurchaseInvoicesBetween(ctx context.Context, from, to string) ([]domain.PurchaseInvoiceRef, error) {
	var rows []procDomain.PurchaseInvoice
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).
		Where("invoice_date >= ? AND invoice_date <= ?", from, to).Order("invoice_date asc, created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PurchaseInvoiceRef, len(rows))
	for i, row := range rows {
		out[i] = purchaseRef(row)
	}
	return out, nil
}

func (r *sourceReader) CustomersByName(ctx context.Context) (map[string]domain.Party, error) {
	var rows []customerDomain.Customer
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]domain.Party, len(rows))
	for _, c := range rows {
		out[strings.ToLower(strings.TrimSpace(c.Name))] = domain.Party{Name: c.Name, NPWP: c.NPWP, NIK: c.NIK, Address: c.Address, Email: c.Email}
	}
	return out, nil
}

func (r *sourceReader) Supplier(ctx context.Context, id uuid.UUID) (*domain.Party, error) {
	var s supplierDomain.Supplier
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.Party{Name: s.Name, NPWP: s.NPWP, NIK: s.NIK, Address: s.Address, Email: s.Email}, nil
}
