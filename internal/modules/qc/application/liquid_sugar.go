package application

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/divinecoid/one-backend/internal/shared/docseq"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// computeMinutesBetween calculates elapsed minutes between two HH:mm strings (handling midnight wrap).
func computeMinutesBetween(startHHMM, endHHMM string) int {
	parseMin := func(s string) (int, bool) {
		s = strings.TrimSpace(s)
		parts := strings.Split(s, ":")
		if len(parts) < 2 {
			return 0, false
		}
		h, err1 := strconv.Atoi(parts[0])
		m, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
			return 0, false
		}
		return h*60 + m, true
	}
	s, ok1 := parseMin(startHHMM)
	e, ok2 := parseMin(endHHMM)
	if !ok1 || !ok2 {
		return 0
	}
	diff := e - s
	if diff < 0 {
		diff += 24 * 60
	}
	return diff
}

func normalizeMSSStage(stage string) string {
	s := strings.ToLower(strings.TrimSpace(stage))
	if s == "tahap_2" || s == "2" || s == "stage_2" {
		return "tahap_2"
	}
	return "tahap_1"
}

func monthFromDate(month, date string) string {
	if strings.TrimSpace(month) != "" {
		return strings.TrimSpace(month)
	}
	if len(date) >= 7 {
		return date[:7]
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// MSS: Sterilization Temperature Monitoring (Form QC-057-MSS)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateSterilizationTempLog(ctx context.Context, dto CreateSterilizationTempLogDTO) (*domain.SterilizationTempLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal monitoring sterilisasi wajib diisi")
	}
	if strings.TrimSpace(dto.PackagingType) == "" {
		return nil, apperrors.NewBadRequest("Jenis kemasan wajib diisi")
	}
	if strings.TrimSpace(dto.InitialTime) == "" || strings.TrimSpace(dto.HeatingTime) == "" || strings.TrimSpace(dto.FinalTime) == "" {
		return nil, apperrors.NewBadRequest("Waktu awal, heating, dan akhir wajib diisi")
	}

	stage := normalizeMSSStage(dto.Stage)
	month := monthFromDate(dto.CheckMonth, dto.CheckDate)
	duration := computeMinutesBetween(dto.HeatingTime, dto.FinalTime)
	standardMet := dto.HeatingTemp >= 70.0 && duration >= 45
	status := "passed"
	if !standardMet || strings.TrimSpace(dto.Issues) != "" {
		status = "action_required"
	}

	docNo, err := docseq.Next(ctx, uc.repo, "MSS", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen MSS")
	}

	item := &domain.SterilizationTempLog{
		DocumentNo:        docNo,
		DocumentCode:      "QC-057-MSS",
		Stage:             stage,
		CheckMonth:        month,
		CheckDate:         dto.CheckDate,
		BatchNo:           strings.TrimSpace(dto.BatchNo),
		ProductionOrderID: dto.ProductionOrderID,
		PackagingType:     strings.TrimSpace(dto.PackagingType),
		InitialTime:       strings.TrimSpace(dto.InitialTime),
		InitialTemp:       dto.InitialTemp,
		HeatingTime:       strings.TrimSpace(dto.HeatingTime),
		HeatingTemp:       dto.HeatingTemp,
		FinalTime:         strings.TrimSpace(dto.FinalTime),
		FinalTemp:         dto.FinalTemp,
		DurationMinutes:   duration,
		IsStandardMet:     standardMet,
		Issues:            strings.TrimSpace(dto.Issues),
		ActionTaken:       strings.TrimSpace(dto.ActionTaken),
		QCName:            strings.TrimSpace(dto.QCName),
		QAVerifiedBy:      strings.TrimSpace(dto.QAVerifiedBy),
		OverallStatus:     status,
		Notes:             strings.TrimSpace(dto.Notes),
	}

	if err := uc.repo.CreateSterilizationTempLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan log monitoring suhu sterilisasi")
	}
	return item, nil
}

