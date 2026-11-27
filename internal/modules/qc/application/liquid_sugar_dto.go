package application

import "github.com/google/uuid"

// ─── MSS: Sterilization Temperature Log (Form QC-057-MSS) ───────────────────

type CreateSterilizationTempLogDTO struct {
	Stage             string     `json:"stage,omitempty"` // tahap_1, tahap_2
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate" validate:"required"`
	BatchNo           string     `json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	PackagingType string  `json:"packagingType" validate:"required"`
	InitialTime   string  `json:"initialTime" validate:"required"`
	InitialTemp   float64 `json:"initialTemp"`
	HeatingTime   string  `json:"heatingTime" validate:"required"`
	HeatingTemp   float64 `json:"heatingTemp"`
	FinalTime     string  `json:"finalTime" validate:"required"`
	FinalTemp     float64 `json:"finalTemp"`

	Issues       string `json:"issues,omitempty"`
	ActionTaken  string `json:"actionTaken,omitempty"`
	QCName       string `json:"qcName,omitempty"`
	QAVerifiedBy string `json:"qaVerifiedBy,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type UpdateSterilizationTempLogDTO struct {
	Stage             string     `json:"stage,omitempty"`
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	PackagingType *string  `json:"packagingType,omitempty"`
	InitialTime   *string  `json:"initialTime,omitempty"`
	InitialTemp   *float64 `json:"initialTemp,omitempty"`
	HeatingTime   *string  `json:"heatingTime,omitempty"`
	HeatingTemp   *float64 `json:"heatingTemp,omitempty"`
	FinalTime     *string  `json:"finalTime,omitempty"`
	FinalTemp     *float64 `json:"finalTemp,omitempty"`

	Issues        *string `json:"issues,omitempty"`
	ActionTaken   *string `json:"actionTaken,omitempty"`
	QCName        *string `json:"qcName,omitempty"`
	QAVerifiedBy  *string `json:"qaVerifiedBy,omitempty"`
	OverallStatus *string `json:"overallStatus,omitempty"`
	Notes         *string `json:"notes,omitempty"`
}

// ─── PVK: Liquid Packaging Visual Check (Form QC-041-PVK Rev. 01) ───────────

type CreateLiquidPackagingVisualCheckDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate" validate:"required"`
	BatchNo           string     `json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	PICResponsible     string `json:"picResponsible,omitempty"`
	PackagingType      string `json:"packagingType" validate:"required"`
	Quantity           int    `json:"quantity"`
	BuyerName          string `json:"buyerName,omitempty"`
	SterilizationInfo  string `json:"sterilizationInfo,omitempty"`
	PackagingCondition string `json:"packagingCondition,omitempty"` // B, K
	Findings           string `json:"findings,omitempty"`
	ActionTaken        string `json:"actionTaken,omitempty"`
	PICName            string `json:"picName" validate:"required"`
	QCName             string `json:"qcName,omitempty"`
	QCKnownBy          string `json:"qcKnownBy,omitempty"`
	QCKnownDate        string `json:"qcKnownDate,omitempty"`
	QAVerifiedBy       string `json:"qaVerifiedBy,omitempty"`
	QAVerifiedDate     string `json:"qaVerifiedDate,omitempty"`
	Notes              string `json:"notes,omitempty"`
}

type UpdateLiquidPackagingVisualCheckDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	PICResponsible     *string `json:"picResponsible,omitempty"`
	PackagingType      *string `json:"packagingType,omitempty"`
	Quantity           *int    `json:"quantity,omitempty"`
	BuyerName          *string `json:"buyerName,omitempty"`
	SterilizationInfo  *string `json:"sterilizationInfo,omitempty"`
	PackagingCondition *string `json:"packagingCondition,omitempty"`
	Findings           *string `json:"findings,omitempty"`
	ActionTaken        *string `json:"actionTaken,omitempty"`
	PICName            *string `json:"picName,omitempty"`
	QCName             *string `json:"qcName,omitempty"`
	QCKnownBy          *string `json:"qcKnownBy,omitempty"`
	QCKnownDate        *string `json:"qcKnownDate,omitempty"`
	QAVerifiedBy       *string `json:"qaVerifiedBy,omitempty"`
	QAVerifiedDate     *string `json:"qaVerifiedDate,omitempty"`
	OverallStatus      *string `json:"overallStatus,omitempty"`
	Notes              *string `json:"notes,omitempty"`
}

// ─── MNC: Liquid Sugar Processing Log (Form QC-014-MNC) ─────────────────────

type CreateLiquidSugarProcessingLogDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate" validate:"required"`
	BatchNo           string     `json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	ProductCode string `json:"productCode" validate:"required"`
	ProductName string `json:"productName,omitempty"`

	InitialTime     string  `json:"initialTime" validate:"required"`
	InitialTemp     float64 `json:"initialTemp"`
	InitialMoisture float64 `json:"initialMoisture"`
	InitialBrix     float64 `json:"initialBrix"`
	InitialPH       float64 `json:"initialPh"`

	HeatingTime string  `json:"heatingTime" validate:"required"`
	HeatingTemp float64 `json:"heatingTemp"`

	FinalTime        string  `json:"finalTime" validate:"required"`
	FinalProductTemp float64 `json:"finalProductTemp"`
	FinalMoisture    float64 `json:"finalMoisture"`
	FinalBrix        float64 `json:"finalBrix"`
	FinalPH          float64 `json:"finalPh"`

	Issues       string `json:"issues,omitempty"`
	ActionTaken  string `json:"actionTaken,omitempty"`
	PICName      string `json:"picName" validate:"required"`
	QCName       string `json:"qcName,omitempty"`
	QAVerifiedBy string `json:"qaVerifiedBy,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type UpdateLiquidSugarProcessingLogDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	ProductCode *string `json:"productCode,omitempty"`
	ProductName *string `json:"productName,omitempty"`

	InitialTime     *string  `json:"initialTime,omitempty"`
	InitialTemp     *float64 `json:"initialTemp,omitempty"`
	InitialMoisture *float64 `json:"initialMoisture,omitempty"`
	InitialBrix     *float64 `json:"initialBrix,omitempty"`
	InitialPH       *float64 `json:"initialPh,omitempty"`

	HeatingTime *string  `json:"heatingTime,omitempty"`
	HeatingTemp *float64 `json:"heatingTemp,omitempty"`

	FinalTime        *string  `json:"finalTime,omitempty"`
	FinalProductTemp *float64 `json:"finalProductTemp,omitempty"`
	FinalMoisture    *float64 `json:"finalMoisture,omitempty"`
	FinalBrix        *float64 `json:"finalBrix,omitempty"`
	FinalPH          *float64 `json:"finalPh,omitempty"`

	Issues        *string `json:"issues,omitempty"`
	ActionTaken   *string `json:"actionTaken,omitempty"`
	PICName       *string `json:"picName,omitempty"`
	QCName        *string `json:"qcName,omitempty"`
	QAVerifiedBy  *string `json:"qaVerifiedBy,omitempty"`
	OverallStatus *string `json:"overallStatus,omitempty"`
	Notes         *string `json:"notes,omitempty"`
}

// ─── PJC: Liquid Sugar Finished Goods Log (Form QC-015-PJC Rev. 02) ─────────

type CreateLiquidSugarFinishedGoodsLogDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate" validate:"required"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	ProductName     string `json:"productName" validate:"required"`
	BatchNo         string `json:"batchNo" validate:"required"`
	RawMaterialCode string `json:"rawMaterialCode,omitempty"`

	CheckPackaging         *bool    `json:"checkPackaging,omitempty"`
	CheckWeight            *bool    `json:"checkWeight,omitempty"`
	WeightValue            *float64 `json:"weightValue,omitempty"`
	CheckShape             *bool    `json:"checkShape,omitempty"`
	CheckAroma             *bool    `json:"checkAroma,omitempty"`
	ColorScore             int      `json:"colorScore"` // 1, 2, 3
	ColorDescription       string   `json:"colorDescription,omitempty"`
	CheckTaste             *bool    `json:"checkTaste,omitempty"`
	CheckFreeForeignMatter *bool    `json:"checkFreeForeignMatter,omitempty"`
	CheckSolubility        *bool    `json:"checkSolubility,omitempty"`
	MoistureContent        float64  `json:"moistureContent"`
	PHValue                float64  `json:"phValue"`
	BrixValue              float64  `json:"brixValue"`

	InspectionDecision string `json:"inspectionDecision,omitempty"` // LOLOS, DITOLAK
	Notes              string `json:"notes,omitempty"`
	QCPersonnelName    string `json:"qcPersonnelName" validate:"required"`
	QAVerifiedBy       string `json:"qaVerifiedBy,omitempty"`
	QAVerifiedDate     string `json:"qaVerifiedDate,omitempty"`
}

