package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Sieving Monitoring (MNA — Form QC-009-MNA.01)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateSievingLog(ctx context.Context, dto CreateSievingLogDTO) (*domain.SievingMonitoringLog, error) {
	if strings.TrimSpace(dto.MachineNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor mesin wajib diisi")
	}
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan wajib diisi")
	}
	if strings.TrimSpace(dto.ProductName) == "" {
		return nil, apperrors.NewBadRequest("Nama produk wajib diisi")
	}
	if strings.TrimSpace(dto.BatchNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor batch wajib diisi")
	}
	if strings.TrimSpace(dto.InitialPICName) == "" || strings.TrimSpace(dto.FinalPICName) == "" {
		return nil, apperrors.NewBadRequest("Nama PIC awal dan akhir wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}
	category := dto.Category
	if category == "" {
		category = "Mentah"
	}
	meshSize := dto.MeshSize
	if meshSize == "" {
		meshSize = "Mesh 18"
	}

	prefixTag := "MNA-"
	if strings.EqualFold(category, "Matang") {
		prefixTag = "MNA-MAT-"
	}
	prefixWithDate := fmt.Sprintf("%s%s-", prefixTag, time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_sieving_monitoring_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	initCond := dto.InitialCondition
	if initCond == "" {
		initCond = "Baik & Bersih"
	}
	finalCond := dto.FinalCondition
	if finalCond == "" {
		finalCond = "Baik & Bersih"
	}

	initPassed := true
	if dto.InitialQCPassed != nil {
		initPassed = *dto.InitialQCPassed
	}
	finalPassed := true
	if dto.FinalQCPassed != nil {
		finalPassed = *dto.FinalQCPassed
	}

	status := dto.OverallStatus
	if status == "" {
		if !initPassed || !finalPassed {
			status = "action_required"
		} else {
			status = "passed"
		}
	}

	item := &domain.SievingMonitoringLog{
		DocumentNo:        docNo,
		RevisionCode:      "00",
		MachineNo:         dto.MachineNo,
		CheckMonth:        month,
		CheckDate:         dto.CheckDate,
		Category:          category,
		ProductionOrderID: dto.ProductionOrderID,
		ProductionStepID:  dto.ProductionStepID,
		RequisitionID:     dto.RequisitionID,
		ProductID:         dto.ProductID,
		ProductName:       dto.ProductName,
		BatchNo:           dto.BatchNo,
		OvenDate:          dto.OvenDate,
		MeshSize:          meshSize,
		InitialCondition:  initCond,
		InitialFindings:   dto.InitialFindings,
		InitialAction:     dto.InitialAction,
		InitialPICName:    dto.InitialPICName,
		InitialQCName:     dto.InitialQCName,
		InitialQCPassed:   initPassed,
		FinalCondition:    finalCond,
		FinalFindings:     dto.FinalFindings,
		FinalAction:       dto.FinalAction,
		FinalPICName:      dto.FinalPICName,
		FinalQCName:       dto.FinalQCName,
		FinalQCPassed:     finalPassed,
		OverallStatus:     status,
		Notes:             dto.Notes,
	}

	if err := uc.repo.CreateSievingLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan monitoring proses ayak")
	}

	if item.OverallStatus == "failed" || !item.InitialQCPassed || !item.FinalQCPassed {
		matType := "BM"
		if item.Category == "Oven" {
			matType = "BO"
		}
		defect := "Gagal QC Ayak: " + item.InitialFindings
		if item.FinalFindings != "" {
			defect += "; " + item.FinalFindings
		}
		if defect == "Gagal QC Ayak: " {
			defect = "Kondisi saringan ayak tidak lolos standar"
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageSieving,
			SourceInspectionID: &item.ID,
			SourceInspectionNo: item.DocumentNo,
			ProductionOrderID:  item.ProductionOrderID,
			ProductID:          item.ProductID,
			ProductName:        item.ProductName,
			BatchNo:            item.BatchNo,
			MachineNo:          item.MachineNo,
			MeshSize:           item.MeshSize,
			MaterialType:       matType,
			DefectReason:       defect,
			ActionDisposition:  domain.ActionRework,
			OperatorName:       item.InitialPICName,
			QCInspectorName:    item.InitialQCName,
			Notes:              item.Notes,
		})
	}

	return item, nil
}