func (uc *qcUseCase) GetSterilizationTempLogByID(ctx context.Context, id uuid.UUID) (*domain.SterilizationTempLog, error) {
	item, err := uc.repo.GetSterilizationTempLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring suhu sterilisasi tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListSterilizationTempLogs(ctx context.Context, month string, stage string, batchNo string) ([]domain.SterilizationTempLog, error) {
	normStage := ""
	if strings.TrimSpace(stage) != "" {
		normStage = normalizeMSSStage(stage)
	}
	list, err := uc.repo.ListSterilizationTempLogs(ctx, month, normStage, batchNo)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar monitoring suhu sterilisasi")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateSterilizationTempLog(ctx context.Context, id uuid.UUID, dto UpdateSterilizationTempLogDTO) (*domain.SterilizationTempLog, error) {
	item, err := uc.repo.GetSterilizationTempLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring suhu sterilisasi tidak ditemukan")
	}

	if dto.Stage != "" {
		item.Stage = normalizeMSSStage(dto.Stage)
	}
	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
		if dto.CheckMonth == "" && len(dto.CheckDate) >= 7 {
			item.CheckMonth = dto.CheckDate[:7]
		}
	}
	if dto.BatchNo != "" {
		item.BatchNo = strings.TrimSpace(dto.BatchNo)
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.PackagingType != nil {
		item.PackagingType = strings.TrimSpace(*dto.PackagingType)
	}
	if dto.InitialTime != nil {
		item.InitialTime = strings.TrimSpace(*dto.InitialTime)
	}
	if dto.InitialTemp != nil {
		item.InitialTemp = *dto.InitialTemp
	}
	if dto.HeatingTime != nil {
		item.HeatingTime = strings.TrimSpace(*dto.HeatingTime)
	}
	if dto.HeatingTemp != nil {
		item.HeatingTemp = *dto.HeatingTemp
	}
	if dto.FinalTime != nil {
		item.FinalTime = strings.TrimSpace(*dto.FinalTime)
	}
	if dto.FinalTemp != nil {
		item.FinalTemp = *dto.FinalTemp
	}
	if dto.Issues != nil {
		item.Issues = strings.TrimSpace(*dto.Issues)
	}
	if dto.ActionTaken != nil {
		item.ActionTaken = strings.TrimSpace(*dto.ActionTaken)
	}
	if dto.QCName != nil {
		item.QCName = strings.TrimSpace(*dto.QCName)
	}
	if dto.QAVerifiedBy != nil {
		item.QAVerifiedBy = strings.TrimSpace(*dto.QAVerifiedBy)
	}
	if dto.Notes != nil {
		item.Notes = strings.TrimSpace(*dto.Notes)
	}

	item.DurationMinutes = computeMinutesBetween(item.HeatingTime, item.FinalTime)
	item.IsStandardMet = item.HeatingTemp >= 70.0 && item.DurationMinutes >= 45
	if dto.OverallStatus != nil && *dto.OverallStatus != "" {
		item.OverallStatus = *dto.OverallStatus
	} else if item.IsStandardMet && item.Issues == "" {
		item.OverallStatus = "passed"
	} else {
		item.OverallStatus = "action_required"
	}

	if err := uc.repo.UpdateSterilizationTempLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui log monitoring suhu sterilisasi")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteSterilizationTempLog(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteSterilizationTempLog(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// PVK: Liquid Packaging Visual Check (Form QC-041-PVK Rev. 01)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateLiquidPackagingVisualCheck(ctx context.Context, dto CreateLiquidPackagingVisualCheckDTO) (*domain.LiquidPackagingVisualCheck, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pengecekan visual kemasan wajib diisi")
	}
	if strings.TrimSpace(dto.PackagingType) == "" {
		return nil, apperrors.NewBadRequest("Jenis kemasan wajib diisi")
	}
	if strings.TrimSpace(dto.PICName) == "" {
		return nil, apperrors.NewBadRequest("Nama/paraf PIC wajib diisi")
	}

	month := monthFromDate(dto.CheckMonth, dto.CheckDate)
	cond := strings.ToUpper(strings.TrimSpace(dto.PackagingCondition))
	if cond != "K" {
		cond = "B"
	}
	picResp := strings.TrimSpace(dto.PICResponsible)
	if picResp == "" {
		picResp = "Sajiman"
	}
	status := "passed"
	if cond != "B" || strings.TrimSpace(dto.Findings) != "" {
		status = "action_required"
	}

	docNo, err := docseq.Next(ctx, uc.repo, "PVK", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen PVK")
	}

	item := &domain.LiquidPackagingVisualCheck{
		DocumentNo:         docNo,
		DocumentCode:       "QC-041-PVK",
		RevisionCode:       "01",
		CheckMonth:         month,
		CheckDate:          dto.CheckDate,
		BatchNo:            strings.TrimSpace(dto.BatchNo),
		ProductionOrderID:  dto.ProductionOrderID,
		PICResponsible:     picResp,
		PackagingType:      strings.TrimSpace(dto.PackagingType),
		Quantity:           dto.Quantity,
		BuyerName:          strings.TrimSpace(dto.BuyerName),
		SterilizationInfo:  strings.TrimSpace(dto.SterilizationInfo),
		PackagingCondition: cond,
		Findings:           strings.TrimSpace(dto.Findings),
		ActionTaken:        strings.TrimSpace(dto.ActionTaken),
		PICName:            strings.TrimSpace(dto.PICName),
		QCName:             strings.TrimSpace(dto.QCName),
		QCKnownBy:          strings.TrimSpace(dto.QCKnownBy),
		QCKnownDate:        strings.TrimSpace(dto.QCKnownDate),
		QAVerifiedBy:       strings.TrimSpace(dto.QAVerifiedBy),
		QAVerifiedDate:     strings.TrimSpace(dto.QAVerifiedDate),
		OverallStatus:      status,
		Notes:              strings.TrimSpace(dto.Notes),
	}

	if err := uc.repo.CreateLiquidPackagingVisualCheck(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan pengecekan visual kemasan gula cair")
	}
	return item, nil
}

