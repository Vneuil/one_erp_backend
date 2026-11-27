package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/divinecoid/one-backend/internal/shared/docseq"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

type QCUseCase interface {
	// Parameters
	ListParameters(ctx context.Context, productID *uuid.UUID, processCode string) ([]QCParameterResponseDTO, error)
	GetParameterByID(ctx context.Context, id uuid.UUID) (*QCParameterResponseDTO, error)
	CreateParameter(ctx context.Context, dto CreateQCParameterDTO) (*QCParameterResponseDTO, error)
	UpdateParameter(ctx context.Context, id uuid.UUID, dto UpdateQCParameterDTO) (*QCParameterResponseDTO, error)
	DeleteParameter(ctx context.Context, id uuid.UUID) error
	SeedDefaultParameters(ctx context.Context) error

	// Inspections
	CreateInspection(ctx context.Context, inspectorID uuid.UUID, inspectorName string, dto CreateQCInspectionDTO) (*QCInspectionResponseDTO, error)
	GetInspectionByID(ctx context.Context, id uuid.UUID) (*QCInspectionResponseDTO, error)
	ListInspections(ctx context.Context, filter domain.ListInspectionFilter) ([]QCInspectionResponseDTO, int64, error)
	VoidInspection(ctx context.Context, id uuid.UUID, actorID uuid.UUID, dto VoidQCInspectionDTO) (*QCInspectionResponseDTO, error)
	GetStats(ctx context.Context) (*QCStatsResponseDTO, error)

	// Hygiene (CPH)
	CreateHygieneCheck(ctx context.Context, dto CreateHygieneCheckDTO) (*domain.PersonalHygieneCheck, error)
	ListHygieneChecks(ctx context.Context, date string, line string) ([]domain.PersonalHygieneCheck, error)
	DeleteHygieneCheck(ctx context.Context, id uuid.UUID) error

	// Pre-Op Sanitation (SSO)
	CreatePreOpCheck(ctx context.Context, dto CreatePreOpCheckDTO) (*domain.PreOpSanitationCheck, error)
	ListPreOpChecks(ctx context.Context, date string, room string) ([]domain.PreOpSanitationCheck, error)
	DeletePreOpCheck(ctx context.Context, id uuid.UUID) error

	// Packaging Quality Check (PBK — QC-003)
	CreatePackagingCheck(ctx context.Context, dto CreatePackagingCheckDTO) (*PackagingCheckResponseDTO, error)
	GetPackagingCheckByID(ctx context.Context, id uuid.UUID) (*PackagingCheckResponseDTO, error)
	ListPackagingChecks(ctx context.Context, month string, lbmNo string) ([]PackagingCheckResponseDTO, error)
	UpdatePackagingCheck(ctx context.Context, id uuid.UUID, dto UpdatePackagingCheckDTO) (*PackagingCheckResponseDTO, error)
	DeletePackagingCheck(ctx context.Context, id uuid.UUID) error

	// Supplier Armada Check (CAS — QC-007)
	CreateArmadaCheck(ctx context.Context, dto CreateArmadaCheckDTO) (*ArmadaCheckResponseDTO, error)
	GetArmadaCheckByID(ctx context.Context, id uuid.UUID) (*ArmadaCheckResponseDTO, error)
	ListArmadaChecks(ctx context.Context, date string, lbmNo string) ([]ArmadaCheckResponseDTO, error)
	UpdateArmadaCheck(ctx context.Context, id uuid.UUID, dto UpdateArmadaCheckDTO) (*ArmadaCheckResponseDTO, error)
	DeleteArmadaCheck(ctx context.Context, id uuid.UUID) error

	// Raw Material Incoming Check (PBD — QC-002-PBD)
	CreateIncomingCheck(ctx context.Context, dto CreateIncomingCheckDTO) (*IncomingCheckResponseDTO, error)
	GetIncomingCheckByID(ctx context.Context, id uuid.UUID) (*IncomingCheckResponseDTO, error)
	ListIncomingChecks(ctx context.Context, month string, date string, lbmNo string) ([]IncomingCheckResponseDTO, error)
	UpdateIncomingCheck(ctx context.Context, id uuid.UUID, dto UpdateIncomingCheckDTO) (*IncomingCheckResponseDTO, error)
	DeleteIncomingCheck(ctx context.Context, id uuid.UUID) error

	// Sieving Monitoring (MNA — Form QC-009-MNA.01)
	CreateSievingLog(ctx context.Context, dto CreateSievingLogDTO) (*domain.SievingMonitoringLog, error)
	GetSievingLogByID(ctx context.Context, id uuid.UUID) (*domain.SievingMonitoringLog, error)
	ListSievingLogs(ctx context.Context, month string, category string, machineNo string, batchNo string) ([]domain.SievingMonitoringLog, error)
	UpdateSievingLog(ctx context.Context, id uuid.UUID, dto UpdateSievingLogDTO) (*domain.SievingMonitoringLog, error)
	DeleteSievingLog(ctx context.Context, id uuid.UUID) error

	// Magnet Trap (CMT — Form QC-038-CMT)
	CreateMagnetTrapLog(ctx context.Context, dto CreateMagnetTrapLogDTO) (*domain.MagnetTrapLog, error)
	GetMagnetTrapLogByID(ctx context.Context, id uuid.UUID) (*domain.MagnetTrapLog, error)
	ListMagnetTrapLogs(ctx context.Context, month string, room string, machineNo string) ([]domain.MagnetTrapLog, error)
	UpdateMagnetTrapLog(ctx context.Context, id uuid.UUID, dto UpdateMagnetTrapLogDTO) (*domain.MagnetTrapLog, error)
	DeleteMagnetTrapLog(ctx context.Context, id uuid.UUID) error

	// Mesh Bolt Monitoring (MNM — Form QC-010-MNM.01)
	CreateMeshBoltLog(ctx context.Context, dto CreateMeshBoltLogDTO) (*domain.MeshBoltMonitoringLog, error)
	GetMeshBoltLogByID(ctx context.Context, id uuid.UUID) (*domain.MeshBoltMonitoringLog, error)
	ListMeshBoltLogs(ctx context.Context, month string, room string, machineNo string) ([]domain.MeshBoltMonitoringLog, error)
	UpdateMeshBoltLog(ctx context.Context, id uuid.UUID, dto UpdateMeshBoltLogDTO) (*domain.MeshBoltMonitoringLog, error)
	DeleteMeshBoltLog(ctx context.Context, id uuid.UUID) error

	// Filter Mesh Monitoring (MNP - Gula Cair & MMV - VCO — Form QC-011-MNP / MMV)
	CreateFilterMeshLog(ctx context.Context, dto CreateFilterMeshLogDTO) (*domain.FilterMeshMonitoringLog, error)
	GetFilterMeshLogByID(ctx context.Context, id uuid.UUID) (*domain.FilterMeshMonitoringLog, error)
	ListFilterMeshLogs(ctx context.Context, month string, category string, location string) ([]domain.FilterMeshMonitoringLog, error)
	UpdateFilterMeshLog(ctx context.Context, id uuid.UUID, dto UpdateFilterMeshLogDTO) (*domain.FilterMeshMonitoringLog, error)
	DeleteFilterMeshLog(ctx context.Context, id uuid.UUID) error

	// Brontol Blade Monitoring (MPB — Form QC-011-MPB.01)
	CreateBrontolBladeLog(ctx context.Context, dto CreateBrontolBladeLogDTO) (*domain.BrontolBladeMonitoringLog, error)
	GetBrontolBladeLogByID(ctx context.Context, id uuid.UUID) (*domain.BrontolBladeMonitoringLog, error)
	ListBrontolBladeLogs(ctx context.Context, month string, productCode string) ([]domain.BrontolBladeMonitoringLog, error)
	UpdateBrontolBladeLog(ctx context.Context, id uuid.UUID, dto UpdateBrontolBladeLogDTO) (*domain.BrontolBladeMonitoringLog, error)
	DeleteBrontolBladeLog(ctx context.Context, id uuid.UUID) error

	// Oven Monitoring (MNO — Form QC-012-MNO.05)
	CreateOvenLog(ctx context.Context, dto CreateOvenLogDTO) (*domain.OvenMonitoringLog, error)
	GetOvenLogByID(ctx context.Context, id uuid.UUID) (*domain.OvenMonitoringLog, error)
	ListOvenLogs(ctx context.Context, month string, ovenNo string, batchNo string) ([]domain.OvenMonitoringLog, error)
	UpdateOvenLog(ctx context.Context, id uuid.UUID, dto UpdateOvenLogDTO) (*domain.OvenMonitoringLog, error)
	DeleteOvenLog(ctx context.Context, id uuid.UUID) error

	// Needle Monitoring (MNJ — Form QC-011-MNJ.01)
	CreateNeedleLog(ctx context.Context, dto CreateNeedleLogDTO) (*domain.NeedleMonitoringLog, error)
	GetNeedleLogByID(ctx context.Context, id uuid.UUID) (*domain.NeedleMonitoringLog, error)
	ListNeedleLogs(ctx context.Context, month string, machineNo string) ([]domain.NeedleMonitoringLog, error)
	UpdateNeedleLog(ctx context.Context, id uuid.UUID, dto UpdateNeedleLogDTO) (*domain.NeedleMonitoringLog, error)
	DeleteNeedleLog(ctx context.Context, id uuid.UUID) error

	// Scissor Monitoring (MNG — Form QC-011-MNJ.01 Sheet MNG)
	CreateScissorLog(ctx context.Context, dto CreateScissorLogDTO) (*domain.ScissorMonitoringLog, error)
	GetScissorLogByID(ctx context.Context, id uuid.UUID) (*domain.ScissorMonitoringLog, error)
	ListScissorLogs(ctx context.Context, month string, location string) ([]domain.ScissorMonitoringLog, error)
	UpdateScissorLog(ctx context.Context, id uuid.UUID, dto UpdateScissorLogDTO) (*domain.ScissorMonitoringLog, error)
	DeleteScissorLog(ctx context.Context, id uuid.UUID) error

	// Metal Detector Cleaning (CPP — Form PRO-007-CPP)
	CreateMetalCleaningLog(ctx context.Context, dto CreateMetalCleaningLogDTO) (*domain.MetalDetectorCleaningLog, error)
	GetMetalCleaningLogByID(ctx context.Context, id uuid.UUID) (*domain.MetalDetectorCleaningLog, error)
	ListMetalCleaningLogs(ctx context.Context, month string, machineNo string, location string) ([]domain.MetalDetectorCleaningLog, error)
	UpdateMetalCleaningLog(ctx context.Context, id uuid.UUID, dto UpdateMetalCleaningLogDTO) (*domain.MetalDetectorCleaningLog, error)
	DeleteMetalCleaningLog(ctx context.Context, id uuid.UUID) error

	// Metal Detector Verification (VMD — Form QC-017-VMD)
	CreateMetalVerificationLog(ctx context.Context, dto CreateMetalVerificationLogDTO) (*domain.MetalDetectorVerificationLog, error)
	GetMetalVerificationLogByID(ctx context.Context, id uuid.UUID) (*domain.MetalDetectorVerificationLog, error)
	ListMetalVerificationLogs(ctx context.Context, month string, productType string, location string) ([]domain.MetalDetectorVerificationLog, error)
	UpdateMetalVerificationLog(ctx context.Context, id uuid.UUID, dto UpdateMetalVerificationLogDTO) (*domain.MetalDetectorVerificationLog, error)
	DeleteMetalVerificationLog(ctx context.Context, id uuid.UUID) error

	// Finished Goods Inspection (PJD — Form QC-004-PJD.05 Rev 05)
	CreateFinishedGoodsLog(ctx context.Context, dto CreateFinishedGoodsDTO) (*domain.FinishedGoodsInspectionLog, error)
	GetFinishedGoodsLogByID(ctx context.Context, id uuid.UUID) (*domain.FinishedGoodsInspectionLog, error)
	ListFinishedGoodsLogs(ctx context.Context, month string, productName string, batchNo string, decision string) ([]domain.FinishedGoodsInspectionLog, error)
	UpdateFinishedGoodsLog(ctx context.Context, id uuid.UUID, dto UpdateFinishedGoodsDTO) (*domain.FinishedGoodsInspectionLog, error)
	DeleteFinishedGoodsLog(ctx context.Context, id uuid.UUID) error

	// Packaging Label Check (PLK — Form QC-016-PLK)
	CreatePackagingLabelCheck(ctx context.Context, dto CreatePackagingLabelCheckDTO) (*domain.PackagingLabelCheckLog, error)
	GetPackagingLabelCheckByID(ctx context.Context, id uuid.UUID) (*domain.PackagingLabelCheckLog, error)
	ListPackagingLabelChecks(ctx context.Context, month string, batchNo string, decision string) ([]domain.PackagingLabelCheckLog, error)
	UpdatePackagingLabelCheck(ctx context.Context, id uuid.UUID, dto UpdatePackagingLabelCheckDTO) (*domain.PackagingLabelCheckLog, error)
	DeletePackagingLabelCheck(ctx context.Context, id uuid.UUID) error

	// Container Checklist (CCO — Form QC-008-CCO.02)
	CreateContainerChecklist(ctx context.Context, dto CreateContainerChecklistDTO) (*domain.ContainerChecklistLog, error)
	GetContainerChecklistByID(ctx context.Context, id uuid.UUID) (*domain.ContainerChecklistLog, error)
	ListContainerChecklists(ctx context.Context, month string, containerNo string, destination string, decision string) ([]domain.ContainerChecklistLog, error)
	UpdateContainerChecklist(ctx context.Context, id uuid.UUID, dto UpdateContainerChecklistDTO) (*domain.ContainerChecklistLog, error)
	DeleteContainerChecklist(ctx context.Context, id uuid.UUID) error

	// Brontol & Reject Return Reports (Form REP-013-KSB)
	CreateBrontolReport(ctx context.Context, dto CreateBrontolReportDTO) (*domain.BrontolReport, error)
	GetBrontolReportByID(ctx context.Context, id uuid.UUID) (*domain.BrontolReport, error)
	ListBrontolReports(ctx context.Context, month, stage, action, status, search string) ([]domain.BrontolReport, error)
	UpdateBrontolReport(ctx context.Context, id uuid.UUID, dto UpdateBrontolReportDTO) (*domain.BrontolReport, error)
	DeleteBrontolReport(ctx context.Context, id uuid.UUID) error

	CreateRejectReturn(ctx context.Context, brontolID uuid.UUID, dto CreateRejectReturnDTO) (*domain.RejectReturnReport, error)
	GetRejectReturnByID(ctx context.Context, id uuid.UUID) (*domain.RejectReturnReport, error)
	ListRejectReturns(ctx context.Context, month, status, search string) ([]domain.RejectReturnReport, error)
	UpdateRejectReturn(ctx context.Context, id uuid.UUID, dto UpdateRejectReturnDTO) (*domain.RejectReturnReport, error)
	DeleteRejectReturn(ctx context.Context, id uuid.UUID) error

	GetBrontolStats(ctx context.Context) (*BrontolStatsResponseDTO, error)
	GetBrontolStockCard(ctx context.Context, month string, productCode string) ([]BrontolStockCardItemDTO, error)
	AutoCreateBrontolOnQCFailure(ctx context.Context, trigger AutoBrontolTrigger) (*domain.BrontolReport, error)
}

