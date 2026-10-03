package domain

import (
	"context"

	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// BillOfMaterial defines the recipe to produce one unit of an output product
type BillOfMaterial struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this BOM to one business unit within the company
	// (see modules/workspace). Nil means the company's default tenant.
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProductID uuid.UUID  `gorm:"type:uuid;not null;index" json:"productId"`
	Name      string     `gorm:"type:varchar(255);not null" json:"name"`
	Version   string     `gorm:"type:varchar(20);default:'v1.0'" json:"version"`
	IsActive  bool       `gorm:"default:true" json:"isActive"`
	Lines     []BOMLine  `gorm:"foreignKey:BOMID" json:"lines,omitempty"`
}

func (BillOfMaterial) TableName() string {
	return "manufacturing_boms"
}

// BOMLine is a component required to produce one unit of the BOM's output product
type BOMLine struct {
	types.BaseEntity
	BOMID              uuid.UUID `gorm:"type:uuid;not null;index" json:"bomId"`
	ComponentProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"componentProductId"`
	QuantityRequired   float64   `gorm:"type:decimal(15,4);not null" json:"quantityRequired"`
	Unit               string    `gorm:"type:varchar(20)" json:"unit"`
}

func (BOMLine) TableName() string {
	return "manufacturing_bom_lines"
}

// Production order statuses
const (
	OrderPlanned    = "planned"
	OrderReleased   = "released"
	OrderInProgress = "in_progress"
	OrderPaused     = "paused"
	OrderCompleted  = "completed"
	OrderCancelled  = "cancelled"
)

// ProductionOrder schedules the production of a BOM's output product
type ProductionOrder struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this production order to one business unit within
	// the company (see modules/workspace). Nil means the default tenant.
	TenantID             *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	BOMID                uuid.UUID  `gorm:"type:uuid;not null;index" json:"bomId"`
	WarehouseID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"warehouseId"`
	QuantityToProduce    int        `gorm:"not null" json:"quantityToProduce"`
	QuantityCompleted    int        `gorm:"default:0" json:"quantityCompleted"`
	Status               string     `gorm:"type:varchar(20);default:'planned'" json:"status"`
	PlannedDate          string     `gorm:"type:varchar(50)" json:"plannedDate"`
	ActualCompletionDate string     `gorm:"type:varchar(50)" json:"actualCompletionDate"`
}

func (ProductionOrder) TableName() string {
	return "manufacturing_production_orders"
}

// RemainingQuantity is the amount still left to produce
func (o ProductionOrder) RemainingQuantity() int {
	remaining := o.QuantityToProduce - o.QuantityCompleted
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ProductionBatch records one completed run against a production order
type ProductionBatch struct {
	types.BaseEntity
	ProductionOrderID uuid.UUID `gorm:"type:uuid;not null;index" json:"productionOrderId"`
	QuantityCompleted int       `gorm:"not null" json:"quantityCompleted"`
	CompletionDate    string    `gorm:"type:varchar(50)" json:"completionDate"`
	Notes             string    `gorm:"type:varchar(255)" json:"notes"`
}

func (ProductionBatch) TableName() string {
	return "manufacturing_production_batches"
}

type ManufacturingRepository interface {
	// BOM
	CreateBOM(ctx context.Context, b *BillOfMaterial) error
	GetBOMByID(ctx context.Context, id uuid.UUID) (*BillOfMaterial, error)
	ListBOMs(ctx context.Context, query types.PaginationQuery) ([]BillOfMaterial, int64, error)
	UpdateBOM(ctx context.Context, b *BillOfMaterial) error
	CountBOMs(ctx context.Context) (int64, error)

	// Production orders
	CreateOrder(ctx context.Context, o *ProductionOrder) error
	GetOrderByID(ctx context.Context, id uuid.UUID) (*ProductionOrder, error)
	ListOrders(ctx context.Context, query types.PaginationQuery) ([]ProductionOrder, int64, error)
	UpdateOrder(ctx context.Context, o *ProductionOrder) error

	// Production batches
	CreateBatch(ctx context.Context, b *ProductionBatch) error
	ListBatchesByOrder(ctx context.Context, orderID uuid.UUID) ([]ProductionBatch, error)

	// Dashboard aggregates
	CountOrdersByStatus(ctx context.Context) (map[string]int64, error)
	CountActiveBOMs(ctx context.Context) (int64, error)
	SumQuantityCompletedSince(ctx context.Context, since string) (int64, error)
	ListUpcomingOrders(ctx context.Context, limit int) ([]ProductionOrder, error)

	// Transactional mutation used by batch completion
	WithTransaction(ctx context.Context, fn func(txRepo ManufacturingRepository) error) error

	// WithTransactionAndInventory wraps both the manufacturing repo mutations
	// (production batch + order status) and the inventory repo mutations
	// (BOM consumption + finished-goods production) in a SINGLE database
	// transaction, so a mid-operation failure can never leave inventory
	// adjusted without the order reflecting completion (or vice versa).
	WithTransactionAndInventory(ctx context.Context, fn func(txRepo ManufacturingRepository, txInv inventoryDomain.InventoryRepository) error) error
}
