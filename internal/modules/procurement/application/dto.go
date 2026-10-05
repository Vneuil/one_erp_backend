package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	"github.com/google/uuid"
)

// Purchase requests

type PurchaseRequestLineDTO struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	Notes     string    `json:"notes"`
}

type CreatePurchaseRequestDTO struct {
	Department   string                   `json:"department"`
	NeededByDate string                   `json:"neededByDate"`
	Lines        []PurchaseRequestLineDTO `json:"lines"`
	// RequestedBy is set by the handler from the authenticated caller, never
	// read from client-supplied JSON - there is no `json:"requestedBy"` tag
	// on purpose, so BodyParser can never populate it from the request body.
	RequestedBy string `json:"-"`
}

type PurchaseRequestLineResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	Notes     string    `json:"notes"`
}

type PurchaseRequestResponseDTO struct {
	ID           uuid.UUID                        `json:"id"`
	RequestNo    string                           `json:"requestNo"`
	RequestedBy  string                           `json:"requestedBy"`
	Department   string                           `json:"department"`
	NeededByDate string                           `json:"neededByDate"`
	Status       string                           `json:"status"`
	Lines        []PurchaseRequestLineResponseDTO `json:"lines,omitempty"`
	CreatedAt    time.Time                        `json:"createdAt"`
}

func ToPurchaseRequestResponse(pr *domain.PurchaseRequest) *PurchaseRequestResponseDTO {
	if pr == nil {
		return nil
	}
	lines := make([]PurchaseRequestLineResponseDTO, len(pr.Lines))
	for i, l := range pr.Lines {
		lines[i] = PurchaseRequestLineResponseDTO{ID: l.ID, ProductID: l.ProductID, Quantity: l.Quantity, Notes: l.Notes}
	}
	return &PurchaseRequestResponseDTO{
		ID: pr.ID, RequestNo: pr.RequestNo, RequestedBy: pr.RequestedBy, Department: pr.Department,
		NeededByDate: pr.NeededByDate, Status: pr.Status, Lines: lines, CreatedAt: pr.CreatedAt,
	}
}

func ToPurchaseRequestResponseList(items []domain.PurchaseRequest) []PurchaseRequestResponseDTO {
	result := make([]PurchaseRequestResponseDTO, len(items))
	for i, pr := range items {
		result[i] = *ToPurchaseRequestResponse(&pr)
	}
	return result
}

// Purchase orders

type PurchaseOrderLineDTO struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
}

type CreatePurchaseOrderDTO struct {
	SupplierID        uuid.UUID  `json:"supplierId"`
	SupplierName      string     `json:"supplierName"`
	PurchaseRequestID *uuid.UUID `json:"purchaseRequestId,omitempty"`
	OrderDate         string     `json:"orderDate"`
	ExpectedDate      string     `json:"expectedDate"`
	RequestedBy       string     `json:"requestedBy"`
	// Currency is the code TotalAmount is denominated in. Empty/IDR (the
	// base currency) skips conversion entirely - see application/usecase.go
	// CreatePurchaseOrder.
	Currency string                 `json:"currency"`
	Lines    []PurchaseOrderLineDTO `json:"lines"`
}

// ApprovalActionDTO carries the acting user's identity when approving or
// rejecting a document that may be routed through modules/approval.
type ApprovalActionDTO struct {
	ActorName string `json:"actorName"`
	ActorRole string `json:"actorRole"`
	Comments  string `json:"comments"`
}

type PurchaseOrderLineResponseDTO struct {
	ID               uuid.UUID `json:"id"`
	ProductID        uuid.UUID `json:"productId"`
	Quantity         float64   `json:"quantity"`
	UnitPrice        float64   `json:"unitPrice"`
	Subtotal         float64   `json:"subtotal"`
	QuantityReceived float64   `json:"quantityReceived"`
}

