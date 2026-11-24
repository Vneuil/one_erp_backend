package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/google/uuid"
)

type CreateQCParameterDTO struct {
	ProductID     *uuid.UUID           `json:"productId,omitempty"`
	ProcessCode   string               `json:"processCode" validate:"required"`
	Name          string               `json:"name" validate:"required"`
	ParamType     domain.ParameterType `json:"paramType" validate:"required"`
	UnitOfMeasure string               `json:"unitOfMeasure"`
	MinValue      *float64             `json:"minValue,omitempty"`
	MaxValue      *float64             `json:"maxValue,omitempty"`
	TargetValue   *float64             `json:"targetValue,omitempty"`
	Tolerance     *float64             `json:"tolerance,omitempty"`
	IsGate        bool                 `json:"isGate"`
	IsMandatory   bool                 `json:"isMandatory"`
	Description   string               `json:"description,omitempty"`
}

type UpdateQCParameterDTO struct {
	ProcessCode   string               `json:"processCode"`
	Name          string               `json:"name"`
	ParamType     domain.ParameterType `json:"paramType"`
	UnitOfMeasure string               `json:"unitOfMeasure"`
	MinValue      *float64             `json:"minValue,omitempty"`
	MaxValue      *float64             `json:"maxValue,omitempty"`
	TargetValue   *float64             `json:"targetValue,omitempty"`
	Tolerance     *float64             `json:"tolerance,omitempty"`
	IsGate        bool                 `json:"isGate"`
	IsMandatory   bool                 `json:"isMandatory"`
	Description   string               `json:"description,omitempty"`
}

type QCParameterResponseDTO struct {
	ID            uuid.UUID            `json:"id"`
	ProductID     *uuid.UUID           `json:"productId,omitempty"`
	ProcessCode   string               `json:"processCode"`
	Name          string               `json:"name"`
	ParamType     domain.ParameterType `json:"paramType"`
	UnitOfMeasure string               `json:"unitOfMeasure"`
	MinValue      *float64             `json:"minValue,omitempty"`
	MaxValue      *float64             `json:"maxValue,omitempty"`
	TargetValue   *float64             `json:"targetValue,omitempty"`
	Tolerance     *float64             `json:"tolerance,omitempty"`
	IsGate        bool                 `json:"isGate"`
	IsMandatory   bool                 `json:"isMandatory"`
	Description   string               `json:"description,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
	UpdatedAt     time.Time            `json:"updatedAt"`
}

type CreateQCInspectionItemDTO struct {
	ParameterID  uuid.UUID `json:"parameterId" validate:"required"`
	NumericValue *float64  `json:"numericValue,omitempty"`
	BooleanValue *bool     `json:"booleanValue,omitempty"`
	Notes        string    `json:"notes,omitempty"`
}

type CreateQCInspectionDTO struct {
	TargetType        domain.TargetType           `json:"targetType" validate:"required"`
	TargetID          uuid.UUID                   `json:"targetId" validate:"required"`
	ProductionOrderID *uuid.UUID                  `json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID                  `json:"productionStepId,omitempty"`
	BatchNo           string                      `json:"batchNo" validate:"required"`
	ProductID         uuid.UUID                   `json:"productId" validate:"required"`
	TestingTool       string                      `json:"testingTool,omitempty"`
	Notes             string                      `json:"notes,omitempty"`
	SugarCondition    string                      `json:"sugarCondition,omitempty"`
	PackageCondition  string                      `json:"packageCondition,omitempty"`
	Items             []CreateQCInspectionItemDTO `json:"items" validate:"required,min=1"`
}

type VoidQCInspectionDTO struct {
	Reason string `json:"reason" validate:"required"`
}

type QCInspectionItemResponseDTO struct {
	ID            uuid.UUID            `json:"id"`
	ParameterID   uuid.UUID            `json:"parameterId"`
	ParameterName string               `json:"parameterName"`
	ParamType     domain.ParameterType `json:"paramType"`
	NumericValue  *float64             `json:"numericValue,omitempty"`
	BooleanValue  *bool                `json:"booleanValue,omitempty"`
	IsPassed      bool                 `json:"isPassed"`
	Notes         string               `json:"notes,omitempty"`
}