type qcUseCase struct {
	repo domain.QCRepository
}

func NewQCUseCase(repo domain.QCRepository) QCUseCase {
	return &qcUseCase{repo: repo}
}

// ─────────────────────────────────────────────────────────────────────────────
// Parameter Management
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) ListParameters(ctx context.Context, productID *uuid.UUID, processCode string) ([]QCParameterResponseDTO, error) {
	params, err := uc.repo.ListParameters(ctx, productID, processCode)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	res := make([]QCParameterResponseDTO, 0, len(params))
	for _, p := range params {
		res = append(res, mapParameterToDTO(&p))
	}
	return res, nil
}

func (uc *qcUseCase) GetParameterByID(ctx context.Context, id uuid.UUID) (*QCParameterResponseDTO, error) {
	p, err := uc.repo.GetParameterByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Parameter QC tidak ditemukan")
	}
	dto := mapParameterToDTO(p)
	return &dto, nil
}

func (uc *qcUseCase) CreateParameter(ctx context.Context, dto CreateQCParameterDTO) (*QCParameterResponseDTO, error) {
	if strings.TrimSpace(dto.ProcessCode) == "" || strings.TrimSpace(dto.Name) == "" {
		return nil, apperrors.NewBadRequest("Kode proses dan nama parameter wajib diisi")
	}

	p := &domain.QCParameter{
		ProductID:     dto.ProductID,
		ProcessCode:   strings.ToUpper(strings.TrimSpace(dto.ProcessCode)),
		Name:          strings.TrimSpace(dto.Name),
		ParamType:     dto.ParamType,
		UnitOfMeasure: strings.TrimSpace(dto.UnitOfMeasure),
		MinValue:      dto.MinValue,
		MaxValue:      dto.MaxValue,
		TargetValue:   dto.TargetValue,
		Tolerance:     dto.Tolerance,
		IsGate:        dto.IsGate,
		IsMandatory:   dto.IsMandatory,
		Description:   strings.TrimSpace(dto.Description),
	}

	if err := uc.repo.CreateParameter(ctx, p); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	dtoRes := mapParameterToDTO(p)
	return &dtoRes, nil
}

