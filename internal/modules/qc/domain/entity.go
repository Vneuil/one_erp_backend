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