func (uc *qcUseCase) GetSievingLogByID(ctx context.Context, id uuid.UUID) (*domain.SievingMonitoringLog, error) {
	item, err := uc.repo.GetSievingLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring proses ayak tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListSievingLogs(ctx context.Context, month string, category string, machineNo string, batchNo string) ([]domain.SievingMonitoringLog, error) {
	list, err := uc.repo.ListSievingLogs(ctx, month, category, machineNo, batchNo)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil daftar monitoring proses ayak")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateSievingLog(ctx context.Context, id uuid.UUID, dto UpdateSievingLogDTO) (*domain.SievingMonitoringLog, error) {
	item, err := uc.repo.GetSievingLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring proses ayak tidak ditemukan")
	}

	if dto.MachineNo != "" {
		item.MachineNo = dto.MachineNo
	}
	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
	}
	if dto.Category != "" {
		item.Category = dto.Category
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.ProductionStepID != nil {
		item.ProductionStepID = dto.ProductionStepID
	}
	if dto.RequisitionID != nil {
		item.RequisitionID = dto.RequisitionID
	}
	if dto.ProductID != nil {
		item.ProductID = dto.ProductID
	}
	if dto.ProductName != "" {
		item.ProductName = dto.ProductName
	}
	if dto.BatchNo != "" {
		item.BatchNo = dto.BatchNo
	}
	if dto.OvenDate != "" {
		item.OvenDate = dto.OvenDate
	}
	if dto.MeshSize != "" {
		item.MeshSize = dto.MeshSize
	}
	if dto.InitialCondition != "" {
		item.InitialCondition = dto.InitialCondition
	}
	if dto.InitialFindings != "" {
		item.InitialFindings = dto.InitialFindings
	}
	if dto.InitialAction != "" {
		item.InitialAction = dto.InitialAction
	}
	if dto.InitialPICName != "" {
		item.InitialPICName = dto.InitialPICName
	}
	if dto.InitialQCName != "" {
		item.InitialQCName = dto.InitialQCName
	}
	if dto.InitialQCPassed != nil {
		item.InitialQCPassed = *dto.InitialQCPassed
	}
	if dto.FinalCondition != "" {
		item.FinalCondition = dto.FinalCondition
	}
	if dto.FinalFindings != "" {
		item.FinalFindings = dto.FinalFindings
	}
	if dto.FinalAction != "" {
		item.FinalAction = dto.FinalAction
	}
	if dto.FinalPICName != "" {
		item.FinalPICName = dto.FinalPICName
	}
	if dto.FinalQCName != "" {
		item.FinalQCName = dto.FinalQCName
	}
	if dto.FinalQCPassed != nil {
		item.FinalQCPassed = *dto.FinalQCPassed
	}
	if dto.OverallStatus != "" {
		item.OverallStatus = dto.OverallStatus
	}
	if dto.Notes != "" {
		item.Notes = dto.Notes
	}

	if err := uc.repo.UpdateSievingLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui data monitoring proses ayak")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteSievingLog(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteSievingLog(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Magnet Trap (CMT — Form QC-038-CMT)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateMagnetTrapLog(ctx context.Context, dto CreateMagnetTrapLogDTO) (*domain.MagnetTrapLog, error) {
	if strings.TrimSpace(dto.MachineNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor mesin wajib diisi")
	}
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan wajib diisi")
	}
	if strings.TrimSpace(dto.PICResponsible) == "" || strings.TrimSpace(dto.PICName) == "" {
		return nil, apperrors.NewBadRequest("Penanggung jawab dan nama PIC wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}
	room := dto.Room
	if room == "" {
		room = "Ayak Mentah"
	}

	prefixTag := "CMT-"
	if strings.Contains(strings.ToLower(room), "matang") {
		prefixTag = "CMT-MAT-"
	}
	prefixWithDate := fmt.Sprintf("%s%s-", prefixTag, time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_magnet_trap_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	isClean := true
	if dto.IsClean != nil {
		isClean = *dto.IsClean
	}
	qcPassed := true
	if dto.QCPassed != nil {
		qcPassed = *dto.QCPassed
	}

	// Gauss meter calculations (mT)
	avg := (dto.GaussPoint1 + dto.GaussPoint2 + dto.GaussPoint3) / 3.0
	avg = math.Round(avg*100) / 100

	// Critical Threshold: Standard >= 700 mT (>= 7000 Gauss). If any point < 700, alert!
	powerSufficient := dto.GaussPoint1 >= 700.0 && dto.GaussPoint2 >= 700.0 && dto.GaussPoint3 >= 700.0

	status := "cleared"
	if !powerSufficient {
		status = "warning"
	} else if !isClean || !qcPassed {
		status = "action_required"
	}

	var qaAt *time.Time
	if dto.QAVerifiedBy != "" {
		now := time.Now()
		qaAt = &now
	}

	item := &domain.MagnetTrapLog{
		DocumentNo:         docNo,
		CheckMonth:         month,
		CheckDate:          dto.CheckDate,
		Room:               room,
		MachineNo:          dto.MachineNo,
		PICResponsible:     dto.PICResponsible,
		ProductionOrderID:  dto.ProductionOrderID,
		ProductionStepID:   dto.ProductionStepID,
		BatchNo:            dto.BatchNo,
		IsClean:            isClean,
		GaussPoint1:        dto.GaussPoint1,
		GaussPoint2:        dto.GaussPoint2,
		GaussPoint3:        dto.GaussPoint3,
		GaussAverage:       avg,
		IsPowerSufficient:  powerSufficient,
		ForeignMatterFound: dto.ForeignMatterFound,
		Issues:             dto.Issues,
		ActionTaken:        dto.ActionTaken,
		PICName:            dto.PICName,
		QCName:             dto.QCName,
		QCPassed:           qcPassed,
		QAVerifiedBy:       dto.QAVerifiedBy,
		QAVerifiedAt:       qaAt,
		OverallStatus:      status,
		Notes:              dto.Notes,
	}

	if err := uc.repo.CreateMagnetTrapLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mencatat pemakaian magnet trap")
	}

	if !item.QCPassed || !item.IsPowerSufficient || item.OverallStatus == "warning" || item.OverallStatus == "replaced" {
		defect := "Magnet Trap: " + item.Issues
		if item.ForeignMatterFound != "" {
			defect += "; Benda Asing: " + item.ForeignMatterFound
		}
		if defect == "Magnet Trap: " {
			defect = "Kekuatan magnet trap tidak mencukupi standar (di bawah 700 mT)"
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageMagnetTrap,
			SourceInspectionID: &item.ID,
			SourceInspectionNo: item.DocumentNo,
			ProductionOrderID:  item.ProductionOrderID,
			BatchNo:            item.BatchNo,
			ProductName:        "Gula Semut (Line " + item.Room + ")",
			MachineNo:          item.MachineNo,
			MaterialType:       "BM",
			DefectReason:       defect,
			ActionDisposition:  domain.ActionHold,
			OperatorName:       item.PICName,
			QCInspectorName:    item.QCName,
			Notes:              item.Notes,
		})
	}

	return item, nil
}

