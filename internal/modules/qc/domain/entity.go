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
	DocumentNo     string `gorm:"type:varchar(50);index" json:"documentNo"`
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
	DocumentNo         string   `gorm:"type:varchar(50);index" json:"documentNo"`
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
	DocumentNo      string     `gorm:"type:varchar(50);index" json:"documentNo"`
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
	DocumentNo        string     `gorm:"type:varchar(50);index" json:"documentNo"`
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

// RawMaterialIncomingCheck records incoming raw material / goods quality inspection (Form QC-002-PBD Rev. 05 / Pemeriksaan Kualitas Barang Datang)
type RawMaterialIncomingCheck struct {
	types.BaseEntity
	DocumentNo         string     `gorm:"type:varchar(50);index" json:"documentNo"`
	DocumentCode       string     `gorm:"type:varchar(30);default:'QC-002-PBD'" json:"documentCode"`
	RevisionCode       string     `gorm:"type:varchar(20);default:'05'" json:"revisionCode"`
	FairtradeStatus    string     `gorm:"type:varchar(30);default:'NON FAIRTRADE'" json:"fairtradeStatus"` // FAIRTRADE, NON FAIRTRADE
	Certifications     string     `gorm:"type:varchar(150);default:'NATURAL'" json:"certifications"`       // NATURAL, ORGANIC EU, ORGANIC USDA, ORGANIC JAS
	GoodsReceiptID     *uuid.UUID `gorm:"type:uuid;index" json:"goodsReceiptId,omitempty"`
	LBMNo              string     `gorm:"type:varchar(50);index" json:"lbmNo,omitempty"`
	CheckDate          string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	CheckMonth         string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	SupplierID         *uuid.UUID `gorm:"type:uuid;index" json:"supplierId,omitempty"`
	SupplierName       string     `gorm:"type:varchar(255);not null" json:"supplierName"`
	DeliveryNoteNo     string     `gorm:"type:varchar(100)" json:"deliveryNoteNo,omitempty"`
	PONo               string     `gorm:"type:varchar(50)" json:"poNo,omitempty"`
	ProductName        string     `gorm:"type:varchar(255);not null" json:"productName"`
	ItemCode           string     `gorm:"type:varchar(100);index" json:"itemCode,omitempty"`               // Kode Barang
	ItemType           string     `gorm:"type:varchar(100);default:'Bahan Baku'" json:"itemType,omitempty"` // Jenis Barang
	BatchNo            string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	// Kuantitas & Sampling (Military Standard AQL 0,1 & 2,5)
	TotalBags          int        `gorm:"not null;default:0" json:"totalBags"`
	SampleBags         int        `gorm:"not null;default:0" json:"sampleBags"` // Jumlah Sampling
	GrossWeight        float64    `gorm:"type:decimal(12,2);not null;default:0" json:"grossWeight"`
	NetWeight          float64    `gorm:"type:decimal(12,2);not null;default:0" json:"netWeight"`
	AcceptedWeight     float64    `gorm:"type:decimal(12,2);not null;default:0" json:"acceptedWeight"`
	RejectedWeight     float64    `gorm:"type:decimal(12,2);not null;default:0" json:"rejectedWeight"`
	// Pengecekan 10 Parameter Resmi (QC-002-PBD)
	CheckPackaging     bool       `gorm:"not null;default:true" json:"checkPackaging"`    // Kemasan
	CheckShape         bool       `gorm:"not null;default:true" json:"checkShape"`        // Bentuk
	CheckTaste         bool       `gorm:"not null;default:true" json:"checkTaste"`        // Rasa
	CheckAroma         bool       `gorm:"not null;default:true" json:"checkAroma"`        // Aroma
	ColorScore         int        `gorm:"not null;default:2" json:"colorScore"`           // Warna* (1: light brown, 2: normal, 3: dark brown)
	MoistureContent    *float64   `gorm:"type:decimal(6,2)" json:"moistureContent,omitempty"` // Kadar air (%)
	BrixValue          *float64   `gorm:"type:decimal(6,2)" json:"brixValue,omitempty"`       // Brix
	CheckSolubility    bool       `gorm:"not null;default:true" json:"checkSolubility"`   // Kelarutan
	PhValue            *float64   `gorm:"type:decimal(5,2)" json:"phValue,omitempty"`         // pH
	CheckPackageLabel  bool       `gorm:"not null;default:true" json:"checkPackageLabel"` // Label Kemasan
	// Tingkat Lolos & Keputusan (LOLOS / DITOLAK)
	PassRatePercent    float64    `gorm:"type:decimal(6,2);not null;default:100" json:"passRatePercent"` // Tingkat lolos (%)
	InspectionDecision string     `gorm:"type:varchar(20);not null;default:'LOLOS'" json:"inspectionDecision"` // LOLOS, DITOLAK
	// Pemeriksaan Fisik & Organoleptik (Kompatibilitas SBB & Brontol)
	SugarCondition     string     `gorm:"type:varchar(100);not null" json:"sugarCondition"`
	ColorAppearance    string     `gorm:"type:varchar(100)" json:"colorAppearance,omitempty"`
	AromaFlavor        string     `gorm:"type:varchar(100)" json:"aromaFlavor,omitempty"`
	ForeignMatterClean bool       `gorm:"not null;default:true" json:"foreignMatterClean"`
	PackagingCondition string     `gorm:"type:varchar(100)" json:"packagingCondition,omitempty"`
	MeshSize           string     `gorm:"type:varchar(50)" json:"meshSize,omitempty"`
	// Rekomendasi & Status Gate SBB
	RecommendedAction  string     `gorm:"type:varchar(150)" json:"recommendedAction,omitempty"`
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

// ─────────────────────────────────────────────────────────────────────────────
// Monitoring Ayak (MNA, CMT, MNM) & Oven (MNO)
// ─────────────────────────────────────────────────────────────────────────────

// SievingMonitoringLog records sieve mesh inspection at start/end of batch (Form QC-009-MNA.01)
type SievingMonitoringLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'00'" json:"revisionCode"`
	MachineNo         string     `gorm:"type:varchar(50);not null;index" json:"machineNo"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	Category          string     `gorm:"type:varchar(20);not null;default:'Mentah'" json:"category"` // Mentah, Oven
	ProductionOrderID *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID `gorm:"type:uuid;index" json:"productionStepId,omitempty"`
	RequisitionID     *uuid.UUID `gorm:"type:uuid;index" json:"requisitionId,omitempty"`
	ProductID         *uuid.UUID `gorm:"type:uuid;index" json:"productId,omitempty"`
	ProductName       string     `gorm:"type:varchar(255);not null" json:"productName"`
	BatchNo           string     `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	OvenDate          string     `gorm:"type:varchar(10)" json:"ovenDate,omitempty"`
	MeshSize          string     `gorm:"type:varchar(50);not null" json:"meshSize"`

	// Pemeriksaan AWAL
	InitialCondition string `gorm:"type:varchar(100);not null;default:'Baik & Bersih'" json:"initialCondition"`
	InitialFindings  string `gorm:"type:text" json:"initialFindings,omitempty"`
	InitialAction    string `gorm:"type:text" json:"initialAction,omitempty"`
	InitialPICName   string `gorm:"type:varchar(150);not null" json:"initialPicName"`
	InitialQCName    string `gorm:"type:varchar(150)" json:"initialQcName,omitempty"`
	InitialQCPassed  bool   `gorm:"not null;default:true" json:"initialQcPassed"`

	// Pemeriksaan AKHIR
	FinalCondition string `gorm:"type:varchar(100);not null;default:'Baik & Bersih'" json:"finalCondition"`
	FinalFindings  string `gorm:"type:text" json:"finalFindings,omitempty"`
	FinalAction    string `gorm:"type:text" json:"finalAction,omitempty"`
	FinalPICName   string `gorm:"type:varchar(150);not null" json:"finalPicName"`
	FinalQCName    string `gorm:"type:varchar(150)" json:"finalQcName,omitempty"`
	FinalQCPassed  bool   `gorm:"not null;default:true" json:"finalQcPassed"`

	OverallStatus string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, failed, action_required
	Notes         string `gorm:"type:text" json:"notes,omitempty"`
}

