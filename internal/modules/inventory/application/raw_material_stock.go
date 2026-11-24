package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/divinecoid/one-backend/internal/shared/docseq"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// InboundReader reads LBM (GoodsReceipt) and the 3 incoming QC forms (CAS, PBD, PBK)
// directly from the tenant DB without introducing circular module imports.
type InboundReader interface {
	ReadLBMContext(ctx context.Context, goodsReceiptID uuid.UUID) (*LBMInboundRaw, error)
}

// Raw read structs populated by infrastructure.InboundReader.

type LBMInboundRaw struct {
	GoodsReceiptID   uuid.UUID
	LBMNo            string
	PONo             string
	ReceivedDate     string
	WarehouseID      *uuid.UUID
	ReceivedBy       string
	Status           string
	DeliveryNoteNo   string
	SupplierID       *uuid.UUID
	SupplierName     string
	SugarCondition   string
	PackageCondition string
	Notes            string
	Lines            []LBMInboundLineRaw
	CAS              *QCCASSummaryDTO
	PBD              *QCPBDSummaryDTO
	PBK              *QCPBKSummaryDTO
}

type LBMInboundLineRaw struct {
	LineID         uuid.UUID
	ProductID      uuid.UUID
	ProductSKU     string
	ProductName    string
	UoM            string
	OrganicType    string
	Certifications string
	LotNumber      string
	WeightGross    *float64
	WeightTare     *float64
	WeightNet      float64
	PackageSize    float64
	BagCount       int
}

// DTOs for LBM + 3 QC forms context

type QCCASSummaryDTO struct {
	ID            uuid.UUID `json:"id"`
	CheckDate     string    `json:"checkDate"`
	DriverName    string    `json:"driverName"`
	LicensePlate  string    `json:"licensePlate,omitempty"`
	VehicleType   string    `json:"vehicleType,omitempty"`
	OverallStatus string    `json:"overallStatus"` // cleared, flagged, rejected
	QCName        string    `json:"qcName,omitempty"`
	QAName        string    `json:"qaName,omitempty"`
}

type QCPBDSummaryDTO struct {
	ID                 uuid.UUID `json:"id"`
	CheckDate          string    `json:"checkDate"`
	ProductName        string    `json:"productName"`
	BatchNo            string    `json:"batchNo,omitempty"`
	TotalBags          int       `json:"totalBags"`
	NetWeight          float64   `json:"netWeight"`
	AcceptedWeight     float64   `json:"acceptedWeight"`
	RejectedWeight     float64   `json:"rejectedWeight"`
	SugarCondition     string    `json:"sugarCondition,omitempty"`
	PackagingCondition string    `json:"packagingCondition,omitempty"`
	MoistureContent    *float64  `json:"moistureContent,omitempty"`
	BrixValue          *float64  `json:"brixValue,omitempty"`
	MeshSize           string    `json:"meshSize,omitempty"`
	RecommendedAction  string    `json:"recommendedAction,omitempty"`
	OverallStatus      string    `json:"overallStatus"` // passed, conditional, rejected
	QCName             string    `json:"qcName,omitempty"`
	QAName             string    `json:"qaName,omitempty"`
}

type QCPBKSummaryDTO struct {
	ID            uuid.UUID `json:"id"`
	CheckDate     string    `json:"checkDate"`
	ItemName      string    `json:"itemName"`
	MaterialType  string    `json:"materialType"`
	TotalQty      int       `json:"totalQty"`
	PassedQty     int       `json:"passedQty"`
	FailedQty     int       `json:"failedQty"`
	OverallStatus string    `json:"overallStatus"` // passed, partial, failed
	QCName        string    `json:"qcName,omitempty"`
	QAName        string    `json:"qaName,omitempty"`
}

type LBMInboundLineDTO struct {
	LineID         uuid.UUID `json:"lineId"`
	ProductID      uuid.UUID `json:"productId"`
	ProductSKU     string    `json:"productSku"`
	ProductName    string    `json:"productName"`
	UoM            string    `json:"uom"`
	OrganicType    string    `json:"organicType,omitempty"`
	Certifications string    `json:"certifications,omitempty"`
	LotNumber      string    `json:"lotNumber"`
	WeightGross    *float64  `json:"weightGross,omitempty"`
	WeightTare     *float64  `json:"weightTare,omitempty"`
	WeightNet      float64   `json:"weightNet"`
	PackageSize    float64   `json:"packageSize"`
	BagCount       int       `json:"bagCount"`
}

type LBMInboundContextDTO struct {
	GoodsReceiptID    uuid.UUID                      `json:"goodsReceiptId"`
	LBMNo             string                         `json:"lbmNo"`
	PONo              string                         `json:"poNo,omitempty"`
	ReceivedDate      string                         `json:"receivedDate"`
	WarehouseID       *uuid.UUID                     `json:"warehouseId,omitempty"`
	WarehouseName     string                         `json:"warehouseName,omitempty"`
	ReceivedBy        string                         `json:"receivedBy,omitempty"`
	Status            string                         `json:"status"`
	DeliveryNoteNo    string                         `json:"deliveryNoteNo,omitempty"`
	SupplierID        *uuid.UUID                     `json:"supplierId,omitempty"`
	SupplierName      string                         `json:"supplierName,omitempty"`
	SugarCondition    string                         `json:"sugarCondition,omitempty"`
	PackageCondition  string                         `json:"packageCondition,omitempty"`
	Notes             string                         `json:"notes,omitempty"`
	Lines             []LBMInboundLineDTO            `json:"lines"`
	CAS               *QCCASSummaryDTO               `json:"cas,omitempty"`
	PBD               *QCPBDSummaryDTO               `json:"pbd,omitempty"`
	PBK               *QCPBKSummaryDTO               `json:"pbk,omitempty"`
	CASStatus         string                         `json:"casStatus"`
	PBDStatus         string                         `json:"pbdStatus"`
	PBKStatus         string                         `json:"pbkStatus"`
	QCGateReady       bool                           `json:"qcGateReady"`
	RecommendedStatus string                         `json:"recommendedStatus"` // available, quarantine, rejected
	ExistingEntries   []RawMaterialStockEntrySummary `json:"existingEntries"`
}

type RawMaterialStockEntrySummary struct {
	ID          uuid.UUID `json:"id"`
	EntryNo     string    `json:"entryNo"`
	ProductID   uuid.UUID `json:"productId"`
	ProductName string    `json:"productName"`
	BatchNo     string    `json:"batchNo"`
	QtyReceived float64   `json:"qtyReceived"`
	Status      string    `json:"status"`
}

// Request / Response DTOs for RawMaterialStockEntry (GDG-002-SBB)

type CreateRawMaterialStockDTO struct {
	EntryDate      string     `json:"entryDate"`
	SourceType     string     `json:"sourceType"` // lbm, opening_balance, return_production, adjustment
	GoodsReceiptID *uuid.UUID `json:"goodsReceiptId,omitempty"`
	LBMNo          string     `json:"lbmNo,omitempty"`
	PONo           string     `json:"poNo,omitempty"`
	DeliveryNoteNo string     `json:"deliveryNoteNo,omitempty"`
	SupplierID     *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName   string     `json:"supplierName,omitempty"`
	ProductID      uuid.UUID  `json:"productId"`
	ProductSKU     string     `json:"productSku,omitempty"`
	ProductName    string     `json:"productName,omitempty"`
	BatchNo        string     `json:"batchNo"`

	// QC Gate overrides/snapshots (auto-pulled when goodsReceiptId is provided)
	CASCheckID       *uuid.UUID `json:"casCheckId,omitempty"`
	CASStatus        string     `json:"casStatus,omitempty"`
	PBDCheckID       *uuid.UUID `json:"pbdCheckId,omitempty"`
	PBDStatus        string     `json:"pbdStatus,omitempty"`
	PBKCheckID       *uuid.UUID `json:"pbkCheckId,omitempty"`
	PBKStatus        string     `json:"pbkStatus,omitempty"`
	MoistureContent  *float64   `json:"moistureContent,omitempty"`
	BrixValue        *float64   `json:"brixValue,omitempty"`
	MeshSize         string     `json:"meshSize,omitempty"`
	QCRecommendation string     `json:"qcRecommendation,omitempty"`

	// Certification & Quality
	OrganicStatus  string `json:"organicStatus,omitempty"`
	HalalStatus    string `json:"halalStatus,omitempty"`
	KosherStatus   string `json:"kosherStatus,omitempty"`
	Certifications string `json:"certifications,omitempty"`
	QualityGrade   string `json:"qualityGrade,omitempty"`

	// Physical Quantities
	UoM           string  `json:"uom,omitempty"`
	PackageUnit   string  `json:"packageUnit,omitempty"`
	QtyLBM        float64 `json:"qtyLbm"`
	QtyReceived   float64 `json:"qtyReceived"`
	QtyRejected   float64 `json:"qtyRejected"`
	TotalPackages int     `json:"totalPackages"`

	// Storage Put-Away & FEFO
	WarehouseID      uuid.UUID `json:"warehouseId"`
	StorageZone      string    `json:"storageZone,omitempty"`
	RackLocation     string    `json:"rackLocation,omitempty"`
	PalletNo         string    `json:"palletNo,omitempty"`
	ProductionDate   string    `json:"productionDate,omitempty"`
	ExpiryDate       string    `json:"expiryDate,omitempty"`
	StorageCondition string    `json:"storageCondition,omitempty"`

	// PIC & Posting
	HandedOverBy    string `json:"handedOverBy,omitempty"`
	ReceivedBy      string `json:"receivedBy,omitempty"`
	Notes           string `json:"notes,omitempty"`
	PostImmediately bool   `json:"postImmediately"`
	TargetStatus    string `json:"targetStatus,omitempty"` // available, quarantine, rejected (used when PostImmediately=true)
}

