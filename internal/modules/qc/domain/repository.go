package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ListInspectionFilter struct {
	TargetType        *TargetType
	TargetID          *uuid.UUID
	ProductionOrderID *uuid.UUID
	ProductionStepID  *uuid.UUID
	BatchNo           string
	ProductID         *uuid.UUID
	Status            *InspectionStatus
	StartDate         *time.Time
	EndDate           *time.Time
	Page              int
	PerPage           int
}

type QCRepository interface {
	// Parameter
	CreateParameter(ctx context.Context, param *QCParameter) error
	GetParameterByID(ctx context.Context, id uuid.UUID) (*QCParameter, error)
	ListParameters(ctx context.Context, productID *uuid.UUID, processCode string) ([]QCParameter, error)
	UpdateParameter(ctx context.Context, param *QCParameter) error
	DeleteParameter(ctx context.Context, id uuid.UUID) error

	// Inspection
	CreateInspection(ctx context.Context, insp *QCInspection) error
	GetInspectionByID(ctx context.Context, id uuid.UUID) (*QCInspection, error)
	GetInspectionByNo(ctx context.Context, inspectionNo string) (*QCInspection, error)
	ListInspections(ctx context.Context, filter ListInspectionFilter) ([]QCInspection, int64, error)
	UpdateInspection(ctx context.Context, insp *QCInspection) error
	GetLatestPassedInspection(ctx context.Context, targetType TargetType, targetID uuid.UUID, processCode string) (*QCInspection, error)
	GetInspectionCountToday(ctx context.Context, datePrefix string) (int64, error)

	// Stats / Reports
	GetInspectionStats(ctx context.Context) (total int64, passed int64, failed int64, voided int64, err error)

	// Hygiene (CPH)
	CreateHygieneCheck(ctx context.Context, check *PersonalHygieneCheck) error
	ListHygieneChecks(ctx context.Context, date string, line string) ([]PersonalHygieneCheck, error)
	DeleteHygieneCheck(ctx context.Context, id uuid.UUID) error

	// Pre-Op Sanitation (SSO)
	CreatePreOpCheck(ctx context.Context, check *PreOpSanitationCheck) error
	ListPreOpChecks(ctx context.Context, date string, room string) ([]PreOpSanitationCheck, error)
	DeletePreOpCheck(ctx context.Context, id uuid.UUID) error

	// Packaging Quality Check (PBK — QC-003)
	CreatePackagingCheck(ctx context.Context, check *PackagingQualityCheck) error
	GetPackagingCheckByID(ctx context.Context, id uuid.UUID) (*PackagingQualityCheck, error)
	ListPackagingChecks(ctx context.Context, month string, lbmNo string) ([]PackagingQualityCheck, error)
	UpdatePackagingCheck(ctx context.Context, check *PackagingQualityCheck) error
	DeletePackagingCheck(ctx context.Context, id uuid.UUID) error

	// Supplier Armada Check (CAS — QC-007)
	CreateArmadaCheck(ctx context.Context, check *SupplierArmadaCheck) error
	GetArmadaCheckByID(ctx context.Context, id uuid.UUID) (*SupplierArmadaCheck, error)
	ListArmadaChecks(ctx context.Context, date string, lbmNo string) ([]SupplierArmadaCheck, error)
	UpdateArmadaCheck(ctx context.Context, check *SupplierArmadaCheck) error
	DeleteArmadaCheck(ctx context.Context, id uuid.UUID) error

	// Raw Material Incoming Check (PBD — QC-002-PBD)
	CreateIncomingCheck(ctx context.Context, check *RawMaterialIncomingCheck) error
	GetIncomingCheckByID(ctx context.Context, id uuid.UUID) (*RawMaterialIncomingCheck, error)
	ListIncomingChecks(ctx context.Context, month string, date string, lbmNo string) ([]RawMaterialIncomingCheck, error)
	UpdateIncomingCheck(ctx context.Context, check *RawMaterialIncomingCheck) error
	DeleteIncomingCheck(ctx context.Context, id uuid.UUID) error

	// Sieving Monitoring (MNA — Form QC-009-MNA.01)
	CreateSievingLog(ctx context.Context, log *SievingMonitoringLog) error
	GetSievingLogByID(ctx context.Context, id uuid.UUID) (*SievingMonitoringLog, error)
	ListSievingLogs(ctx context.Context, month string, category string, machineNo string, batchNo string) ([]SievingMonitoringLog, error)
	UpdateSievingLog(ctx context.Context, log *SievingMonitoringLog) error
	DeleteSievingLog(ctx context.Context, id uuid.UUID) error

	// Magnet Trap (CMT — Form QC-038-CMT)
	CreateMagnetTrapLog(ctx context.Context, log *MagnetTrapLog) error
	GetMagnetTrapLogByID(ctx context.Context, id uuid.UUID) (*MagnetTrapLog, error)
	ListMagnetTrapLogs(ctx context.Context, month string, room string, machineNo string) ([]MagnetTrapLog, error)
	UpdateMagnetTrapLog(ctx context.Context, log *MagnetTrapLog) error
	DeleteMagnetTrapLog(ctx context.Context, id uuid.UUID) error

	// Mesh Bolt Monitoring (MNM / MNB / MBB — Form QC-011)
	CreateMeshBoltLog(ctx context.Context, log *MeshBoltMonitoringLog) error
	GetMeshBoltLogByID(ctx context.Context, id uuid.UUID) (*MeshBoltMonitoringLog, error)
	ListMeshBoltLogs(ctx context.Context, month string, room string, machineNo string) ([]MeshBoltMonitoringLog, error)
	UpdateMeshBoltLog(ctx context.Context, log *MeshBoltMonitoringLog) error
	DeleteMeshBoltLog(ctx context.Context, id uuid.UUID) error

	// Filter Mesh Monitoring (MNP - Gula Cair & MMV - VCO — Form QC-011-MNP / MMV)
	CreateFilterMeshLog(ctx context.Context, log *FilterMeshMonitoringLog) error
	GetFilterMeshLogByID(ctx context.Context, id uuid.UUID) (*FilterMeshMonitoringLog, error)
	ListFilterMeshLogs(ctx context.Context, month string, category string, location string) ([]FilterMeshMonitoringLog, error)
	UpdateFilterMeshLog(ctx context.Context, log *FilterMeshMonitoringLog) error
	DeleteFilterMeshLog(ctx context.Context, id uuid.UUID) error

	// Brontol Blade Monitoring (MPB — Form QC-011-MPB.01)
	CreateBrontolBladeLog(ctx context.Context, log *BrontolBladeMonitoringLog) error
	GetBrontolBladeLogByID(ctx context.Context, id uuid.UUID) (*BrontolBladeMonitoringLog, error)
	ListBrontolBladeLogs(ctx context.Context, month string, productCode string) ([]BrontolBladeMonitoringLog, error)
	UpdateBrontolBladeLog(ctx context.Context, log *BrontolBladeMonitoringLog) error
	DeleteBrontolBladeLog(ctx context.Context, id uuid.UUID) error

	// Oven Monitoring (MNO — Form QC-012-MNO.05)
	CreateOvenLog(ctx context.Context, log *OvenMonitoringLog) error
	GetOvenLogByID(ctx context.Context, id uuid.UUID) (*OvenMonitoringLog, error)
	ListOvenLogs(ctx context.Context, month string, ovenNo string, batchNo string) ([]OvenMonitoringLog, error)
	UpdateOvenLog(ctx context.Context, log *OvenMonitoringLog) error
	DeleteOvenLog(ctx context.Context, id uuid.UUID) error

	// Needle Monitoring (MNJ — Form QC-011-MNJ.01)
	CreateNeedleLog(ctx context.Context, log *NeedleMonitoringLog) error
	GetNeedleLogByID(ctx context.Context, id uuid.UUID) (*NeedleMonitoringLog, error)
	ListNeedleLogs(ctx context.Context, month string, machineNo string) ([]NeedleMonitoringLog, error)
	UpdateNeedleLog(ctx context.Context, log *NeedleMonitoringLog) error
	DeleteNeedleLog(ctx context.Context, id uuid.UUID) error

	// Scissor Monitoring (MNG — Form QC-011-MNJ.01 Sheet MNG)
	CreateScissorLog(ctx context.Context, log *ScissorMonitoringLog) error
	GetScissorLogByID(ctx context.Context, id uuid.UUID) (*ScissorMonitoringLog, error)
	ListScissorLogs(ctx context.Context, month string, location string) ([]ScissorMonitoringLog, error)
	UpdateScissorLog(ctx context.Context, log *ScissorMonitoringLog) error
	DeleteScissorLog(ctx context.Context, id uuid.UUID) error

	// Metal Detector Cleaning (CPP — Form PRO-007-CPP)
	CreateMetalCleaningLog(ctx context.Context, log *MetalDetectorCleaningLog) error
	GetMetalCleaningLogByID(ctx context.Context, id uuid.UUID) (*MetalDetectorCleaningLog, error)
	ListMetalCleaningLogs(ctx context.Context, month string, machineNo string, location string) ([]MetalDetectorCleaningLog, error)
	UpdateMetalCleaningLog(ctx context.Context, log *MetalDetectorCleaningLog) error
	DeleteMetalCleaningLog(ctx context.Context, id uuid.UUID) error

	// Metal Detector Verification (VMD — Form QC-017-VMD Rev 03 2026 / Rev 01)
	CreateMetalVerificationLog(ctx context.Context, log *MetalDetectorVerificationLog) error
	GetMetalVerificationLogByID(ctx context.Context, id uuid.UUID) (*MetalDetectorVerificationLog, error)
	ListMetalVerificationLogs(ctx context.Context, month string, productType string, location string) ([]MetalDetectorVerificationLog, error)
	UpdateMetalVerificationLog(ctx context.Context, log *MetalDetectorVerificationLog) error
	DeleteMetalVerificationLog(ctx context.Context, id uuid.UUID) error

	// Finished Goods Inspection (PJD — Form QC-004-PJD.05 Rev 05)
	CreateFinishedGoodsLog(ctx context.Context, log *FinishedGoodsInspectionLog) error
	GetFinishedGoodsLogByID(ctx context.Context, id uuid.UUID) (*FinishedGoodsInspectionLog, error)
	ListFinishedGoodsLogs(ctx context.Context, month string, productName string, batchNo string, decision string) ([]FinishedGoodsInspectionLog, error)
	UpdateFinishedGoodsLog(ctx context.Context, log *FinishedGoodsInspectionLog) error
	DeleteFinishedGoodsLog(ctx context.Context, id uuid.UUID) error

	// Packaging Label Check (PLK — Form QC-016-PLK)
	CreatePackagingLabelCheck(ctx context.Context, log *PackagingLabelCheckLog) error
	GetPackagingLabelCheckByID(ctx context.Context, id uuid.UUID) (*PackagingLabelCheckLog, error)
	ListPackagingLabelChecks(ctx context.Context, month string, batchNo string, decision string) ([]PackagingLabelCheckLog, error)
	UpdatePackagingLabelCheck(ctx context.Context, log *PackagingLabelCheckLog) error
	DeletePackagingLabelCheck(ctx context.Context, id uuid.UUID) error

	// Container Checklist (CCO — Form QC-008-CCO.02)
	CreateContainerChecklist(ctx context.Context, log *ContainerChecklistLog) error
	GetContainerChecklistByID(ctx context.Context, id uuid.UUID) (*ContainerChecklistLog, error)
	ListContainerChecklists(ctx context.Context, month string, containerNo string, destination string, decision string) ([]ContainerChecklistLog, error)
	UpdateContainerChecklist(ctx context.Context, log *ContainerChecklistLog) error
	DeleteContainerChecklist(ctx context.Context, id uuid.UUID) error

	// Brontol & Reject Return Reports (Form REP-013-KSB)
	CreateBrontolReport(ctx context.Context, report *BrontolReport) error
	GetBrontolReportByID(ctx context.Context, id uuid.UUID) (*BrontolReport, error)
	GetBrontolReportByNo(ctx context.Context, reportNo string) (*BrontolReport, error)
	ListBrontolReports(ctx context.Context, month, stage, action, status, search string) ([]BrontolReport, error)
	UpdateBrontolReport(ctx context.Context, report *BrontolReport) error
	DeleteBrontolReport(ctx context.Context, id uuid.UUID) error

	CreateRejectReturnReport(ctx context.Context, ret *RejectReturnReport) error
	GetRejectReturnReportByID(ctx context.Context, id uuid.UUID) (*RejectReturnReport, error)
	GetRejectReturnReportByBrontolID(ctx context.Context, brontolReportID uuid.UUID) (*RejectReturnReport, error)
	ListRejectReturnReports(ctx context.Context, month, status, search string) ([]RejectReturnReport, error)
	UpdateRejectReturnReport(ctx context.Context, ret *RejectReturnReport) error
	DeleteRejectReturnReport(ctx context.Context, id uuid.UUID) error

	GetBrontolStats(ctx context.Context) (totalCount int64, totalQuantityKg float64, returnCount int64, returnQtyKg float64, reworkCount int64, err error)

	// Document numbering helper
	CountLogsByPrefix(ctx context.Context, tableName string, prefix string) (int64, error)
}
