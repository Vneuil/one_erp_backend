package domain

import (
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Raw material warehouse stock entry statuses (Form Stok Gudang Bahan Baku — GDG-002-SBB).
const (
	RawStockStatusDraft      = "draft"
	RawStockStatusAvailable  = "available"
	RawStockStatusQuarantine = "quarantine"
	RawStockStatusDepleted   = "depleted"
	RawStockStatusRejected   = "rejected"
	RawStockStatusVoided     = "voided"
)

// Source types for a raw material stock entry.
const (
	RawStockSourceLBM              = "lbm"
	RawStockSourceOpeningBalance   = "opening_balance"
	RawStockSourceReturnProduction = "return_production"
	RawStockSourceAdjustment       = "adjustment"
)

// Movement log event types for per-lot warehouse stock card (Kartu Stok Lot).
const (
	RawStockEventPutaway           = "putaway"
	RawStockEventQCReject          = "qc_reject"
	RawStockEventQuarantineRelease = "quarantine_release"
	RawStockEventLocationUpdate    = "location_update"
	RawStockEventBPBReserve        = "bpb_reserve"
	RawStockEventBPBIssue          = "bpb_issue"
	RawStockEventProductionReturn  = "production_return"
	RawStockEventVoid              = "void"
)

// RawMaterialStockEntry records a raw material lot put-away into the raw material warehouse
// (Form GDG-002-SBB), linking LBM receiving and the 3 incoming QC forms (CAS, PBD, PBK)
// to physical bin/rack/pallet storage and running balances consumed by BPB & LBB.
type RawMaterialStockEntry struct {
	types.BaseEntity
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	EntryNo      string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"entryNo"`
	DocumentCode string     `gorm:"type:varchar(30);not null;default:'GDG-002-SBB'" json:"documentCode"`
	EntryDate    string     `gorm:"type:varchar(10);not null;index" json:"entryDate"` // YYYY-MM-DD
	SourceType   string     `gorm:"type:varchar(30);not null;default:'lbm';index" json:"sourceType"`

	// Block 1: Reference & Identity (auto-pulled from LBM / PO)
	GoodsReceiptID *uuid.UUID `gorm:"type:uuid;index" json:"goodsReceiptId,omitempty"`
	LBMNo          string     `gorm:"type:varchar(50);index" json:"lbmNo,omitempty"`
	PONo           string     `gorm:"type:varchar(50)" json:"poNo,omitempty"`
	DeliveryNoteNo string     `gorm:"type:varchar(100)" json:"deliveryNoteNo,omitempty"`
	SupplierID     *uuid.UUID `gorm:"type:uuid;index" json:"supplierId,omitempty"`
	SupplierName   string     `gorm:"type:varchar(255)" json:"supplierName,omitempty"`
	ProductID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"productId"`
	ProductSKU     string     `gorm:"type:varchar(50)" json:"productSku,omitempty"`
	ProductName    string     `gorm:"type:varchar(255);not null" json:"productName"`
	BatchNo        string     `gorm:"type:varchar(100);not null;index" json:"batchNo"`

	// Block 2: 3 QC Forms Verification Gate (CAS, PBD, PBK)
	CASCheckID       *uuid.UUID `gorm:"type:uuid;index" json:"casCheckId,omitempty"`
	CASStatus        string     `gorm:"type:varchar(30);default:'missing'" json:"casStatus"` // cleared, flagged, rejected, missing
	PBDCheckID       *uuid.UUID `gorm:"type:uuid;index" json:"pbdCheckId,omitempty"`
	PBDStatus        string     `gorm:"type:varchar(30);default:'missing'" json:"pbdStatus"` // passed, conditional, rejected, missing
	PBKCheckID       *uuid.UUID `gorm:"type:uuid;index" json:"pbkCheckId,omitempty"`
	PBKStatus        string     `gorm:"type:varchar(30);default:'missing'" json:"pbkStatus"` // passed, partial, failed, missing, not_applicable
	MoistureContent  *float64   `gorm:"type:decimal(6,2)" json:"moistureContent,omitempty"`
	BrixValue        *float64   `gorm:"type:decimal(6,2)" json:"brixValue,omitempty"`
	MeshSize         string     `gorm:"type:varchar(50)" json:"meshSize,omitempty"`
	QCRecommendation string     `gorm:"type:varchar(200)" json:"qcRecommendation,omitempty"`

	// Block 3: Certification & Quality Classification (Organic / Halal / Kosher Audit)
	OrganicStatus  string `gorm:"type:varchar(30);not null;default:'Organik'" json:"organicStatus"` // Organik, Konvensional, Transisi
	HalalStatus    string `gorm:"type:varchar(30);not null;default:'Halal'" json:"halalStatus"`     // Halal, Non-Halal
	KosherStatus   string `gorm:"type:varchar(30);not null;default:'Kosher'" json:"kosherStatus"`   // Kosher, Non-Kosher
	Certifications string `gorm:"type:text" json:"certifications,omitempty"`
	QualityGrade   string `gorm:"type:varchar(50);default:'Standar'" json:"qualityGrade"` // Grade A, Grade B, Standar, Reject

	// Block 4: Physical Quantities & Running Lot Balances (for BPB / LBB)
	UoM               string  `gorm:"type:varchar(20);not null;default:'KG'" json:"uom"`
	PackageUnit       string  `gorm:"type:varchar(30);not null;default:'Sak'" json:"packageUnit"` // Sak, Karung, Jerigen, Drum, Box, Pcs
	QtyLBM            float64 `gorm:"type:decimal(15,3);default:0" json:"qtyLbm"`
	QtyReceived       float64 `gorm:"type:decimal(15,3);not null;default:0" json:"qtyReceived"`
	QtyRejected       float64 `gorm:"type:decimal(15,3);default:0" json:"qtyRejected"`
	TotalPackages     int     `gorm:"not null;default:0" json:"totalPackages"`
	QtyReserved       float64 `gorm:"type:decimal(15,3);default:0" json:"qtyReserved"`
	QtyIssued         float64 `gorm:"type:decimal(15,3);default:0" json:"qtyIssued"`
	QtyRemaining      float64 `gorm:"type:decimal(15,3);not null;default:0" json:"qtyRemaining"`
	RemainingPackages int     `gorm:"not null;default:0" json:"remainingPackages"`

	// Block 5: Physical Storage Put-Away & FEFO
	WarehouseID      uuid.UUID `gorm:"type:uuid;not null;index" json:"warehouseId"`
	StorageZone      string    `gorm:"type:varchar(60);not null;default:'Zona Organik';index" json:"storageZone"`
	RackLocation     string    `gorm:"type:varchar(60)" json:"rackLocation,omitempty"`
	PalletNo         string    `gorm:"type:varchar(60)" json:"palletNo,omitempty"`
	ProductionDate   string    `gorm:"type:varchar(10)" json:"productionDate,omitempty"`       // YYYY-MM-DD
	ExpiryDate       string    `gorm:"type:varchar(10);index" json:"expiryDate,omitempty"`     // YYYY-MM-DD (FEFO)
	StorageCondition string    `gorm:"type:varchar(60);default:'Suhu Ruang Kering'" json:"storageCondition"`

	// Block 6: Status, Audit Trail & Signatures (PIC)
	Status          string     `gorm:"type:varchar(30);not null;default:'draft';index" json:"status"`
	HandedOverBy    string     `gorm:"type:varchar(150)" json:"handedOverBy,omitempty"`
	ReceivedBy      string     `gorm:"type:varchar(150)" json:"receivedBy,omitempty"`
	ApprovedBy      string     `gorm:"type:varchar(150)" json:"approvedBy,omitempty"`
	ApprovedAt      *time.Time `json:"approvedAt,omitempty"`
	StockSynced     bool       `gorm:"not null;default:false" json:"stockSynced"`
	StockAddedBySBB bool       `gorm:"not null;default:false" json:"stockAddedBySbb"` // true when SBB itself incremented inventory_stock_levels
	Notes           string     `gorm:"type:text" json:"notes,omitempty"`

	// Void audit trail
	VoidReason string     `gorm:"type:text" json:"voidReason,omitempty"`
	VoidedBy   string     `gorm:"type:varchar(150)" json:"voidedBy,omitempty"`
	VoidedAt   *time.Time `json:"voidedAt,omitempty"`

	Logs []RawMaterialStockLog `gorm:"foreignKey:EntryID" json:"logs,omitempty"`
}

func (RawMaterialStockEntry) TableName() string {
	return "inventory_raw_material_stock_entries"
}

// RawMaterialStockLog records every quantity or status change on a RawMaterialStockEntry
// (acting as the mini Kartu Stok per Lot).
type RawMaterialStockLog struct {
	types.BaseEntity
	EntryID       uuid.UUID `gorm:"type:uuid;not null;index" json:"entryId"`
	EventType     string    `gorm:"type:varchar(40);not null;index" json:"eventType"`
	RefDocumentNo string    `gorm:"type:varchar(80)" json:"refDocumentNo,omitempty"`
	QtyDelta      float64   `gorm:"type:decimal(15,3);default:0" json:"qtyDelta"`
	PackageDelta  int       `gorm:"default:0" json:"packageDelta"`
	BalanceAfter  float64   `gorm:"type:decimal(15,3);default:0" json:"balanceAfter"`
	PackagesAfter int       `gorm:"default:0" json:"packagesAfter"`
	ActorName     string    `gorm:"type:varchar(150)" json:"actorName,omitempty"`
	Notes         string    `gorm:"type:text" json:"notes,omitempty"`
	OccurredAt    time.Time `gorm:"not null" json:"occurredAt"`
}

func (RawMaterialStockLog) TableName() string {
	return "inventory_raw_material_stock_logs"
}

// RawMaterialStockFilter narrows raw material stock entry queries.
type RawMaterialStockFilter struct {
	Status        string
	WarehouseID   *uuid.UUID
	ProductID     *uuid.UUID
	StorageZone   string
	OrganicStatus string
	LBMNo         string
	GoodsReceiptID *uuid.UUID
	FromDate      string
	ToDate        string
	Search        string
}
