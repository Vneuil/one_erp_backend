package application

import (
	"time"

	"github.com/google/uuid"
)

type CreateBrontolReportDTO struct {
	ReportDate         string     `json:"reportDate" validate:"required"` // YYYY-MM-DD
	SourceStage        string     `json:"sourceStage" validate:"required"`
	SourceInspectionID *uuid.UUID `json:"sourceInspectionId,omitempty"`
	SourceInspectionNo string     `json:"sourceInspectionNo,omitempty"`
	ProductionOrderID  *uuid.UUID `json:"productionOrderId,omitempty"`
	OrderNumber        string     `json:"orderNumber,omitempty"`
	ProductID          *uuid.UUID `json:"productId,omitempty"`
	ProductName        string     `json:"productName" validate:"required"`
	ProductCode        string     `json:"productCode,omitempty"`
	BatchNo            string     `json:"batchNo" validate:"required"`
	MaterialType       string     `json:"materialType,omitempty"` // BM, BO, BK, FG, WIP
	SupplierID         *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName       string     `json:"supplierName,omitempty"`
	MachineNo          string     `json:"machineNo,omitempty"`
	MeshSize           string     `json:"meshSize,omitempty"`
	Quantity           float64    `json:"quantity" validate:"required,min=0"`
	Unit               string     `json:"unit,omitempty"`
	DefectReason       string     `json:"defectReason" validate:"required"`
	ActionDisposition  string     `json:"actionDisposition" validate:"required"` // return_to_supplier, rework, re_oven, disposal, hold
	OperatorName       string     `json:"operatorName,omitempty"`
	QCInspectorName    string     `json:"qcInspectorName,omitempty"`
	Notes              string     `json:"notes,omitempty"`

	// Optional embedded return information when actionDisposition is return_to_supplier
	CreateReturn *CreateRejectReturnDTO `json:"createReturn,omitempty"`
}

type UpdateBrontolReportDTO struct {
	ReportDate        string     `json:"reportDate,omitempty"`
	ProductName       string     `json:"productName,omitempty"`
	ProductCode       string     `json:"productCode,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`
	MaterialType      string     `json:"materialType,omitempty"`
	SupplierID        *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName      string     `json:"supplierName,omitempty"`
	MachineNo         string     `json:"machineNo,omitempty"`
	MeshSize          string     `json:"meshSize,omitempty"`
	Quantity          *float64   `json:"quantity,omitempty"`
	Unit              string     `json:"unit,omitempty"`
	DefectReason      string     `json:"defectReason,omitempty"`
	ActionDisposition string     `json:"actionDisposition,omitempty"`
	OperatorName      string     `json:"operatorName,omitempty"`
	QCInspectorName   string     `json:"qcInspectorName,omitempty"`
	Status            string     `json:"status,omitempty"`
	Notes             string     `json:"notes,omitempty"`
}

type CreateRejectReturnDTO struct {
	ReturnDate     string     `json:"returnDate" validate:"required"` // YYYY-MM-DD
	SupplierID     *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName   string     `json:"supplierName" validate:"required"`
	PONo           string     `json:"poNo,omitempty"`
	GoodsReceiptNo string     `json:"goodsReceiptNo,omitempty"`
	DeliveryNoteNo string     `json:"deliveryNoteNo,omitempty"`
	ReturnQuantity float64    `json:"returnQuantity" validate:"required,min=0"`
	Unit           string     `json:"unit,omitempty"`
	UnitPrice      float64    `json:"unitPrice,omitempty"`
	EstimatedValue float64    `json:"estimatedValue,omitempty"`
	ReturnReason   string     `json:"returnReason" validate:"required"`
	CourierName    string     `json:"courierName,omitempty"`
	VehicleNumber  string     `json:"vehicleNumber,omitempty"`
	DriverName     string     `json:"driverName,omitempty"`
	PreparedBy     string     `json:"preparedBy" validate:"required"`
	ApprovedBy     string     `json:"approvedBy,omitempty"`
	Notes          string     `json:"notes,omitempty"`
}