type UpdateRawMaterialStockDTO struct {
	EntryDate        string   `json:"entryDate,omitempty"`
	BatchNo          string   `json:"batchNo,omitempty"`
	PBKStatus        string   `json:"pbkStatus,omitempty"`
	MoistureContent  *float64 `json:"moistureContent,omitempty"`
	BrixValue        *float64 `json:"brixValue,omitempty"`
	MeshSize         string   `json:"meshSize,omitempty"`
	QCRecommendation string   `json:"qcRecommendation,omitempty"`
	OrganicStatus    string   `json:"organicStatus,omitempty"`
	HalalStatus      string   `json:"halalStatus,omitempty"`
	KosherStatus     string   `json:"kosherStatus,omitempty"`
	Certifications   string   `json:"certifications,omitempty"`
	QualityGrade     string   `json:"qualityGrade,omitempty"`
	UoM              string   `json:"uom,omitempty"`
	PackageUnit      string   `json:"packageUnit,omitempty"`
	QtyLBM           *float64 `json:"qtyLbm,omitempty"`
	QtyReceived      *float64 `json:"qtyReceived,omitempty"`
	QtyRejected      *float64 `json:"qtyRejected,omitempty"`
	TotalPackages    *int     `json:"totalPackages,omitempty"`
	WarehouseID      *uuid.UUID `json:"warehouseId,omitempty"`
	StorageZone      string   `json:"storageZone,omitempty"`
	RackLocation     string   `json:"rackLocation,omitempty"`
	PalletNo         string   `json:"palletNo,omitempty"`
	ProductionDate   string   `json:"productionDate,omitempty"`
	ExpiryDate       string   `json:"expiryDate,omitempty"`
	StorageCondition string   `json:"storageCondition,omitempty"`
	HandedOverBy     string   `json:"handedOverBy,omitempty"`
	ReceivedBy       string   `json:"receivedBy,omitempty"`
	Notes            string   `json:"notes,omitempty"`
}

type PostRawMaterialStockDTO struct {
	TargetStatus string `json:"targetStatus,omitempty"` // available, quarantine, rejected
	ApprovedBy   string `json:"approvedBy,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type ReleaseQuarantineDTO struct {
	Decision   string `json:"decision"` // available | rejected
	ApprovedBy string `json:"approvedBy,omitempty"`
	Notes      string `json:"notes"`
}

type RawMaterialStockLogDTO struct {
	ID            uuid.UUID `json:"id"`
	EntryID       uuid.UUID `json:"entryId"`
	EventType     string    `json:"eventType"`
	RefDocumentNo string    `json:"refDocumentNo,omitempty"`
	QtyDelta      float64   `json:"qtyDelta"`
	PackageDelta  int       `json:"packageDelta"`
	BalanceAfter  float64   `json:"balanceAfter"`
	PackagesAfter int       `json:"packagesAfter"`
	ActorName     string    `json:"actorName,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	OccurredAt    time.Time `json:"occurredAt"`
}

type RawMaterialStockResponseDTO struct {
	ID                uuid.UUID                `json:"id"`
	EntryNo           string                   `json:"entryNo"`
	DocumentCode      string                   `json:"documentCode"`
	EntryDate         string                   `json:"entryDate"`
	SourceType        string                   `json:"sourceType"`
	GoodsReceiptID    *uuid.UUID               `json:"goodsReceiptId,omitempty"`
	LBMNo             string                   `json:"lbmNo,omitempty"`
	PONo              string                   `json:"poNo,omitempty"`
	DeliveryNoteNo    string                   `json:"deliveryNoteNo,omitempty"`
	SupplierID        *uuid.UUID               `json:"supplierId,omitempty"`
	SupplierName      string                   `json:"supplierName,omitempty"`
	ProductID         uuid.UUID                `json:"productId"`
	ProductSKU        string                   `json:"productSku"`
	ProductName       string                   `json:"productName"`
	BatchNo           string                   `json:"batchNo"`
	CASCheckID        *uuid.UUID               `json:"casCheckId,omitempty"`
	CASStatus         string                   `json:"casStatus"`
	PBDCheckID        *uuid.UUID               `json:"pbdCheckId,omitempty"`
	PBDStatus         string                   `json:"pbdStatus"`
	PBKCheckID        *uuid.UUID               `json:"pbkCheckId,omitempty"`
	PBKStatus         string                   `json:"pbkStatus"`
	MoistureContent   *float64                 `json:"moistureContent,omitempty"`
	BrixValue         *float64                 `json:"brixValue,omitempty"`
	MeshSize          string                   `json:"meshSize,omitempty"`
	QCRecommendation  string                   `json:"qcRecommendation,omitempty"`
	OrganicStatus     string                   `json:"organicStatus"`
	HalalStatus       string                   `json:"halalStatus"`
	KosherStatus      string                   `json:"kosherStatus"`
	Certifications    string                   `json:"certifications,omitempty"`
	QualityGrade      string                   `json:"qualityGrade"`
	UoM               string                   `json:"uom"`
	PackageUnit       string                   `json:"packageUnit"`
	QtyLBM            float64                  `json:"qtyLbm"`
	QtyReceived       float64                  `json:"qtyReceived"`
	QtyRejected       float64                  `json:"qtyRejected"`
	TotalPackages     int                      `json:"totalPackages"`
	AvgWeightPerPkg   float64                  `json:"avgWeightPerPackage"`
	QtyReserved       float64                  `json:"qtyReserved"`
	QtyIssued         float64                  `json:"qtyIssued"`
	QtyRemaining      float64                  `json:"qtyRemaining"`
	RemainingPackages int                      `json:"remainingPackages"`
	WarehouseID       uuid.UUID                `json:"warehouseId"`
	WarehouseName     string                   `json:"warehouseName"`
	StorageZone       string                   `json:"storageZone"`
	RackLocation      string                   `json:"rackLocation,omitempty"`
	PalletNo          string                   `json:"palletNo,omitempty"`
	ProductionDate    string                   `json:"productionDate,omitempty"`
	ExpiryDate        string                   `json:"expiryDate,omitempty"`
	ExpiryStatus      string                   `json:"expiryStatus"`
	DaysToExpiry      *int                     `json:"daysToExpiry,omitempty"`
	StorageCondition  string                   `json:"storageCondition"`
	Status            string                   `json:"status"`
	HandedOverBy      string                   `json:"handedOverBy,omitempty"`
	ReceivedBy        string                   `json:"receivedBy,omitempty"`
	ApprovedBy        string                   `json:"approvedBy,omitempty"`
	ApprovedAt        *time.Time               `json:"approvedAt,omitempty"`
	StockSynced       bool                     `json:"stockSynced"`
	StockAddedBySBB   bool                     `json:"stockAddedBySbb"`
	Notes             string                   `json:"notes,omitempty"`
	VoidReason        string                   `json:"voidReason,omitempty"`
	VoidedBy          string                   `json:"voidedBy,omitempty"`
	VoidedAt          *time.Time               `json:"voidedAt,omitempty"`
	Logs              []RawMaterialStockLogDTO `json:"logs,omitempty"`
	CreatedAt         time.Time                `json:"createdAt"`
	UpdatedAt         time.Time                `json:"updatedAt"`
}

type RawMaterialStockSummaryDTO struct {
	TotalEntries             int     `json:"totalEntries"`
	DraftCount               int     `json:"draftCount"`
	AvailableCount           int     `json:"availableCount"`
	QuarantineCount          int     `json:"quarantineCount"`
	DepletedCount            int     `json:"depletedCount"`
	RejectedCount            int     `json:"rejectedCount"`
	TotalAvailableQty        float64 `json:"totalAvailableQty"`
	TotalAvailablePackages   int     `json:"totalAvailablePackages"`
	TotalQuarantineQty       float64 `json:"totalQuarantineQty"`
	OrganicAvailableQty      float64 `json:"organicAvailableQty"`
	ConventionalAvailableQty float64 `json:"conventionalAvailableQty"`
	ExpiringSoonCount        int     `json:"expiringSoonCount"`
	ExpiredCount             int     `json:"expiredCount"`
}