type PurchaseOrderResponseDTO struct {
	ID                uuid.UUID                      `json:"id"`
	OrderNo           string                         `json:"orderNo"`
	SupplierID        uuid.UUID                      `json:"supplierId"`
	SupplierName      string                         `json:"supplierName"`
	PurchaseRequestID *uuid.UUID                     `json:"purchaseRequestId,omitempty"`
	OrderDate         string                         `json:"orderDate"`
	ExpectedDate      string                         `json:"expectedDate"`
	Status            string                         `json:"status"`
	TotalAmount       float64                        `json:"totalAmount"`
	CreatedByEmail    string                         `json:"createdByEmail,omitempty"`
	Currency          string                         `json:"currency"`
	BaseAmount        float64                        `json:"baseAmount"`
	Lines             []PurchaseOrderLineResponseDTO `json:"lines,omitempty"`
	CreatedAt         time.Time                      `json:"createdAt"`
}

func ToPurchaseOrderResponse(po *domain.PurchaseOrder) *PurchaseOrderResponseDTO {
	if po == nil {
		return nil
	}
	lines := make([]PurchaseOrderLineResponseDTO, len(po.Lines))
	for i, l := range po.Lines {
		lines[i] = PurchaseOrderLineResponseDTO{
			ID: l.ID, ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice,
			Subtotal: l.Subtotal, QuantityReceived: l.QuantityReceived,
		}
	}
	return &PurchaseOrderResponseDTO{
		ID: po.ID, OrderNo: po.OrderNo, SupplierID: po.SupplierID, SupplierName: po.SupplierName, PurchaseRequestID: po.PurchaseRequestID,
		OrderDate: po.OrderDate, ExpectedDate: po.ExpectedDate, Status: po.Status, TotalAmount: po.TotalAmount,
		Currency: po.Currency, BaseAmount: po.BaseAmount, CreatedByEmail: po.CreatedByEmail,
		Lines: lines, CreatedAt: po.CreatedAt,
	}
}

func ToPurchaseOrderResponseList(items []domain.PurchaseOrder) []PurchaseOrderResponseDTO {
	result := make([]PurchaseOrderResponseDTO, len(items))
	for i, po := range items {
		result[i] = *ToPurchaseOrderResponse(&po)
	}
	return result
}

// Goods receipts

type GoodsReceiptLineDTO struct {
	ProductID        uuid.UUID `json:"productId"`
	QuantityReceived float64   `json:"quantityReceived"`
}

type CreateGoodsReceiptDTO struct {
	PurchaseOrderID uuid.UUID             `json:"purchaseOrderId"`
	ReceivedDate    string                `json:"receivedDate"`
	WarehouseID     *uuid.UUID            `json:"warehouseId,omitempty"`
	Lines           []GoodsReceiptLineDTO `json:"lines"`
	// ReceivedBy is set by the handler from the authenticated caller, never
	// read from client-supplied JSON - there is no `json:"receivedBy"` tag
	// on purpose, so BodyParser can never populate it from the request body.
	ReceivedBy string `json:"-"`
}

type GoodsReceiptLineResponseDTO struct {
	ID                       uuid.UUID `json:"id"`
	ProductID                uuid.UUID `json:"productId"`
	QuantityReceived         float64   `json:"quantityReceived"`
	QuantityOrderedReference float64   `json:"quantityOrderedReference"`
}

type GoodsReceiptResponseDTO struct {
	ID              uuid.UUID                     `json:"id"`
	ReceiptNo       string                        `json:"receiptNo"`
	PurchaseOrderID uuid.UUID                     `json:"purchaseOrderId"`
	ReceivedDate    string                        `json:"receivedDate"`
	WarehouseID     *uuid.UUID                    `json:"warehouseId,omitempty"`
	ReceivedBy      string                        `json:"receivedBy"`
	Status          string                        `json:"status"`
	Lines           []GoodsReceiptLineResponseDTO `json:"lines,omitempty"`
	CreatedAt       time.Time                     `json:"createdAt"`
}

func ToGoodsReceiptResponse(gr *domain.GoodsReceipt) *GoodsReceiptResponseDTO {
	if gr == nil {
		return nil
	}
	lines := make([]GoodsReceiptLineResponseDTO, len(gr.Lines))
	for i, l := range gr.Lines {
		lines[i] = GoodsReceiptLineResponseDTO{
			ID: l.ID, ProductID: l.ProductID, QuantityReceived: l.QuantityReceived,
			QuantityOrderedReference: l.QuantityOrderedRef,
		}
	}
	return &GoodsReceiptResponseDTO{
		ID: gr.ID, ReceiptNo: gr.ReceiptNo, PurchaseOrderID: gr.PurchaseOrderID, ReceivedDate: gr.ReceivedDate,
		WarehouseID: gr.WarehouseID, ReceivedBy: gr.ReceivedBy, Status: gr.Status, Lines: lines, CreatedAt: gr.CreatedAt,
	}
}

