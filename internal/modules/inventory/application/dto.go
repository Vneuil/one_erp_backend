package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/google/uuid"
)

// Warehouses

type CreateWarehouseDTO struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	IsActive *bool  `json:"isActive,omitempty"`
}

type WarehouseResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

func ToWarehouseResponse(w *domain.Warehouse) *WarehouseResponseDTO {
	if w == nil {
		return nil
	}
	return &WarehouseResponseDTO{
		ID:        w.ID,
		Code:      w.Code,
		Name:      w.Name,
		Address:   w.Address,
		IsActive:  w.IsActive,
		CreatedAt: w.CreatedAt,
	}
}

func ToWarehouseResponseList(items []domain.Warehouse) []WarehouseResponseDTO {
	result := make([]WarehouseResponseDTO, len(items))
	for i, w := range items {
		result[i] = *ToWarehouseResponse(&w)
	}
	return result
}

// Stock levels

type StockLevelResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"productId"`
	ProductSKU    string    `json:"productSku"`
	ProductName   string    `json:"productName"`
	WarehouseID   uuid.UUID `json:"warehouseId"`
	WarehouseName string    `json:"warehouseName"`
	Quantity      int       `json:"quantity"`
	Reserved      int       `json:"reserved"`
	Available     int       `json:"available"`
	MinStock      int       `json:"minStock"`
	MaxStock      int       `json:"maxStock"`
	BufferStatus  string    `json:"bufferStatus,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func ToStockLevelResponse(s *domain.StockLevel, productSKU, productName, warehouseName string) *StockLevelResponseDTO {
	if s == nil {
		return nil
	}
	return &StockLevelResponseDTO{
		ID:            s.ID,
		ProductID:     s.ProductID,
		ProductSKU:    productSKU,
		ProductName:   productName,
		WarehouseID:   s.WarehouseID,
		WarehouseName: warehouseName,
		Quantity:      s.Quantity,
		Reserved:      s.Reserved,
		Available:     s.Available(),
		MinStock:      s.MinStock,
		MaxStock:      s.MaxStock,
		BufferStatus:  s.BufferStatus(),
		UpdatedAt:     s.UpdatedAt,
	}
}

type SetBufferStockDTO struct {
	ProductID   uuid.UUID `json:"productId"`
	WarehouseID uuid.UUID `json:"warehouseId"`
	MinStock    int       `json:"minStock"`
	MaxStock    int       `json:"maxStock"`
}

type AdjustStockDTO struct {
	ProductID   uuid.UUID `json:"productId"`
	WarehouseID uuid.UUID `json:"warehouseId"`
	Quantity    int       `json:"quantity"` // positive = increase, negative = decrease
	Reason      string    `json:"reason"`
	Reference   string    `json:"reference"`
	CreatedBy   string    `json:"createdBy"`
	// BatchNo (with an optional ExpiryDate) files an increase under that lot.
	BatchNo    string `json:"batchNo,omitempty"`
	ExpiryDate string `json:"expiryDate,omitempty"`
	// BatchID takes a decrease from that specific lot; without it, decreases
	// consume the soonest-expiring unexpired lots first.
	BatchID *uuid.UUID `json:"batchId,omitempty"`
}

// Movements

type MovementResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"productId"`
	ProductSKU    string    `json:"productSku"`
	ProductName   string    `json:"productName"`
	WarehouseID   uuid.UUID `json:"warehouseId"`
	WarehouseName string    `json:"warehouseName"`
	Type          string    `json:"type"`
	Quantity      int       `json:"quantity"`
	Balance       int       `json:"balance"`
	Reference     string    `json:"reference"`
	Reason        string    `json:"reason"`
	CreatedBy     string    `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
}

func ToMovementResponse(m *domain.StockMovement, productSKU, productName, warehouseName string) *MovementResponseDTO {
	if m == nil {
		return nil
	}
	return &MovementResponseDTO{
		ID:            m.ID,
		ProductID:     m.ProductID,
		ProductSKU:    productSKU,
		ProductName:   productName,
		WarehouseID:   m.WarehouseID,
		WarehouseName: warehouseName,
		Type:          m.Type,
		Quantity:      m.Quantity,
		Balance:       m.Balance,
		Reference:     m.Reference,
		Reason:        m.Reason,
		CreatedBy:     m.CreatedBy,
		CreatedAt:     m.CreatedAt,
	}
}

