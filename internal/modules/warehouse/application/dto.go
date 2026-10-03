package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/warehouse/domain"
	"github.com/google/uuid"
)

// Pick waves

type PickWaveLineDTO struct {
	ProductID      uuid.UUID `json:"productId"`
	QuantityToPick int       `json:"quantityToPick"`
}

type CreatePickWaveDTO struct {
	WarehouseID  uuid.UUID         `json:"warehouseId"`
	SalesOrderID *uuid.UUID        `json:"salesOrderId,omitempty"`
	AssignedTo   string            `json:"assignedTo"`
	Lines        []PickWaveLineDTO `json:"lines,omitempty"`
}

type PickLineDTO struct {
	ProductID      uuid.UUID `json:"productId"`
	QuantityPicked int       `json:"quantityPicked"`
}

type RecordPickDTO struct {
	Lines []PickLineDTO `json:"lines"`
}

type PickWaveLineResponseDTO struct {
	ID             uuid.UUID `json:"id"`
	ProductID      uuid.UUID `json:"productId"`
	ProductSKU     string    `json:"productSku"`
	ProductName    string    `json:"productName"`
	QuantityToPick int       `json:"quantityToPick"`
	QuantityPicked int       `json:"quantityPicked"`
}

type PickWaveResponseDTO struct {
	ID            uuid.UUID                 `json:"id"`
	WarehouseID   uuid.UUID                 `json:"warehouseId"`
	WarehouseName string                    `json:"warehouseName"`
	SalesOrderID  *uuid.UUID                `json:"salesOrderId,omitempty"`
	Status        string                    `json:"status"`
	AssignedTo    string                    `json:"assignedTo"`
	Lines         []PickWaveLineResponseDTO `json:"lines"`
	CreatedAt     time.Time                 `json:"createdAt"`
}

// Packing sessions

type CreatePackingSessionDTO struct {
	PickWaveID uuid.UUID `json:"pickWaveId"`
	Notes      string    `json:"notes"`
}

type CompletePackingSessionDTO struct {
	PackageCount int    `json:"packageCount"`
	Notes        string `json:"notes"`
}

type PackingSessionResponseDTO struct {
	ID           uuid.UUID `json:"id"`
	PickWaveID   uuid.UUID `json:"pickWaveId"`
	Status       string    `json:"status"`
	PackageCount int       `json:"packageCount"`
	Notes        string    `json:"notes"`
	StartedAt    *string   `json:"startedAt,omitempty"`
	CompletedAt  *string   `json:"completedAt,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Shipments

type CreateShipmentDTO struct {
	PickWaveID         uuid.UUID  `json:"pickWaveId"`
	PackingSessionID   *uuid.UUID `json:"packingSessionId,omitempty"`
	Carrier            string     `json:"carrier"`
	TrackingNumber     string     `json:"trackingNumber"`
	DestinationAddress string     `json:"destinationAddress"`
}

type ShipmentResponseDTO struct {
	ID                 uuid.UUID  `json:"id"`
	PickWaveID         uuid.UUID  `json:"pickWaveId"`
	PackingSessionID   *uuid.UUID `json:"packingSessionId,omitempty"`
	Carrier            string     `json:"carrier"`
	TrackingNumber     string     `json:"trackingNumber"`
	Status             string     `json:"status"`
	DestinationAddress string     `json:"destinationAddress"`
	DispatchedAt       *string    `json:"dispatchedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
}

func ToPackingSessionResponse(s *domain.PackingSession) *PackingSessionResponseDTO {
	if s == nil {
		return nil
	}
	return &PackingSessionResponseDTO{
		ID:           s.ID,
		PickWaveID:   s.PickWaveID,
		Status:       s.Status,
		PackageCount: s.PackageCount,
		Notes:        s.Notes,
		StartedAt:    s.StartedAt,
		CompletedAt:  s.CompletedAt,
		CreatedAt:    s.CreatedAt,
	}
}

func ToShipmentResponse(s *domain.Shipment) *ShipmentResponseDTO {
	if s == nil {
		return nil
	}
	return &ShipmentResponseDTO{
		ID:                 s.ID,
		PickWaveID:         s.PickWaveID,
		PackingSessionID:   s.PackingSessionID,
		Carrier:            s.Carrier,
		TrackingNumber:     s.TrackingNumber,
		Status:             s.Status,
		DestinationAddress: s.DestinationAddress,
		DispatchedAt:       s.DispatchedAt,
		CreatedAt:          s.CreatedAt,
	}
}