func (uc *qcUseCase) GetLiquidPackagingVisualCheckByID(ctx context.Context, id uuid.UUID) (*domain.LiquidPackagingVisualCheck, error) {
	item, err := uc.repo.GetLiquidPackagingVisualCheckByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data pengecekan visual kemasan tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListLiquidPackagingVisualChecks(ctx context.Context, month string, batchNo string) ([]domain.LiquidPackagingVisualCheck, error) {
	list, err := uc.repo.ListLiquidPackagingVisualChecks(ctx, month, batchNo)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar pengecekan visual kemasan")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateLiquidPackagingVisualCheck(ctx context.Context, id uuid.UUID, dto UpdateLiquidPackagingVisualCheckDTO) (*domain.LiquidPackagingVisualCheck, error) {
	item, err := uc.repo.GetLiquidPackagingVisualCheckByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data pengecekan visual kemasan tidak ditemukan")
	}

	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
		if dto.CheckMonth == "" && len(dto.CheckDate) >= 7 {
			item.CheckMonth = dto.CheckDate[:7]
		}
	}
	if dto.BatchNo != "" {
		item.BatchNo = strings.TrimSpace(dto.BatchNo)
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.PICResponsible != nil {
		item.PICResponsible = strings.TrimSpace(*dto.PICResponsible)
	}
	if dto.PackagingType != nil {
		item.PackagingType = strings.TrimSpace(*dto.PackagingType)
	}
	if dto.Quantity != nil {
		item.Quantity = *dto.Quantity
	}
	if dto.BuyerName != nil {
		item.BuyerName = strings.TrimSpace(*dto.BuyerName)
	}
	if dto.SterilizationInfo != nil {
		item.SterilizationInfo = strings.TrimSpace(*dto.SterilizationInfo)
	}
	if dto.PackagingCondition != nil {
		c := strings.ToUpper(strings.TrimSpace(*dto.PackagingCondition))
		if c == "K" {
			item.PackagingCondition = "K"
		} else {
			item.PackagingCondition = "B"
		}
	}
	if dto.Findings != nil {
		item.Findings = strings.TrimSpace(*dto.Findings)
	}
	if dto.ActionTaken != nil {
		item.ActionTaken = strings.TrimSpace(*dto.ActionTaken)
	}
	if dto.PICName != nil {
		item.PICName = strings.TrimSpace(*dto.PICName)
	}
	if dto.QCName != nil {
		item.QCName = strings.TrimSpace(*dto.QCName)
	}
	if dto.QCKnownBy != nil {
		item.QCKnownBy = strings.TrimSpace(*dto.QCKnownBy)
	}
	if dto.QCKnownDate != nil {
		item.QCKnownDate = strings.TrimSpace(*dto.QCKnownDate)
	}
	if dto.QAVerifiedBy != nil {
		item.QAVerifiedBy = strings.TrimSpace(*dto.QAVerifiedBy)
	}
	if dto.QAVerifiedDate != nil {
		item.QAVerifiedDate = strings.TrimSpace(*dto.QAVerifiedDate)
	}
	if dto.Notes != nil {
		item.Notes = strings.TrimSpace(*dto.Notes)
	}

	if dto.OverallStatus != nil && *dto.OverallStatus != "" {
		item.OverallStatus = *dto.OverallStatus
	} else if item.PackagingCondition == "B" && item.Findings == "" {
		item.OverallStatus = "passed"
	} else {
		item.OverallStatus = "action_required"
	}

	if err := uc.repo.UpdateLiquidPackagingVisualCheck(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui pengecekan visual kemasan")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteLiquidPackagingVisualCheck(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteLiquidPackagingVisualCheck(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// MNC: Liquid Sugar Processing Log (Form QC-014-MNC)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateLiquidSugarProcessingLog(ctx context.Context, dto CreateLiquidSugarProcessingLogDTO) (*domain.LiquidSugarProcessingLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pengolahan gula cair wajib diisi")
	}
	if strings.TrimSpace(dto.ProductCode) == "" {
		return nil, apperrors.NewBadRequest("Kode produk wajib diisi")
	}
	if strings.TrimSpace(dto.InitialTime) == "" || strings.TrimSpace(dto.HeatingTime) == "" || strings.TrimSpace(dto.FinalTime) == "" {
		return nil, apperrors.NewBadRequest("Waktu awal, heating, dan akhir wajib diisi")
	}
	if strings.TrimSpace(dto.PICName) == "" {
		return nil, apperrors.NewBadRequest("Nama PIC wajib diisi")
	}

	month := monthFromDate(dto.CheckMonth, dto.CheckDate)
	durMin := computeMinutesBetween(dto.HeatingTime, dto.FinalTime)
	ccpPassed := dto.HeatingTemp >= 90.0 && dto.HeatingTemp <= 95.0 && durMin >= 10
	status := "passed"
	if !ccpPassed {
		status = "ccp_alert"
	} else if strings.TrimSpace(dto.Issues) != "" {
		status = "action_required"
	}

	docNo, err := docseq.Next(ctx, uc.repo, "MNC", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen MNC")
	}

	item := &domain.LiquidSugarProcessingLog{
		DocumentNo:         docNo,
		DocumentCode:       "QC-014-MNC",
		RevisionCode:       "01",
		CheckMonth:         month,
		CheckDate:          dto.CheckDate,
		BatchNo:            strings.TrimSpace(dto.BatchNo),
		ProductionOrderID:  dto.ProductionOrderID,
		ProductCode:        strings.TrimSpace(dto.ProductCode),
		ProductName:        strings.TrimSpace(dto.ProductName),
		InitialTime:        strings.TrimSpace(dto.InitialTime),
		InitialTemp:        dto.InitialTemp,
		InitialMoisture:    dto.InitialMoisture,
		InitialBrix:        dto.InitialBrix,
		InitialPH:          dto.InitialPH,
		HeatingTime:        strings.TrimSpace(dto.HeatingTime),
		HeatingTemp:        dto.HeatingTemp,
		HeatingDurationMin: durMin,
		IsCcpPassed:        ccpPassed,
		FinalTime:          strings.TrimSpace(dto.FinalTime),
		FinalProductTemp:   dto.FinalProductTemp,
		FinalMoisture:      dto.FinalMoisture,
		FinalBrix:          dto.FinalBrix,
		FinalPH:            dto.FinalPH,
		Issues:             strings.TrimSpace(dto.Issues),
		ActionTaken:        strings.TrimSpace(dto.ActionTaken),
		PICName:            strings.TrimSpace(dto.PICName),
		QCName:             strings.TrimSpace(dto.QCName),
		QAVerifiedBy:       strings.TrimSpace(dto.QAVerifiedBy),
		OverallStatus:      status,
		Notes:              strings.TrimSpace(dto.Notes),
	}

	if err := uc.repo.CreateLiquidSugarProcessingLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan monitoring pengolahan gula cair")
	}
	return item, nil
}

