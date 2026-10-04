package application

import (
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/google/uuid"
)

// BOM

type BOMLineDTO struct {
	ComponentProductID uuid.UUID `json:"componentProductId"`
	QuantityRequired   float64   `json:"quantityRequired"`
	Unit               string    `json:"unit"`
}

// BOMProcessDTO is one routing step; steps run in the order given.
type BOMProcessDTO struct {
	Name            string  `json:"name"`
	StandardMinutes float64 `json:"standardMinutes"`
	Notes           string  `json:"notes"`
}

type BOMProcessResponseDTO struct {
	ID              uuid.UUID `json:"id"`
	Sequence        int       `json:"sequence"`
	Name            string    `json:"name"`
	StandardMinutes float64   `json:"standardMinutes"`
	Notes           string    `json:"notes,omitempty"`
}

type CreateBOMDTO struct {
	ProductID uuid.UUID       `json:"productId"`
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	IsActive  *bool           `json:"isActive,omitempty"`
	Lines     []BOMLineDTO    `json:"lines"`
	Processes []BOMProcessDTO `json:"processes,omitempty"`
}

type BOMLineResponseDTO struct {
	ID                 uuid.UUID `json:"id"`
	ComponentProductID uuid.UUID `json:"componentProductId"`
	ComponentSKU       string    `json:"componentSku"`
	ComponentName      string    `json:"componentName"`
	QuantityRequired   float64   `json:"quantityRequired"`
	Unit               string    `json:"unit"`
}

type BOMResponseDTO struct {
	ID          uuid.UUID               `json:"id"`
	ProductID   uuid.UUID               `json:"productId"`
	ProductSKU  string                  `json:"productSku"`
	ProductName string                  `json:"productName"`
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	IsActive    bool                    `json:"isActive"`
	Lines       []BOMLineResponseDTO    `json:"lines"`
	Processes   []BOMProcessResponseDTO `json:"processes"`
	CreatedAt   time.Time               `json:"createdAt"`
}

func ToBOMLineResponse(l *domain.BOMLine, sku, name string) BOMLineResponseDTO {
	return BOMLineResponseDTO{
		ID:                 l.ID,
		ComponentProductID: l.ComponentProductID,
		ComponentSKU:       sku,
		ComponentName:      name,
		QuantityRequired:   l.QuantityRequired,
		Unit:               l.Unit,
	}
}

// Production orders

type CreateProductionOrderDTO struct {
	BOMID             uuid.UUID `json:"bomId"`
	QuantityToProduce int       `json:"quantityToProduce"`
	WarehouseID       uuid.UUID `json:"warehouseId"`
	PlannedDate       string    `json:"plannedDate"`
	// MaterialMode is "backflush" (default: materials are consumed when a batch is completed)
	// or "issued" (materials are issued to the order beforehand with a stock document).
	MaterialMode string `json:"materialMode"`
}

type StepResponseDTO struct {
	ID              uuid.UUID `json:"id"`
	Sequence        int       `json:"sequence"`
	Name            string    `json:"name"`
	StandardMinutes float64   `json:"standardMinutes"`
	QuantityDone    int       `json:"quantityDone"`
	// WaitingQuantity is how many units have reached this step but not finished it yet.
	WaitingQuantity int    `json:"waitingQuantity"`
	Status          string `json:"status"`
	StartedDate     string `json:"startedDate,omitempty"`
	CompletedDate   string `json:"completedDate,omitempty"`
}

type LogStepDTO struct {
	Quantity int    `json:"quantity"`
	Date     string `json:"date"`
	Notes    string `json:"notes"`
}

type ProductionOrderResponseDTO struct {
	ID                   uuid.UUID         `json:"id"`
	OrderNumber          string            `json:"orderNumber"`
	MaterialMode         string            `json:"materialMode"`
	Steps                []StepResponseDTO `json:"steps"`
	BOMID                uuid.UUID         `json:"bomId"`
	BOMName              string            `json:"bomName"`
	ProductID            uuid.UUID         `json:"productId"`
	ProductSKU           string            `json:"productSku"`
	ProductName          string            `json:"productName"`
	WarehouseID          uuid.UUID         `json:"warehouseId"`
	WarehouseName        string            `json:"warehouseName"`
	QuantityToProduce    int               `json:"quantityToProduce"`
	QuantityCompleted    int               `json:"quantityCompleted"`
	RemainingQuantity    int               `json:"remainingQuantity"`
	Status               string            `json:"status"`
	PlannedDate          string            `json:"plannedDate"`
	ActualCompletionDate string            `json:"actualCompletionDate"`
	CreatedAt            time.Time         `json:"createdAt"`
}