func ToGoodsReceiptResponseList(items []domain.GoodsReceipt) []GoodsReceiptResponseDTO {
	result := make([]GoodsReceiptResponseDTO, len(items))
	for i, gr := range items {
		result[i] = *ToGoodsReceiptResponse(&gr)
	}
	return result
}

// Purchase invoices

type CreatePurchaseInvoiceDTO struct {
	SupplierID      uuid.UUID  `json:"supplierId"`
	SupplierName    string     `json:"supplierName"`
	PurchaseOrderID *uuid.UUID `json:"purchaseOrderId,omitempty"`
	InvoiceNumber   string     `json:"invoiceNumber"`
	InvoiceDate     string     `json:"invoiceDate"`
	DueDate         string     `json:"dueDate"`
	// TotalAmount is the amount before adjustments; with none of the fields
	// below it is also the final total (the original behaviour).
	TotalAmount     float64 `json:"totalAmount"`
	DiscountPercent float64 `json:"discountPercent"`
	DiscountAmount  float64 `json:"discountAmount"`
	AdditionalCost  float64 `json:"additionalCost"`
	RoundTo         float64 `json:"roundTo"`
	// ApplyVAT records the supplier's PPN on top of the amount after discount
	// and additional cost. VATRate 0 = statutory rate; VATOtherValueBase
	// defaults to true; VATCreditable defaults to true (PPN Masukan).
	ApplyVAT          bool    `json:"applyVat"`
	VATRate           float64 `json:"vatRate"`
	VATOtherValueBase *bool   `json:"vatOtherValueBase,omitempty"`
	VATCreditable     *bool   `json:"vatCreditable,omitempty"`
}

type RecordInvoicePaymentDTO struct {
	Amount float64 `json:"amount"`
}

type PurchaseInvoiceResponseDTO struct {
	ID                  uuid.UUID  `json:"id"`
	SupplierID          uuid.UUID  `json:"supplierId"`
	SupplierName        string     `json:"supplierName"`
	PurchaseOrderID     *uuid.UUID `json:"purchaseOrderId,omitempty"`
	PurchaseOrderNumber string     `json:"purchaseOrderNumber,omitempty"`
	InvoiceNumber       string     `json:"invoiceNumber"`
	InvoiceDate         string     `json:"invoiceDate"`
	DueDate             string     `json:"dueDate"`
	TotalAmount         float64    `json:"totalAmount"`
	Subtotal            float64    `json:"subtotal"`
	DiscountAmount      float64    `json:"discountAmount"`
	AdditionalCost      float64    `json:"additionalCost"`
	RoundingAmount      float64    `json:"roundingAmount"`
	TaxBase             float64    `json:"taxBase"`
	DPPOtherValue       float64    `json:"dppOtherValue"`
	VATRate             float64    `json:"vatRate"`
	VATOtherValueBase   bool       `json:"vatOtherValueBase"`
	VATAmount           float64    `json:"vatAmount"`
	VATCreditable       bool       `json:"vatCreditable"`
	PaidAmount          float64    `json:"paidAmount"`
	Outstanding         float64    `json:"outstanding"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"createdAt"`
}

func ToPurchaseInvoiceResponse(inv *domain.PurchaseInvoice, poNumber string) *PurchaseInvoiceResponseDTO {
	if inv == nil {
		return nil
	}
	return &PurchaseInvoiceResponseDTO{
		ID: inv.ID, SupplierID: inv.SupplierID, SupplierName: inv.SupplierName, PurchaseOrderID: inv.PurchaseOrderID,
		PurchaseOrderNumber: poNumber,
		InvoiceNumber:       inv.InvoiceNumber, InvoiceDate: inv.InvoiceDate, DueDate: inv.DueDate,
		TotalAmount: inv.TotalAmount, PaidAmount: inv.PaidAmount, Outstanding: inv.TotalAmount - inv.PaidAmount,
		Subtotal: inv.Subtotal, DiscountAmount: inv.DiscountAmount, AdditionalCost: inv.AdditionalCost, RoundingAmount: inv.RoundingAmount,
		TaxBase: inv.TaxBase, DPPOtherValue: inv.DPPOtherValue, VATRate: inv.VATRate, VATOtherValueBase: inv.VATOtherValueBase,
		VATAmount: inv.VATAmount, VATCreditable: inv.VATCreditable,
		Status: inv.Status, CreatedAt: inv.CreatedAt,
	}
}