func (uc *qcUseCase) GetLiquidSugarProcessingLogByID(ctx context.Context, id uuid.UUID) (*domain.LiquidSugarProcessingLog, error) {
	item, err := uc.repo.GetLiquidSugarProcessingLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring pengolahan gula cair tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListLiquidSugarProcessingLogs(ctx context.Context, month string, batchNo string, productCode string) ([]domain.LiquidSugarProcessingLog, error) {
	list, err := uc.repo.ListLiquidSugarProcessingLogs(ctx, month, batchNo, productCode)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar monitoring pengolahan gula cair")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateLiquidSugarProcessingLog(ctx context.Context, id uuid.UUID, dto UpdateLiquidSugarProcessingLogDTO) (*domain.LiquidSugarProcessingLog, error) {
	item, err := uc.repo.GetLiquidSugarProcessingLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data monitoring pengolahan gula cair tidak ditemukan")
	}

	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
		if dto.CheckMonth == "" && len(dto.CheckDate) >= 7 {
			item.CheckMonth = dto.CheckDate[:7]
		}
	}
	if dto.BatchNo != "" {
		item.BatchNo = strings.TrimSpace(dto.BatchNo)
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.ProductCode != nil {
		item.ProductCode = strings.TrimSpace(*dto.ProductCode)
	}
	if dto.ProductName != nil {
		item.ProductName = strings.TrimSpace(*dto.ProductName)
	}
	if dto.InitialTime != nil {
		item.InitialTime = strings.TrimSpace(*dto.InitialTime)
	}
	if dto.InitialTemp != nil {
		item.InitialTemp = *dto.InitialTemp
	}
	if dto.InitialMoisture != nil {
		item.InitialMoisture = *dto.InitialMoisture
	}
	if dto.InitialBrix != nil {
		item.InitialBrix = *dto.InitialBrix
	}
	if dto.InitialPH != nil {
		item.InitialPH = *dto.InitialPH
	}
	if dto.HeatingTime != nil {
		item.HeatingTime = strings.TrimSpace(*dto.HeatingTime)
	}
	if dto.HeatingTemp != nil {
		item.HeatingTemp = *dto.HeatingTemp
	}
	if dto.FinalTime != nil {
		item.FinalTime = strings.TrimSpace(*dto.FinalTime)
	}
	if dto.FinalProductTemp != nil {
		item.FinalProductTemp = *dto.FinalProductTemp
	}
	if dto.FinalMoisture != nil {
		item.FinalMoisture = *dto.FinalMoisture
	}
	if dto.FinalBrix != nil {
		item.FinalBrix = *dto.FinalBrix
	}
	if dto.FinalPH != nil {
		item.FinalPH = *dto.FinalPH
	}
	if dto.Issues != nil {
		item.Issues = strings.TrimSpace(*dto.Issues)
	}
	if dto.ActionTaken != nil {
		item.ActionTaken = strings.TrimSpace(*dto.ActionTaken)
	}
	if dto.PICName != nil {
		item.PICName = strings.TrimSpace(*dto.PICName)
	}
	if dto.QCName != nil {
		item.QCName = strings.TrimSpace(*dto.QCName)
	}
	if dto.QAVerifiedBy != nil {
		item.QAVerifiedBy = strings.TrimSpace(*dto.QAVerifiedBy)
	}
	if dto.Notes != nil {
		item.Notes = strings.TrimSpace(*dto.Notes)
	}

	item.HeatingDurationMin = computeMinutesBetween(item.HeatingTime, item.FinalTime)
	item.IsCcpPassed = item.HeatingTemp >= 90.0 && item.HeatingTemp <= 95.0 && item.HeatingDurationMin >= 10
	if dto.OverallStatus != nil && *dto.OverallStatus != "" {
		item.OverallStatus = *dto.OverallStatus
	} else if !item.IsCcpPassed {
		item.OverallStatus = "ccp_alert"
	} else if item.Issues != "" {
		item.OverallStatus = "action_required"
	} else {
		item.OverallStatus = "passed"
	}

	if err := uc.repo.UpdateLiquidSugarProcessingLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui monitoring pengolahan gula cair")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteLiquidSugarProcessingLog(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteLiquidSugarProcessingLog(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// PJC: Liquid Sugar Finished Goods Inspection (Form QC-015-PJC Rev. 02)
// ─────────────────────────────────────────────────────────────────────────────

func pjcColorDesc(score int, explicit string) (int, string) {
	if score < 1 || score > 3 {
		score = 2
	}
	if strings.TrimSpace(explicit) != "" {
		return score, strings.TrimSpace(explicit)
	}
	switch score {
	case 1:
		return 1, "Light Brown"
	case 3:
		return 3, "Dark Brown"
	default:
		return 2, "Normal"
	}
}

func (uc *qcUseCase) CreateLiquidSugarFinishedGoodsLog(ctx context.Context, dto CreateLiquidSugarFinishedGoodsLogDTO) (*domain.LiquidSugarFinishedGoodsLog, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pemeriksaan produk jadi gula cair wajib diisi")
	}
	if strings.TrimSpace(dto.ProductName) == "" {
		return nil, apperrors.NewBadRequest("Nama produk wajib diisi")
	}
	if strings.TrimSpace(dto.BatchNo) == "" {
		return nil, apperrors.NewBadRequest("Batch number wajib diisi")
	}
	if strings.TrimSpace(dto.QCPersonnelName) == "" {
		return nil, apperrors.NewBadRequest("Nama/paraf QC wajib diisi")
	}

	month := monthFromDate(dto.CheckMonth, dto.CheckDate)
	colorScore, colorDesc := pjcColorDesc(dto.ColorScore, dto.ColorDescription)
	freeForeign := boolOrDefault(dto.CheckFreeForeignMatter, true)

	decision := strings.ToUpper(strings.TrimSpace(dto.InspectionDecision))
	if decision != "DITOLAK" && decision != "LOLOS" {
		if !freeForeign {
			decision = "DITOLAK"
		} else {
			decision = "LOLOS"
		}
	}

	docNo, err := docseq.Next(ctx, uc.repo, "PJC", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen PJC")
	}

	item := &domain.LiquidSugarFinishedGoodsLog{
		DocumentNo:             docNo,
		DocumentCode:           "QC-015-PJC",
		RevisionCode:           "02",
		CheckMonth:             month,
		CheckDate:              dto.CheckDate,
		ProductionOrderID:      dto.ProductionOrderID,
		ProductName:            strings.TrimSpace(dto.ProductName),
		BatchNo:                strings.TrimSpace(dto.BatchNo),
		RawMaterialCode:        strings.TrimSpace(dto.RawMaterialCode),
		CheckPackaging:         boolOrDefault(dto.CheckPackaging, true),
		CheckWeight:            boolOrDefault(dto.CheckWeight, true),
		WeightValue:            dto.WeightValue,
		CheckShape:             boolOrDefault(dto.CheckShape, true),
		CheckAroma:             boolOrDefault(dto.CheckAroma, true),
		ColorScore:             colorScore,
		ColorDescription:       colorDesc,
		CheckTaste:             boolOrDefault(dto.CheckTaste, true),
		CheckFreeForeignMatter: freeForeign,
		CheckSolubility:        boolOrDefault(dto.CheckSolubility, true),
		MoistureContent:        dto.MoistureContent,
		PHValue:                dto.PHValue,
		BrixValue:              dto.BrixValue,
		InspectionDecision:     decision,
		Notes:                  strings.TrimSpace(dto.Notes),
		QCPersonnelName:        strings.TrimSpace(dto.QCPersonnelName),
		QAVerifiedBy:           strings.TrimSpace(dto.QAVerifiedBy),
		QAVerifiedDate:         strings.TrimSpace(dto.QAVerifiedDate),
	}

	if err := uc.repo.CreateLiquidSugarFinishedGoodsLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan pemeriksaan produk jadi gula cair")
	}

	if item.InspectionDecision == "DITOLAK" {
		reason := fmt.Sprintf("Produk Jadi Gula Cair DITOLAK (Brix %.2f, Kadar Air %.2f%%, pH %.2f)", item.BrixValue, item.MoistureContent, item.PHValue)
		if !item.CheckFreeForeignMatter {
			reason = "Produk Jadi Gula Cair DITOLAK: Terdeteksi benda asing; " + reason
		}
		if item.Notes != "" {
			reason += " — " + item.Notes
		}
		_, _ = uc.AutoCreateBrontolOnQCFailure(ctx, AutoBrontolTrigger{
			SourceStage:        domain.StageLiquidSugar,
			SourceInspectionID: &item.ID,
			SourceInspectionNo: item.DocumentNo,
			ProductionOrderID:  item.ProductionOrderID,
			ProductName:        item.ProductName,
			BatchNo:            item.BatchNo,
			MaterialType:       "FG",
			DefectReason:       reason,
			ActionDisposition:  domain.ActionHold,
			QCInspectorName:    item.QCPersonnelName,
			Notes:              item.Notes,
		})
	}

	return item, nil
}

