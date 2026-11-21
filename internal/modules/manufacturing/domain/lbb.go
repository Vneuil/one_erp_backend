package domain

import (
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// RawMaterialRequestReport represents Laporan Permintaan Bahan Baku (LBB — GDG-013-LBB)
// matching the 6-column x 50-row (up to 300 bags/jerrycans) template for GSKN (kg) and VCO (L).
type RawMaterialRequestReport struct {
	types.BaseEntity
	TenantID          *uuid.UUID                   `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ReportNo          string                       `gorm:"type:varchar(50);uniqueIndex;not null" json:"reportNo"` // LBB-YYYYMM-NNNN
	DocumentCode      string                       `gorm:"type:varchar(30);not null;default:'GDG-013-LBB'" json:"documentCode"`
	RequisitionID     *uuid.UUID                   `gorm:"type:uuid;index" json:"requisitionId,omitempty"`     // Link ke BPB
	RequisitionLineID *uuid.UUID                   `gorm:"type:uuid;index" json:"requisitionLineId,omitempty"` // Link ke baris BPB
	BPBNo             string                       `gorm:"type:varchar(50);index" json:"bpbNo,omitempty"`
	RawMaterialStockID *uuid.UUID                  `gorm:"type:uuid;index" json:"rawMaterialStockId,omitempty"` // Link ke Lot SBB
	SBBEntryNo        string                       `gorm:"type:varchar(50)" json:"sbbEntryNo,omitempty"`
	ProductionOrderID *uuid.UUID                   `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`
	OrderNumber       string                       `gorm:"type:varchar(50)" json:"orderNumber,omitempty"`
	ReportDate        string                       `gorm:"type:varchar(10);not null;index" json:"reportDate"` // YYYY-MM-DD
	VariantType       string                       `gorm:"type:varchar(20);not null;default:'GSKN'" json:"variantType"` // GSKN (kg) | VCO (L)
	FairtradeStatus   string                       `gorm:"type:varchar(30);default:'Fairtrade'" json:"fairtradeStatus"` // Fairtrade | Non-Fairtrade
	ProductID         uuid.UUID                    `gorm:"type:uuid;not null;index" json:"productId"`
	MaterialCode      string                       `gorm:"type:varchar(100);not null;index" json:"materialCode"` // Kode Bahan / BatchNo
	ProductName       string                       `gorm:"type:varchar(255);not null" json:"productName"`
	SupplierName      string                       `gorm:"type:varchar(255)" json:"supplierName,omitempty"`
	CustomerName      string                       `gorm:"type:varchar(255)" json:"customerName,omitempty"`
	MachineNo         string                       `gorm:"type:varchar(50)" json:"machineNo,omitempty"`
	MeshSize          string                       `gorm:"type:varchar(50)" json:"meshSize,omitempty"`
	UoM               string                       `gorm:"type:varchar(20);not null;default:'KG'" json:"uom"` // KG (GSKN) | L (VCO)
	PlasticTarePerBag float64                      `gorm:"type:decimal(10,3);default:0" json:"plasticTarePerBag"`
	TotalBags         int                          `gorm:"not null;default:0" json:"totalBags"`
	TotalGrossWeight  float64                      `gorm:"type:decimal(15,3);not null;default:0" json:"totalGrossWeight"`
	TotalPlasticTare  float64                      `gorm:"type:decimal(15,3);not null;default:0" json:"totalPlasticTare"`
	TotalNetWeight    float64                      `gorm:"type:decimal(15,3);not null;default:0" json:"totalNetWeight"`
	OperatorRole      string                       `gorm:"type:varchar(50);default:'Operator Ayak'" json:"operatorRole"` // Operator Ayak (GSKN) | Operator Filling (VCO)
	PreparedBy        string                       `gorm:"type:varchar(150)" json:"preparedBy,omitempty"`                // Dibuat Oleh (Operator Ayak / Filling)
	PreparedDate      string                       `gorm:"type:varchar(10)" json:"preparedDate,omitempty"`
	AcknowledgedBy    string                       `gorm:"type:varchar(150)" json:"acknowledgedBy,omitempty"`            // Diketahui Oleh (Admin Produksi)
	AcknowledgedDate  string                       `gorm:"type:varchar(10)" json:"acknowledgedDate,omitempty"`
	Status            string                       `gorm:"type:varchar(30);not null;default:'draft';index" json:"status"` // draft, submitted, approved
	Notes             string                       `gorm:"type:text" json:"notes,omitempty"`
	BagEntries        []RawMaterialRequestBagEntry `gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE" json:"bagEntries,omitempty"`
}

func (RawMaterialRequestReport) TableName() string {
	return "manufacturing_raw_material_request_reports"
}

// RawMaterialRequestBagEntry stores one row (1..300) in the 6-column x 50-row LBB weighing grid.
type RawMaterialRequestBagEntry struct {
	types.BaseEntity
	ReportID    uuid.UUID `gorm:"type:uuid;not null;index" json:"reportId"`
	RowNumber   int       `gorm:"not null" json:"rowNumber"`                       // 1..300
	ColumnGroup int       `gorm:"not null;default:1" json:"columnGroup"`           // 1..6 (each group holds 50 rows)
	LotCode     string    `gorm:"type:varchar(60)" json:"lotCode,omitempty"`       // Header "Kode :" pada grup kolom tersebut
	WeightValue *float64  `gorm:"type:decimal(10,3)" json:"weightValue,omitempty"` // Berat (kg) atau Berat (L)
}

func (RawMaterialRequestBagEntry) TableName() string {
	return "manufacturing_raw_material_request_bag_entries"
}