func ToPurchaseInvoiceResponseList(items []domain.PurchaseInvoice, poNumbers map[uuid.UUID]string) []PurchaseInvoiceResponseDTO {
	result := make([]PurchaseInvoiceResponseDTO, len(items))
	for i, inv := range items {
		poNumber := ""
		if inv.PurchaseOrderID != nil {
			poNumber = poNumbers[*inv.PurchaseOrderID]
		}
		result[i] = *ToPurchaseInvoiceResponse(&inv, poNumber)
	}
	return result
}

// Purchase down payments

type CreateDownPaymentDTO struct {
	PurchaseOrderID uuid.UUID `json:"purchaseOrderId"`
	SupplierID      uuid.UUID `json:"supplierId"`
	SupplierName    string    `json:"supplierName"`
	Amount          float64   `json:"amount"`
	PaymentDate     string    `json:"paymentDate"`
	PaymentMethod   string    `json:"paymentMethod"`
	Notes           string    `json:"notes"`
}

type PurchaseDownPaymentResponseDTO struct {
	ID                  uuid.UUID `json:"id"`
	DPNumber            string    `json:"dpNumber"`
	PurchaseOrderID     uuid.UUID `json:"purchaseOrderId"`
	PurchaseOrderNumber string    `json:"purchaseOrderNumber,omitempty"`
	SupplierID          uuid.UUID `json:"supplierId"`
	SupplierName        string    `json:"supplierName"`
	Amount              float64   `json:"amount"`
	AppliedAmount       float64   `json:"appliedAmount"`
	RemainingAmount     float64   `json:"remainingAmount"`
	PaymentDate         string    `json:"paymentDate"`
	PaymentMethod       string    `json:"paymentMethod"`
	Status              string    `json:"status"`
	Notes               string    `json:"notes,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
}

func ToDownPaymentResponse(dp *domain.PurchaseDownPayment, poNumber string) *PurchaseDownPaymentResponseDTO {
	if dp == nil {
		return nil
	}
	return &PurchaseDownPaymentResponseDTO{
		ID: dp.ID, DPNumber: dp.DPNumber, PurchaseOrderID: dp.PurchaseOrderID, PurchaseOrderNumber: poNumber,
		SupplierID: dp.SupplierID, SupplierName: dp.SupplierName, Amount: dp.Amount, AppliedAmount: dp.AppliedAmount,
		RemainingAmount: dp.Amount - dp.AppliedAmount, PaymentDate: dp.PaymentDate, PaymentMethod: dp.PaymentMethod,
		Status: dp.Status, Notes: dp.Notes, CreatedAt: dp.CreatedAt,
	}
}

func ToDownPaymentResponseList(items []domain.PurchaseDownPayment, poNumbers map[uuid.UUID]string) []PurchaseDownPaymentResponseDTO {
	result := make([]PurchaseDownPaymentResponseDTO, len(items))
	for i, dp := range items {
		result[i] = *ToDownPaymentResponse(&dp, poNumbers[dp.PurchaseOrderID])
	}
	return result
}

// Purchase returns

type PurchaseReturnLineDTO struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
}

type CreatePurchaseReturnDTO struct {
	GoodsReceiptID uuid.UUID               `json:"goodsReceiptId"`
	ReturnDate     string                  `json:"returnDate"`
	Reason         string                  `json:"reason"`
	Lines          []PurchaseReturnLineDTO `json:"lines"`
}

type PurchaseReturnLineResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
	Subtotal  float64   `json:"subtotal"`
}

type PurchaseReturnResponseDTO struct {
	ID             uuid.UUID                       `json:"id"`
	ReturnNo       string                          `json:"returnNo"`
	GoodsReceiptID uuid.UUID                       `json:"goodsReceiptId"`
	SupplierID     uuid.UUID                       `json:"supplierId"`
	SupplierName   string                          `json:"supplierName"`
	ReturnDate     string                          `json:"returnDate"`
	Reason         string                          `json:"reason"`
	Status         string                          `json:"status"`
	TotalAmount    float64                         `json:"totalAmount"`
	Lines          []PurchaseReturnLineResponseDTO `json:"lines,omitempty"`
	CreatedAt      time.Time                       `json:"createdAt"`
}

func ToPurchaseReturnResponse(r *domain.PurchaseReturn) *PurchaseReturnResponseDTO {
	if r == nil {
		return nil
	}
	lines := make([]PurchaseReturnLineResponseDTO, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = PurchaseReturnLineResponseDTO{
			ID: l.ID, ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice, Subtotal: l.Subtotal,
		}
	}
	return &PurchaseReturnResponseDTO{
		ID: r.ID, ReturnNo: r.ReturnNo, GoodsReceiptID: r.GoodsReceiptID, SupplierID: r.SupplierID,
		SupplierName: r.SupplierName, ReturnDate: r.ReturnDate, Reason: r.Reason, Status: r.Status,
		TotalAmount: r.TotalAmount, Lines: lines, CreatedAt: r.CreatedAt,
	}
}

func ToPurchaseReturnResponseList(items []domain.PurchaseReturn) []PurchaseReturnResponseDTO {
	result := make([]PurchaseReturnResponseDTO, len(items))
	for i, r := range items {
		result[i] = *ToPurchaseReturnResponse(&r)
	}
	return result
}

// Invoice receipts

type InvoiceReceiptLineDTO struct {
	PurchaseInvoiceID *uuid.UUID `json:"purchaseInvoiceId,omitempty"`
	InvoiceNo         string     `json:"invoiceNo"`
	Amount            float64    `json:"amount"`
	Remarks           string     `json:"remarks"`
}

type CreateInvoiceReceiptDTO struct {
	Date         string                  `json:"date"`
	SupplierID   uuid.UUID               `json:"supplierId"`
	SupplierName string                  `json:"supplierName"`
	Notes        string                  `json:"notes"`
	Lines        []InvoiceReceiptLineDTO `json:"lines"`
}

type InvoiceReceiptLineResponseDTO struct {
	ID                uuid.UUID  `json:"id"`
	PurchaseInvoiceID *uuid.UUID `json:"purchaseInvoiceId,omitempty"`
	InvoiceNo         string     `json:"invoiceNo"`
	Amount            float64    `json:"amount"`
	Remarks           string     `json:"remarks"`
}

type InvoiceReceiptResponseDTO struct {
	ID           uuid.UUID                       `json:"id"`
	ReceiptNo    string                          `json:"receiptNo"`
	Date         string                          `json:"date"`
	SupplierID   uuid.UUID                       `json:"supplierId"`
	SupplierName string                          `json:"supplierName"`
	Notes        string                          `json:"notes"`
	Status       string                          `json:"status"`
	Lines        []InvoiceReceiptLineResponseDTO `json:"lines,omitempty"`
	CreatedAt    time.Time                       `json:"createdAt"`
}

func ToInvoiceReceiptResponse(ir *domain.InvoiceReceipt) *InvoiceReceiptResponseDTO {
	if ir == nil {
		return nil
	}
	lines := make([]InvoiceReceiptLineResponseDTO, len(ir.Lines))
	for i, l := range ir.Lines {
		lines[i] = InvoiceReceiptLineResponseDTO{
			ID: l.ID, PurchaseInvoiceID: l.PurchaseInvoiceID, InvoiceNo: l.InvoiceNo, Amount: l.Amount, Remarks: l.Remarks,
		}
	}
	return &InvoiceReceiptResponseDTO{
		ID: ir.ID, ReceiptNo: ir.ReceiptNo, Date: ir.Date, SupplierID: ir.SupplierID,
		SupplierName: ir.SupplierName, Notes: ir.Notes, Status: ir.Status, Lines: lines, CreatedAt: ir.CreatedAt,
	}
}

func ToInvoiceReceiptResponseList(items []domain.InvoiceReceipt) []InvoiceReceiptResponseDTO {
	result := make([]InvoiceReceiptResponseDTO, len(items))
	for i, ir := range items {
		result[i] = *ToInvoiceReceiptResponse(&ir)
	}
	return result
}
