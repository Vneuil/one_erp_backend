package domain

import (
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type ParameterType string

const (
	ParamTypeNumeric ParameterType = "numeric"
	ParamTypeBoolean ParameterType = "boolean"
)

type InspectionStatus string

const (
	QCStatusPending InspectionStatus = "pending"
	QCStatusPassed  InspectionStatus = "passed"
	QCStatusFailed  InspectionStatus = "failed"
	QCStatusHold    InspectionStatus = "hold"
	QCStatusVoided  InspectionStatus = "voided"
)

type TargetType string

const (
	TargetGoodsReceipt    TargetType = "goods_receipt"
	TargetProductionOrder TargetType = "production_order"
	TargetProductionStep  TargetType = "production_step"
)

// QCParameter defines quality acceptance criteria per product or stage
type QCParameter struct {
	types.BaseEntity
	ProductID     *uuid.UUID    `gorm:"type:uuid;index" json:"productId,omitempty"`
	ProcessCode   string        `gorm:"type:varchar(50);not null;index" json:"processCode"`
	Name          string        `gorm:"type:varchar(100);not null" json:"name"`
	ParamType     ParameterType `gorm:"type:varchar(20);not null;default:'numeric'" json:"paramType"`
	UnitOfMeasure string        `gorm:"type:varchar(20)" json:"unitOfMeasure"`
	MinValue      *float64      `gorm:"type:decimal(10,3)" json:"minValue,omitempty"`
	MaxValue      *float64      `gorm:"type:decimal(10,3)" json:"maxValue,omitempty"`
	TargetValue   *float64      `gorm:"type:decimal(10,3)" json:"targetValue,omitempty"`
	Tolerance     *float64      `gorm:"type:decimal(10,3)" json:"tolerance,omitempty"`
	IsGate        bool          `gorm:"default:false;not null" json:"isGate"`
	IsMandatory   bool          `gorm:"default:true;not null" json:"isMandatory"`
	Description   string        `gorm:"type:text" json:"description,omitempty"`
}

func (QCParameter) TableName() string {
	return "qc_parameters"
}

// QCInspection represents a quality inspection record (header)
type QCInspection struct {
	types.BaseEntity
	InspectionNo      string           `gorm:"type:varchar(50);uniqueIndex;not null" json:"inspectionNo"`
	TargetType        TargetType       `gorm:"type:varchar(30);not null;index" json:"targetType"`
	TargetID          uuid.UUID        `gorm:"type:uuid;not null;index" json:"targetId"`
	ProductionOrderID *uuid.UUID       `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID       `gorm:"type:uuid;index" json:"productionStepId,omitempty"`
	BatchNo           string           `gorm:"type:varchar(50);not null;index" json:"batchNo"`
	ProductID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"productId"`
	InspectorID       uuid.UUID        `gorm:"type:uuid;not null" json:"inspectorId"`
	InspectorName     string           `gorm:"type:varchar(150)" json:"inspectorName"`
	InspectedAt       time.Time        `gorm:"not null" json:"inspectedAt"`
	TestingTool       string           `gorm:"type:varchar(100)" json:"testingTool,omitempty"`
	Iteration         int              `gorm:"default:1;not null" json:"iteration"`
	OverallStatus     InspectionStatus `gorm:"type:varchar(20);not null;index;default:'pending'" json:"overallStatus"`
	Notes             string           `gorm:"type:text" json:"notes,omitempty"`
	// Physical condition recorded at incoming QC (moved here from the GRN form):
	// the sugar itself and its sacks/packaging.
	SugarCondition   string `gorm:"type:varchar(100)" json:"sugarCondition,omitempty"`
	PackageCondition string `gorm:"type:varchar(100)" json:"packageCondition,omitempty"`

	// Immutability & Void audit trail (BR-25, BRL-07)
	IsVoided   bool       `gorm:"default:false;not null" json:"isVoided"`
	VoidReason string     `gorm:"type:text" json:"voidReason,omitempty"`
	VoidedBy   *uuid.UUID `gorm:"type:uuid" json:"voidedBy,omitempty"`
	VoidedAt   *time.Time `json:"voidedAt,omitempty"`

	Items []QCInspectionItem `gorm:"foreignKey:InspectionID" json:"items,omitempty"`
}

func (QCInspection) TableName() string {
	return "qc_inspections"
}

// QCInspectionItem represents single parameter evaluation in an inspection
type QCInspectionItem struct {
	types.BaseEntity
	InspectionID  uuid.UUID     `gorm:"type:uuid;not null;index" json:"inspectionId"`
	ParameterID   uuid.UUID     `gorm:"type:uuid;not null;index" json:"parameterId"`
	ParameterName string        `gorm:"type:varchar(100);not null" json:"parameterName"`
	ParamType     ParameterType `gorm:"type:varchar(20);not null" json:"paramType"`
	NumericValue  *float64      `gorm:"type:decimal(10,3)" json:"numericValue,omitempty"`
	BooleanValue  *bool         `json:"booleanValue,omitempty"`
	IsPassed      bool          `gorm:"not null" json:"isPassed"`
	Notes         string        `gorm:"type:varchar(255)" json:"notes,omitempty"`
}

func (QCInspectionItem) TableName() string {
	return "qc_inspection_items"
}

// PersonalHygieneCheck records employee GMP hygiene checklist (Doc CPH / Catatan Personal Hygiene)
type PersonalHygieneCheck struct {
	types.BaseEntity
	CheckDate      string `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	Shift          string `gorm:"type:varchar(20);not null" json:"shift"`
	Line           string `gorm:"type:varchar(100);not null" json:"line"`
	EmployeeName   string `gorm:"type:varchar(150);not null;index" json:"employeeName"`
	WorkWearClean  bool   `gorm:"not null;default:true" json:"workWearClean"`
	HairNet        bool   `gorm:"not null;default:true" json:"hairNet"`
	NailsClean     bool   `gorm:"not null;default:true" json:"nailsClean"`
	NoJewelry      bool   `gorm:"not null;default:true" json:"noJewelry"`
	WorkShoesClean bool   `gorm:"not null;default:true" json:"workShoesClean"`
	HandWashing    bool   `gorm:"not null;default:true" json:"handWashing"`
	HealthFit      bool   `gorm:"not null;default:true" json:"healthFit"`
	IsPassed       bool   `gorm:"not null;default:true" json:"isPassed"`
	ActionTaken    string `gorm:"type:varchar(255)" json:"actionTaken,omitempty"`
	InspectorName  string `gorm:"type:varchar(150);not null" json:"inspectorName"`
	Notes          string `gorm:"type:text" json:"notes,omitempty"`
}

func (PersonalHygieneCheck) TableName() string {
	return "qc_personal_hygiene_checks"
}

// PreOpSanitationCheck records pre-operational room & equipment checklist (Doc SSO / Sarana Produksi Sebelum Operasional)
type PreOpSanitationCheck struct {
	types.BaseEntity
	CheckDate          string   `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	CheckTime          string   `gorm:"type:varchar(10);not null" json:"checkTime"`       // HH:mm
	Shift              string   `gorm:"type:varchar(20);not null" json:"shift"`
	Room               string   `gorm:"type:varchar(100);not null;index" json:"room"`
	FloorClean         bool     `gorm:"not null;default:true" json:"floorClean"`
	WallCeilingClean   bool     `gorm:"not null;default:true" json:"wallCeilingClean"`
	EquipmentSanitized bool     `gorm:"not null;default:true" json:"equipmentSanitized"`
	TrashBinClosed     bool     `gorm:"not null;default:true" json:"trashBinClosed"`
	LightingProtected  bool     `gorm:"not null;default:true" json:"lightingProtected"`
	PestControlIntact  bool     `gorm:"not null;default:true" json:"pestControlIntact"`
	GaussReading       *float64 `gorm:"type:decimal(10,2)" json:"gaussReading,omitempty"`
	MetalTrapClean     *bool    `json:"metalTrapClean,omitempty"`
	Status             string   `gorm:"type:varchar(30);not null;default:'approved'" json:"status"` // approved, hold, action_required
	Notes              string   `gorm:"type:text" json:"notes,omitempty"`
	InspectorName      string   `gorm:"type:varchar(150);not null" json:"inspectorName"`
	SupervisorName     string   `gorm:"type:varchar(150)" json:"supervisorName,omitempty"`
}

func (PreOpSanitationCheck) TableName() string {
	return "qc_preop_sanitation_checks"
}

// PackagingQualityCheck records incoming packaging material inspection (QC-003-PBK)
type PackagingQualityCheck struct {
	types.BaseEntity
	CheckMonth      string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	GoodsReceiptID  *uuid.UUID `gorm:"type:uuid;index" json:"goodsReceiptId,omitempty"`
	LBMNo           string     `gorm:"type:varchar(50);index" json:"lbmNo,omitempty"`
	CheckDate       string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	SupplierID      *uuid.UUID `gorm:"type:uuid;index" json:"supplierId,omitempty"`
	SupplierName    string     `gorm:"type:varchar(255);not null" json:"supplierName"`
	ItemName        string     `gorm:"type:varchar(255);not null" json:"itemName"`
	ItemCode        string     `gorm:"type:varchar(100)" json:"itemCode,omitempty"`
	MaterialType    string     `gorm:"type:varchar(50);not null" json:"materialType"` // plastik, kardus, jerigen, pallet_plastik, pallet_kayu
	BaseColor       string     `gorm:"type:varchar(100)" json:"baseColor,omitempty"`
	Dimension       string     `gorm:"type:varchar(100)" json:"dimension,omitempty"`
	TotalQty        int        `gorm:"not null" json:"totalQty"`
	SampleQty       int        `gorm:"not null" json:"sampleQty"`
	PassedQty       int        `gorm:"not null;default:0" json:"passedQty"`
	PassedCondition string     `gorm:"type:text" json:"passedCondition,omitempty"`
	FailedQty       int        `gorm:"not null;default:0" json:"failedQty"`
	FailedCondition string     `gorm:"type:text" json:"failedCondition,omitempty"`
	OverallStatus   string     `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, failed, partial
	QCName          string     `gorm:"type:varchar(150)" json:"qcName,omitempty"`
	QCDate          *time.Time `json:"qcDate,omitempty"`
	QAName          string     `gorm:"type:varchar(150)" json:"qaName,omitempty"`
	QADate          *time.Time `json:"qaDate,omitempty"`
	Notes           string     `gorm:"type:text" json:"notes,omitempty"`
}

func (PackagingQualityCheck) TableName() string {
	return "qc_packaging_quality_checks"
}

// SupplierArmadaCheck records supplier vehicle & driver checklist (QC-007-CAS)
type SupplierArmadaCheck struct {
	types.BaseEntity
	GoodsReceiptID    *uuid.UUID `gorm:"type:uuid;index" json:"goodsReceiptId,omitempty"`
	LBMNo             string     `gorm:"type:varchar(50);index" json:"lbmNo,omitempty"`
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	IDNumber          string     `gorm:"type:varchar(50)" json:"idNumber,omitempty"`
	DriverName        string     `gorm:"type:varchar(150);not null" json:"driverName"`
	BirthPlace        string     `gorm:"type:varchar(100)" json:"birthPlace,omitempty"`
	BirthDate         string     `gorm:"type:varchar(20)" json:"birthDate,omitempty"`
	Gender            string     `gorm:"type:varchar(20)" json:"gender,omitempty"` // Laki-laki, Perempuan
	Address           string     `gorm:"type:text" json:"address,omitempty"`
	Religion          string     `gorm:"type:varchar(50)" json:"religion,omitempty"`
	MaritalStatus     string     `gorm:"type:varchar(30)" json:"maritalStatus,omitempty"` // Menikah, Belum Menikah
	IDExpiry          string     `gorm:"type:varchar(20)" json:"idExpiry,omitempty"`      // YYYY-MM-DD
	CompanyOrigin     string     `gorm:"type:varchar(255)" json:"companyOrigin,omitempty"`
	CargoDesc         string     `gorm:"type:varchar(255)" json:"cargoDesc,omitempty"`
	LicensePlate      string     `gorm:"type:varchar(30)" json:"licensePlate,omitempty"`
	VehicleType       string     `gorm:"type:varchar(50)" json:"vehicleType,omitempty"` // Truk, Pickup, Motor, Lainnya
	CabinSafe         bool       `gorm:"not null;default:true" json:"cabinSafe"`
	UndercarriageSafe bool       `gorm:"not null;default:true" json:"undercarriageSafe"`
	CargoAreaSafe     bool       `gorm:"not null;default:true" json:"cargoAreaSafe"`
	FloorClean        bool       `gorm:"not null;default:true" json:"floorClean"`
	WallRightClean    bool       `gorm:"not null;default:true" json:"wallRightClean"`
	WallLeftClean     bool       `gorm:"not null;default:true" json:"wallLeftClean"`
	CeilingClean      bool       `gorm:"not null;default:true" json:"ceilingClean"`
	DoorClean         bool       `gorm:"not null;default:true" json:"doorClean"`
	OverallStatus     string     `gorm:"type:varchar(20);not null;default:'cleared'" json:"overallStatus"` // cleared, flagged, rejected
	QCName            string     `gorm:"type:varchar(150)" json:"qcName,omitempty"`
	QCDate            *time.Time `json:"qcDate,omitempty"`
	QAName            string     `gorm:"type:varchar(150)" json:"qaName,omitempty"`
	QADate            *time.Time `json:"qaDate,omitempty"`
	Notes             string     `gorm:"type:text" json:"notes,omitempty"`
}

func (SupplierArmadaCheck) TableName() string {
	return "qc_supplier_armada_checks"
}

// RawMaterialIncomingCheck records incoming raw material quality inspection (QC-001-PBD / Pemeriksaan Bahan Datang)
type RawMaterialIncomingCheck struct {
	types.BaseEntity
	GoodsReceiptID     *uuid.UUID `gorm:"type:uuid;index" json:"goodsReceiptId,omitempty"`
	LBMNo              string     `gorm:"type:varchar(50);index" json:"lbmNo,omitempty"`
	CheckDate          string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	CheckMonth         string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	SupplierID         *uuid.UUID `gorm:"type:uuid;index" json:"supplierId,omitempty"`
	SupplierName       string     `gorm:"type:varchar(255);not null" json:"supplierName"`
	DeliveryNoteNo     string     `gorm:"type:varchar(100)" json:"deliveryNoteNo,omitempty"`
	PONo               string     `gorm:"type:varchar(50)" json:"poNo,omitempty"`
	ProductName        string     `gorm:"type:varchar(255);not null" json:"productName"`
	BatchNo            string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	// Kuantitas kedatangan
	TotalBags          int        `gorm:"not null;default:0" json:"totalBags"`
	SampleBags         int        `gorm:"not null;default:0" json:"sampleBags"`
	GrossWeight        float64    `gorm:"type:decimal(12,2);not null;default:0" json:"grossWeight"`
	NetWeight          float64    `gorm:"type:decimal(12,2);not null;default:0" json:"netWeight"`
	AcceptedWeight     float64    `gorm:"type:decimal(12,2);not null;default:0" json:"acceptedWeight"`
	RejectedWeight     float64    `gorm:"type:decimal(12,2);not null;default:0" json:"rejectedWeight"`
	// Pemeriksaan Fisik & Organoleptik
	SugarCondition     string     `gorm:"type:varchar(100);not null" json:"sugarCondition"`   // Kering, Lembab, Basah, Menggumpal
	ColorAppearance    string     `gorm:"type:varchar(100)" json:"colorAppearance,omitempty"` // Coklat Cerah, Coklat Gelap, Keemasan
	AromaFlavor        string     `gorm:"type:varchar(100)" json:"aromaFlavor,omitempty"`     // Khas Gula Kelapa, Normal, Asam, Sangit
	ForeignMatterClean bool       `gorm:"not null;default:true" json:"foreignMatterClean"`   // true = bebas kotoran/serangga/batu
	PackagingCondition string     `gorm:"type:varchar(100)" json:"packagingCondition,omitempty"` // Utuh, Sobek, Basah, Kotor
	// Parameter Uji Laboratorium / Alat
	MoistureContent    *float64   `gorm:"type:decimal(6,2)" json:"moistureContent,omitempty"` // Kadar air % (Target <= 2.0%)
	BrixValue          *float64   `gorm:"type:decimal(6,2)" json:"brixValue,omitempty"`       // Derajat Brix °Bx (untuk nira/cair)
	MeshSize           string     `gorm:"type:varchar(50)" json:"meshSize,omitempty"`         // Granulometri: Mesh 16, Mesh 18, Halus, Kasar
	// Rekomendasi Alur Produksi
	RecommendedAction  string     `gorm:"type:varchar(150)" json:"recommendedAction,omitempty"` // e.g. Bypass Oven (<=2%), Wajib Oven (>2%), Reject
	OverallStatus      string     `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, conditional, rejected
	// Verifikasi & Petugas
	QCName             string     `gorm:"type:varchar(150)" json:"qcName,omitempty"`
	QCDate             *time.Time `json:"qcDate,omitempty"`
	QAName             string     `gorm:"type:varchar(150)" json:"qaName,omitempty"`
	QADate             *time.Time `json:"qaDate,omitempty"`
	Notes              string     `gorm:"type:text" json:"notes,omitempty"`
}

func (RawMaterialIncomingCheck) TableName() string {
	return "qc_raw_material_incoming_checks"
}