func (uc *qcUseCase) GetMagnetTrapLogByID(ctx context.Context, id uuid.UUID) (*domain.MagnetTrapLog, error) {
	item, err := uc.repo.GetMagnetTrapLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data pemakaian magnet trap tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListMagnetTrapLogs(ctx context.Context, month string, room string, machineNo string) ([]domain.MagnetTrapLog, error) {
	list, err := uc.repo.ListMagnetTrapLogs(ctx, month, room, machineNo)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil daftar pemakaian magnet trap")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateMagnetTrapLog(ctx context.Context, id uuid.UUID, dto UpdateMagnetTrapLogDTO) (*domain.MagnetTrapLog, error) {
	item, err := uc.repo.GetMagnetTrapLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data pemakaian magnet trap tidak ditemukan")
	}

	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
	}
	if dto.Room != "" {
		item.Room = dto.Room
	}
	if dto.MachineNo != "" {
		item.MachineNo = dto.MachineNo
	}
	if dto.PICResponsible != "" {
		item.PICResponsible = dto.PICResponsible
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.ProductionStepID != nil {
		item.ProductionStepID = dto.ProductionStepID
	}
	if dto.BatchNo != "" {
		item.BatchNo = dto.BatchNo
	}
	if dto.IsClean != nil {
		item.IsClean = *dto.IsClean
	}
	if dto.GaussPoint1 != nil {
		item.GaussPoint1 = *dto.GaussPoint1
	}
	if dto.GaussPoint2 != nil {
		item.GaussPoint2 = *dto.GaussPoint2
	}
	if dto.GaussPoint3 != nil {
		item.GaussPoint3 = *dto.GaussPoint3
	}

	avg := (item.GaussPoint1 + item.GaussPoint2 + item.GaussPoint3) / 3.0
	item.GaussAverage = math.Round(avg*100) / 100
	item.IsPowerSufficient = item.GaussPoint1 >= 700.0 && item.GaussPoint2 >= 700.0 && item.GaussPoint3 >= 700.0

	if dto.ForeignMatterFound != "" {
		item.ForeignMatterFound = dto.ForeignMatterFound
	}
	if dto.Issues != "" {
		item.Issues = dto.Issues
	}
	if dto.ActionTaken != "" {
		item.ActionTaken = dto.ActionTaken
	}
	if dto.PICName != "" {
		item.PICName = dto.PICName
	}
	if dto.QCName != "" {
		item.QCName = dto.QCName
	}
	if dto.QCPassed != nil {
		item.QCPassed = *dto.QCPassed
	}
	if dto.QAVerifiedBy != "" {
		item.QAVerifiedBy = dto.QAVerifiedBy
		now := time.Now()
		item.QAVerifiedAt = &now
	}
	if dto.OverallStatus != "" {
		item.OverallStatus = dto.OverallStatus
	} else if !item.IsPowerSufficient {
		item.OverallStatus = "warning"
	}
	if dto.Notes != "" {
		item.Notes = dto.Notes
	}

	if err := uc.repo.UpdateMagnetTrapLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui catatan magnet trap")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteMagnetTrapLog(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteMagnetTrapLog(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Mesh Bolt Monitoring (MNM — Form QC-010-MNM.01)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateMeshBoltLog(ctx context.Context, dto CreateMeshBoltLogDTO) (*domain.MeshBoltMonitoringLog, error) {
	if strings.TrimSpace(dto.MachineNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor mesin wajib diisi")
	}
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan wajib diisi")
	}
	if strings.TrimSpace(dto.MeshSize) == "" {
		return nil, apperrors.NewBadRequest("Ukuran mesh wajib diisi")
	}
	if strings.TrimSpace(dto.PICResponsible) == "" || strings.TrimSpace(dto.InitialPICName) == "" {
		return nil, apperrors.NewBadRequest("Penanggung jawab dan nama PIC wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}
	room := dto.Room
	if room == "" {
		room = "Ayak Mentah"
	}

	prefixTag := "MNM-"
	if strings.Contains(strings.ToLower(room), "matang") {
		prefixTag = "MNM-MAT-"
	}
	prefixWithDate := fmt.Sprintf("%s%s-", prefixTag, time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_mesh_bolt_monitoring_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	initCount := dto.InitialBoltCount
	if initCount <= 0 {
		initCount = 16
	}
	finalCount := dto.FinalBoltCount
	if finalCount <= 0 {
		finalCount = initCount
	}

	initComplete := true
	if dto.InitialIsComplete != nil {
		initComplete = *dto.InitialIsComplete
	}
	initTight := true
	if dto.InitialIsTight != nil {
		initTight = *dto.InitialIsTight
	}
	finalComplete := true
	if dto.FinalIsComplete != nil {
		finalComplete = *dto.FinalIsComplete
	}
	finalTight := true
	if dto.FinalIsTight != nil {
		finalTight = *dto.FinalIsTight
	}

	countMatched := initCount == finalCount

	status := "passed"
	if !countMatched || !finalComplete {
		status = "hazard_alert" // Potensi bahaya kontaminasi fisik mur/baut hilang!
	}

	var qaAt *time.Time
	if dto.QAVerifiedBy != "" {
		now := time.Now()
		qaAt = &now
	}

	initCond := dto.InitialCondition
	if initCond == "" {
		initCond = "SS304 Utuh & Bebas Karat"
	}

	item := &domain.MeshBoltMonitoringLog{
		DocumentNo:        docNo,
		CheckMonth:        month,
		CheckDate:         dto.CheckDate,
		Room:              room,
		MachineNo:         dto.MachineNo,
		MeshSize:          dto.MeshSize,
		PICResponsible:    dto.PICResponsible,
		ProductionOrderID: dto.ProductionOrderID,
		ProductionStepID:  dto.ProductionStepID,
		BatchNo:           dto.BatchNo,
		InitialBoltCount:  initCount,
		InitialIsComplete: initComplete,
		InitialIsTight:    initTight,
		InitialCondition:  initCond,
		InitialFindings:   dto.InitialFindings,
		InitialAction:     dto.InitialAction,
		InitialPICName:    dto.InitialPICName,
		InitialQCName:     dto.InitialQCName,
		FinalBoltCount:    finalCount,
		FinalIsComplete:   finalComplete,
		FinalIsTight:      finalTight,
		IsCountMatched:    countMatched,
		FinalFindings:     dto.FinalFindings,
		FinalAction:       dto.FinalAction,
		FinalPICName:      dto.FinalPICName,
		FinalQCName:       dto.FinalQCName,
		QAVerifiedBy:      dto.QAVerifiedBy,
		QAVerifiedAt:      qaAt,
		OverallStatus:     status,
		Notes:             dto.Notes,
	}

	if err := uc.repo.CreateMeshBoltLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mencatat kondisi mur & baut mesh ayakan")
	}

	if !item.IsCountMatched || item.OverallStatus == "hazard_alert" {
		defect := "Bahaya Fisik Baut Lepas: Jumlah baut awal dan akhir tidak cocok"
		if item.FinalFindings != "" {
			defect += "; " + item.FinalFindings
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageMeshBolt,
			SourceInspectionID: &item.ID,
			SourceInspectionNo: item.DocumentNo,
			ProductionOrderID:  item.ProductionOrderID,
			BatchNo:            item.BatchNo,
			ProductName:        "Gula Semut (Line " + item.Room + ")",
			MachineNo:          item.MachineNo,
			MeshSize:           item.MeshSize,
			MaterialType:       "BM",
			DefectReason:       defect,
			ActionDisposition:  domain.ActionHold,
			OperatorName:       item.FinalPICName,
			QCInspectorName:    item.FinalQCName,
			Notes:              item.Notes,
		})
	}

	return item, nil
}