func (SievingMonitoringLog) TableName() string {
	return "qc_sieving_monitoring_logs"
}

// MagnetTrapLog records magnetic trap inspection & gauss meter tests (Form QC-038-CMT)
type MagnetTrapLog struct {
	types.BaseEntity
	CompanyID          *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID           *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo         string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	CheckMonth         string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate          string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	Room               string     `gorm:"type:varchar(50);not null;default:'Ayak Mentah'" json:"room"` // Ayak Mentah, Ayak Matang
	MachineNo          string     `gorm:"type:varchar(50);not null;index" json:"machineNo"`
	PICResponsible     string     `gorm:"type:varchar(150);not null" json:"picResponsible"`
	ProductionOrderID  *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`
	ProductionStepID   *uuid.UUID `gorm:"type:uuid;index" json:"productionStepId,omitempty"`
	BatchNo            string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`

	// Evaluasi Magnet
	IsClean            bool    `gorm:"not null;default:true" json:"isClean"` // true (√ bersih), false (x kotor)
	GaussPoint1        float64 `gorm:"type:decimal(8,2);not null;default:0" json:"gaussPoint1"` // mT
	GaussPoint2        float64 `gorm:"type:decimal(8,2);not null;default:0" json:"gaussPoint2"` // mT
	GaussPoint3        float64 `gorm:"type:decimal(8,2);not null;default:0" json:"gaussPoint3"` // mT
	GaussAverage       float64 `gorm:"type:decimal(8,2);not null;default:0" json:"gaussAverage"` // mT
	IsPowerSufficient  bool    `gorm:"not null;default:true" json:"isPowerSufficient"` // true jika >= 700 mT

	ForeignMatterFound string     `gorm:"type:text" json:"foreignMatterFound,omitempty"`
	Issues             string     `gorm:"type:text" json:"issues,omitempty"`
	ActionTaken        string     `gorm:"type:text" json:"actionTaken,omitempty"`
	PICName            string     `gorm:"type:varchar(150);not null" json:"picName"`
	QCName             string     `gorm:"type:varchar(150)" json:"qcName,omitempty"`
	QCPassed           bool       `gorm:"not null;default:true" json:"qcPassed"`
	QAVerifiedBy       string     `gorm:"type:varchar(150)" json:"qaVerifiedBy,omitempty"`
	QAVerifiedAt       *time.Time `json:"qaVerifiedAt,omitempty"`

	OverallStatus string `gorm:"type:varchar(20);not null;default:'cleared'" json:"overallStatus"` // cleared, warning, replaced
	Notes         string `gorm:"type:text" json:"notes,omitempty"`
}

