package domain

import (
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// SterilizationTempLog records sterilization temperature monitoring before/during liquid sugar flow (Form QC-057-MSS).
// Supports stage "tahap_1" (4-form prep) and "tahap_2" (2-form pre-process alongside QC-011).
type SterilizationTempLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	DocumentCode      string     `gorm:"type:varchar(30);default:'QC-057-MSS'" json:"documentCode"`
	Stage             string     `gorm:"type:varchar(20);not null;default:'tahap_1';index" json:"stage"` // tahap_1, tahap_2
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"`                // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"`                // YYYY-MM-DD
	BatchNo           string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`

	PackagingType   string  `gorm:"type:varchar(150);not null" json:"packagingType"`          // KEMASAN
	InitialTime     string  `gorm:"type:varchar(10);not null" json:"initialTime"`             // AWAL WAKTU (WIB)
	InitialTemp     float64 `gorm:"type:decimal(6,2);not null;default:0" json:"initialTemp"`  // AWAL SUHU (°C)
	HeatingTime     string  `gorm:"type:varchar(10);not null" json:"heatingTime"`             // HEATING WAKTU (WIB)
	HeatingTemp     float64 `gorm:"type:decimal(6,2);not null;default:0" json:"heatingTemp"`  // HEATING SUHU (°C) >= 70°C
	FinalTime       string  `gorm:"type:varchar(10);not null" json:"finalTime"`               // AKHIR WAKTU (WIB)
	FinalTemp       float64 `gorm:"type:decimal(6,2);not null;default:0" json:"finalTemp"`    // AKHIR SUHU (°C)
	DurationMinutes int     `gorm:"not null;default:0" json:"durationMinutes"`                // Durasi heating -> akhir (min 45 menit)
	IsStandardMet   bool    `gorm:"not null;default:true" json:"isStandardMet"`               // HeatingTemp >= 70 && DurationMinutes >= 45

	Issues        string `gorm:"type:text" json:"issues,omitempty"`        // PERMASALAHAN
	ActionTaken   string `gorm:"type:text" json:"actionTaken,omitempty"`   // TINDAKAN
	QCName        string `gorm:"type:varchar(150)" json:"qcName,omitempty"`
	QAVerifiedBy  string `gorm:"type:varchar(150)" json:"qaVerifiedBy,omitempty"`
	OverallStatus string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, action_required
	Notes         string `gorm:"type:text" json:"notes,omitempty"`
}

func (SterilizationTempLog) TableName() string {
	return "qc_sterilization_temp_logs"
}

// LiquidPackagingVisualCheck records visual inspection of primary liquid sugar packaging (Form QC-041-PVK Rev. 01).
type LiquidPackagingVisualCheck struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	DocumentCode      string     `gorm:"type:varchar(30);default:'QC-041-PVK'" json:"documentCode"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'01'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	BatchNo           string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`

	PICResponsible     string `gorm:"type:varchar(150);not null;default:'Sajiman'" json:"picResponsible"`
	PackagingType      string `gorm:"type:varchar(150);not null" json:"packagingType"`                 // JENIS KEMASAN
	Quantity           int    `gorm:"not null;default:0" json:"quantity"`                              // JUMLAH
	BuyerName          string `gorm:"type:varchar(200)" json:"buyerName,omitempty"`                    // NAMA BUYER
	SterilizationInfo  string `gorm:"type:varchar(150)" json:"sterilizationInfo,omitempty"`            // STERILISASI
	PackagingCondition string `gorm:"type:varchar(10);not null;default:'B'" json:"packagingCondition"` // B (Bersih), K (Kotor)
	Findings           string `gorm:"type:text" json:"findings,omitempty"`                             // TEMUAN
	ActionTaken        string `gorm:"type:text" json:"actionTaken,omitempty"`                          // TINDAKAN
	PICName            string `gorm:"type:varchar(150);not null" json:"picName"`                       // NAMA/PARAF PIC
	QCName             string `gorm:"type:varchar(150)" json:"qcName,omitempty"`                       // PARAF QC
	QCKnownBy          string `gorm:"type:varchar(150)" json:"qcKnownBy,omitempty"`                    // Diketahui Oleh QC
	QCKnownDate        string `gorm:"type:varchar(20)" json:"qcKnownDate,omitempty"`
	QAVerifiedBy       string `gorm:"type:varchar(150)" json:"qaVerifiedBy,omitempty"`                 // Diverifikasi Oleh QA
	QAVerifiedDate     string `gorm:"type:varchar(20)" json:"qaVerifiedDate,omitempty"`
	OverallStatus      string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, action_required
	Notes              string `gorm:"type:text" json:"notes,omitempty"`
}

func (LiquidPackagingVisualCheck) TableName() string {
	return "qc_liquid_packaging_visual_checks"
}

