package application

import (
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

type CreateBOMDTO struct {
	ProductID uuid.UUID    `json:"productId"`
	Name      string       `json:"name"`
	Version   string       `json:"version"`
	IsActive  *bool        `json:"isActive,omitempty"`
	Lines     []BOMLineDTO `json:"lines"`
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
	ID          uuid.UUID            `json:"id"`
	ProductID   uuid.UUID            `json:"productId"`
	ProductSKU  string               `json:"productSku"`
	ProductName string               `json:"productName"`
	Name        string               `json:"name"`
	Version     string               `json:"version"`
	IsActive    bool                 `json:"isActive"`
	Lines       []BOMLineResponseDTO `json:"lines"`
	CreatedAt   time.Time            `json:"createdAt"`
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
}

type ProductionOrderResponseDTO struct {
	ID                   uuid.UUID `json:"id"`
	BOMID                uuid.UUID `json:"bomId"`
	BOMName              string    `json:"bomName"`
	ProductID            uuid.UUID `json:"productId"`
	ProductSKU           string    `json:"productSku"`
	ProductName          string    `json:"productName"`
	WarehouseID          uuid.UUID `json:"warehouseId"`
	WarehouseName        string    `json:"warehouseName"`
	QuantityToProduce    int       `json:"quantityToProduce"`
	QuantityCompleted    int       `json:"quantityCompleted"`
	RemainingQuantity    int       `json:"remainingQuantity"`
	Status               string    `json:"status"`
	PlannedDate          string    `json:"plannedDate"`
	ActualCompletionDate string    `json:"actualCompletionDate"`
	CreatedAt            time.Time `json:"createdAt"`
}

func ToProductionOrderResponse(o *domain.ProductionOrder, bomName string, productID uuid.UUID, productSKU, productName, warehouseName string) *ProductionOrderResponseDTO {
	if o == nil {
		return nil
	}
	return &ProductionOrderResponseDTO{
		ID:                   o.ID,
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
