package domain

import (
	"context"
	"time"

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
	// Processes is the routing: the ordered production steps (cutting, sewing,
	// QC...) every unit goes through. Empty for a BOM without routing.
	Processes []BOMProcess `gorm:"foreignKey:BOMID" json:"processes,omitempty"`
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

// BOMProcess is one step of a BOM's routing. StandardMinutes is the standard
// time to do the step for one unit (0 = not set).
type BOMProcess struct {
	types.BaseEntity
	BOMID           uuid.UUID `gorm:"type:uuid;not null;index" json:"bomId"`
	Sequence        int       `gorm:"not null" json:"sequence"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	StandardMinutes float64   `gorm:"type:decimal(10,2);default:0" json:"standardMinutes"`
	// Conversion cost rates per standard hour: direct labour and factory
	// overhead charged to Work in Process as units pass the step.
	LaborRatePerHour    float64 `gorm:"type:decimal(15,2);default:0" json:"laborRatePerHour"`
	OverheadRatePerHour float64 `gorm:"type:decimal(15,2);default:0" json:"overheadRatePerHour"`
	Notes               string  `gorm:"type:varchar(255)" json:"notes,omitempty"`
}

func (BOMProcess) TableName() string { return "manufacturing_bom_processes" }

// Material modes: how raw materials leave stock for a production order.
const (
	// MaterialBackflush consumes the BOM materials when a batch is completed (the original behaviour).
	MaterialBackflush = "backflush"
	// MaterialIssued expects the materials to be issued to the order beforehand
	// (stock document "Pengambilan Bahan"); completing a batch then consumes nothing.
	MaterialIssued = "issued"
)

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
	// OrderNumber is the document number (PRD-YYYYMM-NNNN); empty on orders created before numbering.
	OrderNumber  string `gorm:"type:varchar(50);index" json:"orderNumber"`
	MaterialMode string `gorm:"type:varchar(20);default:'backflush'" json:"materialMode"`
	// Steps are copied from the BOM's routing when the order is created, so later routing edits never change a running order.
	Steps []ProductionStep `gorm:"foreignKey:OrderID" json:"steps,omitempty"`
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

// Production step statuses.
const (
	StepPending    = "pending"
	StepInProgress = "in_progress"
	StepDone       = "done"
)

// ProductionStep is one routing step of a production order. QuantityDone counts
// the units that have finished this step; a unit can only finish a step after
// it finished the previous one.
type ProductionStep struct {
	types.BaseEntity
	OrderID         uuid.UUID `gorm:"type:uuid;not null;index" json:"orderId"`
	Sequence        int       `gorm:"not null" json:"sequence"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	StandardMinutes float64   `gorm:"type:decimal(10,2);default:0" json:"standardMinutes"`
	// Rates copied from the BOM when the order is created, so later rate changes never rewrite an order's cost.
	LaborRatePerHour    float64 `gorm:"type:decimal(15,2);default:0" json:"laborRatePerHour"`
	OverheadRatePerHour float64 `gorm:"type:decimal(15,2);default:0" json:"overheadRatePerHour"`
	QuantityDone        int     `gorm:"default:0" json:"quantityDone"`
	Status              string  `gorm:"type:varchar(20);default:'pending'" json:"status"`
	StartedDate         string  `gorm:"type:varchar(10)" json:"startedDate,omitempty"`
	CompletedDate       string  `gorm:"type:varchar(10)" json:"completedDate,omitempty"`
}

func (ProductionStep) TableName() string { return "manufacturing_production_steps" }

// UnitLabor / UnitOverhead are the standard direct labour and overhead cost of
// taking one unit through this step: standard hours times the hourly rate.
func (s ProductionStep) UnitLabor() float64    { return s.StandardMinutes / 60 * s.LaborRatePerHour }
func (s ProductionStep) UnitOverhead() float64 { return s.StandardMinutes / 60 * s.OverheadRatePerHour }

// ConversionPerUnit is the labour and overhead of taking one unit through every step of the order.
func (o ProductionOrder) ConversionPerUnit() (labor, overhead float64) {
	for _, s := range o.Steps {
		labor += s.UnitLabor()
		overhead += s.UnitOverhead()
	}
	return labor, overhead
}

// ProductionStepLog records units finishing a step on a date.
type ProductionStepLog struct {
	types.BaseEntity
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index" json:"orderId"`
	StepID    uuid.UUID `gorm:"type:uuid;not null;index" json:"stepId"`
	Quantity  int       `gorm:"not null" json:"quantity"`
	Date      string    `gorm:"type:varchar(10);not null;index" json:"date"`
	Notes     string    `gorm:"type:varchar(255)" json:"notes,omitempty"`
	CreatedBy string    `gorm:"type:varchar(255)" json:"createdBy,omitempty"`
	// Voided logs are kept for the record but no longer count towards the step, the reports or WIP.
	Voided     bool       `gorm:"not null;default:false;index" json:"voided"`
	VoidedAt   *time.Time `json:"voidedAt,omitempty"`
	VoidedBy   string     `gorm:"type:varchar(255)" json:"voidedBy,omitempty"`
	VoidReason string     `gorm:"type:varchar(255)" json:"voidReason,omitempty"`
}

func (ProductionStepLog) TableName() string { return "manufacturing_production_step_logs" }

// ProductionBatch records one completed run against a production order
type ProductionBatch struct {
	types.BaseEntity
	ProductionOrderID uuid.UUID `gorm:"type:uuid;not null;index" json:"productionOrderId"`
	QuantityCompleted int       `gorm:"not null" json:"quantityCompleted"`
	CompletionDate    string    `gorm:"type:varchar(50)" json:"completionDate"`
	Notes             string    `gorm:"type:varchar(255)" json:"notes"`
	// UnitCost is the full cost per finished unit (material plus conversion) folded into the
	// product's cost price when this batch was completed; CostApplied says whether it was.
	UnitCost    float64 `gorm:"type:decimal(15,2);default:0" json:"unitCost"`
	CostApplied bool    `gorm:"not null;default:false" json:"costApplied"`
	// Voided batches stay on record; their stock, cost and journal have been reversed.
	Voided     bool       `gorm:"not null;default:false;index" json:"voided"`
	VoidedAt   *time.Time `json:"voidedAt,omitempty"`
	VoidedBy   string     `gorm:"type:varchar(255)" json:"voidedBy,omitempty"`
	VoidReason string     `gorm:"type:varchar(255)" json:"voidReason,omitempty"`
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

	// Routing steps and their logs
	UpdateStep(ctx context.Context, s *ProductionStep) error
	CreateStepLog(ctx context.Context, l *ProductionStepLog) error
	// ListStepLogs returns the logs dated in the range that have not been voided.
	ListStepLogs(ctx context.Context, from, to string) ([]ProductionStepLog, error)
	// ListOrderStepLogs returns every log of one order, voided ones included, oldest first.
	ListOrderStepLogs(ctx context.Context, orderID uuid.UUID) ([]ProductionStepLog, error)
	GetStepLog(ctx context.Context, id uuid.UUID) (*ProductionStepLog, error)
	VoidStepLog(ctx context.Context, l *ProductionStepLog) error
	ListBatchesBetween(ctx context.Context, from, to string) ([]ProductionBatch, error)
	// ListAllOrders returns every production order with its steps, for reports.
	ListAllOrders(ctx context.Context) ([]ProductionOrder, error)
	CountOrdersByPrefix(ctx context.Context, prefix string) (int64, error)

	// Production batches
	CreateBatch(ctx context.Context, b *ProductionBatch) error
	ListBatchesByOrder(ctx context.Context, orderID uuid.UUID) ([]ProductionBatch, error)
	GetBatch(ctx context.Context, id uuid.UUID) (*ProductionBatch, error)
	VoidBatch(ctx context.Context, b *ProductionBatch) error
	// SetBatchCost records the unit cost folded into the product's cost price.
	SetBatchCost(ctx context.Context, id uuid.UUID, unitCost float64, applied bool) error

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
