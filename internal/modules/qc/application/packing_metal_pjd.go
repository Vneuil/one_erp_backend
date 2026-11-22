package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// MNJ: Needle Monitoring Log (Form QC-011-MNJ.01)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateNeedleLog(ctx context.Context, dto CreateNeedleLogDTO) (*domain.NeedleMonitoringLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan wajib diisi")
	}
	if strings.TrimSpace(dto.SewingMachineNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor mesin jahit wajib diisi")
	}
	if strings.TrimSpace(dto.PICResponsible) == "" || strings.TrimSpace(dto.PICName) == "" {
		return nil, apperrors.NewBadRequest("PIC penanggung jawab dan pemeriksa wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	prefixWithDate := fmt.Sprintf("MNJ-%s-", time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_needle_monitoring_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	initPresent := true
	if dto.InitialIsPresent != nil {
		initPresent = *dto.InitialIsPresent
	}
	initCond := dto.InitialCondition
	if initCond == "" {
		initCond = "Baik"
	}

	finalPresent := true
	if dto.FinalIsPresent != nil {
		finalPresent = *dto.FinalIsPresent
	}
	finalCond := dto.FinalCondition
	if finalCond == "" {
		finalCond = "Baik"
	}

	overallStatus := "passed"
	if !initPresent || !finalPresent || strings.EqualFold(initCond, "Patah") || strings.EqualFold(finalCond, "Patah") {
		overallStatus = "hazard_alert"
	}

	log := &domain.NeedleMonitoringLog{
		DocumentNo:        docNo,
		RevisionCode:      "01",
		CheckMonth:        month,
		CheckDate:         dto.CheckDate,
		SewingMachineNo:   strings.TrimSpace(dto.SewingMachineNo),
		PICResponsible:    strings.TrimSpace(dto.PICResponsible),
		InitialIsPresent:  initPresent,
		InitialCondition:  initCond,
		InitialAnalysis:   dto.InitialAnalysis,
		InitialAction:     dto.InitialAction,
		FinalIsPresent:    finalPresent,
		FinalCondition:    finalCond,
		FinalAnalysis:     dto.FinalAnalysis,
		FinalAction:       dto.FinalAction,
		NeedleStockInRoom: dto.NeedleStockInRoom,
		PICName:           strings.TrimSpace(dto.PICName),
		QCVerifierName:    strings.TrimSpace(dto.QCVerifierName),
		OverallStatus:     overallStatus,
		Notes:             dto.Notes,
	}

	if err := uc.repo.CreateNeedleLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return log, nil
}

func (uc *qcUseCase) GetNeedleLogByID(ctx context.Context, id uuid.UUID) (*domain.NeedleMonitoringLog, error) {
	log, err := uc.repo.GetNeedleLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Log monitoring jarum tidak ditemukan")
	}
	return log, nil
}

func (uc *qcUseCase) ListNeedleLogs(ctx context.Context, month string, machineNo string) ([]domain.NeedleMonitoringLog, error) {
	logs, err := uc.repo.ListNeedleLogs(ctx, month, machineNo)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return logs, nil
}

func (uc *qcUseCase) UpdateNeedleLog(ctx context.Context, id uuid.UUID, dto UpdateNeedleLogDTO) (*domain.NeedleMonitoringLog, error) {
	log, err := uc.repo.GetNeedleLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Log monitoring jarum tidak ditemukan")
	}

	if dto.CheckDate != "" {
		log.CheckDate = dto.CheckDate
	}
	if dto.CheckMonth != "" {
		log.CheckMonth = dto.CheckMonth
	}
	if dto.SewingMachineNo != "" {
		log.SewingMachineNo = dto.SewingMachineNo
	}
	if dto.PICResponsible != "" {
		log.PICResponsible = dto.PICResponsible
	}
	if dto.InitialIsPresent != nil {
		log.InitialIsPresent = *dto.InitialIsPresent
	}
	if dto.InitialCondition != "" {
		log.InitialCondition = dto.InitialCondition
	}
	if dto.InitialAnalysis != "" {
		log.InitialAnalysis = dto.InitialAnalysis
	}
	if dto.InitialAction != "" {
		log.InitialAction = dto.InitialAction
	}
	if dto.FinalIsPresent != nil {
		log.FinalIsPresent = *dto.FinalIsPresent
	}
	if dto.FinalCondition != "" {
		log.FinalCondition = dto.FinalCondition
	}
	if dto.FinalAnalysis != "" {
		log.FinalAnalysis = dto.FinalAnalysis
	}
	if dto.FinalAction != "" {
		log.FinalAction = dto.FinalAction
	}
	if dto.NeedleStockInRoom != nil {
		log.NeedleStockInRoom = *dto.NeedleStockInRoom
	}
	if dto.PICName != "" {
		log.PICName = dto.PICName
	}
	if dto.QCVerifierName != "" {
		log.QCVerifierName = dto.QCVerifierName
	}
	if dto.Notes != "" {
		log.Notes = dto.Notes
	}

	if dto.OverallStatus != "" {
		log.OverallStatus = dto.OverallStatus
	} else {
		if !log.InitialIsPresent || !log.FinalIsPresent || strings.EqualFold(log.InitialCondition, "Patah") || strings.EqualFold(log.FinalCondition, "Patah") {
			log.OverallStatus = "hazard_alert"
		} else {
			log.OverallStatus = "passed"
		}
	}

	if err := uc.repo.UpdateNeedleLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return log, nil
}

