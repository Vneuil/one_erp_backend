package domain

import (
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Production Flow Stages where QC can fail
const (
	StageIncomingRawMaterial = "incoming_raw_material" // QC-002-PBD (Pemeriksaan Kualitas Barang Datang)
	StageIncomingPackaging   = "incoming_packaging"   // QC-003-PBK (Pemeriksaan Bahan Kemas)
	StageSieving             = "sieving"              // QC-009-MNA (Monitoring Proses Ayak)
	StageMagnetTrap          = "magnet_trap"          // QC-038-CMT (Catatan Pemakaian Magnet Trap)
	StageMeshBolt            = "mesh_bolt"            // QC-010-MNM (Monitoring Baut & Mur Ayakan)
	StageOven                = "oven"                 // QC-012-MNO (Monitoring Oven)
	StageMetalDetector       = "metal_detector"       // QC-017-VMD & PRO-007-CPP (Metal Detector CCP)
	StageFinishedGoods       = "finished_goods"       // QC-004-PJD (Pemeriksaan Produk Jadi)
	StagePackagingLabel      = "label_packaging"      // QC-016-PLK (Pemeriksaan Label Kemasan)
	StageBrontol             = "brontol"              // Mesin Brontol (MBB & MPB)
	StageLiquidSugar         = "liquid_sugar"          // Gula Cair (MNP & MNB)
	StageProductionStep      = "production_step"      // Step QC Order Produksi
	StageManual              = "manual"               // Pencatatan Manual di Lantai Produksi
)

// Action dispositions for rejected / brontol material
const (
	ActionReturnToSupplier = "return_to_supplier" // Retur ke Supplier (memicu Laporan Retur)
	ActionRework           = "rework"             // Masak Ulang / Reproses
	ActionReOven           = "re_oven"            // Masuk Oven Ulang
	ActionDisposal         = "disposal"           // Pemusnahan / Afkir / Waste
	ActionHold             = "hold"               // Karantina / Tahan Evaluasi
)

// BrontolReport represents Laporan Brontol / Kartu Stok Brontol (Doc Code: REP-013-KSB)
// Generated automatically when any QC inspection in the production flow fails or rejects material,
// or entered manually from production machines (Ayak, Oven, etc.).
type BrontolReport struct {
	types.BaseEntity
	CompanyID          *uuid.UUID          `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID           *uuid.UUID          `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ReportNo           string              `gorm:"type:varchar(50);uniqueIndex;not null" json:"reportNo"`
	DocumentCode       string              `gorm:"type:varchar(30);default:'REP-013-KSB'" json:"documentCode"`
	ReportDate         string              `gorm:"type:varchar(10);not null;index" json:"reportDate"` // YYYY-MM-DD
	ReportMonth        string              `gorm:"type:varchar(7);not null;index" json:"reportMonth"` // YYYY-MM
	SourceStage        string              `gorm:"type:varchar(50);not null;index" json:"sourceStage"`
	SourceInspectionID *uuid.UUID          `gorm:"type:uuid;index" json:"sourceInspectionId,omitempty"`
	SourceInspectionNo string              `gorm:"type:varchar(100);index" json:"sourceInspectionNo,omitempty"`
	ProductionOrderID  *uuid.UUID          `gorm:"type:uuid;index" json:"productionOrderId,omitempty"`
	OrderNumber        string              `gorm:"type:varchar(50)" json:"orderNumber,omitempty"`
	ProductID          *uuid.UUID          `gorm:"type:uuid;index" json:"productId,omitempty"`
	ProductName        string              `gorm:"type:varchar(255);not null" json:"productName"`
	ProductCode        string              `gorm:"type:varchar(100)" json:"productCode,omitempty"`
	BatchNo            string              `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	MaterialType       string              `gorm:"type:varchar(20);default:'BM'" json:"materialType"` // BM (Bahan Mentah), BO (Bahan Oven), BK (Bahan Kemas), FG (Produk Jadi), WIP
	SupplierID         *uuid.UUID          `gorm:"type:uuid;index" json:"supplierId,omitempty"`
	SupplierName       string              `gorm:"type:varchar(255)" json:"supplierName,omitempty"`
	MachineNo          string              `gorm:"type:varchar(50)" json:"machineNo,omitempty"`
	MeshSize           string              `gorm:"type:varchar(50)" json:"meshSize,omitempty"`
	Quantity           float64             `gorm:"type:decimal(12,3);not null;default:0" json:"quantity"`
	Unit               string              `gorm:"type:varchar(20);default:'KG'" json:"unit"`
	DefectReason       string              `gorm:"type:text;not null" json:"defectReason"`
	ActionDisposition  string              `gorm:"type:varchar(50);not null;default:'hold'" json:"actionDisposition"`
	OperatorName       string              `gorm:"type:varchar(150)" json:"operatorName,omitempty"`
	QCInspectorName    string              `gorm:"type:varchar(150)" json:"qcInspectorName,omitempty"`
	Status             string              `gorm:"type:varchar(30);not null;default:'recorded'" json:"status"` // recorded, returned, reprocessed, disposed, closed
	Notes              string              `gorm:"type:text" json:"notes,omitempty"`

	// Retur linkage
	IsReturn     bool                `gorm:"default:false;not null" json:"isReturn"`
	ReturnReport *RejectReturnReport `gorm:"foreignKey:BrontolReportID;constraint:OnDelete:CASCADE" json:"returnReport,omitempty"`
}

func (BrontolReport) TableName() string {
	return "qc_brontol_reports"
}

// RejectReturnReport represents Laporan Retur Barang Reject ke Supplier
// Generated whenever material is rejected and marked to be returned to supplier (action = return_to_supplier).
type RejectReturnReport struct {
	types.BaseEntity
	CompanyID          *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID           *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	BrontolReportID    uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"brontolReportId"`
	ReturnNo           string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"returnNo"`
	DocumentCode       string     `gorm:"type:varchar(30);default:'RET-REJECT-01'" json:"documentCode"`
	ReturnDate         string     `gorm:"type:varchar(10);not null;index" json:"returnDate"` // YYYY-MM-DD
	SupplierID         *uuid.UUID `gorm:"type:uuid;index" json:"supplierId,omitempty"`
	SupplierName       string     `gorm:"type:varchar(255);not null" json:"supplierName"`
	PONo               string     `gorm:"type:varchar(100)" json:"poNo,omitempty"`
	GoodsReceiptNo     string     `gorm:"type:varchar(100)" json:"goodsReceiptNo,omitempty"`
	DeliveryNoteNo     string     `gorm:"type:varchar(100)" json:"deliveryNoteNo,omitempty"`
	ProductName        string     `gorm:"type:varchar(255);not null" json:"productName"`
	ProductCode        string     `gorm:"type:varchar(100)" json:"productCode,omitempty"`
	BatchNo            string     `gorm:"type:varchar(100);not null;index" json:"batchNo"`
	ReturnQuantity     float64    `gorm:"type:decimal(12,3);not null" json:"returnQuantity"`
	Unit               string     `gorm:"type:varchar(20);default:'KG'" json:"unit"`
	UnitPrice          float64    `gorm:"type:decimal(15,2);default:0" json:"unitPrice"`
	EstimatedValue     float64    `gorm:"type:decimal(15,2);default:0" json:"estimatedValue"`
	ReturnReason       string     `gorm:"type:text;not null" json:"returnReason"`
	CourierName        string     `gorm:"type:varchar(150)" json:"courierName,omitempty"`
	VehicleNumber      string     `gorm:"type:varchar(50)" json:"vehicleNumber,omitempty"`
	DriverName         string     `gorm:"type:varchar(150)" json:"driverName,omitempty"`
	PreparedBy         string     `gorm:"type:varchar(150);not null" json:"preparedBy"`
	ApprovedBy         string     `gorm:"type:varchar(150)" json:"approvedBy,omitempty"`
	ReceivedBySupplier string     `gorm:"type:varchar(150)" json:"receivedBySupplier,omitempty"`
	ShippedDate        *time.Time `json:"shippedDate,omitempty"`
	CompletedDate      *time.Time `json:"completedDate,omitempty"`
	Status             string     `gorm:"type:varchar(30);not null;default:'pending_pickup'" json:"status"` // pending_pickup, shipped, completed, cancelled
	Notes              string     `gorm:"type:text" json:"notes,omitempty"`
}

func (RejectReturnReport) TableName() string {
	return "qc_reject_return_reports"
}