func (uc *qcUseCase) GetLiquidSugarFinishedGoodsLogByID(ctx context.Context, id uuid.UUID) (*domain.LiquidSugarFinishedGoodsLog, error) {
	item, err := uc.repo.GetLiquidSugarFinishedGoodsLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data pemeriksaan produk jadi gula cair tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListLiquidSugarFinishedGoodsLogs(ctx context.Context, month string, batchNo string, decision string) ([]domain.LiquidSugarFinishedGoodsLog, error) {
	list, err := uc.repo.ListLiquidSugarFinishedGoodsLogs(ctx, month, batchNo, decision)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar pemeriksaan produk jadi gula cair")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateLiquidSugarFinishedGoodsLog(ctx context.Context, id uuid.UUID, dto UpdateLiquidSugarFinishedGoodsLogDTO) (*domain.LiquidSugarFinishedGoodsLog, error) {
	item, err := uc.repo.GetLiquidSugarFinishedGoodsLogByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data pemeriksaan produk jadi gula cair tidak ditemukan")
	}

	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
		if dto.CheckMonth == "" && len(dto.CheckDate) >= 7 {
			item.CheckMonth = dto.CheckDate[:7]
		}
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.ProductName != nil {
		item.ProductName = strings.TrimSpace(*dto.ProductName)
	}
	if dto.BatchNo != nil {
		item.BatchNo = strings.TrimSpace(*dto.BatchNo)
	}
	if dto.RawMaterialCode != nil {
		item.RawMaterialCode = strings.TrimSpace(*dto.RawMaterialCode)
	}
	if dto.CheckPackaging != nil {
		item.CheckPackaging = *dto.CheckPackaging
	}
	if dto.CheckWeight != nil {
		item.CheckWeight = *dto.CheckWeight
	}
	if dto.WeightValue != nil {
		item.WeightValue = dto.WeightValue
	}
	if dto.CheckShape != nil {
		item.CheckShape = *dto.CheckShape
	}
	if dto.CheckAroma != nil {
		item.CheckAroma = *dto.CheckAroma
	}
	if dto.ColorScore != nil || dto.ColorDescription != nil {
		cs := item.ColorScore
		if dto.ColorScore != nil {
			cs = *dto.ColorScore
		}
		cd := ""
		if dto.ColorDescription != nil {
			cd = *dto.ColorDescription
		}
		item.ColorScore, item.ColorDescription = pjcColorDesc(cs, cd)
	}
	if dto.CheckTaste != nil {
		item.CheckTaste = *dto.CheckTaste
	}
	if dto.CheckFreeForeignMatter != nil {
		item.CheckFreeForeignMatter = *dto.CheckFreeForeignMatter
	}
	if dto.CheckSolubility != nil {
		item.CheckSolubility = *dto.CheckSolubility
	}
	if dto.MoistureContent != nil {
		item.MoistureContent = *dto.MoistureContent
	}
	if dto.PHValue != nil {
		item.PHValue = *dto.PHValue
	}
	if dto.BrixValue != nil {
		item.BrixValue = *dto.BrixValue
	}
	if dto.InspectionDecision != nil {
		d := strings.ToUpper(strings.TrimSpace(*dto.InspectionDecision))
		if d == "DITOLAK" || d == "LOLOS" {
			item.InspectionDecision = d
		}
	}
	if dto.Notes != nil {
		item.Notes = strings.TrimSpace(*dto.Notes)
	}
	if dto.QCPersonnelName != nil {
		item.QCPersonnelName = strings.TrimSpace(*dto.QCPersonnelName)
	}
	if dto.QAVerifiedBy != nil {
		item.QAVerifiedBy = strings.TrimSpace(*dto.QAVerifiedBy)
	}
	if dto.QAVerifiedDate != nil {
		item.QAVerifiedDate = strings.TrimSpace(*dto.QAVerifiedDate)
	}

	if err := uc.repo.UpdateLiquidSugarFinishedGoodsLog(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui pemeriksaan produk jadi gula cair")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteLiquidSugarFinishedGoodsLog(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteLiquidSugarFinishedGoodsLog(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// VHC: Liquid Sugar Wash Verification (Form QC-055-VHC Rev. 00)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) CreateLiquidSugarWashVerification(ctx context.Context, dto CreateLiquidSugarWashVerificationDTO) (*domain.LiquidSugarWashVerification, error) {
	if strings.TrimSpace(dto.CheckDate) == "" {
		return nil, apperrors.NewBadRequest("Tanggal pencucian wajib diisi")
	}
	if strings.TrimSpace(dto.CirculationStartTime) == "" || strings.TrimSpace(dto.CirculationEndTime) == "" {
		return nil, apperrors.NewBadRequest("Waktu mulai dan selesai sirkulasi air panas wajib diisi")
	}
	if strings.TrimSpace(dto.QCVerifierName) == "" {
		return nil, apperrors.NewBadRequest("Nama verifikator QC wajib diisi")
	}

	month := monthFromDate(dto.CheckMonth, dto.CheckDate)
	loc := strings.TrimSpace(dto.Location)
	if loc == "" {
		loc = "Gula Cair"
	}
	picResp := strings.TrimSpace(dto.PICResponsible)
	if picResp == "" {
		picResp = "QC (Andin)"
	}

	rinseColor := strings.TrimSpace(dto.RinseColor)
	if rinseColor == "" {
		rinseColor = "Bening"
	}
	rinseAroma := strings.TrimSpace(dto.RinseAroma)
	if rinseAroma == "" {
		rinseAroma = "Normal"
	}
	rinseDirt := strings.TrimSpace(dto.RinseDirt)
	if rinseDirt == "" {
		rinseDirt = "Bebas kotoran"
	}

	tempOk := dto.WaterTempActual >= 75.0
	durMin := computeMinutesBetween(dto.CirculationStartTime, dto.CirculationEndTime)
	circOk := durMin >= 10
	rinseOk := strings.EqualFold(rinseColor, "Bening") &&
		strings.EqualFold(rinseAroma, "Normal") &&
		strings.EqualFold(rinseDirt, "Bebas kotoran")

	status := "passed"
	if !tempOk || !circOk || !rinseOk {
		status = "re_wash"
	}

	docNo, err := docseq.Next(ctx, uc.repo, "VHC", dto.CheckDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal membuat nomor dokumen VHC")
	}

	item := &domain.LiquidSugarWashVerification{
		DocumentNo:               docNo,
		DocumentCode:             "QC-055-VHC",
		RevisionCode:             "00",
		CheckMonth:               month,
		CheckDate:                dto.CheckDate,
		BatchNo:                  strings.TrimSpace(dto.BatchNo),
		ProductionOrderID:        dto.ProductionOrderID,
		Location:                 loc,
		PICResponsible:           picResp,
		WaterTempActual:          dto.WaterTempActual,
		IsTempStandardMet:        tempOk,
		CirculationStartTime:     strings.TrimSpace(dto.CirculationStartTime),
		CirculationEndTime:       strings.TrimSpace(dto.CirculationEndTime),
		CirculationDurationMin:   durMin,
		IsCirculationStandardMet: circOk,
		RinseColor:               rinseColor,
		RinseAroma:               rinseAroma,
		RinseDirt:                rinseDirt,
		IsRinseStandardMet:       rinseOk,
		QCVerifierName:           strings.TrimSpace(dto.QCVerifierName),
		QAVerifiedBy:             strings.TrimSpace(dto.QAVerifiedBy),
		QAVerifiedDate:           strings.TrimSpace(dto.QAVerifiedDate),
		OverallStatus:            status,
		Notes:                    strings.TrimSpace(dto.Notes),
	}

	if err := uc.repo.CreateLiquidSugarWashVerification(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal menyimpan verifikasi hasil cuci gula cair")
	}
	return item, nil
}

func (uc *qcUseCase) GetLiquidSugarWashVerificationByID(ctx context.Context, id uuid.UUID) (*domain.LiquidSugarWashVerification, error) {
	item, err := uc.repo.GetLiquidSugarWashVerificationByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data verifikasi hasil cuci tidak ditemukan")
	}
	return item, nil
}