// Helpers

func normalizeOrganicStatus(raw string) string {
	low := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(low, "organik") || strings.Contains(low, "organic"):
		return "Organik"
	case strings.Contains(low, "transisi") || strings.Contains(low, "transition"):
		return "Transisi"
	case strings.Contains(low, "konvensional") || strings.Contains(low, "conventional") || strings.Contains(low, "natural"):
		return "Konvensional"
	default:
		return "Organik"
	}
}

func inferCertStatus(certs, keyword, positive, negative string) string {
	low := strings.ToLower(certs)
	if strings.Contains(low, "non-"+strings.ToLower(keyword)) || strings.Contains(low, "non "+strings.ToLower(keyword)) {
		return negative
	}
	if strings.Contains(low, strings.ToLower(keyword)) {
		return positive
	}
	return positive
}

func recommendStockStatusFromQC(sourceType, casStatus, pbdStatus, pbkStatus string) (recommended string, gateReady bool) {
	if sourceType != domain.RawStockSourceLBM {
		return domain.RawStockStatusAvailable, true
	}
	if pbdStatus == "rejected" || casStatus == "rejected" {
		return domain.RawStockStatusRejected, false
	}
	if pbkStatus == "failed" {
		return domain.RawStockStatusQuarantine, false
	}
	if casStatus == "missing" || pbdStatus == "missing" {
		return domain.RawStockStatusQuarantine, false
	}
	if pbdStatus == "conditional" || casStatus == "flagged" || pbkStatus == "partial" {
		return domain.RawStockStatusQuarantine, true
	}
	return domain.RawStockStatusAvailable, true
}

func validateSmartQCGate(sourceType, casStatus, pbdStatus, pbkStatus, targetStatus string) error {
	if targetStatus != domain.RawStockStatusAvailable &&
		targetStatus != domain.RawStockStatusQuarantine &&
		targetStatus != domain.RawStockStatusRejected {
		return apperrors.NewBadRequest("Target status harus salah satu dari: available, quarantine, atau rejected")
	}
	if sourceType != domain.RawStockSourceLBM {
		return nil
	}
	if targetStatus == domain.RawStockStatusAvailable {
		// Smart Gate (Opsi A): CAS & PBD wajib terisi DAN lolos; PBK opsional (boleh N/A) tetapi tidak boleh Failed.
		if pbdStatus == "rejected" || casStatus == "rejected" {
			return apperrors.NewBadRequest("Lot dengan hasil QC PBD atau CAS berstatus Rejected tidak dapat diposting ke Siap Produksi (Available). Tempatkan ke Karantina atau Ditolak.")
		}
		if casStatus == "missing" || casStatus == "" || pbdStatus == "missing" || pbdStatus == "" {
			return apperrors.NewBadRequest("Form QC-CAS dan QC-PBD wajib diisi sebelum memposting lot ke status Siap Produksi (Available). Anda dapat menyimpannya sebagai Draft atau memposting ke Karantina.")
		}
		if casStatus != "cleared" || pbdStatus != "passed" {
			return apperrors.NewBadRequest("Status Siap Produksi (Available) mensyaratkan QC-CAS = Cleared dan QC-PBD = Passed. Hasil QC bersyarat (Flagged/Conditional) harus diposting ke Karantina lalu dirilis oleh Supervisor.")
		}
		if pbkStatus == "failed" {
			return apperrors.NewBadRequest("Form QC-PBK berstatus Failed. Tempatkan lot ke Karantina terlebih dahulu.")
		}
	}
	return nil
}

// lbmBookedQty returns how much of this lot the confirmed LBM (ConfirmGoodsReceipt) already booked into stock.
func lbmBookedQty(ctx context.Context, repo domain.InventoryRepository, e *domain.RawMaterialStockEntry) (int, error) {
	if strings.TrimSpace(e.LBMNo) == "" {
		return 0, nil
	}
	moves, err := repo.ListMovementsByReference(ctx, e.LBMNo)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, m := range moves {
		if m.ProductID != e.ProductID || m.WarehouseID != e.WarehouseID || m.Quantity <= 0 {
			continue
		}
		if e.BatchNo != "" && m.BatchNo != "" && !strings.EqualFold(m.BatchNo, e.BatchNo) {
			continue
		}
		total += m.Quantity
	}
	return total, nil
}