type QCInspectionResponseDTO struct {
	ID                uuid.UUID                     `json:"id"`
	InspectionNo      string                        `json:"inspectionNo"`
	TargetType        domain.TargetType             `json:"targetType"`
	TargetID          uuid.UUID                     `json:"targetId"`
	ProductionOrderID *uuid.UUID                    `json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID                    `json:"productionStepId,omitempty"`
	BatchNo           string                        `json:"batchNo"`
	ProductID         uuid.UUID                     `json:"productId"`
	InspectorID       uuid.UUID                     `json:"inspectorId"`
	InspectorName     string                        `json:"inspectorName"`
	InspectedAt       time.Time                     `json:"inspectedAt"`
	TestingTool       string                        `json:"testingTool,omitempty"`
	Iteration         int                           `json:"iteration"`
	OverallStatus     domain.InspectionStatus       `json:"overallStatus"`
	Notes             string                        `json:"notes,omitempty"`
	SugarCondition    string                        `json:"sugarCondition,omitempty"`
	PackageCondition  string                        `json:"packageCondition,omitempty"`
	IsVoided          bool                          `json:"isVoided"`
	VoidReason        string                        `json:"voidReason,omitempty"`
	VoidedBy          *uuid.UUID                    `json:"voidedBy,omitempty"`
	VoidedAt          *time.Time                    `json:"voidedAt,omitempty"`
	Items             []QCInspectionItemResponseDTO `json:"items,omitempty"`
	CreatedAt         time.Time                     `json:"createdAt"`
	UpdatedAt         time.Time                     `json:"updatedAt"`
}

type QCStatsResponseDTO struct {
	TotalInspections  int64   `json:"totalInspections"`
	PassedInspections int64   `json:"passedInspections"`
	FailedInspections int64   `json:"failedInspections"`
	VoidedInspections int64   `json:"voidedInspections"`
	PassRate          float64 `json:"passRate"`
}

type CreateHygieneCheckDTO struct {
	CheckDate      string `json:"checkDate" validate:"required"`
	Shift          string `json:"shift" validate:"required"`
	Line           string `json:"line" validate:"required"`
	EmployeeName   string `json:"employeeName" validate:"required"`
	WorkWearClean  bool   `json:"workWearClean"`
	HairNet        bool   `json:"hairNet"`
	NailsClean     bool   `json:"nailsClean"`
	NoJewelry      bool   `json:"noJewelry"`
	WorkShoesClean bool   `json:"workShoesClean"`
	HandWashing    bool   `json:"handWashing"`
	HealthFit      bool   `json:"healthFit"`
	IsPassed       bool   `json:"isPassed"`
	ActionTaken    string `json:"actionTaken,omitempty"`
	InspectorName  string `json:"inspectorName" validate:"required"`
	Notes          string `json:"notes,omitempty"`
}

type CreatePreOpCheckDTO struct {
	CheckDate          string   `json:"checkDate" validate:"required"`
	CheckTime          string   `json:"checkTime" validate:"required"`
	Shift              string   `json:"shift" validate:"required"`
	Room               string   `json:"room" validate:"required"`
	FloorClean         bool     `json:"floorClean"`
	WallCeilingClean   bool     `json:"wallCeilingClean"`
	EquipmentSanitized bool     `json:"equipmentSanitized"`
	TrashBinClosed     bool     `json:"trashBinClosed"`
	LightingProtected  bool     `json:"lightingProtected"`
	PestControlIntact  bool     `json:"pestControlIntact"`
	GaussReading       *float64 `json:"gaussReading,omitempty"`
	MetalTrapClean     *bool    `json:"metalTrapClean,omitempty"`
	Status             string   `json:"status"`
	Notes              string   `json:"notes,omitempty"`
	InspectorName      string   `json:"inspectorName" validate:"required"`
	SupervisorName     string   `json:"supervisorName,omitempty"`
}

// ─── PBK (Packaging Quality Check) DTOs ──────────────────────────────────────

type CreatePackagingCheckDTO struct {
	CheckMonth      string     `json:"checkMonth" validate:"required"` // YYYY-MM
	GoodsReceiptID  *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo           string     `json:"lbmNo,omitempty"`
	CheckDate       string     `json:"checkDate" validate:"required"`
	SupplierID      *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName    string     `json:"supplierName" validate:"required"`
	ItemName        string     `json:"itemName" validate:"required"`
	ItemCode        string     `json:"itemCode,omitempty"`
	MaterialType    string     `json:"materialType" validate:"required"` // plastik, kardus, jerigen, pallet_plastik, pallet_kayu
	BaseColor       string     `json:"baseColor,omitempty"`
	Dimension       string     `json:"dimension,omitempty"`
	TotalQty        int        `json:"totalQty" validate:"required,min=1"`
	SampleQty       int        `json:"sampleQty" validate:"required,min=1"`
	PassedQty       int        `json:"passedQty"`
	PassedCondition string     `json:"passedCondition,omitempty"`
	FailedQty       int        `json:"failedQty"`
	FailedCondition string     `json:"failedCondition,omitempty"`
	QCName          string     `json:"qcName,omitempty"`
	QAName          string     `json:"qaName,omitempty"`
	Notes           string     `json:"notes,omitempty"`
}

type UpdatePackagingCheckDTO struct {
	CheckMonth      string     `json:"checkMonth,omitempty"`
	GoodsReceiptID  *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo           string     `json:"lbmNo,omitempty"`
	CheckDate       string     `json:"checkDate,omitempty"`
	SupplierID      *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName    string     `json:"supplierName,omitempty"`
	ItemName        string     `json:"itemName,omitempty"`
	ItemCode        string     `json:"itemCode,omitempty"`
	MaterialType    string     `json:"materialType,omitempty"`
	BaseColor       string     `json:"baseColor,omitempty"`
	Dimension       string     `json:"dimension,omitempty"`
	TotalQty        int        `json:"totalQty,omitempty"`
	SampleQty       int        `json:"sampleQty,omitempty"`
	PassedQty       int        `json:"passedQty"`
	PassedCondition string     `json:"passedCondition,omitempty"`
	FailedQty       int        `json:"failedQty"`
	FailedCondition string     `json:"failedCondition,omitempty"`
	QCName          string     `json:"qcName,omitempty"`
	QAName          string     `json:"qaName,omitempty"`
	Notes           string     `json:"notes,omitempty"`
}

type PackagingCheckResponseDTO struct {
	ID              uuid.UUID  `json:"id"`
	DocumentNo      string     `json:"documentNo"`
	CheckMonth      string     `json:"checkMonth"`
	GoodsReceiptID  *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo           string     `json:"lbmNo,omitempty"`
	CheckDate       string     `json:"checkDate"`
	SupplierID      *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName    string     `json:"supplierName"`
	ItemName        string     `json:"itemName"`
	ItemCode        string     `json:"itemCode,omitempty"`
	MaterialType    string     `json:"materialType"`
	BaseColor       string     `json:"baseColor,omitempty"`
	Dimension       string     `json:"dimension,omitempty"`
	TotalQty        int        `json:"totalQty"`
	SampleQty       int        `json:"sampleQty"`
	PassedQty       int        `json:"passedQty"`
	PassedCondition string     `json:"passedCondition,omitempty"`
	FailedQty       int        `json:"failedQty"`
	FailedCondition string     `json:"failedCondition,omitempty"`
	OverallStatus   string     `json:"overallStatus"`
	QCName          string     `json:"qcName,omitempty"`
	QCDate          *time.Time `json:"qcDate,omitempty"`
	QAName          string     `json:"qaName,omitempty"`
	QADate          *time.Time `json:"qaDate,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// ─── CAS (Supplier Armada Check) DTOs ────────────────────────────────────────

type CreateArmadaCheckDTO struct {
	GoodsReceiptID    *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo             string     `json:"lbmNo,omitempty"`
	CheckDate         string     `json:"checkDate" validate:"required"`
	IDNumber          string     `json:"idNumber,omitempty"`
	DriverName        string     `json:"driverName" validate:"required"`
	BirthPlace        string     `json:"birthPlace,omitempty"`
	BirthDate         string     `json:"birthDate,omitempty"`
	Gender            string     `json:"gender,omitempty"`
	Address           string     `json:"address,omitempty"`
	Religion          string     `json:"religion,omitempty"`
	MaritalStatus     string     `json:"maritalStatus,omitempty"`
	IDExpiry          string     `json:"idExpiry,omitempty"`
	CompanyOrigin     string     `json:"companyOrigin,omitempty"`
	CargoDesc         string     `json:"cargoDesc,omitempty"`
	LicensePlate      string     `json:"licensePlate,omitempty"`
	VehicleType       string     `json:"vehicleType,omitempty"`
	CabinSafe         bool       `json:"cabinSafe"`
	UndercarriageSafe bool       `json:"undercarriageSafe"`
	CargoAreaSafe     bool       `json:"cargoAreaSafe"`
	FloorClean        bool       `json:"floorClean"`
	WallRightClean    bool       `json:"wallRightClean"`
	WallLeftClean     bool       `json:"wallLeftClean"`
	CeilingClean      bool       `json:"ceilingClean"`
	DoorClean         bool       `json:"doorClean"`
	QCName            string     `json:"qcName,omitempty"`
	QAName            string     `json:"qaName,omitempty"`
	Notes             string     `json:"notes,omitempty"`
}

type UpdateArmadaCheckDTO struct {
	GoodsReceiptID    *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo             string     `json:"lbmNo,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	IDNumber          string     `json:"idNumber,omitempty"`
	DriverName        string     `json:"driverName,omitempty"`
	BirthPlace        string     `json:"birthPlace,omitempty"`
	BirthDate         string     `json:"birthDate,omitempty"`
	Gender            string     `json:"gender,omitempty"`
	Address           string     `json:"address,omitempty"`
	Religion          string     `json:"religion,omitempty"`
	MaritalStatus     string     `json:"maritalStatus,omitempty"`
	IDExpiry          string     `json:"idExpiry,omitempty"`
	CompanyOrigin     string     `json:"companyOrigin,omitempty"`
	CargoDesc         string     `json:"cargoDesc,omitempty"`
	LicensePlate      string     `json:"licensePlate,omitempty"`
	VehicleType       string     `json:"vehicleType,omitempty"`
	CabinSafe         bool       `json:"cabinSafe"`
	UndercarriageSafe bool       `json:"undercarriageSafe"`
	CargoAreaSafe     bool       `json:"cargoAreaSafe"`
	FloorClean        bool       `json:"floorClean"`
	WallRightClean    bool       `json:"wallRightClean"`
	WallLeftClean     bool       `json:"wallLeftClean"`
	CeilingClean      bool       `json:"ceilingClean"`
	DoorClean         bool       `json:"doorClean"`
	QCName            string     `json:"qcName,omitempty"`
	QAName            string     `json:"qaName,omitempty"`
	Notes             string     `json:"notes,omitempty"`
}

type ArmadaCheckResponseDTO struct {
	ID                uuid.UUID  `json:"id"`
	DocumentNo        string     `json:"documentNo"`
	GoodsReceiptID    *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo             string     `json:"lbmNo,omitempty"`
	CheckDate         string     `json:"checkDate"`
	IDNumber          string     `json:"idNumber,omitempty"`
	DriverName        string     `json:"driverName"`
	BirthPlace        string     `json:"birthPlace,omitempty"`
	BirthDate         string     `json:"birthDate,omitempty"`
	Gender            string     `json:"gender,omitempty"`
	Address           string     `json:"address,omitempty"`
	Religion          string     `json:"religion,omitempty"`
	MaritalStatus     string     `json:"maritalStatus,omitempty"`
	IDExpiry          string     `json:"idExpiry,omitempty"`
	CompanyOrigin     string     `json:"companyOrigin,omitempty"`
	CargoDesc         string     `json:"cargoDesc,omitempty"`
	LicensePlate      string     `json:"licensePlate,omitempty"`
	VehicleType       string     `json:"vehicleType,omitempty"`
	CabinSafe         bool       `json:"cabinSafe"`
	UndercarriageSafe bool       `json:"undercarriageSafe"`
	CargoAreaSafe     bool       `json:"cargoAreaSafe"`
	FloorClean        bool       `json:"floorClean"`
	WallRightClean    bool       `json:"wallRightClean"`
	WallLeftClean     bool       `json:"wallLeftClean"`
	CeilingClean      bool       `json:"ceilingClean"`
	DoorClean         bool       `json:"doorClean"`
	OverallStatus     string     `json:"overallStatus"`
	QCName            string     `json:"qcName,omitempty"`
	QCDate            *time.Time `json:"qcDate,omitempty"`
	QAName            string     `json:"qaName,omitempty"`
	QADate            *time.Time `json:"qaDate,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// ─── PBD (Raw Material Incoming Check) DTOs ─────────────────────────────────

type CreateIncomingCheckDTO struct {
	GoodsReceiptID     *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo              string     `json:"lbmNo,omitempty"`
	CheckDate          string     `json:"checkDate" validate:"required"` // YYYY-MM-DD
	CheckMonth         string     `json:"checkMonth,omitempty"`          // YYYY-MM
	SupplierID         *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName       string     `json:"supplierName" validate:"required"`
	DeliveryNoteNo     string     `json:"deliveryNoteNo,omitempty"`
	PONo               string     `json:"poNo,omitempty"`
	ProductName        string     `json:"productName" validate:"required"`
	BatchNo            string     `json:"batchNo,omitempty"`
	TotalBags          int        `json:"totalBags"`
	SampleBags         int        `json:"sampleBags"`
	GrossWeight        float64    `json:"grossWeight"`
	NetWeight          float64    `json:"netWeight"`
	AcceptedWeight     float64    `json:"acceptedWeight"`
	RejectedWeight     float64    `json:"rejectedWeight"`
	SugarCondition     string     `json:"sugarCondition" validate:"required"`
	ColorAppearance    string     `json:"colorAppearance,omitempty"`
	AromaFlavor        string     `json:"aromaFlavor,omitempty"`
	ForeignMatterClean bool       `json:"foreignMatterClean"`
	PackagingCondition string     `json:"packagingCondition,omitempty"`
	MoistureContent    *float64   `json:"moistureContent,omitempty"`
	BrixValue          *float64   `json:"brixValue,omitempty"`
	MeshSize           string     `json:"meshSize,omitempty"`
	RecommendedAction  string     `json:"recommendedAction,omitempty"`
	QCName             string     `json:"qcName,omitempty"`
	QAName             string     `json:"qaName,omitempty"`
	Notes              string     `json:"notes,omitempty"`
}

type UpdateIncomingCheckDTO struct {
	GoodsReceiptID     *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo              string     `json:"lbmNo,omitempty"`
	CheckDate          string     `json:"checkDate,omitempty"`
	CheckMonth         string     `json:"checkMonth,omitempty"`
	SupplierID         *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName       string     `json:"supplierName,omitempty"`
	DeliveryNoteNo     string     `json:"deliveryNoteNo,omitempty"`
	PONo               string     `json:"poNo,omitempty"`
	ProductName        string     `json:"productName,omitempty"`
	BatchNo            string     `json:"batchNo,omitempty"`
	TotalBags          int        `json:"totalBags,omitempty"`
	SampleBags         int        `json:"sampleBags,omitempty"`
	GrossWeight        float64    `json:"grossWeight,omitempty"`
	NetWeight          float64    `json:"netWeight,omitempty"`
	AcceptedWeight     float64    `json:"acceptedWeight,omitempty"`
	RejectedWeight     float64    `json:"rejectedWeight,omitempty"`
	SugarCondition     string     `json:"sugarCondition,omitempty"`
	ColorAppearance    string     `json:"colorAppearance,omitempty"`
	AromaFlavor        string     `json:"aromaFlavor,omitempty"`
	ForeignMatterClean bool       `json:"foreignMatterClean"`
	PackagingCondition string     `json:"packagingCondition,omitempty"`
	MoistureContent    *float64   `json:"moistureContent,omitempty"`
	BrixValue          *float64   `json:"brixValue,omitempty"`
	MeshSize           string     `json:"meshSize,omitempty"`
	RecommendedAction  string     `json:"recommendedAction,omitempty"`
	QCName             string     `json:"qcName,omitempty"`
	QAName             string     `json:"qaName,omitempty"`
	Notes              string     `json:"notes,omitempty"`
}

type IncomingCheckResponseDTO struct {
	ID                 uuid.UUID  `json:"id"`
	DocumentNo         string     `json:"documentNo"`
	GoodsReceiptID     *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo              string     `json:"lbmNo,omitempty"`
	CheckDate          string     `json:"checkDate"`
	CheckMonth         string     `json:"checkMonth"`
	SupplierID         *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName       string     `json:"supplierName"`
	DeliveryNoteNo     string     `json:"deliveryNoteNo,omitempty"`
	PONo               string     `json:"poNo,omitempty"`
	ProductName        string     `json:"productName"`
	BatchNo            string     `json:"batchNo,omitempty"`
	TotalBags          int        `json:"totalBags"`
	SampleBags         int        `json:"sampleBags"`
	GrossWeight        float64    `json:"grossWeight"`
	NetWeight          float64    `json:"netWeight"`
	AcceptedWeight     float64    `json:"acceptedWeight"`
	RejectedWeight     float64    `json:"rejectedWeight"`
	SugarCondition     string     `json:"sugarCondition"`
	ColorAppearance    string     `json:"colorAppearance,omitempty"`
	AromaFlavor        string     `json:"aromaFlavor,omitempty"`
	ForeignMatterClean bool       `json:"foreignMatterClean"`
	PackagingCondition string     `json:"packagingCondition,omitempty"`
	MoistureContent    *float64   `json:"moistureContent,omitempty"`
	BrixValue          *float64   `json:"brixValue,omitempty"`
	MeshSize           string     `json:"meshSize,omitempty"`
	RecommendedAction  string     `json:"recommendedAction,omitempty"`
	OverallStatus      string     `json:"overallStatus"`
	QCName             string     `json:"qcName,omitempty"`
	QCDate             *time.Time `json:"qcDate,omitempty"`
	QAName             string     `json:"qaName,omitempty"`
	QADate             *time.Time `json:"qaDate,omitempty"`
	Notes              string     `json:"notes,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Monitoring Ayak (MNA, CMT, MNM) & Oven (MNO) DTOs
// ─────────────────────────────────────────────────────────────────────────────

// ─── MNA (Sieving Monitoring Log) DTOs ───────────────────────────────────────

type CreateSievingLogDTO struct {
	MachineNo         string     `json:"machineNo" validate:"required"`
	CheckMonth        string     `json:"checkMonth,omitempty"` // YYYY-MM
	CheckDate         string     `json:"checkDate" validate:"required"` // YYYY-MM-DD
	Category          string     `json:"category,omitempty"` // Mentah, Oven (default: Mentah)
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID `json:"productionStepId,omitempty"`
	RequisitionID     *uuid.UUID `json:"requisitionId,omitempty"`
	ProductID         *uuid.UUID `json:"productId,omitempty"`
	ProductName       string     `json:"productName" validate:"required"`
	BatchNo           string     `json:"batchNo" validate:"required"`
	OvenDate          string     `json:"ovenDate,omitempty"`
	MeshSize          string     `json:"meshSize" validate:"required"`

	// Pemeriksaan AWAL
	InitialCondition  string `json:"initialCondition,omitempty"`
	InitialFindings   string `json:"initialFindings,omitempty"`
	InitialAction     string `json:"initialAction,omitempty"`
	InitialPICName    string `json:"initialPicName" validate:"required"`
	InitialQCName     string `json:"initialQcName,omitempty"`
	InitialQCPassed   *bool  `json:"initialQcPassed,omitempty"`

	// Pemeriksaan AKHIR
	FinalCondition    string `json:"finalCondition,omitempty"`
	FinalFindings     string `json:"finalFindings,omitempty"`
	FinalAction       string `json:"finalAction,omitempty"`
	FinalPICName      string `json:"finalPicName" validate:"required"`
	FinalQCName       string `json:"finalQcName,omitempty"`
	FinalQCPassed     *bool  `json:"finalQcPassed,omitempty"`

	OverallStatus     string `json:"overallStatus,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

type UpdateSievingLogDTO struct {
	MachineNo         string     `json:"machineNo,omitempty"`
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	Category          string     `json:"category,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID `json:"productionStepId,omitempty"`
	RequisitionID     *uuid.UUID `json:"requisitionId,omitempty"`
	ProductID         *uuid.UUID `json:"productId,omitempty"`
	ProductName       string     `json:"productName,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`
	OvenDate          string     `json:"ovenDate,omitempty"`
	MeshSize          string     `json:"meshSize,omitempty"`

	InitialCondition  string `json:"initialCondition,omitempty"`
	InitialFindings   string `json:"initialFindings,omitempty"`
	InitialAction     string `json:"initialAction,omitempty"`
	InitialPICName    string `json:"initialPicName,omitempty"`
	InitialQCName     string `json:"initialQcName,omitempty"`
	InitialQCPassed   *bool  `json:"initialQcPassed,omitempty"`

	FinalCondition    string `json:"finalCondition,omitempty"`
	FinalFindings     string `json:"finalFindings,omitempty"`
	FinalAction       string `json:"finalAction,omitempty"`
	FinalPICName      string `json:"finalPicName,omitempty"`
	FinalQCName       string `json:"finalQcName,omitempty"`
	FinalQCPassed     *bool  `json:"finalQcPassed,omitempty"`

	OverallStatus     string `json:"overallStatus,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

// ─── CMT (Magnet Trap Log) DTOs ─────────────────────────────────────────────

type CreateMagnetTrapLogDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate" validate:"required"`
	Room              string     `json:"room,omitempty"` // Ayak Mentah, Ayak Matang
	MachineNo         string     `json:"machineNo" validate:"required"`
	PICResponsible    string     `json:"picResponsible" validate:"required"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID `json:"productionStepId,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`

	IsClean            *bool   `json:"isClean,omitempty"`
	GaussPoint1        float64 `json:"gaussPoint1"`
	GaussPoint2        float64 `json:"gaussPoint2"`
	GaussPoint3        float64 `json:"gaussPoint3"`
	ForeignMatterFound string  `json:"foreignMatterFound,omitempty"`
	Issues             string  `json:"issues,omitempty"`
	ActionTaken        string  `json:"actionTaken,omitempty"`
	PICName            string  `json:"picName" validate:"required"`
	QCName             string  `json:"qcName,omitempty"`
	QCPassed           *bool   `json:"qcPassed,omitempty"`
	QAVerifiedBy       string  `json:"qaVerifiedBy,omitempty"`
	Notes              string  `json:"notes,omitempty"`
}

type UpdateMagnetTrapLogDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	Room              string     `json:"room,omitempty"`
	MachineNo         string     `json:"machineNo,omitempty"`
	PICResponsible    string     `json:"picResponsible,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID `json:"productionStepId,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`

	IsClean            *bool   `json:"isClean,omitempty"`
	GaussPoint1        *float64 `json:"gaussPoint1,omitempty"`
	GaussPoint2        *float64 `json:"gaussPoint2,omitempty"`
	GaussPoint3        *float64 `json:"gaussPoint3,omitempty"`
	ForeignMatterFound string   `json:"foreignMatterFound,omitempty"`
	Issues             string   `json:"issues,omitempty"`
	ActionTaken        string   `json:"actionTaken,omitempty"`
	PICName            string   `json:"picName,omitempty"`
	QCName             string   `json:"qcName,omitempty"`
	QCPassed           *bool    `json:"qcPassed,omitempty"`
	QAVerifiedBy       string   `json:"qaVerifiedBy,omitempty"`
	OverallStatus      string   `json:"overallStatus,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

// ─── MNM (Mesh Bolt Monitoring Log) DTOs ────────────────────────────────────

type CreateMeshBoltLogDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate" validate:"required"`
	Room              string     `json:"room,omitempty"`
	MachineNo         string     `json:"machineNo" validate:"required"`
	MeshSize          string     `json:"meshSize" validate:"required"`
	PICResponsible    string     `json:"picResponsible" validate:"required"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID `json:"productionStepId,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`

	InitialBoltCount  int    `json:"initialBoltCount"`
	InitialIsComplete *bool  `json:"initialIsComplete,omitempty"`
	InitialIsTight    *bool  `json:"initialIsTight,omitempty"`
	InitialCondition  string `json:"initialCondition,omitempty"`
	InitialFindings   string `json:"initialFindings,omitempty"`
	InitialAction     string `json:"initialAction,omitempty"`
	InitialPICName    string `json:"initialPicName" validate:"required"`
	InitialQCName     string `json:"initialQcName,omitempty"`

	FinalBoltCount    int    `json:"finalBoltCount"`
	FinalIsComplete   *bool  `json:"finalIsComplete,omitempty"`
	FinalIsTight      *bool  `json:"finalIsTight,omitempty"`
	FinalFindings     string `json:"finalFindings,omitempty"`
	FinalAction       string `json:"finalAction,omitempty"`
	FinalPICName      string `json:"finalPicName" validate:"required"`
	FinalQCName       string `json:"finalQcName,omitempty"`

	QAVerifiedBy      string `json:"qaVerifiedBy,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

type UpdateMeshBoltLogDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	Room              string     `json:"room,omitempty"`
	MachineNo         string     `json:"machineNo,omitempty"`
	MeshSize          string     `json:"meshSize,omitempty"`
	PICResponsible    string     `json:"picResponsible,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID `json:"productionStepId,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`

	InitialBoltCount  *int   `json:"initialBoltCount,omitempty"`
	InitialIsComplete *bool  `json:"initialIsComplete,omitempty"`
	InitialIsTight    *bool  `json:"initialIsTight,omitempty"`
	InitialCondition  string `json:"initialCondition,omitempty"`
	InitialFindings   string `json:"initialFindings,omitempty"`
	InitialAction     string `json:"initialAction,omitempty"`
	InitialPICName    string `json:"initialPicName,omitempty"`
	InitialQCName     string `json:"initialQcName,omitempty"`

	FinalBoltCount    *int   `json:"finalBoltCount,omitempty"`
	FinalIsComplete   *bool  `json:"finalIsComplete,omitempty"`
	FinalIsTight      *bool  `json:"finalIsTight,omitempty"`
	FinalFindings     string `json:"finalFindings,omitempty"`
	FinalAction       string `json:"finalAction,omitempty"`
	FinalPICName      string `json:"finalPicName,omitempty"`
	FinalQCName       string `json:"finalQcName,omitempty"`

	QAVerifiedBy      string `json:"qaVerifiedBy,omitempty"`
	OverallStatus     string `json:"overallStatus,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

// ─── MNO (Oven Monitoring Log) DTOs ─────────────────────────────────────────

type CreateOvenLogDTO struct {
	CheckMonth         string     `json:"checkMonth,omitempty"`
	CheckDate          string     `json:"checkDate" validate:"required"`
	OvenNo             string     `json:"ovenNo" validate:"required"`
	ProductionOrderID  *uuid.UUID `json:"productionOrderId,omitempty"`
	ProductionStepID   *uuid.UUID `json:"productionStepId,omitempty"`
	BatchNo            string     `json:"batchNo" validate:"required"`
	ProductName        string     `json:"productName" validate:"required"`

	InitialTemperature float64    `json:"initialTemperature"`
	InitialTime        string     `json:"initialTime" validate:"required"`
	InitialMoisture    float64    `json:"initialMoisture"`

	StabilizedTime     string     `json:"stabilizedTime" validate:"required"`

	ProcessTemperature float64    `json:"processTemperature"`
	ProcessMoisture1   *float64   `json:"processMoisture1,omitempty"`
	ProcessMoisture2   *float64   `json:"processMoisture2,omitempty"`
	ProcessMoisture3   *float64   `json:"processMoisture3,omitempty"`
	ProcessMoisture4   *float64   `json:"processMoisture4,omitempty"`

	FinalTemperature   float64    `json:"finalTemperature"`
	FinalTime          string     `json:"finalTime" validate:"required"`
	FinalMoisture      float64    `json:"finalMoisture"`

	Issues             string     `json:"issues,omitempty"`
	ActionTaken        string     `json:"actionTaken,omitempty"`
	PICName            string     `json:"picName" validate:"required"`
	QCVerifierName     string     `json:"qcVerifierName,omitempty"`
	QCPassed           *bool      `json:"qcPassed,omitempty"`
	Notes              string     `json:"notes,omitempty"`
}

type UpdateOvenLogDTO struct {
	CheckMonth         string     `json:"checkMonth,omitempty"`
	CheckDate          string     `json:"checkDate,omitempty"`
	OvenNo             string     `json:"ovenNo,omitempty"`
	ProductionOrderID  *uuid.UUID `json:"productionOrderId,omitempty"`
	ProductionStepID   *uuid.UUID `json:"productionStepId,omitempty"`
	BatchNo            string     `json:"batchNo,omitempty"`
	ProductName        string     `json:"productName,omitempty"`

	InitialTemperature *float64   `json:"initialTemperature,omitempty"`
	InitialTime        string     `json:"initialTime,omitempty"`
	InitialMoisture    *float64   `json:"initialMoisture,omitempty"`

	StabilizedTime     string     `json:"stabilizedTime,omitempty"`

	ProcessTemperature *float64   `json:"processTemperature,omitempty"`
	ProcessMoisture1   *float64   `json:"processMoisture1,omitempty"`
	ProcessMoisture2   *float64   `json:"processMoisture2,omitempty"`
	ProcessMoisture3   *float64   `json:"processMoisture3,omitempty"`
	ProcessMoisture4   *float64   `json:"processMoisture4,omitempty"`

	FinalTemperature   *float64   `json:"finalTemperature,omitempty"`
	FinalTime          string     `json:"finalTime,omitempty"`
	FinalMoisture      *float64   `json:"finalMoisture,omitempty"`

	Issues             string     `json:"issues,omitempty"`
	ActionTaken        string     `json:"actionTaken,omitempty"`
	PICName            string     `json:"picName,omitempty"`
	QCVerifierName     string     `json:"qcVerifierName,omitempty"`
	QCPassed           *bool      `json:"qcPassed,omitempty"`
	OverallStatus      string     `json:"overallStatus,omitempty"`
	Notes              string     `json:"notes,omitempty"`
}

// ─── MNJ (Needle Monitoring Log) DTOs ───────────────────────────────────────

type CreateNeedleLogDTO struct {
	CheckMonth        string `json:"checkMonth,omitempty"`
	CheckDate         string `json:"checkDate" validate:"required"`
	SewingMachineNo   string `json:"sewingMachineNo" validate:"required"`
	PICResponsible    string `json:"picResponsible" validate:"required"`

	InitialIsPresent  *bool  `json:"initialIsPresent,omitempty"`
	InitialCondition  string `json:"initialCondition,omitempty"`
	InitialAnalysis   string `json:"initialAnalysis,omitempty"`
	InitialAction     string `json:"initialAction,omitempty"`

	FinalIsPresent    *bool  `json:"finalIsPresent,omitempty"`
	FinalCondition    string `json:"finalCondition,omitempty"`
	FinalAnalysis     string `json:"finalAnalysis,omitempty"`
	FinalAction       string `json:"finalAction,omitempty"`

	NeedleStockInRoom int    `json:"needleStockInRoom"`
	PICName           string `json:"picName" validate:"required"`
	QCVerifierName    string `json:"qcVerifierName,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

type UpdateNeedleLogDTO struct {
	CheckMonth        string `json:"checkMonth,omitempty"`
	CheckDate         string `json:"checkDate,omitempty"`
	SewingMachineNo   string `json:"sewingMachineNo,omitempty"`
	PICResponsible    string `json:"picResponsible,omitempty"`

	InitialIsPresent  *bool  `json:"initialIsPresent,omitempty"`
	InitialCondition  string `json:"initialCondition,omitempty"`
	InitialAnalysis   string `json:"initialAnalysis,omitempty"`
	InitialAction     string `json:"initialAction,omitempty"`

	FinalIsPresent    *bool  `json:"finalIsPresent,omitempty"`
	FinalCondition    string `json:"finalCondition,omitempty"`
	FinalAnalysis     string `json:"finalAnalysis,omitempty"`
	FinalAction       string `json:"finalAction,omitempty"`

	NeedleStockInRoom *int   `json:"needleStockInRoom,omitempty"`
	PICName           string `json:"picName,omitempty"`
	QCVerifierName    string `json:"qcVerifierName,omitempty"`
	OverallStatus     string `json:"overallStatus,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

// ─── MNG (Scissor Monitoring Log) DTOs ──────────────────────────────────────

type CreateScissorLogDTO struct {
	CheckMonth       string `json:"checkMonth,omitempty"`
	CheckDate        string `json:"checkDate" validate:"required"`
	Location         string `json:"location,omitempty"`
	PICResponsible   string `json:"picResponsible" validate:"required"`

	InitialCount     int    `json:"initialCount"`
	InitialCondition string `json:"initialCondition,omitempty"`
	InitialAnalysis  string `json:"initialAnalysis,omitempty"`
	InitialAction    string `json:"initialAction,omitempty"`

	FinalCount       int    `json:"finalCount"`
	FinalCondition   string `json:"finalCondition,omitempty"`
	FinalAnalysis    string `json:"finalAnalysis,omitempty"`
	FinalAction      string `json:"finalAction,omitempty"`

	PICName          string `json:"picName" validate:"required"`
	QCVerifierName   string `json:"qcVerifierName,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

type UpdateScissorLogDTO struct {
	CheckMonth       string `json:"checkMonth,omitempty"`
	CheckDate        string `json:"checkDate,omitempty"`
	Location         string `json:"location,omitempty"`
	PICResponsible   string `json:"picResponsible,omitempty"`

	InitialCount     *int   `json:"initialCount,omitempty"`
	InitialCondition string `json:"initialCondition,omitempty"`
	InitialAnalysis  string `json:"initialAnalysis,omitempty"`
	InitialAction    string `json:"initialAction,omitempty"`

	FinalCount       *int   `json:"finalCount,omitempty"`
	FinalCondition   string `json:"finalCondition,omitempty"`
	FinalAnalysis    string `json:"finalAnalysis,omitempty"`
	FinalAction      string `json:"finalAction,omitempty"`

	PICName          string `json:"picName,omitempty"`
	QCVerifierName   string `json:"qcVerifierName,omitempty"`
	OverallStatus    string `json:"overallStatus,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

// ─── CPP (Metal Detector Cleaning Log) DTOs ─────────────────────────────────

type CreateMetalCleaningLogDTO struct {
	CheckMonth           string  `json:"checkMonth,omitempty"`
	CheckDate            string  `json:"checkDate" validate:"required"`
	MachineNo            string  `json:"machineNo,omitempty"`
	Location             string  `json:"location,omitempty"`
	CleaningType         string  `json:"cleaningType,omitempty"`

	SensorCleanliness    string  `json:"sensorCleanliness,omitempty"`
	BeltCleanliness      string  `json:"beltCleanliness,omitempty"`
	RejectBinCleanliness string  `json:"rejectBinCleanliness,omitempty"`
	PhysicalCondition    string  `json:"physicalCondition,omitempty"`

	StartTime            string  `json:"startTime" validate:"required"`
	EndTime              string  `json:"endTime" validate:"required"`
	TotalHours           float64 `json:"totalHours,omitempty"`

	OperatorName         string  `json:"operatorName" validate:"required"`
	QCInspectorName      string  `json:"qcInspectorName,omitempty"`
	Notes                string  `json:"notes,omitempty"`
}

type UpdateMetalCleaningLogDTO struct {
	CheckMonth           string   `json:"checkMonth,omitempty"`
	CheckDate            string   `json:"checkDate,omitempty"`
	MachineNo            string   `json:"machineNo,omitempty"`
	Location             string   `json:"location,omitempty"`
	CleaningType         string   `json:"cleaningType,omitempty"`

	SensorCleanliness    string   `json:"sensorCleanliness,omitempty"`
	BeltCleanliness      string   `json:"beltCleanliness,omitempty"`
	RejectBinCleanliness string   `json:"rejectBinCleanliness,omitempty"`
	PhysicalCondition    string   `json:"physicalCondition,omitempty"`

	StartTime            string   `json:"startTime,omitempty"`
	EndTime              string   `json:"endTime,omitempty"`
	TotalHours           *float64 `json:"totalHours,omitempty"`

	OperatorName         string   `json:"operatorName,omitempty"`
	QCInspectorName      string   `json:"qcInspectorName,omitempty"`
	OverallStatus        string   `json:"overallStatus,omitempty"`
	Notes                string   `json:"notes,omitempty"`
}

// ─── VMD (Metal Detector Verification Log) DTOs ─────────────────────────────

type CreateMetalVerificationLogDTO struct {
	CheckMonth           string `json:"checkMonth,omitempty"`
	CheckDate            string `json:"checkDate" validate:"required"`
	ProductType          string `json:"productType,omitempty"`
	Location             string `json:"location,omitempty"`
	PackageCategory      string `json:"packageCategory,omitempty"`
	MonitoringRound      int    `json:"monitoringRound"`
	CheckTime            string `json:"checkTime" validate:"required"`

	FeTestPieceSize      string `json:"feTestPieceSize,omitempty"`
	FeAlarmTriggered     *bool  `json:"feAlarmTriggered,omitempty"`
	FeConveyorStopped    *bool  `json:"feConveyorStopped,omitempty"`

	NonFeTestPieceSize   string `json:"nonFeTestPieceSize,omitempty"`
	NonFeAlarmTriggered  *bool  `json:"nonFeAlarmTriggered,omitempty"`
	NonFeConveyorStopped *bool  `json:"nonFeConveyorStopped,omitempty"`

	SSTestPieceSize      string `json:"ssTestPieceSize,omitempty"`
	SSAlarmTriggered     *bool  `json:"ssAlarmTriggered,omitempty"`
	SSConveyorStopped    *bool  `json:"ssConveyorStopped,omitempty"`

	CorrectiveAction     string `json:"correctiveAction,omitempty"`
	PICName              string `json:"picName" validate:"required"`
	QCVerifierName       string `json:"qcVerifierName,omitempty"`
	Notes                string `json:"notes,omitempty"`
}

type UpdateMetalVerificationLogDTO struct {
	CheckMonth           string `json:"checkMonth,omitempty"`
	CheckDate            string `json:"checkDate,omitempty"`
	ProductType          string `json:"productType,omitempty"`
	Location             string `json:"location,omitempty"`
	PackageCategory      string `json:"packageCategory,omitempty"`
	MonitoringRound      *int   `json:"monitoringRound,omitempty"`
	CheckTime            string `json:"checkTime,omitempty"`

	FeTestPieceSize      string `json:"feTestPieceSize,omitempty"`
	FeAlarmTriggered     *bool  `json:"feAlarmTriggered,omitempty"`
	FeConveyorStopped    *bool  `json:"feConveyorStopped,omitempty"`

	NonFeTestPieceSize   string `json:"nonFeTestPieceSize,omitempty"`
	NonFeAlarmTriggered  *bool  `json:"nonFeAlarmTriggered,omitempty"`
	NonFeConveyorStopped *bool  `json:"nonFeConveyorStopped,omitempty"`

	SSTestPieceSize      string `json:"ssTestPieceSize,omitempty"`
	SSAlarmTriggered     *bool  `json:"ssAlarmTriggered,omitempty"`
	SSConveyorStopped    *bool  `json:"ssConveyorStopped,omitempty"`

	CorrectiveAction     string `json:"correctiveAction,omitempty"`
	PICName              string `json:"picName,omitempty"`
	QCVerifierName       string `json:"qcVerifierName,omitempty"`
	OverallStatus        string `json:"overallStatus,omitempty"`
	Notes                string `json:"notes,omitempty"`
}

// ─── PJD (Finished Goods Inspection Log) DTOs ───────────────────────────────

type CreateFinishedGoodsDTO struct {
	CheckMonth          string     `json:"checkMonth,omitempty"`
	CheckDate           string     `json:"checkDate" validate:"required"`
	ProductName         string     `json:"productName" validate:"required"`
	BatchNo             string     `json:"batchNo" validate:"required"`
	RawMaterialCode     string     `json:"rawMaterialCode,omitempty"`
	ProductionOrderID   *uuid.UUID `json:"productionOrderId,omitempty"`

	PrimaryPackaging    string     `json:"primaryPackaging,omitempty"`
	SecondaryPackaging  string     `json:"secondaryPackaging,omitempty"`
	NetWeight           float64    `json:"netWeight"`
	GrossWeight         float64    `json:"grossWeight"`

	AppearanceShape     string     `json:"appearanceShape,omitempty"`
	Aroma               string     `json:"aroma,omitempty"`
	ColorScore          int        `json:"colorScore"`
	ColorDescription    string     `json:"colorDescription,omitempty"`
	Taste               string     `json:"taste,omitempty"`

	IsFreeForeignMatter *bool      `json:"isFreeForeignMatter,omitempty"`
	ForeignMatterNotes  string     `json:"foreignMatterNotes,omitempty"`
	MoistureContent     float64    `json:"moistureContent"`

	InspectionDecision  string     `json:"inspectionDecision,omitempty"`
	Notes               string     `json:"notes,omitempty"`
	QCPersonnelName     string     `json:"qcPersonnelName" validate:"required"`
	QAVerifiedBy        string     `json:"qaVerifiedBy,omitempty"`
}

type UpdateFinishedGoodsDTO struct {
	CheckMonth          string     `json:"checkMonth,omitempty"`
	CheckDate           string     `json:"checkDate,omitempty"`
	ProductName         string     `json:"productName,omitempty"`
	BatchNo             string     `json:"batchNo,omitempty"`
	RawMaterialCode     string     `json:"rawMaterialCode,omitempty"`
	ProductionOrderID   *uuid.UUID `json:"productionOrderId,omitempty"`

	PrimaryPackaging    string     `json:"primaryPackaging,omitempty"`
	SecondaryPackaging  string     `json:"secondaryPackaging,omitempty"`
	NetWeight           *float64   `json:"netWeight,omitempty"`
	GrossWeight         *float64   `json:"grossWeight,omitempty"`

	AppearanceShape     string     `json:"appearanceShape,omitempty"`
	Aroma               string     `json:"aroma,omitempty"`
	ColorScore          *int       `json:"colorScore,omitempty"`
	ColorDescription    string     `json:"colorDescription,omitempty"`
	Taste               string     `json:"taste,omitempty"`

	IsFreeForeignMatter *bool      `json:"isFreeForeignMatter,omitempty"`
	ForeignMatterNotes  string     `json:"foreignMatterNotes,omitempty"`
	MoistureContent     *float64   `json:"moistureContent,omitempty"`

	InspectionDecision  string     `json:"inspectionDecision,omitempty"`
	Notes               string     `json:"notes,omitempty"`
	QCPersonnelName     string     `json:"qcPersonnelName,omitempty"`
	QAVerifiedBy        string     `json:"qaVerifiedBy,omitempty"`
}