func (uc *qcUseCase) ListLiquidSugarWashVerifications(ctx context.Context, month string, batchNo string) ([]domain.LiquidSugarWashVerification, error) {
	list, err := uc.repo.ListLiquidSugarWashVerifications(ctx, month, batchNo)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memuat daftar verifikasi hasil cuci")
	}
	return list, nil
}

func (uc *qcUseCase) UpdateLiquidSugarWashVerification(ctx context.Context, id uuid.UUID, dto UpdateLiquidSugarWashVerificationDTO) (*domain.LiquidSugarWashVerification, error) {
	item, err := uc.repo.GetLiquidSugarWashVerificationByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewNotFound("Data verifikasi hasil cuci tidak ditemukan")
	}

	if dto.CheckMonth != "" {
		item.CheckMonth = dto.CheckMonth
	}
	if dto.CheckDate != "" {
		item.CheckDate = dto.CheckDate
		if dto.CheckMonth == "" && len(dto.CheckDate) >= 7 {
			item.CheckMonth = dto.CheckDate[:7]
		}
	}
	if dto.BatchNo != "" {
		item.BatchNo = strings.TrimSpace(dto.BatchNo)
	}
	if dto.ProductionOrderID != nil {
		item.ProductionOrderID = dto.ProductionOrderID
	}
	if dto.Location != nil {
		item.Location = strings.TrimSpace(*dto.Location)
	}
	if dto.PICResponsible != nil {
		item.PICResponsible = strings.TrimSpace(*dto.PICResponsible)
	}
	if dto.WaterTempActual != nil {
		item.WaterTempActual = *dto.WaterTempActual
	}
	if dto.CirculationStartTime != nil {
		item.CirculationStartTime = strings.TrimSpace(*dto.CirculationStartTime)
	}
	if dto.CirculationEndTime != nil {
		item.CirculationEndTime = strings.TrimSpace(*dto.CirculationEndTime)
	}
	if dto.RinseColor != nil {
		item.RinseColor = strings.TrimSpace(*dto.RinseColor)
	}
	if dto.RinseAroma != nil {
		item.RinseAroma = strings.TrimSpace(*dto.RinseAroma)
	}
	if dto.RinseDirt != nil {
		item.RinseDirt = strings.TrimSpace(*dto.RinseDirt)
	}
	if dto.QCVerifierName != nil {
		item.QCVerifierName = strings.TrimSpace(*dto.QCVerifierName)
	}
	if dto.QAVerifiedBy != nil {
		item.QAVerifiedBy = strings.TrimSpace(*dto.QAVerifiedBy)
	}
	if dto.QAVerifiedDate != nil {
		item.QAVerifiedDate = strings.TrimSpace(*dto.QAVerifiedDate)
	}
	if dto.Notes != nil {
		item.Notes = strings.TrimSpace(*dto.Notes)
	}

	item.IsTempStandardMet = item.WaterTempActual >= 75.0
	item.CirculationDurationMin = computeMinutesBetween(item.CirculationStartTime, item.CirculationEndTime)
	item.IsCirculationStandardMet = item.CirculationDurationMin >= 10
	item.IsRinseStandardMet = strings.EqualFold(item.RinseColor, "Bening") &&
		strings.EqualFold(item.RinseAroma, "Normal") &&
		strings.EqualFold(item.RinseDirt, "Bebas kotoran")

	if dto.OverallStatus != nil && *dto.OverallStatus != "" {
		item.OverallStatus = *dto.OverallStatus
	} else if item.IsTempStandardMet && item.IsCirculationStandardMet && item.IsRinseStandardMet {
		item.OverallStatus = "passed"
	} else {
		item.OverallStatus = "re_wash"
	}

	if err := uc.repo.UpdateLiquidSugarWashVerification(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Gagal memperbarui verifikasi hasil cuci")
	}
	return item, nil
}