func (uc *qcUseCase) DeleteNeedleLog(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteNeedleLog(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MNG: Scissor Monitoring Log (Form QC-011-MNJ.01 Sheet MNG)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateScissorLog(ctx context.Context, dto CreateScissorLogDTO) (*domain.ScissorMonitoringLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan wajib diisi")
	}
	if strings.TrimSpace(dto.PICResponsible) == "" || strings.TrimSpace(dto.PICName) == "" {
		return nil, apperrors.NewBadRequest("PIC penanggung jawab dan pemeriksa wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	loc := dto.Location
	if loc == "" {
		loc = "Ruang Packing"
	}

	initCond := dto.InitialCondition
	if initCond == "" {
		initCond = "Baik"
	}
	finalCond := dto.FinalCondition
	if finalCond == "" {
		finalCond = "Baik"
	}

	initCount := dto.InitialCount
	if initCount <= 0 {
		initCount = 1
	}
	finalCount := dto.FinalCount
	if finalCount <= 0 {
		finalCount = initCount
	}

	isMatched := (initCount == finalCount)
	overallStatus := "passed"
	if !isMatched || strings.EqualFold(initCond, "Patah") || strings.EqualFold(finalCond, "Patah") {
		overallStatus = "hazard_alert"
	}

	prefixWithDate := fmt.Sprintf("MNG-%s-", time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_scissor_monitoring_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	log := &domain.ScissorMonitoringLog{
		DocumentNo:       docNo,
		RevisionCode:     "01",
		CheckMonth:       month,
		CheckDate:        dto.CheckDate,
		Location:         loc,
		PICResponsible:   strings.TrimSpace(dto.PICResponsible),
		InitialCount:     initCount,
		InitialCondition: initCond,
		InitialAnalysis:  dto.InitialAnalysis,
		InitialAction:    dto.InitialAction,
		FinalCount:       finalCount,
		FinalCondition:   finalCond,
		FinalAnalysis:    dto.FinalAnalysis,
		FinalAction:      dto.FinalAction,
		IsCountMatched:   isMatched,
		PICName:          strings.TrimSpace(dto.PICName),
		QCVerifierName:   strings.TrimSpace(dto.QCVerifierName),
		OverallStatus:    overallStatus,
		Notes:            dto.Notes,
	}

	if err := uc.repo.CreateScissorLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return log, nil
}

func (uc *qcUseCase) GetScissorLogByID(ctx context.Context, id uuid.UUID) (*domain.ScissorMonitoringLog, error) {
	log, err := uc.repo.GetScissorLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Log monitoring gunting tidak ditemukan")
	}
	return log, nil
}

func (uc *qcUseCase) ListScissorLogs(ctx context.Context, month string, location string) ([]domain.ScissorMonitoringLog, error) {
	logs, err := uc.repo.ListScissorLogs(ctx, month, location)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return logs, nil
}

func (uc *qcUseCase) UpdateScissorLog(ctx context.Context, id uuid.UUID, dto UpdateScissorLogDTO) (*domain.ScissorMonitoringLog, error) {
	log, err := uc.repo.GetScissorLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Log monitoring gunting tidak ditemukan")
	}

	if dto.CheckDate != "" {
		log.CheckDate = dto.CheckDate
	}
	if dto.CheckMonth != "" {
		log.CheckMonth = dto.CheckMonth
	}
	if dto.Location != "" {
		log.Location = dto.Location
	}
	if dto.PICResponsible != "" {
		log.PICResponsible = dto.PICResponsible
	}
	if dto.InitialCount != nil {
		log.InitialCount = *dto.InitialCount
	}
	if dto.InitialCondition != "" {
		log.InitialCondition = dto.InitialCondition
	}
	if dto.InitialAnalysis != "" {
		log.InitialAnalysis = dto.InitialAnalysis
	}
	if dto.InitialAction != "" {
		log.InitialAction = dto.InitialAction
	}
	if dto.FinalCount != nil {
		log.FinalCount = *dto.FinalCount
	}
	if dto.FinalCondition != "" {
		log.FinalCondition = dto.FinalCondition
	}
	if dto.FinalAnalysis != "" {
		log.FinalAnalysis = dto.FinalAnalysis
	}
	if dto.FinalAction != "" {
		log.FinalAction = dto.FinalAction
	}
	if dto.PICName != "" {
		log.PICName = dto.PICName
	}
	if dto.QCVerifierName != "" {
		log.QCVerifierName = dto.QCVerifierName
	}
	if dto.Notes != "" {
		log.Notes = dto.Notes
	}

	log.IsCountMatched = (log.InitialCount == log.FinalCount)
	if dto.OverallStatus != "" {
		log.OverallStatus = dto.OverallStatus
	} else {
		if !log.IsCountMatched || strings.EqualFold(log.InitialCondition, "Patah") || strings.EqualFold(log.FinalCondition, "Patah") {
			log.OverallStatus = "hazard_alert"
		} else {
			log.OverallStatus = "passed"
		}
	}

	if err := uc.repo.UpdateScissorLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return log, nil
}

func (uc *qcUseCase) DeleteScissorLog(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteScissorLog(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// CPP: Metal Detector Cleaning Log (Form CPP)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateMetalCleaningLog(ctx context.Context, dto CreateMetalCleaningLogDTO) (*domain.MetalDetectorCleaningLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pembersihan wajib diisi")
	}
	if strings.TrimSpace(dto.StartTime) == "" || strings.TrimSpace(dto.EndTime) == "" {
		return nil, apperrors.NewBadRequest("Jam mulai dan jam selesai wajib diisi")
	}
	if strings.TrimSpace(dto.OperatorName) == "" {
		return nil, apperrors.NewBadRequest("Nama operator pelaksana wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	machineNo := dto.MachineNo
	if machineNo == "" {
		machineNo = "MD-01"
	}
	loc := dto.Location
	if loc == "" {
		loc = "Ruang Packing"
	}
	cleaningType := dto.CleaningType
	if cleaningType == "" {
		cleaningType = "pre_operational"
	}

	sensorClean := dto.SensorCleanliness
	if sensorClean == "" {
		sensorClean = "Bersih"
	}
	beltClean := dto.BeltCleanliness
	if beltClean == "" {
		beltClean = "Bersih"
	}
	binClean := dto.RejectBinCleanliness
	if binClean == "" {
		binClean = "Bersih & Kosong"
	}
	cond := dto.PhysicalCondition
	if cond == "" {
		cond = "Normal"
	}

	overallStatus := "passed"
	if strings.EqualFold(sensorClean, "Perlu Dibersihkan") || strings.EqualFold(beltClean, "Perlu Dibersihkan") {
		overallStatus = "needs_cleaning"
	} else if strings.EqualFold(cond, "Cacat") || strings.EqualFold(cond, "Kabel Terbuka") {
		overallStatus = "maintenance"
	}

	prefixWithDate := fmt.Sprintf("CPP-%s-", time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_metal_detector_cleaning_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	log := &domain.MetalDetectorCleaningLog{
		DocumentNo:           docNo,
		CheckMonth:           month,
		CheckDate:            dto.CheckDate,
		MachineNo:            machineNo,
		Location:             loc,
		CleaningType:         cleaningType,
		SensorCleanliness:    sensorClean,
		BeltCleanliness:      beltClean,
		RejectBinCleanliness: binClean,
		PhysicalCondition:    cond,
		StartTime:            dto.StartTime,
		EndTime:              dto.EndTime,
		TotalHours:           dto.TotalHours,
		OperatorName:         strings.TrimSpace(dto.OperatorName),
		QCInspectorName:      strings.TrimSpace(dto.QCInspectorName),
		OverallStatus:        overallStatus,
		Notes:                dto.Notes,
	}

	if err := uc.repo.CreateMetalCleaningLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return log, nil
}

func (uc *qcUseCase) GetMetalCleaningLogByID(ctx context.Context, id uuid.UUID) (*domain.MetalDetectorCleaningLog, error) {
	log, err := uc.repo.GetMetalCleaningLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Catatan pembersihan metal detector tidak ditemukan")
	}
	return log, nil
}

func (uc *qcUseCase) ListMetalCleaningLogs(ctx context.Context, month string, machineNo string, location string) ([]domain.MetalDetectorCleaningLog, error) {
	logs, err := uc.repo.ListMetalCleaningLogs(ctx, month, machineNo, location)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return logs, nil
}

func (uc *qcUseCase) UpdateMetalCleaningLog(ctx context.Context, id uuid.UUID, dto UpdateMetalCleaningLogDTO) (*domain.MetalDetectorCleaningLog, error) {
	log, err := uc.repo.GetMetalCleaningLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Catatan pembersihan metal detector tidak ditemukan")
	}

	if dto.CheckDate != "" {
		log.CheckDate = dto.CheckDate
	}
	if dto.CheckMonth != "" {
		log.CheckMonth = dto.CheckMonth
	}
	if dto.MachineNo != "" {
		log.MachineNo = dto.MachineNo
	}
	if dto.Location != "" {
		log.Location = dto.Location
	}
	if dto.CleaningType != "" {
		log.CleaningType = dto.CleaningType
	}
	if dto.SensorCleanliness != "" {
		log.SensorCleanliness = dto.SensorCleanliness
	}
	if dto.BeltCleanliness != "" {
		log.BeltCleanliness = dto.BeltCleanliness
	}
	if dto.RejectBinCleanliness != "" {
		log.RejectBinCleanliness = dto.RejectBinCleanliness
	}
	if dto.PhysicalCondition != "" {
		log.PhysicalCondition = dto.PhysicalCondition
	}
	if dto.StartTime != "" {
		log.StartTime = dto.StartTime
	}
	if dto.EndTime != "" {
		log.EndTime = dto.EndTime
	}
	if dto.TotalHours != nil {
		log.TotalHours = *dto.TotalHours
	}
	if dto.OperatorName != "" {
		log.OperatorName = dto.OperatorName
	}
	if dto.QCInspectorName != "" {
		log.QCInspectorName = dto.QCInspectorName
	}
	if dto.Notes != "" {
		log.Notes = dto.Notes
	}

	if dto.OverallStatus != "" {
		log.OverallStatus = dto.OverallStatus
	}

	if err := uc.repo.UpdateMetalCleaningLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return log, nil
}

func (uc *qcUseCase) DeleteMetalCleaningLog(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteMetalCleaningLog(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// VMD: Metal Detector Verification Log (Form QC-017-VMD)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateMetalVerificationLog(ctx context.Context, dto CreateMetalVerificationLogDTO) (*domain.MetalDetectorVerificationLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal verifikasi wajib diisi")
	}
	if strings.TrimSpace(dto.CheckTime) == "" {
		return nil, apperrors.NewBadRequest("Jam verifikasi wajib diisi")
	}
	if strings.TrimSpace(dto.PICName) == "" {
		return nil, apperrors.NewBadRequest("Nama PIC pemeriksa wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	productType := dto.ProductType
	if productType == "" {
		productType = "gula_semut"
	}
	loc := dto.Location
	if loc == "" {
		loc = "Ruang Packing"
	}
	pkgCat := dto.PackageCategory
	if pkgCat == "" {
		pkgCat = "<= 12.5 kg"
	}
	round := dto.MonitoringRound
	if round <= 0 {
		round = 1
	}

	// Test Piece Default Specs based on package weight category
	feSize := dto.FeTestPieceSize
	nonFeSize := dto.NonFeTestPieceSize
	ssSize := dto.SSTestPieceSize

	if feSize == "" {
		if pkgCat == "> 12.5 kg" {
			feSize = "Fe >= 2.5mm"
		} else {
			feSize = "Fe >= 2.0mm"
		}
	}
	if nonFeSize == "" {
		if pkgCat == "> 12.5 kg" {
			nonFeSize = "Non-Fe >= 3.0mm"
		} else {
			nonFeSize = "Non-Fe >= 2.0mm"
		}
	}
	if ssSize == "" {
		ssSize = "SS >= 3.5mm"
	}

	// Fe piece
	feAlarm := true
	if dto.FeAlarmTriggered != nil {
		feAlarm = *dto.FeAlarmTriggered
	}
	feConv := true
	if dto.FeConveyorStopped != nil {
		feConv = *dto.FeConveyorStopped
	}
	fePassed := feAlarm && feConv

	// Non-Fe piece
	nonFeAlarm := true
	if dto.NonFeAlarmTriggered != nil {
		nonFeAlarm = *dto.NonFeAlarmTriggered
	}
	nonFeConv := true
	if dto.NonFeConveyorStopped != nil {
		nonFeConv = *dto.NonFeConveyorStopped
	}
	nonFePassed := nonFeAlarm && nonFeConv

	// SS piece
	ssAlarm := true
	if dto.SSAlarmTriggered != nil {
		ssAlarm = *dto.SSAlarmTriggered
	}
	ssConv := true
	if dto.SSConveyorStopped != nil {
		ssConv = *dto.SSConveyorStopped
	}
	ssPassed := ssAlarm && ssConv

	allPassed := fePassed && nonFePassed && ssPassed
	overallStatus := "passed"
	if !allPassed {
		overallStatus = "failed"
	}

	prefixWithDate := fmt.Sprintf("VMD-%s-", time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_metal_detector_verification_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	log := &domain.MetalDetectorVerificationLog{
		DocumentNo:           docNo,
		RevisionCode:         "03",
		CheckMonth:           month,
		CheckDate:            dto.CheckDate,
		ProductType:          productType,
		Location:             loc,
		PackageCategory:      pkgCat,
		MonitoringRound:      round,
		CheckTime:            dto.CheckTime,
		FeTestPieceSize:      feSize,
		FeAlarmTriggered:     feAlarm,
		FeConveyorStopped:    feConv,
		FePassed:             fePassed,
		NonFeTestPieceSize:   nonFeSize,
		NonFeAlarmTriggered:  nonFeAlarm,
		NonFeConveyorStopped: nonFeConv,
		NonFePassed:          nonFePassed,
		SSTestPieceSize:      ssSize,
		SSAlarmTriggered:     ssAlarm,
		SSConveyorStopped:    ssConv,
		SSPassed:             ssPassed,
		IsAllPiecesPassed:    allPassed,
		CorrectiveAction:     dto.CorrectiveAction,
		PICName:              strings.TrimSpace(dto.PICName),
		QCVerifierName:       strings.TrimSpace(dto.QCVerifierName),
		OverallStatus:        overallStatus,
		Notes:                dto.Notes,
	}

	if err := uc.repo.CreateMetalVerificationLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	if !log.IsAllPiecesPassed || log.OverallStatus == "failed" {
		defect := "Uji CCP Metal Detector gagal: "
		if !log.FePassed {
			defect += "Fe tidak terdeteksi; "
		}
		if !log.NonFePassed {
			defect += "Non-Fe tidak terdeteksi; "
		}
		if !log.SSPassed {
			defect += "SS tidak terdeteksi; "
		}
		if log.CorrectiveAction != "" {
			defect += "Tindakan: " + log.CorrectiveAction
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageMetalDetector,
			SourceInspectionID: &log.ID,
			SourceInspectionNo: log.DocumentNo,
			ProductName:        "Produk Line Packing (" + log.ProductType + ")",
			BatchNo:            log.CheckDate + "-" + log.CheckTime,
			MaterialType:       "FG",
			DefectReason:       defect,
			ActionDisposition:  domain.ActionHold,
			OperatorName:       log.PICName,
			QCInspectorName:    log.QCVerifierName,
			Notes:              log.Notes,
		})
	}

	return log, nil
}

func (uc *qcUseCase) GetMetalVerificationLogByID(ctx context.Context, id uuid.UUID) (*domain.MetalDetectorVerificationLog, error) {
	log, err := uc.repo.GetMetalVerificationLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Verifikasi metal detector tidak ditemukan")
	}
	return log, nil
}

func (uc *qcUseCase) ListMetalVerificationLogs(ctx context.Context, month string, productType string, location string) ([]domain.MetalDetectorVerificationLog, error) {
	logs, err := uc.repo.ListMetalVerificationLogs(ctx, month, productType, location)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return logs, nil
}

func (uc *qcUseCase) UpdateMetalVerificationLog(ctx context.Context, id uuid.UUID, dto UpdateMetalVerificationLogDTO) (*domain.MetalDetectorVerificationLog, error) {
	log, err := uc.repo.GetMetalVerificationLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Verifikasi metal detector tidak ditemukan")
	}

	if dto.CheckDate != "" {
		log.CheckDate = dto.CheckDate
	}
	if dto.CheckMonth != "" {
		log.CheckMonth = dto.CheckMonth
	}
	if dto.ProductType != "" {
		log.ProductType = dto.ProductType
	}
	if dto.Location != "" {
		log.Location = dto.Location
	}
	if dto.PackageCategory != "" {
		log.PackageCategory = dto.PackageCategory
	}
	if dto.MonitoringRound != nil {
		log.MonitoringRound = *dto.MonitoringRound
	}
	if dto.CheckTime != "" {
		log.CheckTime = dto.CheckTime
	}
	if dto.FeTestPieceSize != "" {
		log.FeTestPieceSize = dto.FeTestPieceSize
	}
	if dto.FeAlarmTriggered != nil {
		log.FeAlarmTriggered = *dto.FeAlarmTriggered
	}
	if dto.FeConveyorStopped != nil {
		log.FeConveyorStopped = *dto.FeConveyorStopped
	}
	log.FePassed = log.FeAlarmTriggered && log.FeConveyorStopped

	if dto.NonFeTestPieceSize != "" {
		log.NonFeTestPieceSize = dto.NonFeTestPieceSize
	}
	if dto.NonFeAlarmTriggered != nil {
		log.NonFeAlarmTriggered = *dto.NonFeAlarmTriggered
	}
	if dto.NonFeConveyorStopped != nil {
		log.NonFeConveyorStopped = *dto.NonFeConveyorStopped
	}
	log.NonFePassed = log.NonFeAlarmTriggered && log.NonFeConveyorStopped

	if dto.SSTestPieceSize != "" {
		log.SSTestPieceSize = dto.SSTestPieceSize
	}
	if dto.SSAlarmTriggered != nil {
		log.SSAlarmTriggered = *dto.SSAlarmTriggered
	}
	if dto.SSConveyorStopped != nil {
		log.SSConveyorStopped = *dto.SSConveyorStopped
	}
	log.SSPassed = log.SSAlarmTriggered && log.SSConveyorStopped

	log.IsAllPiecesPassed = log.FePassed && log.NonFePassed && log.SSPassed
	if dto.OverallStatus != "" {
		log.OverallStatus = dto.OverallStatus
	} else {
		if log.IsAllPiecesPassed {
			log.OverallStatus = "passed"
		} else {
			log.OverallStatus = "failed"
		}
	}

	if dto.CorrectiveAction != "" {
		log.CorrectiveAction = dto.CorrectiveAction
	}
	if dto.PICName != "" {
		log.PICName = dto.PICName
	}
	if dto.QCVerifierName != "" {
		log.QCVerifierName = dto.QCVerifierName
	}
	if dto.Notes != "" {
		log.Notes = dto.Notes
	}

	if err := uc.repo.UpdateMetalVerificationLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return log, nil
}

func (uc *qcUseCase) DeleteMetalVerificationLog(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteMetalVerificationLog(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// PJD: Finished Goods Inspection Log (Form QC-004-PJD.05 Rev 05)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateFinishedGoodsLog(ctx context.Context, dto CreateFinishedGoodsDTO) (*domain.FinishedGoodsInspectionLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan wajib diisi")
	}
	if strings.TrimSpace(dto.ProductName) == "" {
		return nil, apperrors.NewBadRequest("Nama produk wajib diisi")
	}
	if strings.TrimSpace(dto.BatchNo) == "" {
		return nil, apperrors.NewBadRequest("Nomor batch wajib diisi")
	}
	if strings.TrimSpace(dto.QCPersonnelName) == "" {
		return nil, apperrors.NewBadRequest("Nama petugas QC wajib diisi")
	}

	month := dto.CheckMonth
	if month == "" && len(dto.CheckDate) >= 7 {
		month = dto.CheckDate[:7]
	}

	primaryPack := dto.PrimaryPackaging
	if primaryPack == "" {
		primaryPack = "Baik"
	}
	secPack := dto.SecondaryPackaging
	if secPack == "" {
		secPack = "Baik"
	}
	shape := dto.AppearanceShape
	if shape == "" {
		shape = "Serbuk/Butiran Kristal Homogen"
	}
	aroma := dto.Aroma
	if aroma == "" {
		aroma = "Khas Nira Gula Kelapa"
	}
	taste := dto.Taste
	if taste == "" {
		taste = "Manis Khas Alami"
	}

	colorScore := dto.ColorScore
	if colorScore <= 0 || colorScore > 3 {
		colorScore = 2
	}
	colorDesc := dto.ColorDescription
	if colorDesc == "" {
		switch colorScore {
		case 1:
			colorDesc = "Light Brown"
		case 2:
			colorDesc = "Normal"
		case 3:
			colorDesc = "Dark Brown"
		}
	}

	isFreeForeign := true
	if dto.IsFreeForeignMatter != nil {
		isFreeForeign = *dto.IsFreeForeignMatter
	}

	isMoistureOk := (dto.MoistureContent <= 2.0 && dto.MoistureContent > 0)
	decision := dto.InspectionDecision
	if decision == "" {
		if !isFreeForeign || !isMoistureOk || !strings.EqualFold(primaryPack, "Baik") {
			decision = "ditolak"
		} else {
			decision = "lolos"
		}
	}

	prefixWithDate := fmt.Sprintf("PJD-%s-", time.Now().Format("200601"))
	count, _ := uc.repo.CountLogsByPrefix(ctx, "qc_finished_goods_inspection_logs", prefixWithDate)
	docNo := fmt.Sprintf("%s%04d", prefixWithDate, count+1)

	log := &domain.FinishedGoodsInspectionLog{
		DocumentNo:          docNo,
		RevisionCode:        "05",
		CheckMonth:          month,
		CheckDate:           dto.CheckDate,
		ProductName:         strings.TrimSpace(dto.ProductName),
		BatchNo:             strings.TrimSpace(dto.BatchNo),
		RawMaterialCode:     strings.TrimSpace(dto.RawMaterialCode),
		ProductionOrderID:   dto.ProductionOrderID,
		PrimaryPackaging:    primaryPack,
		SecondaryPackaging:  secPack,
		NetWeight:           dto.NetWeight,
		GrossWeight:         dto.GrossWeight,
		AppearanceShape:     shape,
		Aroma:               aroma,
		ColorScore:          colorScore,
		ColorDescription:    colorDesc,
		Taste:               taste,
		IsFreeForeignMatter: isFreeForeign,
		ForeignMatterNotes:  dto.ForeignMatterNotes,
		MoistureContent:     dto.MoistureContent,
		IsMoistureStandard:  isMoistureOk,
		InspectionDecision:  decision,
		Notes:               dto.Notes,
		QCPersonnelName:     strings.TrimSpace(dto.QCPersonnelName),
		QAVerifiedBy:        strings.TrimSpace(dto.QAVerifiedBy),
	}

	if err := uc.repo.CreateFinishedGoodsLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}

	if log.InspectionDecision == "ditolak" || !log.IsMoistureStandard {
		defect := "PJD Ditolak: "
		if !log.IsMoistureStandard {
			defect += fmt.Sprintf("Kadar air %.2f%% > 2.0%%; ", log.MoistureContent)
		}
		if !log.IsFreeForeignMatter {
			defect += "Terdeteksi benda asing (" + log.ForeignMatterNotes + "); "
		}
		if !strings.EqualFold(log.PrimaryPackaging, "Baik") {
			defect += "Kemasan primer: " + log.PrimaryPackaging + "; "
		}
		if log.Notes != "" {
			defect += log.Notes
		}
		qty := log.NetWeight
		if qty <= 0 {
			qty = log.GrossWeight
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageFinishedGoods,
			SourceInspectionID: &log.ID,
			SourceInspectionNo: log.DocumentNo,
			ProductionOrderID:  log.ProductionOrderID,
			ProductName:        log.ProductName,
			BatchNo:            log.BatchNo,
			Quantity:           qty,
			Unit:               "KG",
			MaterialType:       "FG",
			DefectReason:       defect,
			ActionDisposition:  domain.ActionRework,
			QCInspectorName:    log.QCPersonnelName,
			Notes:              log.Notes,
		})
	}

	return log, nil
}

func (uc *qcUseCase) GetFinishedGoodsLogByID(ctx context.Context, id uuid.UUID) (*domain.FinishedGoodsInspectionLog, error) {
	log, err := uc.repo.GetFinishedGoodsLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Pemeriksaan produk jadi tidak ditemukan")
	}
	return log, nil
}