func ToProductionOrderResponse(o *domain.ProductionOrder, bomName string, productID uuid.UUID, productSKU, productName, warehouseName string) *ProductionOrderResponseDTO {
	if o == nil {
		return nil
	}
	mode := o.MaterialMode
	if mode == "" {
		mode = domain.MaterialBackflush
	}
	return &ProductionOrderResponseDTO{
		ID:                   o.ID,
		OrderNumber:          OrderLabel(o),
		MaterialMode:         mode,
		Steps:                stepResponses(o),
		BOMID:                o.BOMID,
		BOMName:              bomName,
		ProductID:            productID,
		ProductSKU:           productSKU,
		ProductName:          productName,
		WarehouseID:          o.WarehouseID,
		WarehouseName:        warehouseName,
		QuantityToProduce:    o.QuantityToProduce,
		QuantityCompleted:    o.QuantityCompleted,
		RemainingQuantity:    o.RemainingQuantity(),
		Status:               o.Status,
		PlannedDate:          o.PlannedDate,
		ActualCompletionDate: o.ActualCompletionDate,
		CreatedAt:            o.CreatedAt,
	}
}

// Production batches

type CompleteBatchDTO struct {
	QuantityCompleted int    `json:"quantityCompleted"`
	CompletionDate    string `json:"completionDate"`
	Notes             string `json:"notes"`
}

type ProductionBatchResponseDTO struct {
	ID                uuid.UUID `json:"id"`
	ProductionOrderID uuid.UUID `json:"productionOrderId"`
	QuantityCompleted int       `json:"quantityCompleted"`
	CompletionDate    string    `json:"completionDate"`
	Notes             string    `json:"notes"`
	CreatedAt         time.Time `json:"createdAt"`
}

func ToBatchResponse(b *domain.ProductionBatch) *ProductionBatchResponseDTO {
	if b == nil {
		return nil
	}
	return &ProductionBatchResponseDTO{
		ID:                b.ID,
		ProductionOrderID: b.ProductionOrderID,
		QuantityCompleted: b.QuantityCompleted,
		CompletionDate:    b.CompletionDate,
		Notes:             b.Notes,
		CreatedAt:         b.CreatedAt,
	}
}

// Dashboard

type DashboardSummaryDTO struct {
	ActiveBOMs              int64                        `json:"activeBoms"`
	OrdersByStatus          map[string]int64             `json:"ordersByStatus"`
	UnitsCompletedThisMonth int64                        `json:"unitsCompletedThisMonth"`
	UpcomingOrders          []ProductionOrderResponseDTO `json:"upcomingOrders"`
}

// OrderLabel is the order number, or a number derived from the id for orders created before numbering.
func OrderLabel(o *domain.ProductionOrder) string {
	if o.OrderNumber != "" {
		return o.OrderNumber
	}
	return "MO-" + strings.ToUpper(o.ID.String()[:8])
}

func stepResponses(o *domain.ProductionOrder) []StepResponseDTO {
	out := make([]StepResponseDTO, len(o.Steps))
	for i, s := range o.Steps {
		prev := o.QuantityToProduce
		if i > 0 {
			prev = o.Steps[i-1].QuantityDone
		}
		waiting := prev - s.QuantityDone
		if waiting < 0 {
			waiting = 0
		}
		out[i] = StepResponseDTO{ID: s.ID, Sequence: s.Sequence, Name: s.Name, StandardMinutes: s.StandardMinutes, QuantityDone: s.QuantityDone,
			WaitingQuantity: waiting, Status: s.Status, StartedDate: s.StartedDate, CompletedDate: s.CompletedDate}
	}
	return out
}