func (uc *qcUseCase) DeleteLiquidSugarWashVerification(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteLiquidSugarWashVerification(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Flow Gula Cair Aggregated Status
// ─────────────────────────────────────────────────────────────────────────────

func (uc *qcUseCase) GetLiquidSugarFlowStatus(ctx context.Context, month string, batchNo string) (*LiquidSugarFlowStatusDTO, error) {
	batchTrim := strings.TrimSpace(batchNo)
	matchBatch := func(b string) bool {
		if batchTrim == "" {
			return true
		}
		return strings.EqualFold(strings.TrimSpace(b), batchTrim)
	}

	batchSet := make(map[string]struct{})
	addBatch := func(b string) {
		b = strings.TrimSpace(b)
		if b != "" {
			batchSet[b] = struct{}{}
		}
	}

	mssAll, err := uc.repo.ListSterilizationTempLogs(ctx, month, "", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status MSS")
	}
	mss1Count, mss2Count := 0, 0
	for _, m := range mssAll {
		addBatch(m.BatchNo)
		if !matchBatch(m.BatchNo) {
			continue
		}
		if normalizeMSSStage(m.Stage) == "tahap_2" {
			mss2Count++
		} else {
			mss1Count++
		}
	}

	pvkAll, err := uc.repo.ListLiquidPackagingVisualChecks(ctx, month, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status PVK")
	}
	pvkCount := 0
	for _, p := range pvkAll {
		addBatch(p.BatchNo)
		if matchBatch(p.BatchNo) {
			pvkCount++
		}
	}

	vmdAll, err := uc.repo.ListMetalVerificationLogs(ctx, month, "gula_cair", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status VMD")
	}
	vmdCount := 0
	for _, v := range vmdAll {
		addBatch(v.BatchNo)
		if matchBatch(v.BatchNo) {
			vmdCount++
		}
	}

	cppAll, err := uc.repo.ListMetalCleaningLogs(ctx, month, "", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status CPP")
	}
	cppCount := 0
	for _, c := range cppAll {
		locLower := strings.ToLower(c.Location)
		if !strings.Contains(locLower, "gula cair") {
			continue
		}
		addBatch(c.BatchNo)
		if matchBatch(c.BatchNo) {
			cppCount++
		}
	}

	mnpAll, err := uc.repo.ListFilterMeshLogs(ctx, month, "gula_cair", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status MNP")
	}
	mnpCount := 0
	for _, f := range mnpAll {
		addBatch(f.BatchNo)
		if matchBatch(f.BatchNo) {
			mnpCount++
		}
	}

	mnbAll, err := uc.repo.ListMeshBoltLogs(ctx, month, "", "", "MNB", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status MNB")
	}
	mnbCount := 0
	for _, b := range mnbAll {
		addBatch(b.BatchNo)
		if matchBatch(b.BatchNo) {
			mnbCount++
		}
	}

	mncAll, err := uc.repo.ListLiquidSugarProcessingLogs(ctx, month, "", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status MNC")
	}
	mncCount := 0
	for _, m := range mncAll {
		addBatch(m.BatchNo)
		if matchBatch(m.BatchNo) {
			mncCount++
		}
	}

	pjcAll, err := uc.repo.ListLiquidSugarFinishedGoodsLogs(ctx, month, "", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status PJC")
	}
	pjcCount := 0
	for _, p := range pjcAll {
		addBatch(p.BatchNo)
		if matchBatch(p.BatchNo) {
			pjcCount++
		}
	}

	vhcAll, err := uc.repo.ListLiquidSugarWashVerifications(ctx, month, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Gagal mengambil status VHC")
	}
	vhcCount := 0
	for _, v := range vhcAll {
		addBatch(v.BatchNo)
		if matchBatch(v.BatchNo) {
			vhcCount++
		}
	}

	stage1Done := mss1Count > 0 && pvkCount > 0 && vmdCount > 0 && cppCount > 0
	mnpDone := mnpCount > 0
	mnbDone := mnbCount > 0
	qc011Done := mnpDone || mnbDone
	stage2Done := mss2Count > 0 && qc011Done
	stage3Done := mncCount > 0
	stage4Done := pjcCount > 0
	stage5Done := vhcCount > 0

	completed := 0
	for _, done := range []bool{stage1Done, stage2Done, stage3Done, stage4Done, stage5Done} {
		if done {
			completed++
		}
	}

	current := 1
	if stage1Done {
		current = 2
	}
	if stage1Done && stage2Done {
		current = 3
	}
	if stage1Done && stage2Done && stage3Done {
		current = 4
	}
	if stage1Done && stage2Done && stage3Done && stage4Done {
		current = 5
	}

	batches := make([]string, 0, len(batchSet))
	for b := range batchSet {
		batches = append(batches, b)
	}
	sort.Strings(batches)

	return &LiquidSugarFlowStatusDTO{
		Month:                month,
		BatchNo:              batchTrim,
		MssStage1Count:       mss1Count,
		PvkCount:             pvkCount,
		VmdCount:             vmdCount,
		CppCount:             cppCount,
		MssStage2Count:       mss2Count,
		MnpCount:             mnpCount,
		MnbCount:             mnbCount,
		MncCount:             mncCount,
		PjcCount:             pjcCount,
		VhcCount:             vhcCount,
		Stage1Done:           stage1Done,
		MnpDone:              mnpDone,
		MnbDone:              mnbDone,
		Qc011Done:            qc011Done,
		Stage2Done:           stage2Done,
		Stage3Done:           stage3Done,
		Stage4Done:           stage4Done,
		Stage5Done:           stage5Done,
		CompletedStagesCount: completed,
		CurrentStage:         current,
		BatchNumbers:         batches,
	}, nil
}
