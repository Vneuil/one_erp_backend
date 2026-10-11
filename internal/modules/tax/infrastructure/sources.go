package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	customerDomain "github.com/divinecoid/one-backend/internal/modules/customer/domain"
	posDomain "github.com/divinecoid/one-backend/internal/modules/pos/domain"
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

func (r *sourceReader) ReturnVATBetween(ctx context.Context, from, to string) (output, input []domain.ReturnVATRef, err error) {
	var sales []salesDomain.SalesReturn
	if err = tenantctx.Scope(ctx, r.db.WithContext(ctx)).
		Where("return_date >= ? AND return_date <= ? AND vat_amount > 0", from, to).Find(&sales).Error; err != nil {
		return nil, nil, err
	}
	for _, s := range sales {
		output = append(output, domain.ReturnVATRef{Number: s.ReturnNo, Party: s.CustomerName, Date: s.ReturnDate, VAT: s.VATAmount})
	}
	var purchases []procDomain.PurchaseReturn
	if err = tenantctx.Scope(ctx, r.db.WithContext(ctx)).
		Where("return_date >= ? AND return_date <= ? AND creditable_vat_amount > 0", from, to).Find(&purchases).Error; err != nil {
		return nil, nil, err
	}
	for _, p := range purchases {
		input = append(input, domain.ReturnVATRef{Number: p.ReturnNo, Party: p.SupplierName, Date: p.ReturnDate, VAT: p.CreditableVATAmount})
	}
	// POS refunds reverse part of the PPN of a sale. Fully voided sales are
	// excluded from the POS sources altogether, so their refund is skipped here.
	tid := tenantctx.FromContext(ctx)
	var refunds []struct {
		Date string
		Tax  float64
		No   string
		Cust string
	}
	if err = r.db.WithContext(ctx).Table("pos_refunds AS f").
		Select("(f.created_at AT TIME ZONE 'Asia/Jakarta')::date::text AS date, f.tax_amount AS tax, t.order_no AS no, t.customer AS cust").
		Joins("JOIN pos_transactions AS t ON t.id = f.transaction_id").
		Where("(?::uuid IS NULL OR f.tenant_id = ?::uuid) AND f.tax_amount > 0 AND f.kind = 'refund' AND (f.created_at AT TIME ZONE 'Asia/Jakarta')::date >= ? AND (f.created_at AT TIME ZONE 'Asia/Jakarta')::date <= ?", tid, tid, from, to).
		Scan(&refunds).Error; err != nil {
		return nil, nil, err
	}
	for _, f := range refunds {
		output = append(output, domain.ReturnVATRef{Number: f.No, Party: f.Cust, Date: f.Date, VAT: f.Tax})
	}
	return output, input, nil
}

func jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*3600)
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

const posLocalDate = "(created_at AT TIME ZONE 'Asia/Jakarta')::date"

func posRef(t posDomain.POSTransaction) domain.POSSaleRef {
	ref := domain.POSSaleRef{ID: t.ID, Number: t.OrderNo, Customer: t.Customer, Date: t.CreatedAt.In(jakarta()).Format("2006-01-02"),
		Total: t.TotalAmount, Tax: t.TaxAmount, Discount: t.DiscountAmount}
	for _, l := range t.Lines {
		ref.Lines = append(ref.Lines, domain.SourceLine{Description: l.Name, Quantity: l.Quantity, Subtotal: l.Subtotal})
	}
	return ref
}

func (r *sourceReader) POSSale(ctx context.Context, id uuid.UUID) (*domain.POSSaleRef, error) {
	var t posDomain.POSTransaction
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Lines").Where("id = ? AND tax_amount > 0 AND status <> 'Voided'", id).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	ref := posRef(t)
	return &ref, nil
}

func (r *sourceReader) POSSalesBetween(ctx context.Context, from, to string) ([]domain.POSSaleRef, error) {
	var rows []posDomain.POSTransaction
	if err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).
		Where("tax_amount > 0 AND status <> 'Voided' AND "+posLocalDate+" >= ? AND "+posLocalDate+" <= ?", from, to).
		Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.POSSaleRef, len(rows))
	for i, row := range rows {
		out[i] = posRef(row)
	}
	return out, nil
}
