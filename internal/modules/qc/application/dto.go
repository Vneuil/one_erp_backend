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