func (uc *qcUseCase) GetMeshBoltLogByID(ctx context.Context, id uuid.UUID) (*domain.MeshBoltMonitoringLog, error) {
	item, err := uc.repo.GetMeshBoltLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring mur & baut tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListMeshBoltLogs(ctx context.Context, month string, room string, machineNo string) ([]domain.MeshBoltMonitoringLog, error) {
	list, err := uc.repo.ListMeshBoltLogs(ctx, month, room, machineNo)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil daftar monitoring mur & baut")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateMeshBoltLog(ctx context.Context, id uuid.UUID, dto UpdateMeshBoltLogDTO) (*domain.MeshBoltMonitoringLog, error) {
	item, err := uc.repo.GetMeshBoltLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring mur & baut tidak ditemukan")
	}

	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
	}
	if dto.Room != "" {
		item.Room = dto.Room
	}
	if dto.MachineNo != "" {
		item.MachineNo = dto.MachineNo
	}
	if dto.MeshSize != "" {
		item.MeshSize = dto.MeshSize
	}
	if dto.PICResponsible != "" {
		item.PICResponsible = dto.PICResponsible
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.ProductionStepID != nil {
		item.ProductionStepID = dto.ProductionStepID
	}
	if dto.BatchNo != "" {
		item.BatchNo = dto.BatchNo
	}
	if dto.InitialBoltCount != nil {
		item.InitialBoltCount = *dto.InitialBoltCount
	}
	if dto.InitialIsComplete != nil {
		item.InitialIsComplete = *dto.InitialIsComplete
	}
	if dto.InitialIsTight != nil {
		item.InitialIsTight = *dto.InitialIsTight
	}
	if dto.InitialCondition != "" {
		item.InitialCondition = dto.InitialCondition
	}
	if dto.InitialFindings != "" {
		item.InitialFindings = dto.InitialFindings
	}
	if dto.InitialAction != "" {
		item.InitialAction = dto.InitialAction
	}
	if dto.InitialPICName != "" {
		item.InitialPICName = dto.InitialPICName
	}
	if dto.InitialQCName != "" {
		item.InitialQCName = dto.InitialQCName
	}
	if dto.FinalBoltCount != nil {
		item.FinalBoltCount = *dto.FinalBoltCount
	}
	if dto.FinalIsComplete != nil {
		item.FinalIsComplete = *dto.FinalIsComplete
	}
	if dto.FinalIsTight != nil {
		item.FinalIsTight = *dto.FinalIsTight
	}

	item.IsCountMatched = item.InitialBoltCount == item.FinalBoltCount

	if dto.FinalFindings != "" {
		item.FinalFindings = dto.FinalFindings
	}
	if dto.FinalAction != "" {
		item.FinalAction = dto.FinalAction
	}
	if dto.FinalPICName != "" {
		item.FinalPICName = dto.FinalPICName
	}
	if dto.FinalQCName != "" {
		item.FinalQCName = dto.FinalQCName
	}
	if dto.QAVerifiedBy != "" {
		item.QAVerifiedBy = dto.QAVerifiedBy
		now := time.Now()
		item.QAVerifiedAt = &now
	}
	if dto.OverallStatus != "" {
		item.OverallStatus = dto.OverallStatus
	} else if !item.IsCountMatched || !item.FinalIsComplete {
		item.OverallStatus = "hazard_alert"
	}
	if dto.Notes != "" {
		item.Notes = dto.Notes
	}

	if err := uc.repo.UpdateMeshBoltLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui catatan mur & baut")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteMeshBoltLog(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteMeshBoltLog(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Oven Monitoring (MNO — Form QC-012-MNO.05)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateOvenLog(ctx context.Context, dto CreateOvenLogDTO) (*domain.OvenMonitoringLog, error) {
	if strings.TrimSpace(dto.OvenNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor oven wajib diisi")
	}
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan wajib diisi")
	}
	if strings.TrimSpace(dto.BatchNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor batch wajib diisi")
	}
	if strings.TrimSpace(dto.ProductName) == "" {
		return nil, apperrors.NewBadRequest("Nama produk wajib diisi")
	}
	if strings.TrimSpace(dto.InitialTime) == "" || strings.TrimSpace(dto.StabilizedTime) == "" || strings.TrimSpace(dto.FinalTime) == "" {
		return nil, apperrors.NewBadRequest("Jam masuk, stabil, dan jam keluar oven wajib diisi")
	}
	if strings.TrimSpace(dto.PICName) == "" {
		return nil, apperrors.NewBadRequest("Nama operator/PIC wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	prefixWithDate := fmt.Sprintf("MNO-%s-", time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_oven_monitoring_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	// Target Moisture Content <= 2.0%
	targetReached := dto.FinalMoisture <= 2.0 && dto.FinalMoisture > 0

	status := "passed"
	if !targetReached {
		status = "re_oven" // Kadar air masih melebihi toleransi maksimal 2.0%!
	}

	qcPassed := targetReached
	if dto.QCPassed != nil {
		qcPassed = *dto.QCPassed
	}

	item := &domain.OvenMonitoringLog{
		DocumentNo:              docNo,
		RevisionCode:            "05",
		CheckMonth:              month,
		CheckDate:               dto.CheckDate,
		OvenNo:                  dto.OvenNo,
		ProductionOrderID:       dto.ProductionOrderID,
		ProductionStepID:        dto.ProductionStepID,
		BatchNo:                 dto.BatchNo,
		ProductName:             dto.ProductName,
		InitialTemperature:      dto.InitialTemperature,
		InitialTime:             dto.InitialTime,
		InitialMoisture:         dto.InitialMoisture,
		StabilizedTime:          dto.StabilizedTime,
		ProcessTemperature:      dto.ProcessTemperature,
		ProcessMoisture1:        dto.ProcessMoisture1,
		ProcessMoisture2:        dto.ProcessMoisture2,
		ProcessMoisture3:        dto.ProcessMoisture3,
		ProcessMoisture4:        dto.ProcessMoisture4,
		FinalTemperature:        dto.FinalTemperature,
		FinalTime:               dto.FinalTime,
		FinalMoisture:           dto.FinalMoisture,
		IsTargetMoistureReached: targetReached,
		Issues:                  dto.Issues,
		ActionTaken:             dto.ActionTaken,
		PICName:                 dto.PICName,
		QCVerifierName:          dto.QCVerifierName,
		QCPassed:                qcPassed,
		OverallStatus:           status,
		Notes:                   dto.Notes,
	}

	if err := uc.repo.CreateOvenLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mencatat monitoring oven")
	}

	if !item.QCPassed || !item.IsTargetMoistureReached || item.OverallStatus == "failed" || item.OverallStatus == "re_oven" {
		defect := fmt.Sprintf("Kadar air oven %.2f%% (target <= 2.0%%)", item.FinalMoisture)
		if item.Issues != "" {
			defect += "; " + item.Issues
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageOven,
			SourceInspectionID: &item.ID,
			SourceInspectionNo: item.DocumentNo,
			ProductionOrderID:  item.ProductionOrderID,
			ProductName:        item.ProductName,
			BatchNo:            item.BatchNo,
			MachineNo:          item.OvenNo,
			MaterialType:       "BO",
			DefectReason:       defect,
			ActionDisposition:  domain.ActionReOven,
			OperatorName:       item.PICName,
			QCInspectorName:    item.QCVerifierName,
			Notes:              item.Notes,
		})
	}

	return item, nil
}

