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
	datePrefix := fmt.Sprintf("QC-%s", now.Format("200601"))
	count, _ := uc.repo.GetInspectionCountToday(ctx, datePrefix)
	inspNo := fmt.Sprintf("%s-%04d", datePrefix, count+1)

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
	isPassed := dto.WorkWearClean && dto.HairNet && dto.NailsClean && dto.NoJewelry && dto.WorkShoesClean && dto.HandWashing && dto.HealthFit
	check := &domain.PersonalHygieneCheck{
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