func (MagnetTrapLog) TableName() string {
	return "qc_magnet_trap_logs"
}

// MeshBoltMonitoringLog records bolt & nut integrity on sieving mesh and processing equipment to prevent physical hazard (Form QC-011-MNM/MNB/MBB)
type MeshBoltMonitoringLog struct {
	types.BaseEntity
	CompanyID          *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID           *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo         string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	DocumentCode       string     `gorm:"type:varchar(30);default:'QC-011-MNM'" json:"documentCode"`
	FormType           string     `gorm:"type:varchar(20);default:'MNM';index" json:"formType"` // MNM, MNB, MBB
	StageCategory      string     `gorm:"type:varchar(30);default:'mentah';index" json:"stageCategory"` // mentah, matang, gula_cair, brontol
	CheckMonth         string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"`
	CheckDate          string     `gorm:"type:varchar(10);not null;index" json:"checkDate"`
	Room               string     `gorm:"type:varchar(50);not null;default:'Ayak Mentah'" json:"room"`
	MachineNo          string     `gorm:"type:varchar(50);index" json:"machineNo,omitempty"`
	MeshSize           string     `gorm:"type:varchar(50)" json:"meshSize,omitempty"`
	PICResponsible     string     `gorm:"type:varchar(150);not null" json:"picResponsible"`
	ProductionOrderID  *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`
	ProductionStepID   *uuid.UUID `gorm:"type:uuid;index" json:"productionStepId,omitempty"`
	BatchNo            string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`

	// Pemeriksaan AWAL
	InitialBoltCount   int    `gorm:"not null;default:16" json:"initialBoltCount"`
	InitialIsComplete  bool   `gorm:"not null;default:true" json:"initialIsComplete"`
	InitialIsTight     bool   `gorm:"not null;default:true" json:"initialIsTight"`
	InitialCondition   string `gorm:"type:varchar(100);not null;default:'SS304 Utuh & Bebas Karat'" json:"initialCondition"`
	InitialFindings    string `gorm:"type:text" json:"initialFindings,omitempty"`
	InitialAction      string `gorm:"type:text" json:"initialAction,omitempty"`
	InitialPICName     string `gorm:"type:varchar(150);not null" json:"initialPicName"`
	InitialQCName      string `gorm:"type:varchar(150)" json:"initialQcName,omitempty"`

	// Pemeriksaan AKHIR
	FinalBoltCount     int    `gorm:"not null;default:16" json:"finalBoltCount"`
	FinalIsComplete    bool   `gorm:"not null;default:true" json:"finalIsComplete"`
	FinalIsTight       bool   `gorm:"not null;default:true" json:"finalIsTight"`
	IsCountMatched     bool   `gorm:"not null;default:true" json:"isCountMatched"` // InitialBoltCount == FinalBoltCount
	FinalFindings      string `gorm:"type:text" json:"finalFindings,omitempty"`
	FinalAction        string `gorm:"type:text" json:"finalAction,omitempty"`
	FinalPICName       string `gorm:"type:varchar(150);not null" json:"finalPicName"`
	FinalQCName        string `gorm:"type:varchar(150)" json:"finalQcName,omitempty"`

	QAVerifiedBy  string     `gorm:"type:varchar(150)" json:"qaVerifiedBy,omitempty"`
	QAVerifiedAt  *time.Time `json:"qaVerifiedAt,omitempty"`
	OverallStatus string     `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, hazard_alert
	Notes         string     `gorm:"type:text" json:"notes,omitempty"`
}

func (MeshBoltMonitoringLog) TableName() string {
	return "qc_mesh_bolt_monitoring_logs"
}

// FilterMeshMonitoringLog records filter screen / mesh checks for liquid sugar & VCO lines (Form QC-011-MNP & QC-011-MMV)
type FilterMeshMonitoringLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	DocumentCode      string     `gorm:"type:varchar(30);not null;default:'QC-011-MNP'" json:"documentCode"` // QC-011-MNP, QC-011-MMV
	Category          string     `gorm:"type:varchar(50);not null;default:'gula_cair'" json:"category"` // gula_cair, vco
	Location          string     `gorm:"type:varchar(100);not null;default:'Ruang Gula Cair'" json:"location"` // Ruang Gula Cair, Ruang VCO
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	DayOfWeek         string     `gorm:"type:varchar(20)" json:"dayOfWeek,omitempty"`
	ProductName       string     `gorm:"type:varchar(255);not null" json:"productName"`
	ProductCode       string     `gorm:"type:varchar(100)" json:"productCode,omitempty"`
	BatchNo           string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	MeshSize          string     `gorm:"type:varchar(50);not null" json:"meshSize"`
	PICResponsible    string     `gorm:"type:varchar(150);not null" json:"picResponsible"`

	// Pengecekan AWAL
	InitialCondition  string `gorm:"type:varchar(100);not null;default:'Baik'" json:"initialCondition"`
	InitialFindings   string `gorm:"type:text" json:"initialFindings,omitempty"`
	InitialAction     string `gorm:"type:text" json:"initialAction,omitempty"`
	InitialPICName    string `gorm:"type:varchar(150);not null" json:"initialPicName"`
	InitialQCName     string `gorm:"type:varchar(150)" json:"initialQcName,omitempty"`

	// Pengecekan AKHIR
	FinalCondition    string `gorm:"type:varchar(100);not null;default:'Baik'" json:"finalCondition"`
	FinalFindings     string `gorm:"type:text" json:"finalFindings,omitempty"`
	FinalAction       string `gorm:"type:text" json:"finalAction,omitempty"`
	FinalPICName      string `gorm:"type:varchar(150);not null" json:"finalPicName"`
	FinalQCName       string `gorm:"type:varchar(150)" json:"finalQcName,omitempty"`

	OverallStatus     string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, action_required
	Notes             string `gorm:"type:text" json:"notes,omitempty"`
}

func (FilterMeshMonitoringLog) TableName() string {
	return "qc_filter_mesh_monitoring_logs"
}

// BrontolBladeMonitoringLog records knife/blade integrity checks on Brontol machines (Form QC-011-MPB.01)
type BrontolBladeMonitoringLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	DocumentCode      string     `gorm:"type:varchar(30);default:'QC-011-MPB'" json:"documentCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	DayOfWeek         string     `gorm:"type:varchar(20)" json:"dayOfWeek,omitempty"`
	Location          string     `gorm:"type:varchar(100);not null;default:'Ruang ayak matang'" json:"location"`
	PICResponsible    string     `gorm:"type:varchar(150);not null;default:'Sajiman'" json:"picResponsible"`
	ProductName       string     `gorm:"type:varchar(255);not null" json:"productName"`
	ProductCode       string     `gorm:"type:varchar(100)" json:"productCode,omitempty"`

	// Pengecekan AWAL
	InitialCondition  string `gorm:"type:varchar(100);not null;default:'Baik'" json:"initialCondition"` // Baik, Gompal, Tumpul, Retak
	InitialFindings   string `gorm:"type:text" json:"initialFindings,omitempty"`
	InitialAction     string `gorm:"type:text" json:"initialAction,omitempty"`
	InitialPICName    string `gorm:"type:varchar(150);not null" json:"initialPicName"`
	InitialQCName     string `gorm:"type:varchar(150)" json:"initialQcName,omitempty"`

	// Pengecekan AKHIR
	FinalCondition    string `gorm:"type:varchar(100);not null;default:'Baik'" json:"finalCondition"` // Baik, Gompal, Tumpul, Retak
	FinalFindings     string `gorm:"type:text" json:"finalFindings,omitempty"`
	FinalAction       string `gorm:"type:text" json:"finalAction,omitempty"`
	FinalPICName      string `gorm:"type:varchar(150);not null" json:"finalPicName"`
	FinalQCName       string `gorm:"type:varchar(150)" json:"finalQcName,omitempty"`

	OverallStatus     string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, hazard_alert
	Notes             string `gorm:"type:text" json:"notes,omitempty"`
}