type UpdateLiquidSugarFinishedGoodsLogDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	ProductName     *string `json:"productName,omitempty"`
	BatchNo         *string `json:"batchNo,omitempty"`
	RawMaterialCode *string `json:"rawMaterialCode,omitempty"`

	CheckPackaging         *bool    `json:"checkPackaging,omitempty"`
	CheckWeight            *bool    `json:"checkWeight,omitempty"`
	WeightValue            *float64 `json:"weightValue,omitempty"`
	CheckShape             *bool    `json:"checkShape,omitempty"`
	CheckAroma             *bool    `json:"checkAroma,omitempty"`
	ColorScore             *int     `json:"colorScore,omitempty"`
	ColorDescription       *string  `json:"colorDescription,omitempty"`
	CheckTaste             *bool    `json:"checkTaste,omitempty"`
	CheckFreeForeignMatter *bool    `json:"checkFreeForeignMatter,omitempty"`
	CheckSolubility        *bool    `json:"checkSolubility,omitempty"`
	MoistureContent        *float64 `json:"moistureContent,omitempty"`
	PHValue                *float64 `json:"phValue,omitempty"`
	BrixValue              *float64 `json:"brixValue,omitempty"`

	InspectionDecision *string `json:"inspectionDecision,omitempty"`
	Notes              *string `json:"notes,omitempty"`
	QCPersonnelName    *string `json:"qcPersonnelName,omitempty"`
	QAVerifiedBy       *string `json:"qaVerifiedBy,omitempty"`
	QAVerifiedDate     *string `json:"qaVerifiedDate,omitempty"`
}

// ─── VHC: Liquid Sugar Wash Verification (Form QC-055-VHC Rev. 00) ──────────

type CreateLiquidSugarWashVerificationDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate" validate:"required"`
	BatchNo           string     `json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	Location       string `json:"location,omitempty"`
	PICResponsible string `json:"picResponsible,omitempty"`

	WaterTempActual      float64 `json:"waterTempActual"`
	CirculationStartTime string  `json:"circulationStartTime" validate:"required"`
	CirculationEndTime   string  `json:"circulationEndTime" validate:"required"`
	RinseColor           string  `json:"rinseColor,omitempty"`
	RinseAroma           string  `json:"rinseAroma,omitempty"`
	RinseDirt            string  `json:"rinseDirt,omitempty"`

	QCVerifierName string `json:"qcVerifierName" validate:"required"`
	QAVerifiedBy   string `json:"qaVerifiedBy,omitempty"`
	QAVerifiedDate string `json:"qaVerifiedDate,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type UpdateLiquidSugarWashVerificationDTO struct {
	CheckMonth        string     `json:"checkMonth,omitempty"`
	CheckDate         string     `json:"checkDate,omitempty"`
	BatchNo           string     `json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `json:"productionOrderId,omitempty"`

	Location       *string `json:"location,omitempty"`
	PICResponsible *string `json:"picResponsible,omitempty"`

	WaterTempActual      *float64 `json:"waterTempActual,omitempty"`
	CirculationStartTime *string  `json:"circulationStartTime,omitempty"`
	CirculationEndTime   *string  `json:"circulationEndTime,omitempty"`
	RinseColor           *string  `json:"rinseColor,omitempty"`
	RinseAroma           *string  `json:"rinseAroma,omitempty"`
	RinseDirt            *string  `json:"rinseDirt,omitempty"`

	QCVerifierName *string `json:"qcVerifierName,omitempty"`
	QAVerifiedBy   *string `json:"qaVerifiedBy,omitempty"`
	QAVerifiedDate *string `json:"qaVerifiedDate,omitempty"`
	OverallStatus  *string `json:"overallStatus,omitempty"`
	Notes          *string `json:"notes,omitempty"`
}

// ─── Flow Status Aggregation DTO ────────────────────────────────────────────

type LiquidSugarFlowStatusDTO struct {
	Month                string   `json:"month"`
	BatchNo              string   `json:"batchNo,omitempty"`
	MssStage1Count       int      `json:"mssStage1Count"`
	PvkCount             int      `json:"pvkCount"`
	VmdCount             int      `json:"vmdCount"`
	CppCount             int      `json:"cppCount"`
	MssStage2Count       int      `json:"mssStage2Count"`
	MnpCount             int      `json:"mnpCount"`
	MnbCount             int      `json:"mnbCount"`
	MncCount             int      `json:"mncCount"`
	PjcCount             int      `json:"pjcCount"`
	VhcCount             int      `json:"vhcCount"`
	Stage1Done           bool     `json:"stage1Done"`
	MnpDone              bool     `json:"mnpDone"`
	MnbDone              bool     `json:"mnbDone"`
	Qc011Done            bool     `json:"qc011Done"`
	Stage2Done           bool     `json:"stage2Done"`
	Stage3Done           bool     `json:"stage3Done"`
	Stage4Done           bool     `json:"stage4Done"`
	Stage5Done           bool     `json:"stage5Done"`
	CompletedStagesCount int      `json:"completedStagesCount"`
	CurrentStage         int      `json:"currentStage"`
	BatchNumbers         []string `json:"batchNumbers"`
}