func (uc *inventoryUseCase) toRawMaterialStockResponse(ctx context.Context, e *domain.RawMaterialStockEntry, whCache map[uuid.UUID]string) RawMaterialStockResponseDTO {
	whName := ""
	if whCache != nil {
		if cached, ok := whCache[e.WarehouseID]; ok {
			whName = cached
		} else {
			whName = uc.warehouseName(ctx, e.WarehouseID)
			whCache[e.WarehouseID] = whName
		}
	} else {
		whName = uc.warehouseName(ctx, e.WarehouseID)
	}

	avgPkg := 0.0
	if e.TotalPackages > 0 && e.QtyReceived > 0 {
		avgPkg = math.Round((e.QtyReceived/float64(e.TotalPackages))*1000) / 1000
	}
	expStatus, expDays := BatchStatus(e.ExpiryDate, today())

	logs := make([]RawMaterialStockLogDTO, 0, len(e.Logs))
	for _, l := range e.Logs {
		logs = append(logs, RawMaterialStockLogDTO{
			ID:            l.ID,
			EntryID:       l.EntryID,
			EventType:     l.EventType,
			RefDocumentNo: l.RefDocumentNo,
			QtyDelta:      l.QtyDelta,
			PackageDelta:  l.PackageDelta,
			BalanceAfter:  l.BalanceAfter,
			PackagesAfter: l.PackagesAfter,
			ActorName:     l.ActorName,
			Notes:         l.Notes,
			OccurredAt:    l.OccurredAt,
		})
	}

	return RawMaterialStockResponseDTO{
		ID:                e.ID,
		EntryNo:           e.EntryNo,
		DocumentCode:      e.DocumentCode,
		EntryDate:         e.EntryDate,
		SourceType:        e.SourceType,
		GoodsReceiptID:    e.GoodsReceiptID,
		LBMNo:             e.LBMNo,
		PONo:              e.PONo,
		DeliveryNoteNo:    e.DeliveryNoteNo,
		SupplierID:        e.SupplierID,
		SupplierName:      e.SupplierName,
		ProductID:         e.ProductID,
		ProductSKU:        e.ProductSKU,
		ProductName:       e.ProductName,
		BatchNo:           e.BatchNo,
		CASCheckID:        e.CASCheckID,
		CASStatus:         e.CASStatus,
		PBDCheckID:        e.PBDCheckID,
		PBDStatus:         e.PBDStatus,
		PBKCheckID:        e.PBKCheckID,
		PBKStatus:         e.PBKStatus,
		MoistureContent:   e.MoistureContent,
		BrixValue:         e.BrixValue,
		MeshSize:          e.MeshSize,
		QCRecommendation:  e.QCRecommendation,
		OrganicStatus:     e.OrganicStatus,
		HalalStatus:       e.HalalStatus,
		KosherStatus:      e.KosherStatus,
		Certifications:    e.Certifications,
		QualityGrade:      e.QualityGrade,
		UoM:               e.UoM,
		PackageUnit:       e.PackageUnit,
		QtyLBM:            e.QtyLBM,
		QtyReceived:       e.QtyReceived,
		QtyRejected:       e.QtyRejected,
		TotalPackages:     e.TotalPackages,
		AvgWeightPerPkg:   avgPkg,
		QtyReserved:       e.QtyReserved,
		QtyIssued:         e.QtyIssued,
		QtyRemaining:      e.QtyRemaining,
		RemainingPackages: e.RemainingPackages,
		WarehouseID:       e.WarehouseID,
		WarehouseName:     whName,
		StorageZone:       e.StorageZone,
		RackLocation:      e.RackLocation,
		PalletNo:          e.PalletNo,
		ProductionDate:    e.ProductionDate,
		ExpiryDate:        e.ExpiryDate,
		ExpiryStatus:      expStatus,
		DaysToExpiry:      expDays,
		StorageCondition:  e.StorageCondition,
		Status:            e.Status,
		HandedOverBy:      e.HandedOverBy,
		ReceivedBy:        e.ReceivedBy,
		ApprovedBy:        e.ApprovedBy,
		ApprovedAt:        e.ApprovedAt,
		StockSynced:       e.StockSynced,
		StockAddedBySBB:   e.StockAddedBySBB,
		Notes:             e.Notes,
		VoidReason:        e.VoidReason,
		VoidedBy:          e.VoidedBy,
		VoidedAt:          e.VoidedAt,
		Logs:              logs,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

// GetLBMInboundContext pulls LBM details, PO/supplier info, lines, the 3 QC form statuses (CAS, PBD, PBK),
// and any already-recorded SBB entries for the given goodsReceiptId.
func (uc *inventoryUseCase) GetLBMInboundContext(ctx context.Context, goodsReceiptID uuid.UUID) (*LBMInboundContextDTO, error) {
	if goodsReceiptID == uuid.Nil {
		return nil, apperrors.NewBadRequest("goodsReceiptId wajib diisi")
	}
	if uc.inboundReader == nil {
		return nil, apperrors.NewInternal(nil, "Inbound reader tidak tersedia")
	}
	raw, err := uc.inboundReader.ReadLBMContext(ctx, goodsReceiptID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membaca konteks LBM & QC")
	}
	if raw == nil {
		return nil, apperrors.NewNotFound("Laporan Barang Masuk (LBM) tidak ditemukan")
	}

	casStatus := "missing"
	if raw.CAS != nil && raw.CAS.OverallStatus != "" {
		casStatus = raw.CAS.OverallStatus
	}
	pbdStatus := "missing"
	if raw.PBD != nil && raw.PBD.OverallStatus != "" {
		pbdStatus = raw.PBD.OverallStatus
	}
	pbkStatus := "missing"
	if raw.PBK != nil && raw.PBK.OverallStatus != "" {
		pbkStatus = raw.PBK.OverallStatus
	}

	recommended, gateReady := recommendStockStatusFromQC(domain.RawStockSourceLBM, casStatus, pbdStatus, pbkStatus)

	lines := make([]LBMInboundLineDTO, 0, len(raw.Lines))
	for _, l := range raw.Lines {
		lines = append(lines, LBMInboundLineDTO{
			LineID:         l.LineID,
			ProductID:      l.ProductID,
			ProductSKU:     l.ProductSKU,
			ProductName:    l.ProductName,
			UoM:            l.UoM,
			OrganicType:    l.OrganicType,
			Certifications: l.Certifications,
			LotNumber:      l.LotNumber,
			WeightGross:    l.WeightGross,
			WeightTare:     l.WeightTare,
			WeightNet:      l.WeightNet,
			PackageSize:    l.PackageSize,
			BagCount:       l.BagCount,
		})
	}

	whName := ""
	if raw.WarehouseID != nil {
		whName = uc.warehouseName(ctx, *raw.WarehouseID)
	}

	existingEntries := []RawMaterialStockEntrySummary{}
	if existing, err := uc.repo.ListRawMaterialStockEntries(ctx, domain.RawMaterialStockFilter{
		GoodsReceiptID: &goodsReceiptID,
	}); err == nil {
		for _, e := range existing {
			if e.Status == domain.RawStockStatusVoided {
				continue
			}
			existingEntries = append(existingEntries, RawMaterialStockEntrySummary{
				ID:          e.ID,
				EntryNo:     e.EntryNo,
				ProductID:   e.ProductID,
				ProductName: e.ProductName,
				BatchNo:     e.BatchNo,
				QtyReceived: e.QtyReceived,
				Status:      e.Status,
			})
		}
	}

	return &LBMInboundContextDTO{
		GoodsReceiptID:    raw.GoodsReceiptID,
		LBMNo:             raw.LBMNo,
		PONo:              raw.PONo,
		ReceivedDate:      raw.ReceivedDate,
		WarehouseID:       raw.WarehouseID,
		WarehouseName:     whName,
		ReceivedBy:        raw.ReceivedBy,
		Status:            raw.Status,
		DeliveryNoteNo:    raw.DeliveryNoteNo,
		SupplierID:        raw.SupplierID,
		SupplierName:      raw.SupplierName,
		SugarCondition:    raw.SugarCondition,
		PackageCondition:  raw.PackageCondition,
		Notes:             raw.Notes,
		Lines:             lines,
		CAS:               raw.CAS,
		PBD:               raw.PBD,
		PBK:               raw.PBK,
		CASStatus:         casStatus,
		PBDStatus:         pbdStatus,
		PBKStatus:         pbkStatus,
		QCGateReady:       gateReady,
		RecommendedStatus: recommended,
		ExistingEntries:   existingEntries,
	}, nil
}

func (uc *inventoryUseCase) nextRawMaterialEntryNo(ctx context.Context, entryDate string) (string, error) {
	return docseq.Next(ctx, uc.repo, "SBB", entryDate)
}

// CreateRawMaterialStockEntry creates a draft (or immediately posted) Form Stok Gudang Bahan Baku (GDG-002-SBB).
func (uc *inventoryUseCase) CreateRawMaterialStockEntry(ctx context.Context, dto CreateRawMaterialStockDTO) (*RawMaterialStockResponseDTO, error) {
	if dto.ProductID == uuid.Nil {
		return nil, apperrors.NewBadRequest("Produk bahan baku wajib dipilih")
	}
	if dto.WarehouseID == uuid.Nil {
		return nil, apperrors.NewBadRequest("Gudang tujuan wajib dipilih")
	}
	if strings.TrimSpace(dto.BatchNo) == "" {
		return nil, apperrors.NewBadRequest("Kode Bahan Baku / Nomor Lot (Batch) wajib diisi")
	}
	if dto.QtyReceived < 0 || dto.QtyRejected < 0 {
		return nil, apperrors.NewBadRequest("Kuantitas tidak boleh bernilai negatif")
	}
	if dto.QtyReceived == 0 && dto.QtyRejected == 0 {
		return nil, apperrors.NewBadRequest("Kuantitas diterima atau ditolak harus lebih dari 0")
	}
	if dto.ExpiryDate != "" {
		if _, err := time.Parse("2006-01-02", dto.ExpiryDate); err != nil {
			return nil, apperrors.NewBadRequest("Format tanggal kedaluwarsa harus YYYY-MM-DD")
		}
	}

	entryDate := strings.TrimSpace(dto.EntryDate)
	if entryDate == "" {
		entryDate = today()
	}

	sourceType := strings.TrimSpace(dto.SourceType)
	if sourceType == "" {
		if dto.GoodsReceiptID != nil || strings.TrimSpace(dto.LBMNo) != "" {
			sourceType = domain.RawStockSourceLBM
		} else {
			sourceType = domain.RawStockSourceOpeningBalance
		}
	}

	// Enrich product metadata
	prodSKU, prodName := strings.TrimSpace(dto.ProductSKU), strings.TrimSpace(dto.ProductName)
	uom := strings.TrimSpace(dto.UoM)
	organicStatus := strings.TrimSpace(dto.OrganicStatus)
	certs := strings.TrimSpace(dto.Certifications)
	if uc.productRepo != nil {
		if p, err := uc.productRepo.GetByID(ctx, dto.ProductID); err == nil && p != nil {
			if prodSKU == "" {
				prodSKU = p.SKU
			}
			if prodName == "" {
				prodName = p.Name
			}
			if uom == "" {
				uom = p.Unit
			}
			if organicStatus == "" {
				organicStatus = normalizeOrganicStatus(p.OrganicType)
			}
			if certs == "" {
				certs = p.Certifications
			}
		}
	}
	if prodName == "" {
		prodName = "Bahan Baku"
	}
	if uom == "" {
		uom = "KG"
	}
	if organicStatus == "" {
		organicStatus = "Organik"
	}

	halalStatus := strings.TrimSpace(dto.HalalStatus)
	if halalStatus == "" {
		halalStatus = inferCertStatus(certs, "halal", "Halal", "Non-Halal")
	}
	kosherStatus := strings.TrimSpace(dto.KosherStatus)
	if kosherStatus == "" {
		kosherStatus = inferCertStatus(certs, "kosher", "Kosher", "Non-Kosher")
	}
	qualityGrade := strings.TrimSpace(dto.QualityGrade)
	if qualityGrade == "" {
		qualityGrade = "Standar"
	}
	pkgUnit := strings.TrimSpace(dto.PackageUnit)
	if pkgUnit == "" {
		if strings.EqualFold(uom, "L") || strings.EqualFold(uom, "LITER") {
			pkgUnit = "Jerigen"
		} else {
			pkgUnit = "Sak"
		}
	}
	storageZone := strings.TrimSpace(dto.StorageZone)
	if storageZone == "" {
		if organicStatus == "Organik" {
			storageZone = "Zona Organik"
		} else {
			storageZone = "Zona Konvensional"
		}
	}
	storageCond := strings.TrimSpace(dto.StorageCondition)
	if storageCond == "" {
		storageCond = "Suhu Ruang Kering"
	}

	// Auto-pull latest QC snapshot if goodsReceiptId is provided
	casID, casStatus := dto.CASCheckID, strings.TrimSpace(dto.CASStatus)
	pbdID, pbdStatus := dto.PBDCheckID, strings.TrimSpace(dto.PBDStatus)
	pbkID, pbkStatus := dto.PBKCheckID, strings.TrimSpace(dto.PBKStatus)
	moisture := dto.MoistureContent
	brix := dto.BrixValue
	mesh := strings.TrimSpace(dto.MeshSize)
	qcRec := strings.TrimSpace(dto.QCRecommendation)
	lbmNo := strings.TrimSpace(dto.LBMNo)
	poNo := strings.TrimSpace(dto.PONo)
	dnNo := strings.TrimSpace(dto.DeliveryNoteNo)
	supplierID := dto.SupplierID
	supplierName := strings.TrimSpace(dto.SupplierName)
	handedOverBy := strings.TrimSpace(dto.HandedOverBy)

	if dto.GoodsReceiptID != nil && *dto.GoodsReceiptID != uuid.Nil && uc.inboundReader != nil {
		if raw, err := uc.inboundReader.ReadLBMContext(ctx, *dto.GoodsReceiptID); err == nil && raw != nil {
			if lbmNo == "" {
				lbmNo = raw.LBMNo
			}
			if poNo == "" {
				poNo = raw.PONo
			}
			if dnNo == "" {
				dnNo = raw.DeliveryNoteNo
			}
			if supplierID == nil {
				supplierID = raw.SupplierID
			}
			if supplierName == "" {
				supplierName = raw.SupplierName
			}
			if raw.CAS != nil {
				if casID == nil {
					idCopy := raw.CAS.ID
					casID = &idCopy
				}
				if casStatus == "" || casStatus == "missing" {
					casStatus = raw.CAS.OverallStatus
				}
			}
			if raw.PBD != nil {
				if pbdID == nil {
					idCopy := raw.PBD.ID
					pbdID = &idCopy
				}
				if pbdStatus == "" || pbdStatus == "missing" {
					pbdStatus = raw.PBD.OverallStatus
				}
				if moisture == nil {
					moisture = raw.PBD.MoistureContent
				}
				if brix == nil {
					brix = raw.PBD.BrixValue
				}
				if mesh == "" {
					mesh = raw.PBD.MeshSize
				}
				if qcRec == "" {
					qcRec = raw.PBD.RecommendedAction
				}
				if handedOverBy == "" {
					handedOverBy = raw.PBD.QCName
				}
			}
			if raw.PBK != nil {
				if pbkID == nil {
					idCopy := raw.PBK.ID
					pbkID = &idCopy
				}
				if pbkStatus == "" || pbkStatus == "missing" {
					pbkStatus = raw.PBK.OverallStatus
				}
			}
		}
	}

	if sourceType == domain.RawStockSourceLBM {
		if casStatus == "" {
			casStatus = "missing"
		}
		if pbdStatus == "" {
			pbdStatus = "missing"
		}
		if pbkStatus == "" {
			pbkStatus = "missing"
		}
	} else {
		if casStatus == "" {
			casStatus = "not_applicable"
		}
		if pbdStatus == "" {
			pbdStatus = "not_applicable"
		}
		if pbkStatus == "" {
			pbkStatus = "not_applicable"
		}
	}

	receivedBy := strings.TrimSpace(dto.ReceivedBy)
	if receivedBy == "" {
		receivedBy = actor.EmailFrom(ctx)
	}

	batchNoNorm := strings.ToUpper(strings.TrimSpace(dto.BatchNo))

	// Prevent accidental duplicate put-away of the same LBM lot (one active SBB per LBM + product + lot).
	if sourceType == domain.RawStockSourceLBM && dto.GoodsReceiptID != nil && *dto.GoodsReceiptID != uuid.Nil {
		existing, err := uc.repo.ListRawMaterialStockEntries(ctx, domain.RawMaterialStockFilter{GoodsReceiptID: dto.GoodsReceiptID})
		if err != nil {
			return nil, apperrors.NewInternal(err, "Gagal memeriksa duplikasi Form SBB")
		}
		for _, x := range existing {
			if x.Status != domain.RawStockStatusVoided && x.ProductID == dto.ProductID && strings.EqualFold(x.BatchNo, batchNoNorm) {
				return nil, apperrors.NewConflict(fmt.Sprintf("Lot %s dari LBM %s sudah tercatat pada %s (status: %s). Gunakan dokumen tersebut atau void terlebih dahulu.", batchNoNorm, lbmNo, x.EntryNo, x.Status))
			}
		}
	}

	// Validate the Smart QC Gate up-front so a failed "Post" never leaves an orphan draft behind.
	if dto.PostImmediately {
		target := strings.TrimSpace(dto.TargetStatus)
		if target == "" {
			target, _ = recommendStockStatusFromQC(sourceType, casStatus, pbdStatus, pbkStatus)
		}
		if err := validateSmartQCGate(sourceType, casStatus, pbdStatus, pbkStatus, target); err != nil {
			return nil, err
		}
	}

	entryNo, err := uc.nextRawMaterialEntryNo(ctx, entryDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menghasilkan nomor dokumen SBB")
	}

	entry := &domain.RawMaterialStockEntry{
		EntryNo:           entryNo,
		DocumentCode:      "GDG-002-SBB",
		EntryDate:         entryDate,
		SourceType:        sourceType,
		GoodsReceiptID:    dto.GoodsReceiptID,
		LBMNo:             lbmNo,
		PONo:              poNo,
		DeliveryNoteNo:    dnNo,
		SupplierID:        supplierID,
		SupplierName:      supplierName,
		ProductID:         dto.ProductID,
		ProductSKU:        prodSKU,
		ProductName:       prodName,
		BatchNo:           batchNoNorm,
		CASCheckID:        casID,
		CASStatus:         casStatus,
		PBDCheckID:        pbdID,
		PBDStatus:         pbdStatus,
		PBKCheckID:        pbkID,
		PBKStatus:         pbkStatus,
		MoistureContent:   moisture,
		BrixValue:         brix,
		MeshSize:          mesh,
		QCRecommendation:  qcRec,
		OrganicStatus:     organicStatus,
		HalalStatus:       halalStatus,
		KosherStatus:      kosherStatus,
		Certifications:    certs,
		QualityGrade:      qualityGrade,
		UoM:               uom,
		PackageUnit:       pkgUnit,
		QtyLBM:            dto.QtyLBM,
		QtyReceived:       dto.QtyReceived,
		QtyRejected:       dto.QtyRejected,
		TotalPackages:     dto.TotalPackages,
		QtyReserved:       0,
		QtyIssued:         0,
		QtyRemaining:      dto.QtyReceived,
		RemainingPackages: dto.TotalPackages,
		WarehouseID:       dto.WarehouseID,
		StorageZone:       storageZone,
		RackLocation:      strings.ToUpper(strings.TrimSpace(dto.RackLocation)),
		PalletNo:          strings.ToUpper(strings.TrimSpace(dto.PalletNo)),
		ProductionDate:    strings.TrimSpace(dto.ProductionDate),
		ExpiryDate:        strings.TrimSpace(dto.ExpiryDate),
		StorageCondition:  storageCond,
		Status:            domain.RawStockStatusDraft,
		HandedOverBy:      handedOverBy,
		ReceivedBy:        receivedBy,
		Notes:             strings.TrimSpace(dto.Notes),
	}

	if err := uc.repo.CreateRawMaterialStockEntry(ctx, entry); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan Form Stok Gudang Bahan Baku")
	}

	if dto.PostImmediately {
		target := strings.TrimSpace(dto.TargetStatus)
		if target == "" {
			target, _ = recommendStockStatusFromQC(entry.SourceType, entry.CASStatus, entry.PBDStatus, entry.PBKStatus)
		}
		return uc.PostRawMaterialStockEntry(ctx, entry.ID, PostRawMaterialStockDTO{
			TargetStatus: target,
			ApprovedBy:   receivedBy,
			Notes:        entry.Notes,
		})
	}

	res := uc.toRawMaterialStockResponse(ctx, entry, nil)
	return &res, nil
}

// UpdateRawMaterialStockEntry updates draft entries fully, or allows bin/rack/pallet/expiry/notes updates on posted lots.
func (uc *inventoryUseCase) UpdateRawMaterialStockEntry(ctx context.Context, id uuid.UUID, dto UpdateRawMaterialStockDTO) (*RawMaterialStockResponseDTO, error) {
	e, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat Form Stok Gudang Bahan Baku")
	}
	if e == nil {
		return nil, apperrors.NewNotFound("Form Stok Gudang Bahan Baku tidak ditemukan")
	}
	if e.Status == domain.RawStockStatusVoided || e.Status == domain.RawStockStatusDepleted {
		return nil, apperrors.NewConflict("Dokumen berstatus " + e.Status + " tidak dapat diubah")
	}

	if dto.ExpiryDate != "" {
		if _, err := time.Parse("2006-01-02", dto.ExpiryDate); err != nil {
			return nil, apperrors.NewBadRequest("Format tanggal kedaluwarsa harus YYYY-MM-DD")
		}
	}

	oldLoc := fmt.Sprintf("%s / %s / %s", e.StorageZone, e.RackLocation, e.PalletNo)

	// Location, storage condition, expiry, and notes can be edited on both draft and active posted lots.
	if dto.StorageZone != "" {
		e.StorageZone = strings.TrimSpace(dto.StorageZone)
	}
	if dto.RackLocation != "" {
		e.RackLocation = strings.ToUpper(strings.TrimSpace(dto.RackLocation))
	}
	if dto.PalletNo != "" {
		e.PalletNo = strings.ToUpper(strings.TrimSpace(dto.PalletNo))
	}
	if dto.ProductionDate != "" {
		e.ProductionDate = strings.TrimSpace(dto.ProductionDate)
	}
	if dto.ExpiryDate != "" {
		e.ExpiryDate = strings.TrimSpace(dto.ExpiryDate)
	}
	if dto.StorageCondition != "" {
		e.StorageCondition = strings.TrimSpace(dto.StorageCondition)
	}
	if dto.Notes != "" {
		e.Notes = strings.TrimSpace(dto.Notes)
	}

	if e.Status == domain.RawStockStatusDraft {
		if dto.EntryDate != "" {
			e.EntryDate = strings.TrimSpace(dto.EntryDate)
		}
		if dto.BatchNo != "" {
			e.BatchNo = strings.ToUpper(strings.TrimSpace(dto.BatchNo))
		}
		if dto.PBKStatus != "" {
			e.PBKStatus = strings.TrimSpace(dto.PBKStatus)
		}
		if dto.MoistureContent != nil {
			e.MoistureContent = dto.MoistureContent
		}
		if dto.BrixValue != nil {
			e.BrixValue = dto.BrixValue
		}
		if dto.MeshSize != "" {
			e.MeshSize = strings.TrimSpace(dto.MeshSize)
		}
		if dto.QCRecommendation != "" {
			e.QCRecommendation = strings.TrimSpace(dto.QCRecommendation)
		}
		if dto.OrganicStatus != "" {
			e.OrganicStatus = strings.TrimSpace(dto.OrganicStatus)
		}
		if dto.HalalStatus != "" {
			e.HalalStatus = strings.TrimSpace(dto.HalalStatus)
		}
		if dto.KosherStatus != "" {
			e.KosherStatus = strings.TrimSpace(dto.KosherStatus)
		}
		if dto.Certifications != "" {
			e.Certifications = strings.TrimSpace(dto.Certifications)
		}
		if dto.QualityGrade != "" {
			e.QualityGrade = strings.TrimSpace(dto.QualityGrade)
		}
		if dto.UoM != "" {
			e.UoM = strings.TrimSpace(dto.UoM)
		}
		if dto.PackageUnit != "" {
			e.PackageUnit = strings.TrimSpace(dto.PackageUnit)
		}
		if dto.QtyLBM != nil {
			e.QtyLBM = *dto.QtyLBM
		}
		if dto.QtyReceived != nil {
			if *dto.QtyReceived < 0 {
				return nil, apperrors.NewBadRequest("Kuantitas diterima tidak boleh negatif")
			}
			e.QtyReceived = *dto.QtyReceived
			e.QtyRemaining = *dto.QtyReceived
		}
		if dto.QtyRejected != nil {
			if *dto.QtyRejected < 0 {
				return nil, apperrors.NewBadRequest("Kuantitas reject tidak boleh negatif")
			}
			e.QtyRejected = *dto.QtyRejected
		}
		if dto.TotalPackages != nil {
			e.TotalPackages = *dto.TotalPackages
			e.RemainingPackages = *dto.TotalPackages
		}
		if dto.WarehouseID != nil && *dto.WarehouseID != uuid.Nil {
			e.WarehouseID = *dto.WarehouseID
		}
		if dto.HandedOverBy != "" {
			e.HandedOverBy = strings.TrimSpace(dto.HandedOverBy)
		}
		if dto.ReceivedBy != "" {
			e.ReceivedBy = strings.TrimSpace(dto.ReceivedBy)
		}
	}

	if err := uc.repo.UpdateRawMaterialStockEntry(ctx, e); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui Form Stok Gudang Bahan Baku")
	}

	// Keep StockBatch.ExpiryDate synchronized if already posted.
	if e.StockSynced && e.ExpiryDate != "" {
		if b, err := uc.repo.GetBatchByNo(ctx, e.ProductID, e.WarehouseID, e.BatchNo); err == nil && b != nil && b.ExpiryDate != e.ExpiryDate {
			b.ExpiryDate = e.ExpiryDate
			_ = uc.repo.SaveBatch(ctx, b)
		}
	}

	newLoc := fmt.Sprintf("%s / %s / %s", e.StorageZone, e.RackLocation, e.PalletNo)
	if e.Status != domain.RawStockStatusDraft && oldLoc != newLoc {
		actorName := actor.EmailFrom(ctx)
		if actorName == "" {
			actorName = e.ReceivedBy
		}
		_ = uc.repo.CreateRawMaterialStockLog(ctx, &domain.RawMaterialStockLog{
			EntryID:       e.ID,
			EventType:     domain.RawStockEventLocationUpdate,
			RefDocumentNo: e.EntryNo,
			QtyDelta:      0,
			PackageDelta:  0,
			BalanceAfter:  e.QtyRemaining,
			PackagesAfter: e.RemainingPackages,
			ActorName:     actorName,
			Notes:         fmt.Sprintf("Pindah lokasi: %s -> %s", oldLoc, newLoc),
			OccurredAt:    time.Now(),
		})
	}

	reloaded, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil || reloaded == nil {
		res := uc.toRawMaterialStockResponse(ctx, e, nil)
		return &res, nil
	}
	res := uc.toRawMaterialStockResponse(ctx, reloaded, nil)
	return &res, nil
}