func (BrontolBladeMonitoringLog) TableName() string {
	return "qc_brontol_blade_monitoring_logs"
}

// OvenMonitoringLog records temperature & moisture during oven drying cycle (Form QC-012-MNO.05)
type OvenMonitoringLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'05'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	OvenNo            string     `gorm:"type:varchar(50);not null;index" json:"ovenNo"` // Oven 01, Oven 02
	ProductionOrderID *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`
	ProductionStepID  *uuid.UUID `gorm:"type:uuid;index" json:"productionStepId,omitempty"`
	BatchNo           string     `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	ProductName       string     `gorm:"type:varchar(255);not null" json:"productName"`

	// Fase AWAL
	InitialTemperature float64 `gorm:"type:decimal(5,2);not null;default:0" json:"initialTemperature"` // °C
	InitialTime        string  `gorm:"type:varchar(10);not null" json:"initialTime"` // HH:mm
	InitialMoisture    float64 `gorm:"type:decimal(5,2);not null;default:0" json:"initialMoisture"` // %

	// Fase STABIL (100°C)
	StabilizedTime string `gorm:"type:varchar(10);not null" json:"stabilizedTime"` // HH:mm

	// Fase PROSES (Pemantauan berkala ± 1 jam setelah stabil)
	ProcessTemperature float64  `gorm:"type:decimal(5,2);not null;default:0" json:"processTemperature"` // °C
	ProcessMoisture1   *float64 `gorm:"type:decimal(5,2)" json:"processMoisture1,omitempty"` // %
	ProcessMoisture2   *float64 `gorm:"type:decimal(5,2)" json:"processMoisture2,omitempty"` // %
	ProcessMoisture3   *float64 `gorm:"type:decimal(5,2)" json:"processMoisture3,omitempty"` // %
	ProcessMoisture4   *float64 `gorm:"type:decimal(5,2)" json:"processMoisture4,omitempty"` // %

	// Fase AKHIR (Siap diturunkan)
	FinalTemperature       float64 `gorm:"type:decimal(5,2);not null;default:0" json:"finalTemperature"` // °C
	FinalTime              string  `gorm:"type:varchar(10);not null" json:"finalTime"` // HH:mm
	FinalMoisture          float64 `gorm:"type:decimal(5,2);not null;default:0" json:"finalMoisture"` // % target <= 2.0%
	IsTargetMoistureReached bool    `gorm:"not null;default:true" json:"isTargetMoistureReached"` // true jika <= 2.0%

	Issues         string `gorm:"type:text" json:"issues,omitempty"`
	ActionTaken    string `gorm:"type:text" json:"actionTaken,omitempty"`
	PICName        string `gorm:"type:varchar(150);not null" json:"picName"`
	QCVerifierName string `gorm:"type:varchar(150)" json:"qcVerifierName,omitempty"`
	QCPassed       bool   `gorm:"not null;default:true" json:"qcPassed"`

	OverallStatus string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, failed, re_oven
	Notes         string `gorm:"type:text" json:"notes,omitempty"`
}

