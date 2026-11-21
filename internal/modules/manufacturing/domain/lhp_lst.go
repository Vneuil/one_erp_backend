package domain

import (
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// ProductionOutputReport represents LHP (Laporan Hasil Produksi — PRO-002-LHP)
type ProductionOutputReport struct {
	types.BaseEntity
	ReportNo            string                       `gorm:"type:varchar(50);uniqueIndex;not null" json:"reportNo"`
	ProductionOrderID   *uuid.UUID                   `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`
	OrderNumber         string                       `gorm:"type:varchar(50)" json:"orderNumber,omitempty"`
	FairtradeStatus     string                       `gorm:"type:varchar(50);not null;default:'NON FAIRTRADE'" json:"fairtradeStatus"` // FAIRTRADE or NON FAIRTRADE
	Certifications      string                       `gorm:"type:varchar(255);default:'NATURAL'" json:"certifications"`                // NATURAL, ORGANIC EU, ORGANIC USDA / COR-NOP, ORGANIC JAS
	PackingDate         string                       `gorm:"type:varchar(10);not null;index" json:"packingDate"`
	PackingStartTime    string                       `gorm:"type:varchar(20)" json:"packingStartTime,omitempty"`
	PackingEndTime      string                       `gorm:"type:varchar(20)" json:"packingEndTime,omitempty"`
	EvaluationNotes     string                       `gorm:"type:text" json:"evaluationNotes,omitempty"`
	ProductionPIC       string                       `gorm:"type:varchar(150);not null" json:"productionPic"`
	ProductionSignDate  *string                      `gorm:"type:varchar(20)" json:"productionSignDate,omitempty"`
	AdminPIC            string                       `gorm:"type:varchar(150);not null" json:"adminPic"`
	AdminSignDate       *string                      `gorm:"type:varchar(20)" json:"adminSignDate,omitempty"`
	Status              string                       `gorm:"type:varchar(30);not null;default:'draft'" json:"status"` // draft, completed, transferred
	TotalWeightKg       float64                      `gorm:"type:decimal(15,3);default:0" json:"totalWeightKg"`
	TotalQuantityPcs    int                          `gorm:"default:0" json:"totalQuantityPcs"`
	Items               []ProductionOutputReportItem `gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

func (ProductionOutputReport) TableName() string { return "manufacturing_production_output_reports" }

// ProductionOutputReportItem represents one row in the LHP table
type ProductionOutputReportItem struct {
	types.BaseEntity
	ReportID               uuid.UUID `gorm:"type:uuid;not null;index" json:"reportId"`
	ItemNo                 int       `gorm:"not null" json:"itemNo"`
	RawMaterialCode        string    `gorm:"type:varchar(100);not null" json:"rawMaterialCode"`
	ProductName            string    `gorm:"type:varchar(255);not null" json:"productName"`
	BatchNo                string    `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	UnitPcs                int       `gorm:"not null;default:0" json:"unitPcs"`
	PrimaryPackagingType   string    `gorm:"type:varchar(100)" json:"primaryPackagingType,omitempty"`
	PrimaryPackagingCode   string    `gorm:"type:varchar(100)" json:"primaryPackagingCode,omitempty"`
	SecondaryPackagingType string    `gorm:"type:varchar(100)" json:"secondaryPackagingType,omitempty"`
	SecondaryPackagingCode string    `gorm:"type:varchar(100)" json:"secondaryPackagingCode,omitempty"`
	WeightPerPackKg        float64   `gorm:"type:decimal(10,3);not null;default:0" json:"weightPerPackKg"`
	TotalWeightKg          float64   `gorm:"type:decimal(15,3);not null;default:0" json:"totalWeightKg"`
	StickerNo              string    `gorm:"type:varchar(100)" json:"stickerNo,omitempty"`
	RemainingStockKg       float64   `gorm:"type:decimal(12,3);default:0" json:"remainingStockKg"`
	BuyerName              string    `gorm:"type:varchar(255)" json:"buyerName,omitempty"`
	Notes                  string    `gorm:"type:varchar(255)" json:"notes,omitempty"`
}

func (ProductionOutputReportItem) TableName() string { return "manufacturing_production_output_report_items" }

// HandoverReport represents LST (Laporan Serah Terima — GDG-009-LST)
type HandoverReport struct {
	types.BaseEntity
	ReportNo            string               `gorm:"type:varchar(50);uniqueIndex;not null" json:"reportNo"`
	LHPID               *uuid.UUID           `gorm:"type:uuid;index" json:"lhpId,omitempty"`
	LHPReportNo         string               `gorm:"type:varchar(50)" json:"lhpReportNo,omitempty"`
	FairtradeStatus     string               `gorm:"type:varchar(50);not null;default:'NON FAIRTRADE'" json:"fairtradeStatus"` // FAIRTRADE or NON FAIRTRADE
	FromDepartment      string               `gorm:"type:varchar(100);not null;default:'Produksi'" json:"fromDepartment"`
	ToDepartment        string               `gorm:"type:varchar(100);not null;default:'Gudang'" json:"toDepartment"`
	HandoverDate        string               `gorm:"type:varchar(10);not null;index" json:"handoverDate"`
	Notes               string               `gorm:"type:text" json:"notes,omitempty"`
	HandedOverBy        string               `gorm:"type:varchar(150);not null" json:"handedOverBy"` // Produksi
	HandedOverSignDate  *string              `gorm:"type:varchar(20)" json:"handedOverSignDate,omitempty"`
	VerifiedByQC        string               `gorm:"type:varchar(150)" json:"verifiedByQc,omitempty"` // QC
	VerifiedSignDate    *string              `gorm:"type:varchar(20)" json:"verifiedSignDate,omitempty"`
	ReceivedByWarehouse string               `gorm:"type:varchar(150)" json:"receivedByWarehouse,omitempty"` // Gudang
	ReceivedSignDate    *string              `gorm:"type:varchar(20)" json:"receivedSignDate,omitempty"`
	Status              string               `gorm:"type:varchar(30);not null;default:'draft'" json:"status"` // draft, verified_qc, accepted_warehouse, rejected
	TotalQuantityPcs    int                  `gorm:"default:0" json:"totalQuantityPcs"`
	TotalWeightKg       float64              `gorm:"type:decimal(15,3);default:0" json:"totalWeightKg"`
	Items               []HandoverReportItem `gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

func (HandoverReport) TableName() string { return "manufacturing_handover_reports" }

// HandoverReportItem represents one item line in LST
type HandoverReportItem struct {
	types.BaseEntity
	ReportID     uuid.UUID `gorm:"type:uuid;not null;index" json:"reportId"`
	ItemNo       int       `gorm:"not null" json:"itemNo"`
	ItemName     string    `gorm:"type:varchar(255);not null" json:"itemName"`
	ItemCode     string    `gorm:"type:varchar(100);not null" json:"itemCode"`
	BatchNo      string    `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	UnitKg       float64   `gorm:"type:decimal(12,3);not null;default:0" json:"unitKg"`
	Packaging    string    `gorm:"type:varchar(100);not null" json:"packaging"`
	BuyerName    string    `gorm:"type:varchar(255)" json:"buyerName,omitempty"`
	QuantityPcs  int       `gorm:"not null;default:0" json:"quantityPcs"`
	Notes        string    `gorm:"type:varchar(255)" json:"notes,omitempty"`
}

func (HandoverReportItem) TableName() string { return "manufacturing_handover_report_items" }
