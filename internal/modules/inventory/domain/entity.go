package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Warehouse is a physical stock location
type Warehouse struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this warehouse to one business unit within the
	// company (see modules/workspace). Nil means the default tenant.
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Code     string     `gorm:"type:varchar(50);not null;index" json:"code"`
	Name     string     `gorm:"type:varchar(255);not null" json:"name"`
	Address  string     `gorm:"type:varchar(255)" json:"address"`
	IsActive bool       `gorm:"default:true" json:"isActive"`
}

func (Warehouse) TableName() string {
	return "inventory_warehouses"
}

// StockLevel tracks quantity on hand for a product at a warehouse
type StockLevel struct {
	types.BaseEntity
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProductID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"productId"`
	WarehouseID uuid.UUID  `gorm:"type:uuid;not null;index" json:"warehouseId"`
	Quantity    int        `gorm:"default:0" json:"quantity"`
	Reserved    int        `gorm:"default:0" json:"reserved"`
	MinStock    int        `gorm:"default:0" json:"minStock"`
	// MaxStock is the buffer stock ceiling; 0 means "not set" (no ceiling
	// alert). Paired with MinStock this gives min/max buffer stock alerting.
	MaxStock int `gorm:"default:0" json:"maxStock"`
}

// BufferStatus classifies this stock level against its min/max buffer
// (empty string when no buffer is configured for this row).
func (s StockLevel) BufferStatus() string {
	switch {
	case s.MinStock > 0 && s.Quantity <= s.MinStock:
		return "below_min"
	case s.MaxStock > 0 && s.Quantity >= s.MaxStock:
		return "above_max"
	case s.MinStock > 0 || s.MaxStock > 0:
		return "within_range"
	default:
		return ""
	}
}

func (StockLevel) TableName() string {
	return "inventory_stock_levels"
}

// Available is the quantity not reserved
func (s StockLevel) Available() int {
	avail := s.Quantity - s.Reserved
	if avail < 0 {
		return 0
	}
	return avail
}

// Movement types for the stock movement ledger
const (
	MovementIn          = "in"
	MovementOut         = "out"
	MovementAdjustment  = "adjustment"
	MovementTransferIn  = "transfer_in"
	MovementTransferOut = "transfer_out"
)

// StockMovement is an immutable ledger entry for a stock-level-changing event
type StockMovement struct {
	types.BaseEntity
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProductID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"productId"`
	WarehouseID uuid.UUID  `gorm:"type:uuid;not null;index" json:"warehouseId"`
	Type        string     `gorm:"type:varchar(20);not null" json:"type"`
	Quantity    int        `gorm:"not null" json:"quantity"`
	Balance     int        `gorm:"not null" json:"balance"`
	Reference   string     `gorm:"type:varchar(100)" json:"reference"`
	Reason      string     `gorm:"type:varchar(255)" json:"reason"`
	CreatedBy   string     `gorm:"type:varchar(150)" json:"createdBy"`
	// BatchNo names the lot a movement touched (empty when untracked or spread over several).
	BatchNo string `gorm:"type:varchar(100)" json:"batchNo,omitempty"`
}

// StockBatch is one received lot of a product in a warehouse, with an optional
// expiry date. Quantity is what remains of it. Stock levels stay the total; a
// batch adds traceability and first-expired-first-out consumption on top.
type StockBatch struct {
	types.BaseEntity
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProductID   uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_stock_batch" json:"productId"`
	WarehouseID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_stock_batch" json:"warehouseId"`
	BatchNo     string     `gorm:"type:varchar(100);not null;uniqueIndex:uq_stock_batch" json:"batchNo"`
	ExpiryDate  string     `gorm:"type:varchar(10);index" json:"expiryDate,omitempty"` // YYYY-MM-DD; empty = does not expire
	ReceivedAt  string     `gorm:"type:varchar(10)" json:"receivedAt"`
	InitialQty  int        `gorm:"not null" json:"initialQty"`
	Quantity    int        `gorm:"not null" json:"quantity"`
}

func (StockBatch) TableName() string { return "inventory_stock_batches" }

func (StockMovement) TableName() string {
	return "inventory_stock_movements"
}

// Stock transfer statuses
const (
	TransferPending   = "pending"
	TransferInTransit = "in_transit"
	TransferCompleted = "completed"
	TransferCancelled = "cancelled"
)

// StockTransfer moves stock of a product between two warehouses
type StockTransfer struct {
	types.BaseEntity
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProductID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"productId"`
	FromWarehouseID uuid.UUID  `gorm:"type:uuid;not null;index" json:"fromWarehouseId"`
	ToWarehouseID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"toWarehouseId"`
	Quantity        int        `gorm:"not null" json:"quantity"`
	Status          string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	RequestedBy     string     `gorm:"type:varchar(150)" json:"requestedBy"`
	Notes           string     `gorm:"type:varchar(255)" json:"notes"`
}