func (uc *qcUseCase) UpdateParameter(ctx context.Context, id uuid.UUID, dto UpdateQCParameterDTO) (*QCParameterResponseDTO, error) {
	p, err := uc.repo.GetParameterByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Parameter QC tidak ditemukan")
	}

	if strings.TrimSpace(dto.ProcessCode) != "" {
		p.ProcessCode = strings.ToUpper(strings.TrimSpace(dto.ProcessCode))
	}
	if strings.TrimSpace(dto.Name) != "" {
		p.Name = strings.TrimSpace(dto.Name)
	}
	if dto.ParamType != "" {
		p.ParamType = dto.ParamType
	}
	p.UnitOfMeasure = strings.TrimSpace(dto.UnitOfMeasure)
	p.MinValue = dto.MinValue
	p.MaxValue = dto.MaxValue
	p.TargetValue = dto.TargetValue
	p.Tolerance = dto.Tolerance
	p.IsGate = dto.IsGate
	p.IsMandatory = dto.IsMandatory
	p.Description = strings.TrimSpace(dto.Description)

	if err := uc.repo.UpdateParameter(ctx, p); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	dtoRes := mapParameterToDTO(p)
	return &dtoRes, nil
}

func (uc *qcUseCase) DeleteParameter(ctx context.Context, id uuid.UUID) error {
	_, err := uc.repo.GetParameterByID(ctx, id)
	if err != nil {
		return apperrors.NewNotFound("Parameter QC tidak ditemukan")
	}
	return uc.repo.DeleteParameter(ctx, id)
}

func (uc *qcUseCase) SeedDefaultParameters(ctx context.Context) error {
	existing, err := uc.repo.ListParameters(ctx, nil, "")
	if err == nil && len(existing) > 0 {
		return nil
	}

	maxAir := 2.0
	targetBrix := 70.0
	tolBrix := 2.0

	defaults := []domain.QCParameter{
		{
			ProcessCode:   "CEK_AIR",
			Name:          "Kadar Air (Moisture)",
			ParamType:     domain.ParamTypeNumeric,
			UnitOfMeasure: "%",
			MaxValue:      &maxAir,
			IsGate:        true,
			IsMandatory:   true,
			Description:   "BRL-01: Kadar air maksimal 2.0% untuk gula semut (<=2% skip oven; >2% wajib oven)",
		},
		{
			ProcessCode:   "METAL_DETECTOR",
			Name:          "Uji Metal Detector",
			ParamType:     domain.ParamTypeBoolean,
			UnitOfMeasure: "boolean",
			IsGate:        true,
			IsMandatory:   true,
			Description:   "BRL-03: Batch tidak boleh masuk FG / Pack tanpa hasil uji metal detector lulus",
		},
		{
			ProcessCode:   "BRIX",
			Name:          "Kadar Brix Gula Cair",
			ParamType:     domain.ParamTypeNumeric,
			UnitOfMeasure: "°Bx",
			TargetValue:   &targetBrix,
			Tolerance:     &tolBrix,
			IsGate:        true,
			IsMandatory:   true,
			Description:   "BRL-04: Target Brix 70 ± 2°Bx untuk gula cair sebelum proses filling",
		},
	}

	for _, p := range defaults {
		_ = uc.repo.CreateParameter(ctx, &p)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Inspection
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateInspection(ctx context.Context, inspectorID uuid.UUID, inspectorName string, dto CreateQCInspectionDTO) (*QCInspectionResponseDTO, error) {
	if len(dto.Items) == 0 {
		return nil, apperrors.NewBadRequest("Hasil inspeksi harus memiliki minimal 1 parameter yang diuji")
	}
	if strings.TrimSpace(dto.BatchNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor batch wajib diisi")
	}

	now := time.Now()
	inspNo, err := docseq.Next(ctx, uc.repo, "QC", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor inspeksi QC")
	}

	insp := &domain.QCInspection{
		InspectionNo:      inspNo,
		TargetType:        dto.TargetType,
		TargetID:          dto.TargetID,
		ProductionOrderID: dto.ProductionOrderID,
		ProductionStepID:  dto.ProductionStepID,
		BatchNo:           strings.TrimSpace(dto.BatchNo),
		ProductID:         dto.ProductID,
		InspectorID:       inspectorID,
		InspectorName:     inspectorName,
		InspectedAt:       now,
		TestingTool:       strings.TrimSpace(dto.TestingTool),
		Iteration:         1,
		OverallStatus:     domain.QCStatusPassed,
		Notes:             strings.TrimSpace(dto.Notes),
		SugarCondition:    strings.TrimSpace(dto.SugarCondition),
		PackageCondition:  strings.TrimSpace(dto.PackageCondition),
		Items:             make([]domain.QCInspectionItem, 0, len(dto.Items)),
	}

	hasFailed := false

	for _, itmDTO := range dto.Items {
		param, err := uc.repo.GetParameterByID(ctx, itmDTO.ParameterID)
		if err != nil {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Parameter ID %s tidak valid", itmDTO.ParameterID))
		}

		isPassed := true
		if param.ParamType == domain.ParamTypeNumeric {
			if itmDTO.NumericValue == nil {
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Nilai angka wajib diisi untuk parameter %s", param.Name))
			}
			val := *itmDTO.NumericValue
			if param.MinValue != nil && val < *param.MinValue {
				isPassed = false
			}
			if param.MaxValue != nil && val > *param.MaxValue {
				isPassed = false
			}
			if param.TargetValue != nil && param.Tolerance != nil {
				diff := math.Abs(val - *param.TargetValue)
				if diff > *param.Tolerance {
					isPassed = false
				}
			}
		} else if param.ParamType == domain.ParamTypeBoolean {
			if itmDTO.BooleanValue == nil || !*itmDTO.BooleanValue {
				isPassed = false
			}
		}

		if !isPassed {
			hasFailed = true
		}

		insp.Items = append(insp.Items, domain.QCInspectionItem{
			ParameterID:   param.ID,
			ParameterName: param.Name,
			ParamType:     param.ParamType,
			NumericValue:  itmDTO.NumericValue,
			BooleanValue:  itmDTO.BooleanValue,
			IsPassed:      isPassed,
			Notes:         strings.TrimSpace(itmDTO.Notes),
		})
	}

	if hasFailed {
		insp.OverallStatus = domain.QCStatusFailed
	} else {
		insp.OverallStatus = domain.QCStatusPassed
	}

	if err := uc.repo.CreateInspection(ctx, insp); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	if insp.OverallStatus == domain.QCStatusFailed {
		action := domain.ActionRework
		if insp.TargetType == domain.TargetGoodsReceipt {
			action = domain.ActionReturnToSupplier
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageProductionStep,
			SourceInspectionID: &insp.ID,
			SourceInspectionNo: insp.InspectionNo,
			ProductionOrderID:  insp.ProductionOrderID,
			BatchNo:            insp.BatchNo,
			ProductID:          &insp.ProductID,
			DefectReason:       fmt.Sprintf("Gagal inspeksi QC parameter (%s)", insp.InspectionNo),
			ActionDisposition:  action,
			QCInspectorName:    insp.InspectorName,
			Notes:              insp.Notes,
		})
	}

	dtoRes := mapInspectionToDTO(insp)
	return &dtoRes, nil
}

func (uc *qcUseCase) GetInspectionByID(ctx context.Context, id uuid.UUID) (*QCInspectionResponseDTO, error) {
	insp, err := uc.repo.GetInspectionByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Inspeksi QC tidak ditemukan")
	}
	dto := mapInspectionToDTO(insp)
	return &dto, nil
}