func (OvenMonitoringLog) TableName() string {
	return "qc_oven_monitoring_logs"
}

// ─────────────────────────────────────────────────────────────────────────────
// Kontrol Ruang Packing (MNJ, MNG), Metal Detector CCP (CPP, VMD), & Produk Jadi (PJD)
// ─────────────────────────────────────────────────────────────────────────────

// NeedleMonitoringLog records sewing machine needle status & stock to prevent needle fragments (Form QC-011-MNJ.01)
type NeedleMonitoringLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'01'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	SewingMachineNo   string     `gorm:"type:varchar(50);not null;index" json:"sewingMachineNo"` // Mesin Jahit 01, dll.
	PICResponsible    string     `gorm:"type:varchar(150);not null" json:"picResponsible"`

	// Pengecekan AWAL
	InitialIsPresent  bool   `gorm:"not null;default:true" json:"initialIsPresent"`
	InitialCondition  string `gorm:"type:varchar(50);not null;default:'Baik'" json:"initialCondition"` // Baik, Rusak, Patah
	InitialAnalysis   string `gorm:"type:text" json:"initialAnalysis,omitempty"`
	InitialAction     string `gorm:"type:text" json:"initialAction,omitempty"`

	// Pengecekan AKHIR
	FinalIsPresent    bool   `gorm:"not null;default:true" json:"finalIsPresent"`
	FinalCondition    string `gorm:"type:varchar(50);not null;default:'Baik'" json:"finalCondition"` // Baik, Rusak, Patah
	FinalAnalysis     string `gorm:"type:text" json:"finalAnalysis,omitempty"`
	FinalAction       string `gorm:"type:text" json:"finalAction,omitempty"`

	// Stok jarum di ruang packing
	NeedleStockInRoom int    `gorm:"not null;default:0" json:"needleStockInRoom"`

	PICName           string `gorm:"type:varchar(150);not null" json:"picName"`
	QCVerifierName    string `gorm:"type:varchar(150)" json:"qcVerifierName,omitempty"`
	OverallStatus     string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, hazard_alert
	Notes             string `gorm:"type:text" json:"notes,omitempty"`
}