// LiquidSugarProcessingLog records cooking/heating parameters during liquid sugar production (Form QC-014-MNC).
type LiquidSugarProcessingLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	DocumentCode      string     `gorm:"type:varchar(30);default:'QC-014-MNC'" json:"documentCode"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'01'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	BatchNo           string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`

	ProductCode string `gorm:"type:varchar(100);not null" json:"productCode"` // KODE PRODUK
	ProductName string `gorm:"type:varchar(200)" json:"productName,omitempty"`

	// Fase AWAL (api mulai dinyalakan)
	InitialTime     string  `gorm:"type:varchar(10);not null" json:"initialTime"`
	InitialTemp     float64 `gorm:"type:decimal(6,2);not null;default:0" json:"initialTemp"`
	InitialMoisture float64 `gorm:"type:decimal(6,2);not null;default:0" json:"initialMoisture"` // KADAR AIR (%)
	InitialBrix     float64 `gorm:"type:decimal(6,2);not null;default:0" json:"initialBrix"`
	InitialPH       float64 `gorm:"type:decimal(5,2);not null;default:0" json:"initialPh"`

	// Fase HEATING (suhu mencapai CCP 90-95°C, minimal 10 menit)
	HeatingTime        string  `gorm:"type:varchar(10);not null" json:"heatingTime"`
	HeatingTemp        float64 `gorm:"type:decimal(6,2);not null;default:0" json:"heatingTemp"`
	HeatingDurationMin int     `gorm:"not null;default:0" json:"heatingDurationMin"`
	IsCcpPassed        bool    `gorm:"not null;default:true" json:"isCcpPassed"`

	// Fase AKHIR (siap diturunkan)
	FinalTime        string  `gorm:"type:varchar(10);not null" json:"finalTime"`
	FinalProductTemp float64 `gorm:"type:decimal(6,2);not null;default:0" json:"finalProductTemp"` // SUHU PRODUK (°C)
	FinalMoisture    float64 `gorm:"type:decimal(6,2);not null;default:0" json:"finalMoisture"`    // KADAR AIR (%)
	FinalBrix        float64 `gorm:"type:decimal(6,2);not null;default:0" json:"finalBrix"`
	FinalPH          float64 `gorm:"type:decimal(5,2);not null;default:0" json:"finalPh"`

	Issues        string `gorm:"type:text" json:"issues,omitempty"`
	ActionTaken   string `gorm:"type:text" json:"actionTaken,omitempty"`
	PICName       string `gorm:"type:varchar(150);not null" json:"picName"`
	QCName        string `gorm:"type:varchar(150)" json:"qcName,omitempty"`
	QAVerifiedBy  string `gorm:"type:varchar(150)" json:"qaVerifiedBy,omitempty"`
	OverallStatus string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, ccp_alert, action_required
	Notes         string `gorm:"type:text" json:"notes,omitempty"`
}

func (LiquidSugarProcessingLog) TableName() string {
	return "qc_liquid_sugar_processing_logs"
}

