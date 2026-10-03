package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/pos/domain"
	"github.com/google/uuid"
)

type CheckoutLineDTO struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
}

type CheckoutDTO struct {
	Outlet        string  `json:"outlet"`
	Cashier       string  `json:"cashier"`
	Customer      string  `json:"customer"`
	CustomerPhone string  `json:"customerPhone"`
	TotalItems    int     `json:"totalItems"`
	TotalAmount   float64 `json:"totalAmount"`
	PaymentMethod string  `json:"paymentMethod"`
	// WarehouseID + Lines are optional: when both are set, stock is
	// deducted per line from this warehouse - see usecase.Checkout.
	WarehouseID *uuid.UUID        `json:"warehouseId,omitempty"`
	Lines       []CheckoutLineDTO `json:"lines,omitempty"`
	// DiscountAmount is taken off the cart subtotal (lines only).
	DiscountAmount float64 `json:"discountAmount"`
	// AmountTendered is the cash handed over; required (and at least the total)
	// for cash sales made from a cart.
	AmountTendered float64 `json:"amountTendered"`
}

// RefundLineDTO names one sale line and how many units come back.
type RefundLineDTO struct {
	LineID   uuid.UUID `json:"lineId"`
	Quantity float64   `json:"quantity"`
}

type RefundDTO struct {
	Lines  []RefundLineDTO `json:"lines"`
	Reason string          `json:"reason"`
	// Restock puts the returned units back into stock (default true).
	Restock *bool `json:"restock,omitempty"`
}

type UpdateSettingsDTO struct {
	TaxPercent         float64 `json:"taxPercent"`
	TaxInclusive       bool    `json:"taxInclusive"`
	RoundTo            float64 `json:"roundTo"`
	DefaultOutlet      string  `json:"defaultOutlet"`
	ReceiptHeader      string  `json:"receiptHeader"`
	ReceiptFooter      string  `json:"receiptFooter"`
	NonCashAccountCode string  `json:"nonCashAccountCode"`
}

type POSTransactionResponseDTO struct {
	ID                   uuid.UUID                   `json:"id"`
	OrderNo              string                      `json:"orderNo"`
	Outlet               string                      `json:"outlet"`
	Cashier              string                      `json:"cashier"`
	Customer             string                      `json:"customer"`
	CustomerPhone        string                      `json:"customerPhone,omitempty"`
	TotalItems           int                         `json:"totalItems"`
	TotalAmount          float64                     `json:"totalAmount"`
	PaymentMethod        string                      `json:"paymentMethod"`
	Status               string                      `json:"status"`
	SalesOrderID         *uuid.UUID                  `json:"salesOrderId,omitempty"`
	SalesOrderNumber     string                      `json:"salesOrderNumber,omitempty"`
	LoyaltyMemberCode    string                      `json:"loyaltyMemberCode,omitempty"`
	LoyaltyPointsEarned  int                         `json:"loyaltyPointsEarned,omitempty"`
	LoyaltyPointsBalance int                         `json:"loyaltyPointsBalance,omitempty"`
	Subtotal             float64                     `json:"subtotal"`
	DiscountAmount       float64                     `json:"discountAmount"`
	TaxAmount            float64                     `json:"taxAmount"`
	RefundedAmount       float64                     `json:"refundedAmount"`
	AmountTendered       float64                     `json:"amountTendered"`
	ChangeAmount         float64                     `json:"changeAmount"`
	CashierEmail         string                      `json:"cashierEmail,omitempty"`
	VoidReason           string                      `json:"voidReason,omitempty"`
	Lines                []domain.POSTransactionLine `json:"lines,omitempty"`
	CreatedAt            time.Time                   `json:"createdAt"`
}

func ToPOSTransactionResponse(tx *domain.POSTransaction) *POSTransactionResponseDTO {
	if tx == nil {
		return nil
	}
	return &POSTransactionResponseDTO{
		ID:                   tx.ID,
		OrderNo:              tx.OrderNo,
		Outlet:               tx.Outlet,
		Cashier:              tx.Cashier,
		Customer:             tx.Customer,
		CustomerPhone:        tx.CustomerPhone,
		TotalItems:           tx.TotalItems,
		TotalAmount:          tx.TotalAmount,
		PaymentMethod:        tx.PaymentMethod,
		Status:               tx.Status,
		SalesOrderID:         tx.SalesOrderID,
		SalesOrderNumber:     tx.SalesOrderNumber,
		LoyaltyMemberCode:    tx.LoyaltyMemberCode,
		LoyaltyPointsEarned:  tx.LoyaltyPointsEarned,
		LoyaltyPointsBalance: tx.LoyaltyPointsBalance,
		Subtotal:             tx.Subtotal,
		DiscountAmount:       tx.DiscountAmount,
		TaxAmount:            tx.TaxAmount,
		RefundedAmount:       tx.RefundedAmount,
		AmountTendered:       tx.AmountTendered,
		ChangeAmount:         tx.ChangeAmount,
		CashierEmail:         tx.CashierEmail,
		VoidReason:           tx.VoidReason,
		Lines:                tx.Lines,
		CreatedAt:            tx.CreatedAt,
	}
}

func ToPOSTransactionResponseList(transactions []domain.POSTransaction) []POSTransactionResponseDTO {
	result := make([]POSTransactionResponseDTO, len(transactions))
	for i, tx := range transactions {
		result[i] = *ToPOSTransactionResponse(&tx)
	}
	return result
}