func (StockTransfer) TableName() string {
	return "inventory_stock_transfers"
}

// Stock opname statuses
const (
	OpnameDraft      = "draft"
	OpnameInProgress = "in_progress"
	OpnameCompleted  = "completed"
)

// StockOpname is a physical audit of a warehouse's stock
type StockOpname struct {
	types.BaseEntity
	TenantID    *uuid.UUID        `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	WarehouseID uuid.UUID         `gorm:"type:uuid;not null;index" json:"warehouseId"`
	AuditDate   string            `gorm:"type:varchar(50)" json:"auditDate"`
	Status      string            `gorm:"type:varchar(20);default:'draft'" json:"status"`
	Lines       []StockOpnameLine `gorm:"foreignKey:OpnameID" json:"lines,omitempty"`
}

func (StockOpname) TableName() string {
	return "inventory_stock_opnames"
}

// StockOpnameLine is a per-product count line within an opname
type StockOpnameLine struct {
	types.BaseEntity
	OpnameID   uuid.UUID `gorm:"type:uuid;not null;index" json:"opnameId"`
	ProductID  uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	SystemQty  int       `gorm:"not null" json:"systemQty"`
	CountedQty int       `gorm:"default:0" json:"countedQty"`
	Counted    bool      `gorm:"default:false" json:"counted"`
}

func (StockOpnameLine) TableName() string {
	return "inventory_stock_opname_lines"
}

// Variance is counted minus system quantity
func (l StockOpnameLine) Variance() int {
	return l.CountedQty - l.SystemQty
}

type InventoryRepository interface {
	// Warehouses
	CreateWarehouse(ctx context.Context, w *Warehouse) error
	GetWarehouseByID(ctx context.Context, id uuid.UUID) (*Warehouse, error)
	ListWarehouses(ctx context.Context, query types.PaginationQuery) ([]Warehouse, int64, error)
	ListAllWarehouses(ctx context.Context) ([]Warehouse, error)
	UpdateWarehouse(ctx context.Context, w *Warehouse) error
	DeleteWarehouse(ctx context.Context, id uuid.UUID) error
	CountWarehouses(ctx context.Context) (int64, error)

	// Stock levels
	GetStockLevel(ctx context.Context, productID, warehouseID uuid.UUID) (*StockLevel, error)
	HasAnyStockLevel(ctx context.Context, productID uuid.UUID) (bool, error)
	ListStockLevels(ctx context.Context, query types.PaginationQuery) ([]StockLevel, int64, error)
	ListStockLevelsByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]StockLevel, error)
	UpsertStockLevel(ctx context.Context, s *StockLevel) error

	// Batches
	GetBatch(ctx context.Context, id uuid.UUID) (*StockBatch, error)
	GetBatchByNo(ctx context.Context, productID, warehouseID uuid.UUID, batchNo string) (*StockBatch, error)
	SaveBatch(ctx context.Context, b *StockBatch) error
	ListBatches(ctx context.Context, productID, warehouseID *uuid.UUID, includeEmpty bool) ([]StockBatch, error)
	// ListBatchesFEFO lists a product's remaining batches in one warehouse, soonest
	// expiry first (batches that never expire last).
	ListBatchesFEFO(ctx context.Context, productID, warehouseID uuid.UUID) ([]StockBatch, error)
	// ListBatchesExpiringBy lists batches with stock left that expire on or before date.
	ListBatchesExpiringBy(ctx context.Context, date string) ([]StockBatch, error)

	// Movements
	CreateMovement(ctx context.Context, m *StockMovement) error
	ListMovements(ctx context.Context, query types.PaginationQuery) ([]StockMovement, int64, error)
	MovementsBetween(ctx context.Context, from, to time.Time, warehouseID *uuid.UUID) ([]StockMovement, error)

	// Transfers
	CreateTransfer(ctx context.Context, t *StockTransfer) error
	GetTransferByID(ctx context.Context, id uuid.UUID) (*StockTransfer, error)
	ListTransfers(ctx context.Context, query types.PaginationQuery) ([]StockTransfer, int64, error)
	UpdateTransfer(ctx context.Context, t *StockTransfer) error

	// Stock opname
	CreateOpname(ctx context.Context, o *StockOpname) error
	GetOpnameByID(ctx context.Context, id uuid.UUID) (*StockOpname, error)
	ListOpnames(ctx context.Context, query types.PaginationQuery) ([]StockOpname, int64, error)
	UpdateOpname(ctx context.Context, o *StockOpname) error
	UpdateOpnameLine(ctx context.Context, l *StockOpnameLine) error

	// Transactional stock mutation used by transfers and opname finalization
	WithTransaction(ctx context.Context, fn func(txRepo InventoryRepository) error) error
}
