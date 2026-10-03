package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Pick wave statuses
const (
	PickWavePending    = "pending"
	PickWaveInProgress = "in_progress"
	PickWaveCompleted  = "completed"
	PickWaveCancelled  = "cancelled"
)

// PickWave groups the lines to be picked from a warehouse, optionally against a sales order
type PickWave struct {
	types.BaseEntity
	TenantID     *uuid.UUID     `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	WarehouseID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"warehouseId"`
	SalesOrderID *uuid.UUID     `gorm:"type:uuid;index" json:"salesOrderId,omitempty"`
	Status       string         `gorm:"type:varchar(20);default:'pending'" json:"status"`
	AssignedTo   string         `gorm:"type:varchar(150)" json:"assignedTo"`
	Lines        []PickWaveLine `gorm:"foreignKey:PickWaveID" json:"lines,omitempty"`
}

func (PickWave) TableName() string {
	return "warehouse_pick_waves"
}

// PickWaveLine is a per-product pick line within a pick wave
type PickWaveLine struct {
	types.BaseEntity
	PickWaveID     uuid.UUID `gorm:"type:uuid;not null;index" json:"pickWaveId"`
	ProductID      uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	QuantityToPick int       `gorm:"not null" json:"quantityToPick"`
	QuantityPicked int       `gorm:"default:0" json:"quantityPicked"`
}

func (PickWaveLine) TableName() string {
	return "warehouse_pick_wave_lines"
}

// Packing session statuses
const (
	PackingOpen      = "open"
	PackingPacking   = "packing"
	PackingCompleted = "completed"
)

// PackingSession represents a packing station's work against a completed (or in-progress) pick wave
type PackingSession struct {
	types.BaseEntity
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	PickWaveID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"pickWaveId"`
	Status       string     `gorm:"type:varchar(20);default:'open'" json:"status"`
	PackageCount int        `gorm:"default:0" json:"packageCount"`
	Notes        string     `gorm:"type:varchar(255)" json:"notes"`
	StartedAt    *string    `gorm:"type:varchar(50)" json:"startedAt,omitempty"`
	CompletedAt  *string    `gorm:"type:varchar(50)" json:"completedAt,omitempty"`
}

func (PackingSession) TableName() string {
	return "warehouse_packing_sessions"
}

// Shipment statuses
const (
	ShipmentBooked     = "booked"
	ShipmentDispatched = "dispatched"
	ShipmentDelivered  = "delivered"
	ShipmentCancelled  = "cancelled"
)

// Shipment is the final outbound leg, tied to a packing session (or directly to a pick wave)
type Shipment struct {
	types.BaseEntity
	TenantID           *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	PackingSessionID   *uuid.UUID `gorm:"type:uuid;index" json:"packingSessionId,omitempty"`
	PickWaveID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"pickWaveId"`
	Carrier            string     `gorm:"type:varchar(150)" json:"carrier"`
	TrackingNumber     string     `gorm:"type:varchar(100)" json:"trackingNumber"`
	Status             string     `gorm:"type:varchar(20);default:'booked'" json:"status"`
	DestinationAddress string     `gorm:"type:varchar(255)" json:"destinationAddress"`
	DispatchedAt       *string    `gorm:"type:varchar(50)" json:"dispatchedAt,omitempty"`
}

func (Shipment) TableName() string {
	return "warehouse_shipments"
}

type WarehouseRepository interface {
	// Pick waves
	CreatePickWave(ctx context.Context, w *PickWave) error
	GetPickWaveByID(ctx context.Context, id uuid.UUID) (*PickWave, error)
	ListPickWaves(ctx context.Context, query types.PaginationQuery) ([]PickWave, int64, error)
	UpdatePickWave(ctx context.Context, w *PickWave) error
	UpdatePickWaveLine(ctx context.Context, l *PickWaveLine) error

	// Packing sessions
	CreatePackingSession(ctx context.Context, s *PackingSession) error
	GetPackingSessionByID(ctx context.Context, id uuid.UUID) (*PackingSession, error)
	ListPackingSessions(ctx context.Context, query types.PaginationQuery) ([]PackingSession, int64, error)
	UpdatePackingSession(ctx context.Context, s *PackingSession) error

	// Shipments
	CreateShipment(ctx context.Context, s *Shipment) error
	GetShipmentByID(ctx context.Context, id uuid.UUID) (*Shipment, error)
	ListShipments(ctx context.Context, query types.PaginationQuery) ([]Shipment, int64, error)
	UpdateShipment(ctx context.Context, s *Shipment) error

	WithTransaction(ctx context.Context, fn func(txRepo WarehouseRepository) error) error
}