type UpdateRejectReturnDTO struct {
	ReturnDate         string     `json:"returnDate,omitempty"`
	SupplierName       string     `json:"supplierName,omitempty"`
	PONo               string     `json:"poNo,omitempty"`
	GoodsReceiptNo     string     `json:"goodsReceiptNo,omitempty"`
	DeliveryNoteNo     string     `json:"deliveryNoteNo,omitempty"`
	ReturnQuantity     *float64   `json:"returnQuantity,omitempty"`
	UnitPrice          *float64   `json:"unitPrice,omitempty"`
	EstimatedValue     *float64   `json:"estimatedValue,omitempty"`
	ReturnReason       string     `json:"returnReason,omitempty"`
	CourierName        string     `json:"courierName,omitempty"`
	VehicleNumber      string     `json:"vehicleNumber,omitempty"`
	DriverName         string     `json:"driverName,omitempty"`
	ApprovedBy         string     `json:"approvedBy,omitempty"`
	ReceivedBySupplier string     `json:"receivedBySupplier,omitempty"`
	ShippedDate        *time.Time `json:"shippedDate,omitempty"`
	CompletedDate      *time.Time `json:"completedDate,omitempty"`
	Status             string     `json:"status,omitempty"` // pending_pickup, shipped, completed, cancelled
	Notes              string     `json:"notes,omitempty"`
}

type BrontolStatsResponseDTO struct {
	TotalCount      int64   `json:"totalCount"`
	TotalQuantityKg float64 `json:"totalQuantityKg"`
	ReturnCount     int64   `json:"returnCount"`
	ReturnQtyKg     float64 `json:"returnQtyKg"`
	ReworkCount     int64   `json:"reworkCount"`
}

// BrontolStockCardItemDTO corresponds to one line of Form REP-013-KSB
type BrontolStockCardItemDTO struct {
	ID                uuid.UUID `json:"id"`
	RowNo             int       `json:"rowNo"`
	ReportNo          string    `json:"reportNo"`
	BatchNo           string    `json:"batchNo"`
	ProductCode       string    `json:"productCode"`
	ProductName       string    `json:"productName"`
	IsBM              bool      `json:"isBM"` // true if BM
	IsBO              bool      `json:"isBO"` // true if BO
	InDate            string    `json:"inDate"`
	InQty             float64   `json:"inQty"`
	OutDate           string    `json:"outDate,omitempty"`
	OutQty            float64   `json:"outQty"`
	BalanceQty        float64   `json:"balanceQty"`
	MachineNo         string    `json:"machineNo,omitempty"`
	MeshSize          string    `json:"meshSize,omitempty"`
	OperatorName      string    `json:"operatorName,omitempty"`
	ActionDisposition string    `json:"actionDisposition"`
	Status            string    `json:"status"`
	HasReturnReport   bool      `json:"hasReturnReport"`
	ReturnNo          string    `json:"returnNo,omitempty"`
	SupplierName      string    `json:"supplierName,omitempty"`
}

// AutoBrontolTrigger carries details from a failed QC checkpoint across the production flow
type AutoBrontolTrigger struct {
	SourceStage        string
	SourceInspectionID *uuid.UUID
	SourceInspectionNo string
	ProductionOrderID  *uuid.UUID
	OrderNumber        string
	ProductID          *uuid.UUID
	ProductName        string
	ProductCode        string
	BatchNo            string
	MaterialType       string
	SupplierID         *uuid.UUID
	SupplierName       string
	MachineNo          string
	MeshSize           string
	Quantity           float64
	Unit               string
	DefectReason       string
	ActionDisposition  string
	OperatorName       string
	QCInspectorName    string
	Notes              string
	PONo               string
	GoodsReceiptNo     string
	DeliveryNoteNo     string
	UnitPrice          float64
}
