package application

import (
	"context"
	"sort"
	"strings"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// Purchase report (Laporan Pembelian)

type purchaseInvoiceRow struct {
	ID             uuid.UUID
	InvoiceNumber  string
	SupplierName   string
	InvoiceDate    string
	Subtotal       float64
	DiscountAmount float64
	AdditionalCost float64
	VATAmount      float64
	RoundingAmount float64
	TotalAmount    float64
	PaidAmount     float64
	Status         string
}

type PurchaseInvoiceLine struct {
	InvoiceID     uuid.UUID `json:"invoiceId"`
	Date          string    `json:"date"`
	InvoiceNumber string    `json:"invoiceNumber"`
	Supplier      string    `json:"supplier"`
	Subtotal      float64   `json:"subtotal"`
	Discount      float64   `json:"discount"`
	Additional    float64   `json:"additionalCost"`
	VAT           float64   `json:"vat"`
	Total         float64   `json:"total"`
	Paid          float64   `json:"paid"`
	Outstanding   float64   `json:"outstanding"`
	Status        string    `json:"status"`
}

type SupplierPurchaseRow struct {
	Rank        int     `json:"rank"`
	Supplier    string  `json:"supplier"`
	Invoices    int     `json:"invoices"`
	Subtotal    float64 `json:"subtotal"`
	Discount    float64 `json:"discount"`
	Additional  float64 `json:"additionalCost"`
	VAT         float64 `json:"vat"`
	Total       float64 `json:"total"`
	Paid        float64 `json:"paid"`
	Outstanding float64 `json:"outstanding"`
}

type purchaseLineRow struct {
	ProductID uuid.UUID
	SKU       string
	Name      string
	Category  string
	Quantity  float64
	Subtotal  float64
}

type ProductPurchaseRow struct {
	ProductID uuid.UUID `json:"productId"`
	SKU       string    `json:"sku"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Quantity  float64   `json:"quantity"`
	Amount    float64   `json:"amount"`
	AvgPrice  float64   `json:"avgPrice"`
}

type PurchaseReportDTO struct {
	From        string                `json:"from"`
	To          string                `json:"to"`
	Invoices    []PurchaseInvoiceLine `json:"invoices"`
	BySupplier  []SupplierPurchaseRow `json:"bySupplier"`
	ByProduct   []ProductPurchaseRow  `json:"byProduct"`
	Total       float64               `json:"total"`
	VAT         float64               `json:"vat"`
	Paid        float64               `json:"paid"`
	Outstanding float64               `json:"outstanding"`
	Note        string                `json:"note"`
}

func buildPurchaseReport(from, to string, invoices []purchaseInvoiceRow, lines []purchaseLineRow) *PurchaseReportDTO {
	out := &PurchaseReportDTO{From: from, To: to, Invoices: []PurchaseInvoiceLine{}, BySupplier: []SupplierPurchaseRow{}, ByProduct: []ProductPurchaseRow{},
		Note: "Invoices are by invoice date. The by-product breakdown comes from purchase orders dated in the range that are no longer draft, cancelled or rejected, so it is on an ordered (not invoiced) basis."}
	bySupplier := map[string]*SupplierPurchaseRow{}
	for _, i := range invoices {
		out.Invoices = append(out.Invoices, PurchaseInvoiceLine{InvoiceID: i.ID, Date: i.InvoiceDate, InvoiceNumber: i.InvoiceNumber, Supplier: i.SupplierName,
			Subtotal: i.Subtotal, Discount: i.DiscountAmount, Additional: i.AdditionalCost, VAT: i.VATAmount, Total: i.TotalAmount, Paid: i.PaidAmount,
			Outstanding: round2(i.TotalAmount - i.PaidAmount), Status: i.Status})
		k := strings.ToLower(strings.TrimSpace(i.SupplierName))
		s := bySupplier[k]
		if s == nil {
			s = &SupplierPurchaseRow{Supplier: strings.TrimSpace(i.SupplierName)}
			bySupplier[k] = s
		}
		sub := i.Subtotal
		if sub == 0 { // legacy invoices carry only the total
			sub = i.TotalAmount
		}
		s.Invoices++
		s.Subtotal += sub
		s.Discount += i.DiscountAmount
		s.Additional += i.AdditionalCost
		s.VAT += i.VATAmount
		s.Total += i.TotalAmount
		s.Paid += i.PaidAmount
		out.Total += i.TotalAmount
		out.VAT += i.VATAmount
		out.Paid += i.PaidAmount
	}
	for _, s := range bySupplier {
		s.Subtotal, s.Discount, s.Additional, s.VAT, s.Total, s.Paid = round2(s.Subtotal), round2(s.Discount), round2(s.Additional), round2(s.VAT), round2(s.Total), round2(s.Paid)
		s.Outstanding = round2(s.Total - s.Paid)
		out.BySupplier = append(out.BySupplier, *s)
	}
	sort.Slice(out.BySupplier, func(i, j int) bool {
		if out.BySupplier[i].Total != out.BySupplier[j].Total {
			return out.BySupplier[i].Total > out.BySupplier[j].Total
		}
		return out.BySupplier[i].Supplier < out.BySupplier[j].Supplier
	})
	for i := range out.BySupplier {
		out.BySupplier[i].Rank = i + 1
	}
	out.Total, out.VAT, out.Paid = round2(out.Total), round2(out.VAT), round2(out.Paid)
	out.Outstanding = round2(out.Total - out.Paid)

	byProduct := map[uuid.UUID]*ProductPurchaseRow{}
	for _, l := range lines {
		p := byProduct[l.ProductID]
		if p == nil {
			p = &ProductPurchaseRow{ProductID: l.ProductID, SKU: l.SKU, Name: l.Name, Category: l.Category}
			byProduct[l.ProductID] = p
		}
		p.Quantity += l.Quantity
		p.Amount += l.Subtotal
	}
	for _, p := range byProduct {
		p.Quantity, p.Amount = round2(p.Quantity), round2(p.Amount)
		if p.Quantity > 0 {
			p.AvgPrice = round2(p.Amount / p.Quantity)
		}
		out.ByProduct = append(out.ByProduct, *p)
	}
	sort.Slice(out.ByProduct, func(i, j int) bool { return out.ByProduct[i].Amount > out.ByProduct[j].Amount })
	return out
}

func (q *ReportQuery) PurchaseReport(ctx context.Context, from, to string) (*PurchaseReportDTO, error) {
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	var invoices []purchaseInvoiceRow
	if err := scope(ctx, q.db.WithContext(ctx).Table("procurement_purchase_invoices i").
		Where("i.deleted_at IS NULL AND i.invoice_date >= ? AND i.invoice_date <= ?", from, to), "i").
		Select(`i.id, i.invoice_number, i.supplier_name, i.invoice_date, i.subtotal, i.discount_amount, i.additional_cost,
			i.vat_amount, i.rounding_amount, i.total_amount, i.paid_amount, i.status`).
		Order("i.invoice_date asc, i.created_at asc").Scan(&invoices).Error; err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load purchase invoices")
	}
	var lines []purchaseLineRow
	if err := scope(ctx, q.db.WithContext(ctx).Table("procurement_purchase_order_lines l").
		Joins("JOIN procurement_purchase_orders o ON o.id = l.purchase_order_id AND o.deleted_at IS NULL").
		Joins("LEFT JOIN products p ON p.id = l.product_id").
		Where("l.deleted_at IS NULL AND LOWER(o.status) NOT IN ?", []string{"draft", "cancelled", "rejected"}).
		Where("o.order_date >= ? AND o.order_date <= ?", from, to), "o").
		Select("l.product_id, COALESCE(p.sku,'') AS sku, COALESCE(p.name,'(produk dihapus)') AS name, COALESCE(p.category,'') AS category, l.quantity, l.subtotal").
		Scan(&lines).Error; err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load purchase order lines")
	}
	return buildPurchaseReport(from, to, invoices, lines), nil
}