func (uc *qcUseCase) ListFinishedGoodsLogs(ctx context.Context, month string, productName string, batchNo string, decision string) ([]domain.FinishedGoodsInspectionLog, error) {
	logs, err := uc.repo.ListFinishedGoodsLogs(ctx, month, productName, batchNo, decision)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return logs, nil
}

func (uc *qcUseCase) UpdateFinishedGoodsLog(ctx context.Context, id uuid.UUID, dto UpdateFinishedGoodsDTO) (*domain.FinishedGoodsInspectionLog, error) {
	log, err := uc.repo.GetFinishedGoodsLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Pemeriksaan produk jadi tidak ditemukan")
	}

	if dto.CheckDate != "" {
		log.CheckDate = dto.CheckDate
	}
	if dto.CheckMonth != "" {
		log.CheckMonth = dto.CheckMonth
	}
	if dto.ProductName != "" {
		log.ProductName = dto.ProductName
	}
	if dto.BatchNo != "" {
		log.BatchNo = dto.BatchNo
	}
	if dto.RawMaterialCode != "" {
		log.RawMaterialCode = dto.RawMaterialCode
	}
	if dto.ProductionOrderID != nil {
		log.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.PrimaryPackaging != "" {
		log.PrimaryPackaging = dto.PrimaryPackaging
	}
	if dto.SecondaryPackaging != "" {
		log.SecondaryPackaging = dto.SecondaryPackaging
	}
	if dto.NetWeight != nil {
		log.NetWeight = *dto.NetWeight
	}
	if dto.GrossWeight != nil {
		log.GrossWeight = *dto.GrossWeight
	}
	if dto.AppearanceShape != "" {
		log.AppearanceShape = dto.AppearanceShape
	}
	if dto.Aroma != "" {
		log.Aroma = dto.Aroma
	}
	if dto.ColorScore != nil {
		log.ColorScore = *dto.ColorScore
		switch log.ColorScore {
		case 1:
			log.ColorDescription = "Light Brown"
		case 2:
			log.ColorDescription = "Normal"
		case 3:
			log.ColorDescription = "Dark Brown"
		}
	}
	if dto.ColorDescription != "" {
		log.ColorDescription = dto.ColorDescription
	}
	if dto.Taste != "" {
		log.Taste = dto.Taste
	}
	if dto.IsFreeForeignMatter != nil {
		log.IsFreeForeignMatter = *dto.IsFreeForeignMatter
	}
	if dto.ForeignMatterNotes != "" {
		log.ForeignMatterNotes = dto.ForeignMatterNotes
	}
	if dto.MoistureContent != nil {
		log.MoistureContent = *dto.MoistureContent
		log.IsMoistureStandard = (log.MoistureContent <= 2.0 && log.MoistureContent > 0)
	}
	if dto.InspectionDecision != "" {
		log.InspectionDecision = dto.InspectionDecision
	}
	if dto.Notes != "" {
		log.Notes = dto.Notes
	}
	if dto.QCPersonnelName != "" {
		log.QCPersonnelName = dto.QCPersonnelName
	}
	if dto.QAVerifiedBy != "" {
		log.QAVerifiedBy = dto.QAVerifiedBy
	}

	if err := uc.repo.UpdateFinishedGoodsLog(ctx, log); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return log, nil
}

func (uc *qcUseCase) DeleteFinishedGoodsLog(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteFinishedGoodsLog(ctx, id); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}