func (uc *qcUseCase) GetOvenLogByID(ctx context.Context, id uuid.UUID) (*domain.OvenMonitoringLog, error) {
	item, err := uc.repo.GetOvenLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring oven tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListOvenLogs(ctx context.Context, month string, ovenNo string, batchNo string) ([]domain.OvenMonitoringLog, error) {
	list, err := uc.repo.ListOvenLogs(ctx, month, ovenNo, batchNo)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil daftar monitoring oven")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateOvenLog(ctx context.Context, id uuid.UUID, dto UpdateOvenLogDTO) (*domain.OvenMonitoringLog, error) {
	item, err := uc.repo.GetOvenLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring oven tidak ditemukan")
	}

	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
	}
	if dto.OvenNo != "" {
		item.OvenNo = dto.OvenNo
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.ProductionStepID != nil {
		item.ProductionStepID = dto.ProductionStepID
	}
	if dto.BatchNo != "" {
		item.BatchNo = dto.BatchNo
	}
	if dto.ProductName != "" {
		item.ProductName = dto.ProductName
	}
	if dto.InitialTemperature != nil {
		item.InitialTemperature = *dto.InitialTemperature
	}
	if dto.InitialTime != "" {
		item.InitialTime = dto.InitialTime
	}
	if dto.InitialMoisture != nil {
		item.InitialMoisture = *dto.InitialMoisture
	}
	if dto.StabilizedTime != "" {
		item.StabilizedTime = dto.StabilizedTime
	}
	if dto.ProcessTemperature != nil {
		item.ProcessTemperature = *dto.ProcessTemperature
	}
	if dto.ProcessMoisture1 != nil {
		item.ProcessMoisture1 = dto.ProcessMoisture1
	}
	if dto.ProcessMoisture2 != nil {
		item.ProcessMoisture2 = dto.ProcessMoisture2
	}
	if dto.ProcessMoisture3 != nil {
		item.ProcessMoisture3 = dto.ProcessMoisture3
	}
	if dto.ProcessMoisture4 != nil {
		item.ProcessMoisture4 = dto.ProcessMoisture4
	}
	if dto.FinalTemperature != nil {
		item.FinalTemperature = *dto.FinalTemperature
	}
	if dto.FinalTime != "" {
		item.FinalTime = dto.FinalTime
	}
	if dto.FinalMoisture != nil {
		item.FinalMoisture = *dto.FinalMoisture
		item.IsTargetMoistureReached = item.FinalMoisture <= 2.0 && item.FinalMoisture > 0
		if !item.IsTargetMoistureReached {
			item.OverallStatus = "re_oven"
		} else {
			item.OverallStatus = "passed"
		}
	}
	if dto.Issues != "" {
		item.Issues = dto.Issues
	}
	if dto.ActionTaken != "" {
		item.ActionTaken = dto.ActionTaken
	}
	if dto.PICName != "" {
		item.PICName = dto.PICName
	}
	if dto.QCVerifierName != "" {
		item.QCVerifierName = dto.QCVerifierName
	}
	if dto.QCPassed != nil {
		item.QCPassed = *dto.QCPassed
	}
	if dto.OverallStatus != "" {
		item.OverallStatus = dto.OverallStatus
	}
	if dto.Notes != "" {
		item.Notes = dto.Notes
	}

	if err := uc.repo.UpdateOvenLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui catatan monitoring oven")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteOvenLog(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteOvenLog(ctx, id)
}