func (uc *qcUseCase) ListInspections(ctx context.Context, filter domain.ListInspectionFilter) ([]QCInspectionResponseDTO, int64, error) {
	items, total, err := uc.repo.ListInspections(ctx, filter)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err)
	}
	res := make([]QCInspectionResponseDTO, 0, len(items))
	for _, itm := range items {
		res = append(res, mapInspectionToDTO(&itm))
	}
	return res, total, nil
}

func (uc *qcUseCase) VoidInspection(ctx context.Context, id uuid.UUID, actorID uuid.UUID, dto VoidQCInspectionDTO) (*QCInspectionResponseDTO, error) {
	if strings.TrimSpace(dto.Reason) == "" {
		return nil, apperrors.NewBadRequest("Alasan pembatalan (void) wajib diisi (BRL-07)")
	}

	insp, err := uc.repo.GetInspectionByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Inspeksi QC tidak ditemukan")
	}

	if insp.IsVoided {
		return nil, apperrors.NewBadRequest("Inspeksi QC ini sudah dibatalkan sebelumnya")
	}

	now := time.Now()
	insp.IsVoided = true
	insp.VoidReason = strings.TrimSpace(dto.Reason)
	insp.VoidedBy = &actorID
	insp.VoidedAt = &now
	insp.OverallStatus = domain.QCStatusVoided

	if err := uc.repo.UpdateInspection(ctx, insp); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	dtoRes := mapInspectionToDTO(insp)
	return &dtoRes, nil
}