// Transfers

type CreateTransferDTO struct {
	ProductID       uuid.UUID `json:"productId"`
	FromWarehouseID uuid.UUID `json:"fromWarehouseId"`
	ToWarehouseID   uuid.UUID `json:"toWarehouseId"`
	Quantity        int       `json:"quantity"`
	RequestedBy     string    `json:"requestedBy"`
	Notes           string    `json:"notes"`
}

type TransferResponseDTO struct {
	ID                uuid.UUID `json:"id"`
	ProductID         uuid.UUID `json:"productId"`
	ProductSKU        string    `json:"productSku"`
	ProductName       string    `json:"productName"`
	FromWarehouseID   uuid.UUID `json:"fromWarehouseId"`
	FromWarehouseName string    `json:"fromWarehouseName"`
	ToWarehouseID     uuid.UUID `json:"toWarehouseId"`
	ToWarehouseName   string    `json:"toWarehouseName"`
	Quantity          int       `json:"quantity"`
	Status            string    `json:"status"`
	RequestedBy       string    `json:"requestedBy"`
	Notes             string    `json:"notes"`
	CreatedAt         time.Time `json:"createdAt"`
}

func ToTransferResponse(t *domain.StockTransfer, productSKU, productName, fromName, toName string) *TransferResponseDTO {
	if t == nil {
		return nil
	}
	return &TransferResponseDTO{
		ID:                t.ID,
		ProductID:         t.ProductID,
		ProductSKU:        productSKU,
		ProductName:       productName,
		FromWarehouseID:   t.FromWarehouseID,
		FromWarehouseName: fromName,
		ToWarehouseID:     t.ToWarehouseID,
		ToWarehouseName:   toName,
		Quantity:          t.Quantity,
		Status:            t.Status,
		RequestedBy:       t.RequestedBy,
		Notes:             t.Notes,
		CreatedAt:         t.CreatedAt,
	}
}

// Stock opname

type CreateOpnameDTO struct {
	WarehouseID uuid.UUID   `json:"warehouseId"`
	AuditDate   string      `json:"auditDate"`
	ProductIDs  []uuid.UUID `json:"productIds"`
}

type CountOpnameLineDTO struct {
	ProductID  uuid.UUID `json:"productId"`
	CountedQty int       `json:"countedQty"`
}

type OpnameLineResponseDTO struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"productId"`
	ProductSKU  string    `json:"productSku"`
	ProductName string    `json:"productName"`
	SystemQty   int       `json:"systemQty"`
	CountedQty  int       `json:"countedQty"`
	Variance    int       `json:"variance"`
	Counted     bool      `json:"counted"`
}

type OpnameResponseDTO struct {
	ID            uuid.UUID               `json:"id"`
	WarehouseID   uuid.UUID               `json:"warehouseId"`
	WarehouseName string                  `json:"warehouseName"`
	AuditDate     string                  `json:"auditDate"`
	Status        string                  `json:"status"`
	Lines         []OpnameLineResponseDTO `json:"lines"`
	CreatedAt     time.Time               `json:"createdAt"`
}

type ReceiveBatchDTO struct {
	ProductID   uuid.UUID `json:"productId"`
	WarehouseID uuid.UUID `json:"warehouseId"`
	BatchNo     string    `json:"batchNo"`
	ExpiryDate  string    `json:"expiryDate"` // YYYY-MM-DD, optional
	Quantity    int       `json:"quantity"`
	Reference   string    `json:"reference"`
}

// BatchResponseDTO is a lot with its product/warehouse names and expiry status.
type BatchResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"productId"`
	ProductSKU    string    `json:"productSku"`
	ProductName   string    `json:"productName"`
	WarehouseID   uuid.UUID `json:"warehouseId"`
	WarehouseName string    `json:"warehouseName"`
	BatchNo       string    `json:"batchNo"`
	ExpiryDate    string    `json:"expiryDate,omitempty"`
	ReceivedAt    string    `json:"receivedAt"`
	InitialQty    int       `json:"initialQty"`
	Quantity      int       `json:"quantity"`
	// Status is expired, expiring_soon (within 30 days), ok or no_expiry.
	Status       string `json:"status"`
	DaysToExpiry *int   `json:"daysToExpiry,omitempty"`
}
