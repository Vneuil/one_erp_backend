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