func (uc *qcUseCase) GetStats(ctx context.Context) (*QCStatsResponseDTO, error) {
	total, passed, failed, voided, err := uc.repo.GetInspectionStats(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	var passRate float64
	active := passed + failed
	if active > 0 {
		passRate = math.Round((float64(passed)/float64(active))*1000) / 10
	}
	return &QCStatsResponseDTO{
		TotalInspections:  total,
		PassedInspections: passed,
		FailedInspections: failed,
		VoidedInspections: voided,
		PassRate:          passRate,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func mapParameterToDTO(p *domain.QCParameter) QCParameterResponseDTO {
	return QCParameterResponseDTO{
		ID:            p.ID,
		ProductID:     p.ProductID,
		ProcessCode:   p.ProcessCode,
		Name:          p.Name,
		ParamType:     p.ParamType,
		UnitOfMeasure: p.UnitOfMeasure,
		MinValue:      p.MinValue,
		MaxValue:      p.MaxValue,
		TargetValue:   p.TargetValue,
		Tolerance:     p.Tolerance,
		IsGate:        p.IsGate,
		IsMandatory:   p.IsMandatory,
		Description:   p.Description,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func mapInspectionToDTO(insp *domain.QCInspection) QCInspectionResponseDTO {
	items := make([]QCInspectionItemResponseDTO, 0, len(insp.Items))
	for _, itm := range insp.Items {
		items = append(items, QCInspectionItemResponseDTO{
			ID:            itm.ID,
			ParameterID:   itm.ParameterID,
			ParameterName: itm.ParameterName,
			ParamType:     itm.ParamType,
			NumericValue:  itm.NumericValue,
			BooleanValue:  itm.BooleanValue,
			IsPassed:      itm.IsPassed,
			Notes:         itm.Notes,
		})
	}

	return QCInspectionResponseDTO{
		ID:                insp.ID,
		InspectionNo:      insp.InspectionNo,
		TargetType:        insp.TargetType,
		TargetID:          insp.TargetID,
		ProductionOrderID: insp.ProductionOrderID,
		ProductionStepID:  insp.ProductionStepID,
		BatchNo:           insp.BatchNo,
		ProductID:         insp.ProductID,
		InspectorID:       insp.InspectorID,
		InspectorName:     insp.InspectorName,
		InspectedAt:       insp.InspectedAt,
		TestingTool:       insp.TestingTool,
		Iteration:         insp.Iteration,
		OverallStatus:     insp.OverallStatus,
		Notes:             insp.Notes,
		SugarCondition:    insp.SugarCondition,
		PackageCondition:  insp.PackageCondition,
		IsVoided:          insp.IsVoided,
		VoidReason:        insp.VoidReason,
		VoidedBy:          insp.VoidedBy,
		VoidedAt:          insp.VoidedAt,
		Items:             items,
		CreatedAt:         insp.CreatedAt,
		UpdatedAt:         insp.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Personal Hygiene (Doc CPH) & Pre-Op Sanitation (Doc SSO)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateHygieneCheck(ctx context.Context, dto CreateHygieneCheckDTO) (*domain.PersonalHygieneCheck, error) {
	docNo, err := docseq.Next(ctx, uc.repo, "CPH", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen CPH")
	}
	isPassed := dto.WorkWearClean && dto.HairNet && dto.NailsClean && dto.NoJewelry && dto.WorkShoesClean && dto.HandWashing && dto.HealthFit
	check := &domain.PersonalHygieneCheck{
		DocumentNo:     docNo,
		CheckDate:      dto.CheckDate,
		Shift:          dto.Shift,
		Line:           dto.Line,
		EmployeeName:   dto.EmployeeName,
		WorkWearClean:  dto.WorkWearClean,
		HairNet:        dto.HairNet,
		NailsClean:     dto.NailsClean,
		NoJewelry:      dto.NoJewelry,
		WorkShoesClean: dto.WorkShoesClean,
		HandWashing:    dto.HandWashing,
		HealthFit:      dto.HealthFit,
		IsPassed:       isPassed,
		ActionTaken:    dto.ActionTaken,
		InspectorName:  dto.InspectorName,
		Notes:          dto.Notes,
	}
	if err := uc.repo.CreateHygieneCheck(ctx, check); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return check, nil
}

func (uc *qcUseCase) ListHygieneChecks(ctx context.Context, date string, line string) ([]domain.PersonalHygieneCheck, error) {
	checks, err := uc.repo.ListHygieneChecks(ctx, date, line)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return checks, nil
}

func (uc *qcUseCase) DeleteHygieneCheck(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteHygieneCheck(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

func (uc *qcUseCase) CreatePreOpCheck(ctx context.Context, dto CreatePreOpCheckDTO) (*domain.PreOpSanitationCheck, error) {
	docNo, err := docseq.Next(ctx, uc.repo, "SSO", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen SSO")
	}
	status := dto.Status
	if status == "" {
		allOk := dto.FloorClean && dto.WallCeilingClean && dto.EquipmentSanitized && dto.TrashBinClosed && dto.LightingProtected && dto.PestControlIntact
		if allOk {
			status = "approved"
		} else {
			status = "action_required"
		}
	}
	check := &domain.PreOpSanitationCheck{
		DocumentNo:         docNo,
		CheckDate:          dto.CheckDate,
		CheckTime:          dto.CheckTime,
		Shift:              dto.Shift,
		Room:               dto.Room,
		FloorClean:         dto.FloorClean,
		WallCeilingClean:   dto.WallCeilingClean,
		EquipmentSanitized: dto.EquipmentSanitized,
		TrashBinClosed:     dto.TrashBinClosed,
		LightingProtected:  dto.LightingProtected,
		PestControlIntact:  dto.PestControlIntact,
		GaussReading:       dto.GaussReading,
		MetalTrapClean:     dto.MetalTrapClean,
		Status:             status,
		Notes:              dto.Notes,
		InspectorName:      dto.InspectorName,
		SupervisorName:     dto.SupervisorName,
	}
	if err := uc.repo.CreatePreOpCheck(ctx, check); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return check, nil
}

func (uc *qcUseCase) ListPreOpChecks(ctx context.Context, date string, room string) ([]domain.PreOpSanitationCheck, error) {
	checks, err := uc.repo.ListPreOpChecks(ctx, date, room)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return checks, nil
}

func (uc *qcUseCase) DeletePreOpCheck(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeletePreOpCheck(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Packaging Quality Check (PBK — QC-003)
// ─────────────────────────────────────────────────────────────────────────────

func computePBKStatus(passedQty, failedQty int) string {
	if failedQty == 0 {
		return "passed"
	}
	if passedQty == 0 {
		return "failed"
	}
	return "partial"
}

func mapPackagingCheckToDTO(c *domain.PackagingQualityCheck) PackagingCheckResponseDTO {
	return PackagingCheckResponseDTO{
		ID:              c.ID,
		DocumentNo:      c.DocumentNo,
		CheckMonth:      c.CheckMonth,
		GoodsReceiptID:  c.GoodsReceiptID,
		LBMNo:           c.LBMNo,
		CheckDate:       c.CheckDate,
		SupplierID:      c.SupplierID,
		SupplierName:    c.SupplierName,
		ItemName:        c.ItemName,
		ItemCode:        c.ItemCode,
		MaterialType:    c.MaterialType,
		BaseColor:       c.BaseColor,
		Dimension:       c.Dimension,
		TotalQty:        c.TotalQty,
		SampleQty:       c.SampleQty,
		PassedQty:       c.PassedQty,
		PassedCondition: c.PassedCondition,
		FailedQty:       c.FailedQty,
		FailedCondition: c.FailedCondition,
		OverallStatus:   c.OverallStatus,
		QCName:          c.QCName,
		QCDate:          c.QCDate,
		QAName:          c.QAName,
		QADate:          c.QADate,
		Notes:           c.Notes,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

func (uc *qcUseCase) CreatePackagingCheck(ctx context.Context, dto CreatePackagingCheckDTO) (*PackagingCheckResponseDTO, error) {
	if strings.TrimSpace(dto.CheckMonth) == "" || strings.TrimSpace(dto.SupplierName) == "" || strings.TrimSpace(dto.ItemName) == "" {
		return nil, apperrors.NewBadRequest("Bulan, supplier, dan nama barang wajib diisi")
	}

	overallStatus := computePBKStatus(dto.PassedQty, dto.FailedQty)
	now := time.Now()
	var qcDate *time.Time
	if dto.QCName != "" {
		qcDate = &now
	}
	var qaDate *time.Time
	if dto.QAName != "" {
		qaDate = &now
	}

	docNo, err := docseq.Next(ctx, uc.repo, "PBK", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen PBK")
	}
	check := &domain.PackagingQualityCheck{
		DocumentNo:      docNo,
		CheckMonth:      dto.CheckMonth,
		GoodsReceiptID:  dto.GoodsReceiptID,
		LBMNo:           dto.LBMNo,
		CheckDate:       dto.CheckDate,
		SupplierID:      dto.SupplierID,
		SupplierName:    dto.SupplierName,
		ItemName:        dto.ItemName,
		ItemCode:        dto.ItemCode,
		MaterialType:    dto.MaterialType,
		BaseColor:       dto.BaseColor,
		Dimension:       dto.Dimension,
		TotalQty:        dto.TotalQty,
		SampleQty:       dto.SampleQty,
		PassedQty:       dto.PassedQty,
		PassedCondition: dto.PassedCondition,
		FailedQty:       dto.FailedQty,
		FailedCondition: dto.FailedCondition,
		OverallStatus:   overallStatus,
		QCName:          dto.QCName,
		QCDate:          qcDate,
		QAName:          dto.QAName,
		QADate:          qaDate,
		Notes:           dto.Notes,
	}

	if err := uc.repo.CreatePackagingCheck(ctx, check); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	if check.OverallStatus == "failed" || check.FailedQty > 0 {
		failQty := float64(check.FailedQty)
		if failQty <= 0 {
			failQty = float64(check.TotalQty)
		}
		defect := check.FailedCondition
		if defect == "" {
			defect = "Bahan kemas cacat / gagal spesifikasi QC-003-PBK"
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageIncomingPackaging,
			SourceInspectionID: &check.ID,
			SourceInspectionNo: check.LBMNo,
			ProductName:        check.ItemName,
			ProductCode:        check.ItemCode,
			BatchNo:            check.CheckMonth,
			MaterialType:       "BK",
			SupplierID:         check.SupplierID,
			SupplierName:       check.SupplierName,
			Quantity:           failQty,
			Unit:               "PCS",
			DefectReason:       defect,
			ActionDisposition:  domain.ActionReturnToSupplier,
			QCInspectorName:    check.QCName,
			Notes:              check.Notes,
			GoodsReceiptNo:     check.LBMNo,
		})
	}

	res := mapPackagingCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) GetPackagingCheckByID(ctx context.Context, id uuid.UUID) (*PackagingCheckResponseDTO, error) {
	check, err := uc.repo.GetPackagingCheckByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Pemeriksaan kemasan tidak ditemukan")
	}
	res := mapPackagingCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) ListPackagingChecks(ctx context.Context, month string, lbmNo string) ([]PackagingCheckResponseDTO, error) {
	checks, err := uc.repo.ListPackagingChecks(ctx, month, lbmNo)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	res := make([]PackagingCheckResponseDTO, 0, len(checks))
	for _, c := range checks {
		res = append(res, mapPackagingCheckToDTO(&c))
	}
	return res, nil
}

func (uc *qcUseCase) UpdatePackagingCheck(ctx context.Context, id uuid.UUID, dto UpdatePackagingCheckDTO) (*PackagingCheckResponseDTO, error) {
	check, err := uc.repo.GetPackagingCheckByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Pemeriksaan kemasan tidak ditemukan")
	}

	if dto.CheckMonth != "" {
		check.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		check.CheckDate = dto.CheckDate
	}
	if dto.GoodsReceiptID != nil {
		check.GoodsReceiptID = dto.GoodsReceiptID
	}
	if dto.LBMNo != "" {
		check.LBMNo = dto.LBMNo
	}
	if dto.SupplierName != "" {
		check.SupplierName = dto.SupplierName
	}
	if dto.SupplierID != nil {
		check.SupplierID = dto.SupplierID
	}
	if dto.ItemName != "" {
		check.ItemName = dto.ItemName
	}
	if dto.ItemCode != "" {
		check.ItemCode = dto.ItemCode
	}
	if dto.MaterialType != "" {
		check.MaterialType = dto.MaterialType
	}
	if dto.BaseColor != "" {
		check.BaseColor = dto.BaseColor
	}
	if dto.Dimension != "" {
		check.Dimension = dto.Dimension
	}
	if dto.TotalQty > 0 {
		check.TotalQty = dto.TotalQty
	}
	if dto.SampleQty > 0 {
		check.SampleQty = dto.SampleQty
	}
	check.PassedQty = dto.PassedQty
	check.PassedCondition = dto.PassedCondition
	check.FailedQty = dto.FailedQty
	check.FailedCondition = dto.FailedCondition
	check.OverallStatus = computePBKStatus(check.PassedQty, check.FailedQty)

	now := time.Now()
	if dto.QCName != "" {
		check.QCName = dto.QCName
		if check.QCDate == nil {
			check.QCDate = &now
		}
	}
	if dto.QAName != "" {
		check.QAName = dto.QAName
		if check.QADate == nil {
			check.QADate = &now
		}
	}
	if dto.Notes != "" {
		check.Notes = dto.Notes
	}

	if err := uc.repo.UpdatePackagingCheck(ctx, check); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	res := mapPackagingCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) DeletePackagingCheck(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeletePackagingCheck(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Supplier Armada Check (CAS — QC-007)
// ─────────────────────────────────────────────────────────────────────────────

func computeCASStatus(cabinSafe, undercarriageSafe, cargoAreaSafe, floorClean, wallRightClean, wallLeftClean, ceilingClean, doorClean bool) string {
	if !cabinSafe || !undercarriageSafe || !cargoAreaSafe {
		return "rejected"
	}
	if !floorClean || !wallRightClean || !wallLeftClean || !ceilingClean || !doorClean {
		return "flagged"
	}
	return "cleared"
}

func mapArmadaCheckToDTO(c *domain.SupplierArmadaCheck) ArmadaCheckResponseDTO {
	return ArmadaCheckResponseDTO{
		ID:                c.ID,
		DocumentNo:        c.DocumentNo,
		GoodsReceiptID:    c.GoodsReceiptID,
		LBMNo:             c.LBMNo,
		CheckDate:         c.CheckDate,
		IDNumber:          c.IDNumber,
		DriverName:        c.DriverName,
		BirthPlace:        c.BirthPlace,
		BirthDate:         c.BirthDate,
		Gender:            c.Gender,
		Address:           c.Address,
		Religion:          c.Religion,
		MaritalStatus:     c.MaritalStatus,
		IDExpiry:          c.IDExpiry,
		CompanyOrigin:     c.CompanyOrigin,
		CargoDesc:         c.CargoDesc,
		LicensePlate:      c.LicensePlate,
		VehicleType:       c.VehicleType,
		CabinSafe:         c.CabinSafe,
		UndercarriageSafe: c.UndercarriageSafe,
		CargoAreaSafe:     c.CargoAreaSafe,
		FloorClean:        c.FloorClean,
		WallRightClean:    c.WallRightClean,
		WallLeftClean:     c.WallLeftClean,
		CeilingClean:      c.CeilingClean,
		DoorClean:         c.DoorClean,
		OverallStatus:     c.OverallStatus,
		QCName:            c.QCName,
		QCDate:            c.QCDate,
		QAName:            c.QAName,
		QADate:            c.QADate,
		Notes:             c.Notes,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func (uc *qcUseCase) CreateArmadaCheck(ctx context.Context, dto CreateArmadaCheckDTO) (*ArmadaCheckResponseDTO, error) {
	if strings.TrimSpace(dto.DriverName) == "" || strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Nama pengemudi dan tanggal checklist wajib diisi")
	}

	overallStatus := computeCASStatus(
		dto.CabinSafe, dto.UndercarriageSafe, dto.CargoAreaSafe,
		dto.FloorClean, dto.WallRightClean, dto.WallLeftClean, dto.CeilingClean, dto.DoorClean,
	)

	now := time.Now()
	var qcDate *time.Time
	if dto.QCName != "" {
		qcDate = &now
	}
	var qaDate *time.Time
	if dto.QAName != "" {
		qaDate = &now
	}

	docNo, err := docseq.Next(ctx, uc.repo, "CAS", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen CAS")
	}
	check := &domain.SupplierArmadaCheck{
		DocumentNo:        docNo,
		GoodsReceiptID:    dto.GoodsReceiptID,
		LBMNo:             dto.LBMNo,
		CheckDate:         dto.CheckDate,
		IDNumber:          dto.IDNumber,
		DriverName:        dto.DriverName,
		BirthPlace:        dto.BirthPlace,
		BirthDate:         dto.BirthDate,
		Gender:            dto.Gender,
		Address:           dto.Address,
		Religion:          dto.Religion,
		MaritalStatus:     dto.MaritalStatus,
		IDExpiry:          dto.IDExpiry,
		CompanyOrigin:     dto.CompanyOrigin,
		CargoDesc:         dto.CargoDesc,
		LicensePlate:      dto.LicensePlate,
		VehicleType:       dto.VehicleType,
		CabinSafe:         dto.CabinSafe,
		UndercarriageSafe: dto.UndercarriageSafe,
		CargoAreaSafe:     dto.CargoAreaSafe,
		FloorClean:        dto.FloorClean,
		WallRightClean:    dto.WallRightClean,
		WallLeftClean:     dto.WallLeftClean,
		CeilingClean:      dto.CeilingClean,
		DoorClean:         dto.DoorClean,
		OverallStatus:     overallStatus,
		QCName:            dto.QCName,
		QCDate:            qcDate,
		QAName:            dto.QAName,
		QADate:            qaDate,
		Notes:             dto.Notes,
	}

	if err := uc.repo.CreateArmadaCheck(ctx, check); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	res := mapArmadaCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) GetArmadaCheckByID(ctx context.Context, id uuid.UUID) (*ArmadaCheckResponseDTO, error) {
	check, err := uc.repo.GetArmadaCheckByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Checklist armada tidak ditemukan")
	}
	res := mapArmadaCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) ListArmadaChecks(ctx context.Context, date string, lbmNo string) ([]ArmadaCheckResponseDTO, error) {
	checks, err := uc.repo.ListArmadaChecks(ctx, date, lbmNo)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	res := make([]ArmadaCheckResponseDTO, 0, len(checks))
	for _, c := range checks {
		res = append(res, mapArmadaCheckToDTO(&c))
	}
	return res, nil
}

func (uc *qcUseCase) UpdateArmadaCheck(ctx context.Context, id uuid.UUID, dto UpdateArmadaCheckDTO) (*ArmadaCheckResponseDTO, error) {
	check, err := uc.repo.GetArmadaCheckByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Checklist armada tidak ditemukan")
	}

	if dto.GoodsReceiptID != nil {
		check.GoodsReceiptID = dto.GoodsReceiptID
	}
	if dto.LBMNo != "" {
		check.LBMNo = dto.LBMNo
	}
	if dto.CheckDate != "" {
		check.CheckDate = dto.CheckDate
	}
	if dto.IDNumber != "" {
		check.IDNumber = dto.IDNumber
	}
	if dto.DriverName != "" {
		check.DriverName = dto.DriverName
	}
	if dto.BirthPlace != "" {
		check.BirthPlace = dto.BirthPlace
	}
	if dto.BirthDate != "" {
		check.BirthDate = dto.BirthDate
	}
	if dto.Gender != "" {
		check.Gender = dto.Gender
	}
	if dto.Address != "" {
		check.Address = dto.Address
	}
	if dto.Religion != "" {
		check.Religion = dto.Religion
	}
	if dto.MaritalStatus != "" {
		check.MaritalStatus = dto.MaritalStatus
	}
	if dto.IDExpiry != "" {
		check.IDExpiry = dto.IDExpiry
	}
	if dto.CompanyOrigin != "" {
		check.CompanyOrigin = dto.CompanyOrigin
	}
	if dto.CargoDesc != "" {
		check.CargoDesc = dto.CargoDesc
	}
	if dto.LicensePlate != "" {
		check.LicensePlate = dto.LicensePlate
	}
	if dto.VehicleType != "" {
		check.VehicleType = dto.VehicleType
	}

	check.CabinSafe = dto.CabinSafe
	check.UndercarriageSafe = dto.UndercarriageSafe
	check.CargoAreaSafe = dto.CargoAreaSafe
	check.FloorClean = dto.FloorClean
	check.WallRightClean = dto.WallRightClean
	check.WallLeftClean = dto.WallLeftClean
	check.CeilingClean = dto.CeilingClean
	check.DoorClean = dto.DoorClean

	check.OverallStatus = computeCASStatus(
		check.CabinSafe, check.UndercarriageSafe, check.CargoAreaSafe,
		check.FloorClean, check.WallRightClean, check.WallLeftClean, check.CeilingClean, check.DoorClean,
	)

	now := time.Now()
	if dto.QCName != "" {
		check.QCName = dto.QCName
		if check.QCDate == nil {
			check.QCDate = &now
		}
	}
	if dto.QAName != "" {
		check.QAName = dto.QAName
		if check.QADate == nil {
			check.QADate = &now
		}
	}
	if dto.Notes != "" {
		check.Notes = dto.Notes
	}

	if err := uc.repo.UpdateArmadaCheck(ctx, check); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	res := mapArmadaCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) DeleteArmadaCheck(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteArmadaCheck(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Raw Material Incoming Check (PBD — QC-002-PBD Rev. 05)
// ─────────────────────────────────────────────────────────────────────────────

func boolOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func colorScoreLabel(score int) string {
	switch score {
	case 1:
		return "1 - Light Brown"
	case 3:
		return "3 - Dark Brown"
	default:
		return "2 - Normal"
	}
}

func computePBDStatus(acceptedWeight, rejectedWeight float64, moisture *float64, foreignClean bool) (status string, recommendedAction string) {
	if !foreignClean || (rejectedWeight > 0 && acceptedWeight == 0) {
		return "rejected", "Ditolak / Retur ke Supplier (Tercemar atau Rusak Total)"
	}
	if moisture != nil && *moisture > 2.0 {
		if rejectedWeight > 0 {
			return "conditional", fmt.Sprintf("Diterima Bersyarat (Wajib Masuk Oven, Kadar Air %.2f%%; %.0f kg Ditolak)", *moisture, rejectedWeight)
		}
		return "conditional", fmt.Sprintf("Diterima Bersyarat - Wajib Line Oven (Kadar Air %.2f%% > 2.0%%)", *moisture)
	}
	if rejectedWeight > 0 {
		return "conditional", fmt.Sprintf("Diterima Sebagian (%.0f kg Ditolak)", rejectedWeight)
	}
	return "passed", "Lolos Standar - Dapat Langsung ke Ayak Matang / Produksi (Kadar Air <= 2.0%)"
}

func evaluateIncomingCheckOutcome(
	check *domain.RawMaterialIncomingCheck,
	explicitDecision string,
	explicitPassRate *float64,
	explicitRecAction string,
	explicitSugarCond string,
	explicitPackCond string,
) {
	status, autoAction := computePBDStatus(check.AcceptedWeight, check.RejectedWeight, check.MoistureContent, check.ForeignMatterClean)

	passRate := check.PassRatePercent
	if explicitPassRate != nil {
		passRate = math.Max(0, math.Min(100, *explicitPassRate))
	} else if check.NetWeight > 0 && (check.AcceptedWeight > 0 || check.RejectedWeight > 0) {
		passRate = math.Round((check.AcceptedWeight/check.NetWeight)*10000) / 100
		if passRate > 100 {
			passRate = 100
		}
	} else if status == "rejected" {
		passRate = 0
	} else if passRate <= 0 && explicitDecision != "DITOLAK" {
		passRate = 100
	}
	check.PassRatePercent = passRate

	decision := "LOLOS"
	if strings.EqualFold(strings.TrimSpace(explicitDecision), "DITOLAK") ||
		(explicitPassRate != nil && *explicitPassRate <= 0) ||
		!check.CheckTaste || !check.CheckAroma || status == "rejected" {
		status = "rejected"
		decision = "DITOLAK"
		if !strings.HasPrefix(autoAction, "Ditolak") {
			autoAction = "Ditolak / Retur ke Supplier (Tidak Memenuhi Spesifikasi QC-002-PBD)"
		}
	} else if (!check.CheckPackaging || !check.CheckShape || !check.CheckSolubility || !check.CheckPackageLabel || (passRate > 0 && passRate < 100)) && status == "passed" {
		status = "conditional"
		autoAction = "Diterima Bersyarat (Catatan Pemeriksaan Fisik/Kemasan QC-002-PBD)"
	}

	check.OverallStatus = status
	check.InspectionDecision = decision

	if strings.TrimSpace(explicitRecAction) != "" {
		check.RecommendedAction = explicitRecAction
	} else if check.RecommendedAction == "" || explicitDecision != "" || explicitPassRate != nil {
		check.RecommendedAction = autoAction
	}

	if strings.TrimSpace(explicitSugarCond) != "" {
		check.SugarCondition = explicitSugarCond
	} else if check.SugarCondition == "" {
		if status == "rejected" {
			check.SugarCondition = "Tidak Sesuai Spesifikasi"
		} else if check.MoistureContent != nil && *check.MoistureContent > 2.0 {
			check.SugarCondition = "Lembab (Kadar Air > 2.0%)"
		} else {
			check.SugarCondition = "Baik, Sesuai Spesifikasi"
		}
	}

	if strings.TrimSpace(explicitPackCond) != "" {
		check.PackagingCondition = explicitPackCond
	} else if check.PackagingCondition == "" {
		if check.CheckPackaging && check.CheckPackageLabel {
			check.PackagingCondition = "Kemasan & Label Baik"
		} else {
			check.PackagingCondition = "Kemasan / Label Perlu Perhatian"
		}
	}

	if check.ColorAppearance == "" {
		check.ColorAppearance = colorScoreLabel(check.ColorScore)
	}
}

func mapIncomingCheckToDTO(c *domain.RawMaterialIncomingCheck) IncomingCheckResponseDTO {
	docCode := c.DocumentCode
	if docCode == "" {
		docCode = "QC-002-PBD"
	}
	revCode := c.RevisionCode
	if revCode == "" {
		revCode = "05"
	}
	fairtrade := c.FairtradeStatus
	if fairtrade == "" {
		fairtrade = "NON FAIRTRADE"
	}
	certs := c.Certifications
	if certs == "" {
		certs = "NATURAL"
	}
	itemType := c.ItemType
	if itemType == "" {
		itemType = "Bahan Baku"
	}
	itemCode := c.ItemCode
	if itemCode == "" {
		itemCode = c.BatchNo
	}
	colorScore := c.ColorScore
	if colorScore < 1 || colorScore > 3 {
		colorScore = 2
	}
	decision := c.InspectionDecision
	if decision == "" {
		if c.OverallStatus == "rejected" {
			decision = "DITOLAK"
		} else {
			decision = "LOLOS"
		}
	}

	return IncomingCheckResponseDTO{
		ID:                 c.ID,
		DocumentNo:         c.DocumentNo,
		DocumentCode:       docCode,
		RevisionCode:       revCode,
		FairtradeStatus:    fairtrade,
		Certifications:     certs,
		GoodsReceiptID:     c.GoodsReceiptID,
		LBMNo:              c.LBMNo,
		CheckDate:          c.CheckDate,
		CheckMonth:         c.CheckMonth,
		SupplierID:         c.SupplierID,
		SupplierName:       c.SupplierName,
		DeliveryNoteNo:     c.DeliveryNoteNo,
		PONo:               c.PONo,
		ProductName:        c.ProductName,
		ItemCode:           itemCode,
		ItemType:           itemType,
		BatchNo:            c.BatchNo,
		TotalBags:          c.TotalBags,
		SampleBags:         c.SampleBags,
		GrossWeight:        c.GrossWeight,
		NetWeight:          c.NetWeight,
		AcceptedWeight:     c.AcceptedWeight,
		RejectedWeight:     c.RejectedWeight,
		CheckPackaging:     c.CheckPackaging,
		CheckShape:         c.CheckShape,
		CheckTaste:         c.CheckTaste,
		CheckAroma:         c.CheckAroma,
		ColorScore:         colorScore,
		MoistureContent:    c.MoistureContent,
		BrixValue:          c.BrixValue,
		CheckSolubility:    c.CheckSolubility,
		PhValue:            c.PhValue,
		CheckPackageLabel:  c.CheckPackageLabel,
		PassRatePercent:    c.PassRatePercent,
		InspectionDecision: decision,
		SugarCondition:     c.SugarCondition,
		ColorAppearance:    c.ColorAppearance,
		AromaFlavor:        c.AromaFlavor,
		ForeignMatterClean: c.ForeignMatterClean,
		PackagingCondition: c.PackagingCondition,
		MeshSize:           c.MeshSize,
		RecommendedAction:  c.RecommendedAction,
		OverallStatus:      c.OverallStatus,
		QCName:             c.QCName,
		QCDate:             c.QCDate,
		QAName:             c.QAName,
		QADate:             c.QADate,
		Notes:              c.Notes,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

func (uc *qcUseCase) CreateIncomingCheck(ctx context.Context, dto CreateIncomingCheckDTO) (*IncomingCheckResponseDTO, error) {
	if strings.TrimSpace(dto.SupplierName) == "" || strings.TrimSpace(dto.ProductName) == "" || strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal, supplier, dan nama barang datang wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	fairtrade := strings.TrimSpace(dto.FairtradeStatus)
	if fairtrade == "" {
		fairtrade = "NON FAIRTRADE"
	}
	certs := strings.TrimSpace(dto.Certifications)
	if certs == "" {
		certs = "NATURAL"
	}
	itemType := strings.TrimSpace(dto.ItemType)
	if itemType == "" {
		itemType = "Bahan Baku"
	}
	itemCode := strings.TrimSpace(dto.ItemCode)
	batchNo := strings.TrimSpace(dto.BatchNo)
	if itemCode == "" && batchNo != "" {
		itemCode = batchNo
	} else if batchNo == "" && itemCode != "" {
		batchNo = itemCode
	}
	colorScore := dto.ColorScore
	if colorScore < 1 || colorScore > 3 {
		colorScore = 2
	}

	now := time.Now()
	var qcDate *time.Time
	if dto.QCName != "" {
		qcDate = &now
	}
	var qaDate *time.Time
	if dto.QAName != "" {
		qaDate = &now
	}

	docNo, err := docseq.Next(ctx, uc.repo, "PBD", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen PBD")
	}
	check := &domain.RawMaterialIncomingCheck{
		DocumentNo:         docNo,
		DocumentCode:       "QC-002-PBD",
		RevisionCode:       "05",
		FairtradeStatus:    fairtrade,
		Certifications:     certs,
		GoodsReceiptID:     dto.GoodsReceiptID,
		LBMNo:              dto.LBMNo,
		CheckDate:          dto.CheckDate,
		CheckMonth:         month,
		SupplierID:         dto.SupplierID,
		SupplierName:       dto.SupplierName,
		DeliveryNoteNo:     dto.DeliveryNoteNo,
		PONo:               dto.PONo,
		ProductName:        dto.ProductName,
		ItemCode:           itemCode,
		ItemType:           itemType,
		BatchNo:            batchNo,
		TotalBags:          dto.TotalBags,
		SampleBags:         dto.SampleBags,
		GrossWeight:        dto.GrossWeight,
		NetWeight:          dto.NetWeight,
		AcceptedWeight:     dto.AcceptedWeight,
		RejectedWeight:     dto.RejectedWeight,
		CheckPackaging:     boolOrDefault(dto.CheckPackaging, true),
		CheckShape:         boolOrDefault(dto.CheckShape, true),
		CheckTaste:         boolOrDefault(dto.CheckTaste, true),
		CheckAroma:         boolOrDefault(dto.CheckAroma, true),
		ColorScore:         colorScore,
		MoistureContent:    dto.MoistureContent,
		BrixValue:          dto.BrixValue,
		CheckSolubility:    boolOrDefault(dto.CheckSolubility, true),
		PhValue:            dto.PhValue,
		CheckPackageLabel:  boolOrDefault(dto.CheckPackageLabel, true),
		ColorAppearance:    dto.ColorAppearance,
		AromaFlavor:        dto.AromaFlavor,
		ForeignMatterClean: dto.ForeignMatterClean,
		MeshSize:           dto.MeshSize,
		QCName:             dto.QCName,
		QCDate:             qcDate,
		QAName:             dto.QAName,
		QADate:             qaDate,
		Notes:              dto.Notes,
	}

	evaluateIncomingCheckOutcome(
		check,
		dto.InspectionDecision,
		dto.PassRatePercent,
		dto.RecommendedAction,
		dto.SugarCondition,
		dto.PackagingCondition,
	)

	if err := uc.repo.CreateIncomingCheck(ctx, check); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	if check.OverallStatus == "rejected" || check.RejectedWeight > 0 {
		rejQty := check.RejectedWeight
		if rejQty <= 0 {
			rejQty = check.NetWeight
		}
		defect := fmt.Sprintf("Kondisi: %s; %s", check.SugarCondition, check.RecommendedAction)
		if check.MoistureContent != nil && *check.MoistureContent > 2.0 {
			defect += fmt.Sprintf(" (Kadar Air %.2f%%)", *check.MoistureContent)
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageIncomingRawMaterial,
			SourceInspectionID: &check.ID,
			SourceInspectionNo: check.LBMNo,
			ProductName:        check.ProductName,
			ProductCode:        check.ItemCode,
			BatchNo:            check.BatchNo,
			MaterialType:       "BM",
			SupplierID:         check.SupplierID,
			SupplierName:       check.SupplierName,
			Quantity:           rejQty,
			Unit:               "KG",
			DefectReason:       defect,
			ActionDisposition:  domain.ActionReturnToSupplier,
			QCInspectorName:    check.QCName,
			Notes:              check.Notes,
			PONo:               check.PONo,
			GoodsReceiptNo:     check.LBMNo,
			DeliveryNoteNo:     check.DeliveryNoteNo,
		})
	}

	res := mapIncomingCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) GetIncomingCheckByID(ctx context.Context, id uuid.UUID) (*IncomingCheckResponseDTO, error) {
	check, err := uc.repo.GetIncomingCheckByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Catatan pemeriksaan barang datang tidak ditemukan")
	}
	res := mapIncomingCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) ListIncomingChecks(ctx context.Context, month string, date string, lbmNo string) ([]IncomingCheckResponseDTO, error) {
	checks, err := uc.repo.ListIncomingChecks(ctx, month, date, lbmNo)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	res := make([]IncomingCheckResponseDTO, 0, len(checks))
	for _, c := range checks {
		res = append(res, mapIncomingCheckToDTO(&c))
	}
	return res, nil
}

func (uc *qcUseCase) UpdateIncomingCheck(ctx context.Context, id uuid.UUID, dto UpdateIncomingCheckDTO) (*IncomingCheckResponseDTO, error) {
	check, err := uc.repo.GetIncomingCheckByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Catatan pemeriksaan barang datang tidak ditemukan")
	}

	if check.DocumentCode == "" {
		check.DocumentCode = "QC-002-PBD"
	}
	if check.RevisionCode == "" {
		check.RevisionCode = "05"
	}
	if dto.FairtradeStatus != "" {
		check.FairtradeStatus = dto.FairtradeStatus
	}
	if dto.Certifications != "" {
		check.Certifications = dto.Certifications
	}
	if dto.GoodsReceiptID != nil {
		check.GoodsReceiptID = dto.GoodsReceiptID
	}
	if dto.LBMNo != "" {
		check.LBMNo = dto.LBMNo
	}
	if dto.CheckDate != "" {
		check.CheckDate = dto.CheckDate
		if len(dto.CheckDate) >= 7 {
			check.CheckMonth = dto.CheckDate[:7]
		}
	}
	if dto.CheckMonth != "" {
		check.CheckMonth = dto.CheckMonth
	}
	if dto.SupplierID != nil {
		check.SupplierID = dto.SupplierID
	}
	if dto.SupplierName != "" {
		check.SupplierName = dto.SupplierName
	}
	if dto.DeliveryNoteNo != "" {
		check.DeliveryNoteNo = dto.DeliveryNoteNo
	}
	if dto.PONo != "" {
		check.PONo = dto.PONo
	}
	if dto.ProductName != "" {
		check.ProductName = dto.ProductName
	}
	if dto.ItemCode != "" {
		check.ItemCode = dto.ItemCode
	}
	if dto.ItemType != "" {
		check.ItemType = dto.ItemType
	}
	if dto.BatchNo != "" {
		check.BatchNo = dto.BatchNo
	}
	if check.ItemCode == "" && check.BatchNo != "" {
		check.ItemCode = check.BatchNo
	}
	if dto.TotalBags > 0 {
		check.TotalBags = dto.TotalBags
	}
	if dto.SampleBags > 0 {
		check.SampleBags = dto.SampleBags
	}
	if dto.GrossWeight > 0 {
		check.GrossWeight = dto.GrossWeight
	}
	if dto.NetWeight > 0 {
		check.NetWeight = dto.NetWeight
	}
	check.AcceptedWeight = dto.AcceptedWeight
	check.RejectedWeight = dto.RejectedWeight

	if dto.CheckPackaging != nil {
		check.CheckPackaging = *dto.CheckPackaging
	}
	if dto.CheckShape != nil {
		check.CheckShape = *dto.CheckShape
	}
	if dto.CheckTaste != nil {
		check.CheckTaste = *dto.CheckTaste
	}
	if dto.CheckAroma != nil {
		check.CheckAroma = *dto.CheckAroma
	}
	if dto.ColorScore >= 1 && dto.ColorScore <= 3 {
		check.ColorScore = dto.ColorScore
		if dto.ColorAppearance == "" {
			check.ColorAppearance = colorScoreLabel(dto.ColorScore)
		}
	}
	if dto.CheckSolubility != nil {
		check.CheckSolubility = *dto.CheckSolubility
	}
	if dto.PhValue != nil {
		check.PhValue = dto.PhValue
	}
	if dto.CheckPackageLabel != nil {
		check.CheckPackageLabel = *dto.CheckPackageLabel
	}
	if dto.ColorAppearance != "" {
		check.ColorAppearance = dto.ColorAppearance
	}
	if dto.AromaFlavor != "" {
		check.AromaFlavor = dto.AromaFlavor
	}
	check.ForeignMatterClean = dto.ForeignMatterClean
	if dto.MoistureContent != nil {
		check.MoistureContent = dto.MoistureContent
	}
	if dto.BrixValue != nil {
		check.BrixValue = dto.BrixValue
	}
	if dto.MeshSize != "" {
		check.MeshSize = dto.MeshSize
	}

	evaluateIncomingCheckOutcome(
		check,
		dto.InspectionDecision,
		dto.PassRatePercent,
		dto.RecommendedAction,
		dto.SugarCondition,
		dto.PackagingCondition,
	)

	now := time.Now()
	if dto.QCName != "" {
		check.QCName = dto.QCName
		if check.QCDate == nil {
			check.QCDate = &now
		}
	}
	if dto.QAName != "" {
		check.QAName = dto.QAName
		if check.QADate == nil {
			check.QADate = &now
		}
	}
	if dto.Notes != "" {
		check.Notes = dto.Notes
	}

	if err := uc.repo.UpdateIncomingCheck(ctx, check); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	res := mapIncomingCheckToDTO(check)
	return &res, nil
}

func (uc *qcUseCase) DeleteIncomingCheck(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteIncomingCheck(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}