// LiquidSugarFinishedGoodsLog records finished liquid sugar QC release inspection (Form QC-015-PJC Rev. 02).
type LiquidSugarFinishedGoodsLog struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	DocumentCode      string     `gorm:"type:varchar(30);default:'QC-015-PJC'" json:"documentCode"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'02'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD
	ProductionOrderID *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`

	ProductName     string `gorm:"type:varchar(255);not null" json:"productName"`
	BatchNo         string `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	RawMaterialCode string `gorm:"type:varchar(100)" json:"rawMaterialCode,omitempty"` // Kode Bahan

	// 11 Kolom Pengecekan sesuai Form QC-015-PJC Rev 02
	CheckPackaging         bool     `gorm:"not null;default:true" json:"checkPackaging"`               // Kemasan
	CheckWeight            bool     `gorm:"not null;default:true" json:"checkWeight"`                  // Berat (ceklis)
	WeightValue            *float64 `gorm:"type:decimal(10,2)" json:"weightValue,omitempty"`           // Opsional berat aktual (kg)
	CheckShape             bool     `gorm:"not null;default:true" json:"checkShape"`                   // Bentuk
	CheckAroma             bool     `gorm:"not null;default:true" json:"checkAroma"`                   // Aroma
	ColorScore             int      `gorm:"not null;default:2" json:"colorScore"`                      // Warna*: 1 (light brown), 2 (normal), 3 (dark brown)
	ColorDescription       string   `gorm:"type:varchar(50);default:'Normal'" json:"colorDescription"` // Light Brown / Normal / Dark Brown
	CheckTaste             bool     `gorm:"not null;default:true" json:"checkTaste"`                   // Rasa
	CheckFreeForeignMatter bool     `gorm:"not null;default:true" json:"checkFreeForeignMatter"`       // Bebas Benda Asing
	CheckSolubility        bool     `gorm:"not null;default:true" json:"checkSolubility"`              // Kelarutan
	MoistureContent        float64  `gorm:"type:decimal(6,2);not null;default:0" json:"moistureContent"` // Kadar Air (%)
	PHValue                float64  `gorm:"type:decimal(5,2);not null;default:0" json:"phValue"`         // pH
	BrixValue              float64  `gorm:"type:decimal(6,2);not null;default:0" json:"brixValue"`       // Brix

	InspectionDecision string `gorm:"type:varchar(20);not null;default:'LOLOS'" json:"inspectionDecision"` // LOLOS, DITOLAK
	Notes              string `gorm:"type:text" json:"notes,omitempty"`                                      // Keterangan
	QCPersonnelName    string `gorm:"type:varchar(150);not null" json:"qcPersonnelName"`                   // Paraf QC
	QAVerifiedBy       string `gorm:"type:varchar(150)" json:"qaVerifiedBy,omitempty"`                     // Diverifikasi Oleh QA
	QAVerifiedDate     string `gorm:"type:varchar(20)" json:"qaVerifiedDate,omitempty"`
}

func (LiquidSugarFinishedGoodsLog) TableName() string {
	return "qc_liquid_sugar_finished_goods_logs"
}

// LiquidSugarWashVerification records post-production wash & sanitation verification in Ruang Gula Cair (Form QC-055-VHC Rev. 00).
type LiquidSugarWashVerification struct {
	types.BaseEntity
	CompanyID         *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentNo        string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"documentNo"`
	DocumentCode      string     `gorm:"type:varchar(30);default:'QC-055-VHC'" json:"documentCode"`
	RevisionCode      string     `gorm:"type:varchar(20);default:'00'" json:"revisionCode"`
	CheckMonth        string     `gorm:"type:varchar(7);not null;index" json:"checkMonth"` // YYYY-MM
	CheckDate         string     `gorm:"type:varchar(10);not null;index" json:"checkDate"` // YYYY-MM-DD (Tanggal Pencucian)
	BatchNo           string     `gorm:"type:varchar(100);index" json:"batchNo,omitempty"`
	ProductionOrderID *uuid.UUID `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`

	Location       string `gorm:"type:varchar(100);not null;default:'Gula Cair'" json:"location"`
	PICResponsible string `gorm:"type:varchar(150);not null;default:'QC (Andin)'" json:"picResponsible"`

	// Point 1: Pemasakan air di dalam tangki gula cair hingga suhu minimal 75°C
	WaterTempActual   float64 `gorm:"type:decimal(6,2);not null;default:0" json:"waterTempActual"`
	IsTempStandardMet bool    `gorm:"not null;default:true" json:"isTempStandardMet"`

	// Point 2: Sirkulasi air panas di pipa gula cair dan ditahan selama minimal 10 menit
	CirculationStartTime     string `gorm:"type:varchar(10);not null" json:"circulationStartTime"`
	CirculationEndTime       string `gorm:"type:varchar(10);not null" json:"circulationEndTime"`
	CirculationDurationMin   int    `gorm:"not null;default:0" json:"circulationDurationMin"`
	IsCirculationStandardMet bool   `gorm:"not null;default:true" json:"isCirculationStandardMet"`

	// Point 3: Pembilasan ke-2 hingga air bilasan berwarna bening, tidak berbau, dan tidak ada kotoran
	RinseColor         string `gorm:"type:varchar(30);not null;default:'Bening'" json:"rinseColor"`        // Bening, Keruh
	RinseAroma         string `gorm:"type:varchar(50);not null;default:'Normal'" json:"rinseAroma"`        // Normal, Aroma sabun, Aroma gula
	RinseDirt          string `gorm:"type:varchar(50);not null;default:'Bebas kotoran'" json:"rinseDirt"` // Bebas kotoran, Kotoran terdeteksi
	IsRinseStandardMet bool   `gorm:"not null;default:true" json:"isRinseStandardMet"`

	QCVerifierName string `gorm:"type:varchar(150);not null" json:"qcVerifierName"`              // Verifikasi QC
	QAVerifiedBy   string `gorm:"type:varchar(150)" json:"qaVerifiedBy,omitempty"`                 // Diketahui Oleh QA
	QAVerifiedDate string `gorm:"type:varchar(20)" json:"qaVerifiedDate,omitempty"`
	OverallStatus  string `gorm:"type:varchar(20);not null;default:'passed'" json:"overallStatus"` // passed, re_wash
	Notes          string `gorm:"type:text" json:"notes,omitempty"`
	_              *time.Time
}

func (LiquidSugarWashVerification) TableName() string {
	return "qc_liquid_sugar_wash_verifications"
}