func (NeedleMonitoringLog) TableName() string {
	return "qc_needle_monitoring_logs"
}

// ScissorMonitoringLog records scissor count & condition in packing room (Form QC-011-MNJ.01 Sheet MNG)
type ScissorMonitoringLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'01'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	Location          string     `gorm:"type:varchar(100);not null;default:'Ruang Packing'" json:"location"`
	PICResponsible    string     `gorm:"type:varchar(150);not null" json:"picResponsible"`

	// Pengecekan AWAL
	InitialCount      int    `gorm:"not null;default:1" json:"initialCount"`
	InitialCondition  string `gorm:"type:varchar(50);not null;default:'Baik'" json:"initialCondition"` // Baik, Rusak, Patah
	InitialAnalysis   string `gorm:"type:text" json:"initialAnalysis,omitempty"`
	InitialAction     string `gorm:"type:text" json:"initialAction,omitempty"`

	// Pengecekan AKHIR
	FinalCount        int    `gorm:"not null;default:1" json:"finalCount"`
	FinalCondition    string `gorm:"type:varchar(50);not null;default:'Baik'" json:"finalCondition"` // Baik, Rusak, Patah
	FinalAnalysis     string `gorm:"type:text" json:"finalAnalysis,omitempty"`
	FinalAction       string `gorm:"type:text" json:"finalAction,omitempty"`

	IsCountMatched    bool   `gorm:"not null;default:true" json:"isCountMatched"` // InitialCount == FinalCount
	PICName           string `gorm:"type:varchar(150);not null" json:"picName"`
	QCVerifierName    string `gorm:"type:varchar(150)" json:"qcVerifierName,omitempty"`
	OverallStatus     string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, hazard_alert
	Notes             string `gorm:"type:text" json:"notes,omitempty"`
}

func (ScissorMonitoringLog) TableName() string {
	return "qc_scissor_monitoring_logs"
}