// PostRawMaterialStockEntry posts a draft SBB entry into warehouse inventory with Smart QC Gate
// and reconciled anti double-counting synchronization against inventory_stock_levels/batches.
func (uc *inventoryUseCase) PostRawMaterialStockEntry(ctx context.Context, id uuid.UUID, dto PostRawMaterialStockDTO) (*RawMaterialStockResponseDTO, error) {
	e, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat Form Stok Gudang Bahan Baku")
	}
	if e == nil {
		return nil, apperrors.NewNotFound("Form Stok Gudang Bahan Baku tidak ditemukan")
	}
	if e.Status != domain.RawStockStatusDraft {
		return nil, apperrors.NewConflict("Hanya dokumen berstatus Draft yang dapat di-post")
	}

	// Refresh latest QC snapshot from inbound reader right before posting (so newly completed QC forms are recognized).
	if e.GoodsReceiptID != nil && *e.GoodsReceiptID != uuid.Nil && uc.inboundReader != nil {
		if raw, rerr := uc.inboundReader.ReadLBMContext(ctx, *e.GoodsReceiptID); rerr == nil && raw != nil {
			if raw.CAS != nil {
				cid := raw.CAS.ID
				e.CASCheckID = &cid
				e.CASStatus = raw.CAS.OverallStatus
			}
			if raw.PBD != nil {
				pid := raw.PBD.ID
				e.PBDCheckID = &pid
				e.PBDStatus = raw.PBD.OverallStatus
				if e.MoistureContent == nil {
					e.MoistureContent = raw.PBD.MoistureContent
				}
				if e.BrixValue == nil {
					e.BrixValue = raw.PBD.BrixValue
				}
				if e.MeshSize == "" {
					e.MeshSize = raw.PBD.MeshSize
				}
				if e.QCRecommendation == "" {
					e.QCRecommendation = raw.PBD.RecommendedAction
				}
			}
			if raw.PBK != nil {
				kid := raw.PBK.ID
				e.PBKCheckID = &kid
				if e.PBKStatus != "not_applicable" {
					e.PBKStatus = raw.PBK.OverallStatus
				}
			}
		}
	}

	target := strings.TrimSpace(dto.TargetStatus)
	if target == "" {
		target, _ = recommendStockStatusFromQC(e.SourceType, e.CASStatus, e.PBDStatus, e.PBKStatus)
	}
	if err := validateSmartQCGate(e.SourceType, e.CASStatus, e.PBDStatus, e.PBKStatus, target); err != nil {
		return nil, err
	}

	actorName := strings.TrimSpace(dto.ApprovedBy)
	if actorName == "" {
		actorName = actor.EmailFrom(ctx)
	}
	if actorName == "" {
		actorName = e.ReceivedBy
	}

	now := time.Now()

	err = uc.repo.WithTransaction(ctx, func(txRepo domain.InventoryRepository) error {
		e.Status = target
		e.ApprovedBy = actorName
		e.ApprovedAt = &now
		if strings.TrimSpace(dto.Notes) != "" {
			e.Notes = strings.TrimSpace(dto.Notes)
		}

		if target == domain.RawStockStatusRejected {
			e.QtyRemaining = 0
			e.RemainingPackages = 0
		} else {
			e.QtyRemaining = e.QtyReceived
			e.RemainingPackages = e.TotalPackages
		}

		// Anti Double-Counting Reconciled Put-Away Sync (Opsi A)
		if !e.StockSynced {
			bookedQty := 0
			if e.SourceType == domain.RawStockSourceLBM {
				bookedQty, _ = lbmBookedQty(ctx, txRepo, e)
			}

			if bookedQty > 0 {
				// LBM already increased StockLevel & StockBatch.
				// Enrich batch expiry date if provided.
				if e.ExpiryDate != "" && e.BatchNo != "" {
					if b, berr := txRepo.GetBatchByNo(ctx, e.ProductID, e.WarehouseID, e.BatchNo); berr == nil && b != nil && b.ExpiryDate == "" {
						b.ExpiryDate = e.ExpiryDate
						_ = txRepo.SaveBatch(ctx, b)
					}
				}
				// Reconcile any QC rejection weight so StockLevel reflects accepted stock accurately.
				deductQty := 0
				if target == domain.RawStockStatusRejected {
					deductQty = bookedQty
				} else if e.QtyRejected > 0 {
					deductQty = int(math.Round(e.QtyRejected))
				}
				if deductQty > 0 {
					if err := uc.applyTxStockDelta(ctx, txRepo, e.ProductID, e.WarehouseID, -deductQty, domain.MovementReject, e.BatchNo, "", e.EntryNo, "Rekonsiliasi Reject QC (Form SBB)", actorName); err != nil {
						return err
					}
				}
				e.StockSynced = true
				e.StockAddedBySBB = false
			} else {
				// Non-LBM source (opening_balance / manual / return) or LBM not yet booked to stock:
				// Book accepted quantity into StockLevel & StockBatch directly.
				if target == domain.RawStockStatusAvailable || target == domain.RawStockStatusQuarantine {
					addQty := int(math.Round(e.QtyReceived))
					if addQty > 0 {
						if err := uc.applyTxStockDelta(ctx, txRepo, e.ProductID, e.WarehouseID, addQty, domain.MovementIn, e.BatchNo, e.ExpiryDate, e.EntryNo, "Masuk Gudang Bahan Baku ("+e.SourceType+")", actorName); err != nil {
							return err
						}
						e.StockAddedBySBB = true
					}
				}
				e.StockSynced = true
			}
		}

		if err := txRepo.UpdateRawMaterialStockEntry(ctx, e); err != nil {
			return apperrors.NewInternal(err, "Gagal memperbarui status Form Stok Gudang Bahan Baku")
		}

		// Record mini Kartu Stok log entry
		eventType := domain.RawStockEventPutaway
		delta := e.QtyReceived
		pkgDelta := e.TotalPackages
		logNote := fmt.Sprintf("Put-away ke %s (Rak: %s, Pallet: %s) — Status: %s", e.StorageZone, e.RackLocation, e.PalletNo, e.Status)
		if target == domain.RawStockStatusRejected {
			eventType = domain.RawStockEventQCReject
			delta = 0
			pkgDelta = 0
			logNote = "Ditolak pada penerimaan gudang bahan baku (QC Rejected)"
		}
		if err := txRepo.CreateRawMaterialStockLog(ctx, &domain.RawMaterialStockLog{
			EntryID:       e.ID,
			EventType:     eventType,
			RefDocumentNo: e.EntryNo,
			QtyDelta:      delta,
			PackageDelta:  pkgDelta,
			BalanceAfter:  e.QtyRemaining,
			PackagesAfter: e.RemainingPackages,
			ActorName:     actorName,
			Notes:         logNote,
			OccurredAt:    now,
		}); err != nil {
			return apperrors.NewInternal(err, "Gagal mencatat log kartu stok lot")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	reloaded, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil || reloaded == nil {
		res := uc.toRawMaterialStockResponse(ctx, e, nil)
		return &res, nil
	}
	res := uc.toRawMaterialStockResponse(ctx, reloaded, nil)
	return &res, nil
}

// applyTxStockDelta safely mutates StockLevel, StockBatch, and appends a StockMovement inside an existing txRepo.
// For negative deltas (e.g. reject/void reconciliation), it clamps against current available stock so it never fails with negative stock.
func (uc *inventoryUseCase) applyTxStockDelta(
	ctx context.Context,
	txRepo domain.InventoryRepository,
	productID, warehouseID uuid.UUID,
	delta int,
	movementType, batchNo, expiryDate, reference, reason, actorName string,
) error {
	if delta == 0 {
		return nil
	}
	level, err := txRepo.GetStockLevel(ctx, productID, warehouseID)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get stock level")
	}
	if level == nil {
		level = &domain.StockLevel{ProductID: productID, WarehouseID: warehouseID}
	}
	if delta < 0 && level.Quantity+delta < 0 {
		delta = -level.Quantity
		if delta == 0 {
			return nil
		}
	}

	// Update StockBatch
	if delta > 0 && strings.TrimSpace(batchNo) != "" {
		if _, err := uc.receiveIntoBatch(ctx, txRepo, AdjustStockDTO{
			ProductID:   productID,
			WarehouseID: warehouseID,
			Quantity:    delta,
			BatchNo:     batchNo,
			ExpiryDate:  expiryDate,
		}); err != nil {
			return err
		}
	} else if delta < 0 && strings.TrimSpace(batchNo) != "" {
		if b, berr := txRepo.GetBatchByNo(ctx, productID, warehouseID, strings.TrimSpace(batchNo)); berr == nil && b != nil && b.Quantity > 0 {
			take := min(-delta, b.Quantity)
			b.Quantity -= take
			if err := txRepo.SaveBatch(ctx, b); err != nil {
				return apperrors.NewInternal(err, "Failed to update batch quantity")
			}
		}
	}

	level.Quantity += delta
	if err := txRepo.UpsertStockLevel(ctx, level); err != nil {
		return apperrors.NewInternal(err, "Failed to update stock level")
	}

	mv := &domain.StockMovement{
		ProductID:   productID,
		WarehouseID: warehouseID,
		Type:        movementType,
		Quantity:    delta,
		Balance:     level.Quantity,
		Reference:   reference,
		Reason:      reason,
		CreatedBy:   actorName,
		BatchNo:     strings.TrimSpace(batchNo),
	}
	if err := txRepo.CreateMovement(ctx, mv); err != nil {
		return apperrors.NewInternal(err, "Failed to record stock movement")
	}
	return nil
}

// ReleaseQuarantine transitions a quarantined lot to either available (ready for BPB/LBB) or rejected.
func (uc *inventoryUseCase) ReleaseQuarantine(ctx context.Context, id uuid.UUID, dto ReleaseQuarantineDTO) (*RawMaterialStockResponseDTO, error) {
	decision := strings.TrimSpace(dto.Decision)
	if decision != domain.RawStockStatusAvailable && decision != domain.RawStockStatusRejected {
		return nil, apperrors.NewBadRequest("Keputusan rilis karantina harus 'available' atau 'rejected'")
	}
	notes := strings.TrimSpace(dto.Notes)
	if notes == "" {
		return nil, apperrors.NewBadRequest("Catatan / alasan rilis karantina wajib diisi")
	}

	e, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat Form Stok Gudang Bahan Baku")
	}
	if e == nil {
		return nil, apperrors.NewNotFound("Form Stok Gudang Bahan Baku tidak ditemukan")
	}
	if e.Status != domain.RawStockStatusQuarantine {
		return nil, apperrors.NewConflict("Hanya lot berstatus Karantina yang dapat dirilis")
	}

	actorName := strings.TrimSpace(dto.ApprovedBy)
	if actorName == "" {
		actorName = actor.EmailFrom(ctx)
	}
	now := time.Now()

	err = uc.repo.WithTransaction(ctx, func(txRepo domain.InventoryRepository) error {
		e.Status = decision
		e.ApprovedBy = actorName
		e.ApprovedAt = &now

		qtyDelta := 0.0
		pkgDelta := 0
		if decision == domain.RawStockStatusRejected && e.QtyRemaining > 0 {
			qtyDelta = -e.QtyRemaining
			pkgDelta = -e.RemainingPackages
			deductInt := int(math.Round(e.QtyRemaining))
			if e.StockSynced && deductInt > 0 {
				if err := uc.applyTxStockDelta(ctx, txRepo, e.ProductID, e.WarehouseID, -deductInt, domain.MovementReject, e.BatchNo, "", e.EntryNo, "Reject dari Karantina: "+notes, actorName); err != nil {
					return err
				}
			}
			e.QtyRejected += e.QtyRemaining
			e.QtyRemaining = 0
			e.RemainingPackages = 0
		}

		if err := txRepo.UpdateRawMaterialStockEntry(ctx, e); err != nil {
			return apperrors.NewInternal(err, "Gagal menyimpan status rilis karantina")
		}

		if err := txRepo.CreateRawMaterialStockLog(ctx, &domain.RawMaterialStockLog{
			EntryID:       e.ID,
			EventType:     domain.RawStockEventQuarantineRelease,
			RefDocumentNo: e.EntryNo,
			QtyDelta:      qtyDelta,
			PackageDelta:  pkgDelta,
			BalanceAfter:  e.QtyRemaining,
			PackagesAfter: e.RemainingPackages,
			ActorName:     actorName,
			Notes:         fmt.Sprintf("Rilis Karantina -> %s: %s", decision, notes),
			OccurredAt:    now,
		}); err != nil {
			return apperrors.NewInternal(err, "Gagal mencatat log rilis karantina")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	reloaded, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil || reloaded == nil {
		res := uc.toRawMaterialStockResponse(ctx, e, nil)
		return &res, nil
	}
	res := uc.toRawMaterialStockResponse(ctx, reloaded, nil)
	return &res, nil
}

// VoidRawMaterialStockEntry voids an unused SBB document with immutable audit trail.
func (uc *inventoryUseCase) VoidRawMaterialStockEntry(ctx context.Context, id uuid.UUID, reason string) (*RawMaterialStockResponseDTO, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperrors.NewBadRequest("Alasan pembatalan (void) wajib diisi")
	}
	e, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat Form Stok Gudang Bahan Baku")
	}
	if e == nil {
		return nil, apperrors.NewNotFound("Form Stok Gudang Bahan Baku tidak ditemukan")
	}
	if e.Status == domain.RawStockStatusVoided {
		return nil, apperrors.NewConflict("Dokumen sudah berstatus void")
	}
	if e.QtyIssued > 0 || e.QtyReserved > 0 {
		return nil, apperrors.NewConflict("Lot ini sudah digunakan atau dialokasikan ke BPB/LBB sehingga tidak dapat di-void")
	}

	actorName := actor.EmailFrom(ctx)
	now := time.Now()

	err = uc.repo.WithTransaction(ctx, func(txRepo domain.InventoryRepository) error {
		prevRemaining := e.QtyRemaining
		prevPkgs := e.RemainingPackages

		// Reverse stock only if this SBB entry itself added stock (non-LBM / unbooked LBM).
		if e.StockSynced && e.StockAddedBySBB && prevRemaining > 0 {
			revQty := int(math.Round(prevRemaining))
			if revQty > 0 {
				if err := uc.applyTxStockDelta(ctx, txRepo, e.ProductID, e.WarehouseID, -revQty, domain.MovementAdjustment, e.BatchNo, "", e.EntryNo+" (VOID)", "Void Form SBB: "+reason, actorName); err != nil {
					return err
				}
			}
		}

		e.Status = domain.RawStockStatusVoided
		e.QtyRemaining = 0
		e.RemainingPackages = 0
		e.VoidReason = reason
		e.VoidedBy = actorName
		e.VoidedAt = &now

		if err := txRepo.UpdateRawMaterialStockEntry(ctx, e); err != nil {
			return apperrors.NewInternal(err, "Gagal membatalkan Form Stok Gudang Bahan Baku")
		}

		if err := txRepo.CreateRawMaterialStockLog(ctx, &domain.RawMaterialStockLog{
			EntryID:       e.ID,
			EventType:     domain.RawStockEventVoid,
			RefDocumentNo: e.EntryNo,
			QtyDelta:      -prevRemaining,
			PackageDelta:  -prevPkgs,
			BalanceAfter:  0,
			PackagesAfter: 0,
			ActorName:     actorName,
			Notes:         "VOID: " + reason,
			OccurredAt:    now,
		}); err != nil {
			return apperrors.NewInternal(err, "Gagal mencatat log void")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	reloaded, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil || reloaded == nil {
		res := uc.toRawMaterialStockResponse(ctx, e, nil)
		return &res, nil
	}
	res := uc.toRawMaterialStockResponse(ctx, reloaded, nil)
	return &res, nil
}

func (uc *inventoryUseCase) DeleteRawMaterialStockEntry(ctx context.Context, id uuid.UUID) error {
	e, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Gagal memuat Form Stok Gudang Bahan Baku")
	}
	if e == nil {
		return apperrors.NewNotFound("Form Stok Gudang Bahan Baku tidak ditemukan")
	}
	if e.Status != domain.RawStockStatusDraft {
		return apperrors.NewConflict("Hanya dokumen berstatus Draft yang dapat dihapus. Gunakan fitur Void untuk dokumen yang sudah diposting.")
	}
	if err := uc.repo.DeleteRawMaterialStockEntry(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Gagal menghapus draft Form Stok Gudang Bahan Baku")
	}
	return nil
}

func (uc *inventoryUseCase) GetRawMaterialStockEntry(ctx context.Context, id uuid.UUID) (*RawMaterialStockResponseDTO, error) {
	e, err := uc.repo.GetRawMaterialStockEntry(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat detail Form Stok Gudang Bahan Baku")
	}
	if e == nil {
		return nil, apperrors.NewNotFound("Form Stok Gudang Bahan Baku tidak ditemukan")
	}
	res := uc.toRawMaterialStockResponse(ctx, e, nil)
	return &res, nil
}

func (uc *inventoryUseCase) ListRawMaterialStockEntries(ctx context.Context, f domain.RawMaterialStockFilter) ([]RawMaterialStockResponseDTO, error) {
	items, err := uc.repo.ListRawMaterialStockEntries(ctx, f)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar Stok Gudang Bahan Baku")
	}
	whCache := map[uuid.UUID]string{}
	out := make([]RawMaterialStockResponseDTO, 0, len(items))
	for i := range items {
		out = append(out, uc.toRawMaterialStockResponse(ctx, &items[i], whCache))
	}
	return out, nil
}

// ListAvailableRawMaterialLots lists ready-to-issue raw material lots in FEFO order (consumed by BPB & LBB).
func (uc *inventoryUseCase) ListAvailableRawMaterialLots(ctx context.Context, productID, warehouseID *uuid.UUID) ([]RawMaterialStockResponseDTO, error) {
	items, err := uc.repo.ListAvailableRawMaterialLots(ctx, productID, warehouseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar lot bahan baku tersedia")
	}
	whCache := map[uuid.UUID]string{}
	out := make([]RawMaterialStockResponseDTO, 0, len(items))
	for i := range items {
		out = append(out, uc.toRawMaterialStockResponse(ctx, &items[i], whCache))
	}
	return out, nil
}

func (uc *inventoryUseCase) GetRawMaterialStockSummary(ctx context.Context, warehouseID *uuid.UUID) (*RawMaterialStockSummaryDTO, error) {
	items, err := uc.repo.ListRawMaterialStockEntries(ctx, domain.RawMaterialStockFilter{
		WarehouseID: warehouseID,
	})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menghitung ringkasan Stok Gudang Bahan Baku")
	}
	sum := &RawMaterialStockSummaryDTO{}
	nowStr := today()
	for _, e := range items {
		if e.Status == domain.RawStockStatusVoided {
			continue
		}
		sum.TotalEntries++
		switch e.Status {
		case domain.RawStockStatusDraft:
			sum.DraftCount++
		case domain.RawStockStatusAvailable:
			sum.AvailableCount++
			sum.TotalAvailableQty += e.QtyRemaining
			sum.TotalAvailablePackages += e.RemainingPackages
			if strings.EqualFold(e.OrganicStatus, "Organik") {
				sum.OrganicAvailableQty += e.QtyRemaining
			} else {
				sum.ConventionalAvailableQty += e.QtyRemaining
			}
			expStatus, _ := BatchStatus(e.ExpiryDate, nowStr)
			if expStatus == "expiring_soon" {
				sum.ExpiringSoonCount++
			} else if expStatus == "expired" {
				sum.ExpiredCount++
			}
		case domain.RawStockStatusQuarantine:
			sum.QuarantineCount++
			sum.TotalQuarantineQty += e.QtyRemaining
		case domain.RawStockStatusDepleted:
			sum.DepletedCount++
		case domain.RawStockStatusRejected:
			sum.RejectedCount++
		}
	}
	return sum, nil
}