// MetalDetectorCleaningLog records usage & cleaning logbook for metal detector / X-Ray (Form PRO-007-CPP Revisi 01).
// StartTime/EndTime = jam Pemakaian (Mulai/Selesai); CleaningStartTime/CleaningEndTime = jam Pembersihan;
// OperatorName = PIC; QCInspectorName = "Diketahui Oleh: QA".
type MetalDetectorCleaningLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	MachineNo         string     `gorm:"type:varchar(50);not null;default:'MD-01'" json:"machineNo"`
	Location          string     `gorm:"type:varchar(100);not null;default:'Ruang Packing'" json:"location"`
	CleaningType      string     `gorm:"type:varchar(50);not null;default:'pre_operational'" json:"cleaningType"` // pre_operational, batch_change, post_operational

	SensorCleanliness    string  `gorm:"type:varchar(50);not null;default:'Bersih'" json:"sensorCleanliness"` // Bersih, Perlu Dibersihkan
	BeltCleanliness      string  `gorm:"type:varchar(50);not null;default:'Bersih'" json:"beltCleanliness"` // Bersih, Perlu Dibersihkan
	RejectBinCleanliness string  `gorm:"type:varchar(50);not null;default:'Bersih & Kosong'" json:"rejectBinCleanliness"`
	PhysicalCondition    string  `gorm:"type:varchar(50);not null;default:'Normal'" json:"physicalCondition"` // Normal, Cacat, Kabel Terbuka

	StartTime         string     `gorm:"type:varchar(10);not null" json:"startTime"` // HH:mm
	EndTime           string     `gorm:"type:varchar(10);not null" json:"endTime"`   // HH:mm
	TotalHours        float64    `gorm:"type:decimal(5,2);default:0" json:"totalHours"`

	OperatorName      string     `gorm:"type:varchar(150);not null" json:"operatorName"`
	QCInspectorName   string     `gorm:"type:varchar(150)" json:"qcInspectorName,omitempty"`
	OverallStatus     string     `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, needs_cleaning, maintenance
	Notes             string     `gorm:"type:text" json:"notes,omitempty"`

	// ── Form PRO-007-CPP Revisi 01 (Catatan Pemakaian dan Pembersihan Alat) ──
	// Sheet varian: "MD" (Metal Detector – R. Packing), "MD GC" (Metal Detector – R. Gula Cair), "X-Ray" (X-Ray – R. Packing).
	DocumentCode      string `gorm:"type:varchar(30);default:'PRO-007-CPP'" json:"documentCode"`
	RevisionCode      string `gorm:"type:varchar(10);default:'01'" json:"revisionCode"`
	EquipmentName     string `gorm:"type:varchar(50);default:'Metal Detector'" json:"equipmentName"` // Nama Alat: Metal Detector, X-Ray
	ProductName       string `gorm:"type:varchar(200)" json:"productName,omitempty"`                 // Nama Produk
	BatchNo           string `gorm:"type:varchar(100)" json:"batchNo,omitempty"`                     // No. Batch
	CustomerName      string `gorm:"type:varchar(200)" json:"customerName,omitempty"`                // Customer
	Quantity          string `gorm:"type:varchar(50)" json:"quantity,omitempty"`                     // Jumlah
	Finding           string `gorm:"type:text" json:"finding,omitempty"`                             // Pemakaian - Temuan
	AnalysisResult    string `gorm:"type:text" json:"analysisResult,omitempty"`                      // Pemakaian - Hasil Analisa
	CorrectiveAction  string `gorm:"type:text" json:"correctiveAction,omitempty"`                    // Pemakaian - Tindakan Perbaikan
	CleaningStartTime string `gorm:"type:varchar(10)" json:"cleaningStartTime,omitempty"`            // Pembersihan - Jam Mulai (WIB)
	CleaningEndTime   string `gorm:"type:varchar(10)" json:"cleaningEndTime,omitempty"`              // Pembersihan - Jam Selesai (WIB)
	QAVerifiedDate    string `gorm:"type:varchar(10)" json:"qaVerifiedDate,omitempty"`               // Diketahui Oleh QA - Tgl.
}

func (MetalDetectorCleaningLog) TableName() string {
	return "qc_metal_detector_cleaning_logs"
}

// MetalDetectorVerificationLog records test piece verification checks for CCP Metal Detector (Form QC-017-VMD Rev 03 2026 / Rev 01 Cair)
type MetalDetectorVerificationLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'03'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	ProductType       string     `gorm:"type:varchar(50);not null;default:'gula_semut'" json:"productType"` // gula_semut, gula_cair
	Location          string     `gorm:"type:varchar(100);not null;default:'Ruang Packing'" json:"location"`
	BatchNo           string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	PackageCategory   string     `gorm:"type:varchar(30);not null;default:'<= 12.5 kg'" json:"packageCategory"` // <= 12.5 kg, > 12.5 kg
	MonitoringRound   int        `gorm:"not null;default:1" json:"monitoringRound"` // 1, 2, 3...
	CheckTime         string     `gorm:"type:varchar(10);not null" json:"checkTime"` // HH:mm

	// Fe Test Piece (Ferrous)
	FeTestPieceSize   string `gorm:"type:varchar(50);not null;default:'Fe >= 2.0mm'" json:"feTestPieceSize"`
	FeAlarmTriggered  bool   `gorm:"not null;default:true" json:"feAlarmTriggered"` // Harus Bunyi
	FeConveyorStopped bool   `gorm:"not null;default:true" json:"feConveyorStopped"` // Harus Berhenti
	FePassed          bool   `gorm:"not null;default:true" json:"fePassed"`

	// Non-Fe Test Piece (Non-Ferrous)
	NonFeTestPieceSize   string `gorm:"type:varchar(50);not null;default:'Non-Fe >= 2.0mm'" json:"nonFeTestPieceSize"`
	NonFeAlarmTriggered  bool   `gorm:"not null;default:true" json:"nonFeAlarmTriggered"`
	NonFeConveyorStopped bool   `gorm:"not null;default:true" json:"nonFeConveyorStopped"`
	NonFePassed          bool   `gorm:"not null;default:true" json:"nonFePassed"`

	// SS Test Piece (Stainless Steel)
	SSTestPieceSize   string `gorm:"type:varchar(50);not null;default:'SS >= 3.5mm'" json:"ssTestPieceSize"`
	SSAlarmTriggered  bool   `gorm:"not null;default:true" json:"ssAlarmTriggered"`
	SSConveyorStopped bool   `gorm:"not null;default:true" json:"ssConveyorStopped"`
	SSPassed          bool   `gorm:"not null;default:true" json:"ssPassed"`

	IsAllPiecesPassed bool   `gorm:"not null;default:true" json:"isAllPiecesPassed"` // Fe && Non-Fe && SS
	CorrectiveAction  string `gorm:"type:text" json:"correctiveAction,omitempty"`
	PICName           string `gorm:"type:varchar(150);not null" json:"picName"`
	QCVerifierName    string `gorm:"type:varchar(150)" json:"qcVerifierName,omitempty"`
	OverallStatus     string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, failed
	Notes             string `gorm:"type:text" json:"notes,omitempty"`
}

func (MetalDetectorVerificationLog) TableName() string {
	return "qc_metal_detector_verification_logs"
}

// FinishedGoodsInspectionLog records finished goods inspection (Form QC-004-PJD.05 Rev 05)
type FinishedGoodsInspectionLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'05'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	ProductName       string     `gorm:"type:varchar(255);not null" json:"productName"`
	BatchNo           string     `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	RawMaterialCode   string     `gorm:"type:varchar(100)" json:"rawMaterialCode,omitempty"`
	ProductionOrderID *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`

	// Pemeriksaan Fisik & Kemasan
	PrimaryPackaging   string  `gorm:"type:varchar(50);not null;default:'Baik'" json:"primaryPackaging"` // Baik, Bocor, Cacat
	SecondaryPackaging string  `gorm:"type:varchar(50);not null;default:'Baik'" json:"secondaryPackaging"` // Baik, Rusak, Sobek
	NetWeight          float64 `gorm:"type:decimal(10,2);not null;default:0" json:"netWeight"` // kg / gram
	GrossWeight        float64 `gorm:"type:decimal(10,2);not null;default:0" json:"grossWeight"` // kg / gram (gula + kemasan primer + sekunder)

	// Pemeriksaan Organoleptik
	AppearanceShape    string  `gorm:"type:varchar(100);not null;default:'Serbuk/Butiran Kristal Homogen'" json:"appearanceShape"`
	Aroma              string  `gorm:"type:varchar(100);not null;default:'Khas Nira Gula Kelapa'" json:"aroma"` // Khas Nira, Sangit, Asam
	ColorScore         int     `gorm:"not null;default:2" json:"colorScore"` // 1: Light brown, 2: Normal, 3: Dark brown
	ColorDescription   string  `gorm:"type:varchar(50);default:'Normal'" json:"colorDescription"`
	Taste              string  `gorm:"type:varchar(100);not null;default:'Manis Khas Alami'" json:"taste"` // Manis Khas Alami, Asam, Pahit/Apek

	// Keamanan Pangan
	IsFreeForeignMatter bool    `gorm:"not null;default:true" json:"isFreeForeignMatter"` // Bebas benda asing
	ForeignMatterNotes  string  `gorm:"type:varchar(255)" json:"foreignMatterNotes,omitempty"`
	MoistureContent     float64 `gorm:"type:decimal(5,2);not null;default:0" json:"moistureContent"` // % target <= 2.0%
	IsMoistureStandard  bool    `gorm:"not null;default:true" json:"isMoistureStandard"` // true jika <= 2.0%

	// Keputusan Mutu
	InspectionDecision string  `gorm:"type:varchar(20);not null;default:'lolos'" json:"inspectionDecision"` // lolos, ditolak
	Notes              string  `gorm:"type:text" json:"notes,omitempty"`
	QCPersonnelName    string  `gorm:"type:varchar(150);not null" json:"qcPersonnelName"`
	QAVerifiedBy       string  `gorm:"type:varchar(150)" json:"qaVerifiedBy,omitempty"`
}

func (FinishedGoodsInspectionLog) TableName() string {
	return "qc_finished_goods_inspection_logs"
}

