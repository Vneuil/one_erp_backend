package infrastructure

import (
	"context"
	"errors"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type qcRepository struct {
	db *gorm.DB
}

func NewQCRepository(db *gorm.DB) domain.QCRepository {
	return &qcRepository{db: db}
}

// ─────────────────────────────────────────────────────────────────────────────
// Parameter
// ─────────────────────────────────────────────────────────────────────────────

func (r *qcRepository) CreateParameter(ctx context.Context, param *domain.QCParameter) error {
	return r.db.WithContext(ctx).Create(param).Error
}

func (r *qcRepository) GetParameterByID(ctx context.Context, id uuid.UUID) (*domain.QCParameter, error) {
	var param domain.QCParameter
	if err := r.db.WithContext(ctx).First(&param, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &param, nil
}

func (r *qcRepository) ListParameters(ctx context.Context, productID *uuid.UUID, processCode string) ([]domain.QCParameter, error) {
	var params []domain.QCParameter
	q := r.db.WithContext(ctx)

	if productID != nil && *productID != uuid.Nil {
		// return product-specific OR global (product_id is null)
		q = q.Where("product_id = ? OR product_id IS NULL", *productID)
	}
	if strings.TrimSpace(processCode) != "" {
		q = q.Where("LOWER(process_code) = ?", strings.ToLower(strings.TrimSpace(processCode)))
	}

	if err := q.Order("process_code ASC, name ASC").Find(&params).Error; err != nil {
		return nil, err
	}
	return params, nil
}

func (r *qcRepository) UpdateParameter(ctx context.Context, param *domain.QCParameter) error {
	return r.db.WithContext(ctx).Save(param).Error
}

func (r *qcRepository) DeleteParameter(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.QCParameter{}, "id = ?", id).Error
}

// ─────────────────────────────────────────────────────────────────────────────
// Inspection
// ─────────────────────────────────────────────────────────────────────────────

func (r *qcRepository) CreateInspection(ctx context.Context, insp *domain.QCInspection) error {
	return r.db.WithContext(ctx).Create(insp).Error
}

func (r *qcRepository) GetInspectionByID(ctx context.Context, id uuid.UUID) (*domain.QCInspection, error) {
	var insp domain.QCInspection
	if err := r.db.WithContext(ctx).Preload("Items").First(&insp, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &insp, nil
}

func (r *qcRepository) GetInspectionByNo(ctx context.Context, inspectionNo string) (*domain.QCInspection, error) {
	var insp domain.QCInspection
	if err := r.db.WithContext(ctx).Preload("Items").First(&insp, "inspection_no = ?", inspectionNo).Error; err != nil {
		return nil, err
	}
	return &insp, nil
}

func (r *qcRepository) ListInspections(ctx context.Context, filter domain.ListInspectionFilter) ([]domain.QCInspection, int64, error) {
	var items []domain.QCInspection
	var total int64

	q := r.db.WithContext(ctx).Model(&domain.QCInspection{})

	if filter.TargetType != nil && *filter.TargetType != "" {
		q = q.Where("target_type = ?", *filter.TargetType)
	}
	if filter.TargetID != nil && *filter.TargetID != uuid.Nil {
		q = q.Where("target_id = ?", *filter.TargetID)
	}
	if filter.ProductionOrderID != nil && *filter.ProductionOrderID != uuid.Nil {
		q = q.Where("production_order_id = ?", *filter.ProductionOrderID)
	}
	if filter.ProductionStepID != nil && *filter.ProductionStepID != uuid.Nil {
		q = q.Where("production_step_id = ?", *filter.ProductionStepID)
	}
	if strings.TrimSpace(filter.BatchNo) != "" {
		q = q.Where("batch_no ILIKE ?", "%"+strings.TrimSpace(filter.BatchNo)+"%")
	}
	if filter.ProductID != nil && *filter.ProductID != uuid.Nil {
		q = q.Where("product_id = ?", *filter.ProductID)
	}
	if filter.Status != nil && *filter.Status != "" {
		q = q.Where("overall_status = ?", *filter.Status)
	}
	if filter.StartDate != nil {
		q = q.Where("inspected_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		q = q.Where("inspected_at <= ?", *filter.EndDate)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	if err := q.Preload("Items").Order("inspected_at DESC, created_at DESC").Offset(offset).Limit(perPage).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *qcRepository) UpdateInspection(ctx context.Context, insp *domain.QCInspection) error {
	return r.db.WithContext(ctx).Save(insp).Error
}

func (r *qcRepository) GetLatestPassedInspection(ctx context.Context, targetType domain.TargetType, targetID uuid.UUID, processCode string) (*domain.QCInspection, error) {
	var insp domain.QCInspection
	q := r.db.WithContext(ctx).
		Preload("Items").
		Where("target_type = ? AND target_id = ? AND overall_status = ? AND is_voided = false", targetType, targetID, domain.QCStatusPassed)

	if strings.TrimSpace(processCode) != "" {
		// join or check items
		q = q.Where("id IN (SELECT inspection_id FROM qc_inspection_items WHERE parameter_id IN (SELECT id FROM qc_parameters WHERE LOWER(process_code) = ?))", strings.ToLower(strings.TrimSpace(processCode)))
	}

	err := q.Order("inspected_at DESC").First(&insp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &insp, nil
}

func (r *qcRepository) GetInspectionCountToday(ctx context.Context, datePrefix string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.QCInspection{}).
		Where("inspection_no LIKE ?", datePrefix+"%").
		Count(&count).Error
	return count, err
}

func (r *qcRepository) GetInspectionStats(ctx context.Context) (total int64, passed int64, failed int64, voided int64, err error) {
	q := r.db.WithContext(ctx).Model(&domain.QCInspection{})
	if err = q.Count(&total).Error; err != nil {
		return
	}
	_ = r.db.WithContext(ctx).Model(&domain.QCInspection{}).Where("overall_status = ? AND is_voided = false", domain.QCStatusPassed).Count(&passed)
	_ = r.db.WithContext(ctx).Model(&domain.QCInspection{}).Where("overall_status = ? AND is_voided = false", domain.QCStatusFailed).Count(&failed)
	_ = r.db.WithContext(ctx).Model(&domain.QCInspection{}).Where("is_voided = true", true).Count(&voided)
	return
}

func (r *qcRepository) CreateHygieneCheck(ctx context.Context, check *domain.PersonalHygieneCheck) error {
	return r.db.WithContext(ctx).Create(check).Error
}

func (r *qcRepository) ListHygieneChecks(ctx context.Context, date string, line string) ([]domain.PersonalHygieneCheck, error) {
	var checks []domain.PersonalHygieneCheck
	q := r.db.WithContext(ctx).Model(&domain.PersonalHygieneCheck{})
	if date != "" {
		q = q.Where("check_date = ?", date)
	}
	if line != "" {
		q = q.Where("line = ?", line)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&checks).Error
	return checks, err
}

func (r *qcRepository) DeleteHygieneCheck(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.PersonalHygieneCheck{}, "id = ?", id).Error
}

func (r *qcRepository) CreatePreOpCheck(ctx context.Context, check *domain.PreOpSanitationCheck) error {
	return r.db.WithContext(ctx).Create(check).Error
}

func (r *qcRepository) ListPreOpChecks(ctx context.Context, date string, room string) ([]domain.PreOpSanitationCheck, error) {
	var checks []domain.PreOpSanitationCheck
	q := r.db.WithContext(ctx).Model(&domain.PreOpSanitationCheck{})
	if date != "" {
		q = q.Where("check_date = ?", date)
	}
	if room != "" {
		q = q.Where("room = ?", room)
	}
	err := q.Order("check_date DESC, check_time DESC, created_at DESC").Find(&checks).Error
	return checks, err
}

func (r *qcRepository) DeletePreOpCheck(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.PreOpSanitationCheck{}, "id = ?", id).Error
}

// ─── Packaging Quality Check (PBK — QC-003) ──────────────────────────────────

func (r *qcRepository) CreatePackagingCheck(ctx context.Context, check *domain.PackagingQualityCheck) error {
	return r.db.WithContext(ctx).Create(check).Error
}

func (r *qcRepository) GetPackagingCheckByID(ctx context.Context, id uuid.UUID) (*domain.PackagingQualityCheck, error) {
	var check domain.PackagingQualityCheck
	if err := r.db.WithContext(ctx).First(&check, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &check, nil
}

func (r *qcRepository) ListPackagingChecks(ctx context.Context, month string, lbmNo string) ([]domain.PackagingQualityCheck, error) {
	var checks []domain.PackagingQualityCheck
	q := r.db.WithContext(ctx).Model(&domain.PackagingQualityCheck{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if lbmNo != "" {
		q = q.Where("lbm_no = ?", lbmNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&checks).Error
	return checks, err
}

func (r *qcRepository) UpdatePackagingCheck(ctx context.Context, check *domain.PackagingQualityCheck) error {
	return r.db.WithContext(ctx).Save(check).Error
}

func (r *qcRepository) DeletePackagingCheck(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.PackagingQualityCheck{}, "id = ?", id).Error
}

// ─── Supplier Armada Check (CAS — QC-007) ────────────────────────────────────

func (r *qcRepository) CreateArmadaCheck(ctx context.Context, check *domain.SupplierArmadaCheck) error {
	return r.db.WithContext(ctx).Create(check).Error
}

func (r *qcRepository) GetArmadaCheckByID(ctx context.Context, id uuid.UUID) (*domain.SupplierArmadaCheck, error) {
	var check domain.SupplierArmadaCheck
	if err := r.db.WithContext(ctx).First(&check, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &check, nil
}

func (r *qcRepository) ListArmadaChecks(ctx context.Context, date string, lbmNo string) ([]domain.SupplierArmadaCheck, error) {
	var checks []domain.SupplierArmadaCheck
	q := r.db.WithContext(ctx).Model(&domain.SupplierArmadaCheck{})
	if date != "" {
		q = q.Where("check_date = ?", date)
	}
	if lbmNo != "" {
		q = q.Where("lbm_no = ?", lbmNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&checks).Error
	return checks, err
}

func (r *qcRepository) UpdateArmadaCheck(ctx context.Context, check *domain.SupplierArmadaCheck) error {
	return r.db.WithContext(ctx).Save(check).Error
}

func (r *qcRepository) DeleteArmadaCheck(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.SupplierArmadaCheck{}, "id = ?", id).Error
}

// ─── Raw Material Incoming Check (PBD — QC-001) ──────────────────────────────

func (r *qcRepository) CreateIncomingCheck(ctx context.Context, check *domain.RawMaterialIncomingCheck) error {
	return r.db.WithContext(ctx).Create(check).Error
}

func (r *qcRepository) GetIncomingCheckByID(ctx context.Context, id uuid.UUID) (*domain.RawMaterialIncomingCheck, error) {
	var check domain.RawMaterialIncomingCheck
	if err := r.db.WithContext(ctx).First(&check, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &check, nil
}

func (r *qcRepository) ListIncomingChecks(ctx context.Context, date string, lbmNo string) ([]domain.RawMaterialIncomingCheck, error) {
	var checks []domain.RawMaterialIncomingCheck
	q := r.db.WithContext(ctx).Model(&domain.RawMaterialIncomingCheck{})
	if date != "" {
		q = q.Where("check_date = ?", date)
	}
	if lbmNo != "" {
		q = q.Where("lbm_no = ?", lbmNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&checks).Error
	return checks, err
}

func (r *qcRepository) UpdateIncomingCheck(ctx context.Context, check *domain.RawMaterialIncomingCheck) error {
	return r.db.WithContext(ctx).Save(check).Error
}

func (r *qcRepository) DeleteIncomingCheck(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.RawMaterialIncomingCheck{}, "id = ?", id).Error
}

// ─── Sieving Monitoring (MNA — Form QC-009-MNA.01) ───────────────────────────

func (r *qcRepository) CreateSievingLog(ctx context.Context, log *domain.SievingMonitoringLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetSievingLogByID(ctx context.Context, id uuid.UUID) (*domain.SievingMonitoringLog, error) {
	var item domain.SievingMonitoringLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListSievingLogs(ctx context.Context, month string, category string, machineNo string, batchNo string) ([]domain.SievingMonitoringLog, error) {
	var logs []domain.SievingMonitoringLog
	q := r.db.WithContext(ctx).Model(&domain.SievingMonitoringLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if machineNo != "" {
		q = q.Where("machine_no = ?", machineNo)
	}
	if batchNo != "" {
		q = q.Where("batch_no = ?", batchNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateSievingLog(ctx context.Context, log *domain.SievingMonitoringLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteSievingLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.SievingMonitoringLog{}, "id = ?", id).Error
}

// ─── Magnet Trap (CMT — Form QC-038-CMT) ────────────────────────────────────

func (r *qcRepository) CreateMagnetTrapLog(ctx context.Context, log *domain.MagnetTrapLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetMagnetTrapLogByID(ctx context.Context, id uuid.UUID) (*domain.MagnetTrapLog, error) {
	var item domain.MagnetTrapLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListMagnetTrapLogs(ctx context.Context, month string, room string, machineNo string) ([]domain.MagnetTrapLog, error) {
	var logs []domain.MagnetTrapLog
	q := r.db.WithContext(ctx).Model(&domain.MagnetTrapLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if room != "" {
		q = q.Where("room = ?", room)
	}
	if machineNo != "" {
		q = q.Where("machine_no = ?", machineNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateMagnetTrapLog(ctx context.Context, log *domain.MagnetTrapLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteMagnetTrapLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.MagnetTrapLog{}, "id = ?", id).Error
}

// ─── Mesh Bolt Monitoring (MNM — Form QC-010-MNM.01) ─────────────────────────

func (r *qcRepository) CreateMeshBoltLog(ctx context.Context, log *domain.MeshBoltMonitoringLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetMeshBoltLogByID(ctx context.Context, id uuid.UUID) (*domain.MeshBoltMonitoringLog, error) {
	var item domain.MeshBoltMonitoringLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListMeshBoltLogs(ctx context.Context, month string, room string, machineNo string) ([]domain.MeshBoltMonitoringLog, error) {
	var logs []domain.MeshBoltMonitoringLog
	q := r.db.WithContext(ctx).Model(&domain.MeshBoltMonitoringLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if room != "" {
		q = q.Where("room = ?", room)
	}
	if machineNo != "" {
		q = q.Where("machine_no = ?", machineNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateMeshBoltLog(ctx context.Context, log *domain.MeshBoltMonitoringLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteMeshBoltLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.MeshBoltMonitoringLog{}, "id = ?", id).Error
}

// ─── Oven Monitoring (MNO — Form QC-012-MNO.05) ──────────────────────────────

func (r *qcRepository) CreateOvenLog(ctx context.Context, log *domain.OvenMonitoringLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetOvenLogByID(ctx context.Context, id uuid.UUID) (*domain.OvenMonitoringLog, error) {
	var item domain.OvenMonitoringLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListOvenLogs(ctx context.Context, month string, ovenNo string, batchNo string) ([]domain.OvenMonitoringLog, error) {
	var logs []domain.OvenMonitoringLog
	q := r.db.WithContext(ctx).Model(&domain.OvenMonitoringLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if ovenNo != "" {
		q = q.Where("oven_no = ?", ovenNo)
	}
	if batchNo != "" {
		q = q.Where("batch_no = ?", batchNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateOvenLog(ctx context.Context, log *domain.OvenMonitoringLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteOvenLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.OvenMonitoringLog{}, "id = ?", id).Error
}

// ─── Needle Monitoring (MNJ — Form QC-011-MNJ.01) ────────────────────────────

func (r *qcRepository) CreateNeedleLog(ctx context.Context, log *domain.NeedleMonitoringLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetNeedleLogByID(ctx context.Context, id uuid.UUID) (*domain.NeedleMonitoringLog, error) {
	var item domain.NeedleMonitoringLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListNeedleLogs(ctx context.Context, month string, machineNo string) ([]domain.NeedleMonitoringLog, error) {
	var logs []domain.NeedleMonitoringLog
	q := r.db.WithContext(ctx).Model(&domain.NeedleMonitoringLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if machineNo != "" {
		q = q.Where("sewing_machine_no = ?", machineNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateNeedleLog(ctx context.Context, log *domain.NeedleMonitoringLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteNeedleLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.NeedleMonitoringLog{}, "id = ?", id).Error
}

// ─── Scissor Monitoring (MNG — Form QC-011-MNJ.01 Sheet MNG) ─────────────────

func (r *qcRepository) CreateScissorLog(ctx context.Context, log *domain.ScissorMonitoringLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetScissorLogByID(ctx context.Context, id uuid.UUID) (*domain.ScissorMonitoringLog, error) {
	var item domain.ScissorMonitoringLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListScissorLogs(ctx context.Context, month string, location string) ([]domain.ScissorMonitoringLog, error) {
	var logs []domain.ScissorMonitoringLog
	q := r.db.WithContext(ctx).Model(&domain.ScissorMonitoringLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if location != "" {
		q = q.Where("location = ?", location)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateScissorLog(ctx context.Context, log *domain.ScissorMonitoringLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteScissorLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.ScissorMonitoringLog{}, "id = ?", id).Error
}

// ─── Metal Detector Cleaning (CPP — Form CPP) ────────────────────────────────

func (r *qcRepository) CreateMetalCleaningLog(ctx context.Context, log *domain.MetalDetectorCleaningLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetMetalCleaningLogByID(ctx context.Context, id uuid.UUID) (*domain.MetalDetectorCleaningLog, error) {
	var item domain.MetalDetectorCleaningLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListMetalCleaningLogs(ctx context.Context, month string, machineNo string, location string) ([]domain.MetalDetectorCleaningLog, error) {
	var logs []domain.MetalDetectorCleaningLog
	q := r.db.WithContext(ctx).Model(&domain.MetalDetectorCleaningLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if machineNo != "" {
		q = q.Where("machine_no = ?", machineNo)
	}
	if location != "" {
		q = q.Where("location = ?", location)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateMetalCleaningLog(ctx context.Context, log *domain.MetalDetectorCleaningLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteMetalCleaningLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.MetalDetectorCleaningLog{}, "id = ?", id).Error
}

// ─── Metal Detector Verification (VMD — Form QC-017-VMD) ─────────────────────

func (r *qcRepository) CreateMetalVerificationLog(ctx context.Context, log *domain.MetalDetectorVerificationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetMetalVerificationLogByID(ctx context.Context, id uuid.UUID) (*domain.MetalDetectorVerificationLog, error) {
	var item domain.MetalDetectorVerificationLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListMetalVerificationLogs(ctx context.Context, month string, productType string, location string) ([]domain.MetalDetectorVerificationLog, error) {
	var logs []domain.MetalDetectorVerificationLog
	q := r.db.WithContext(ctx).Model(&domain.MetalDetectorVerificationLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if productType != "" {
		q = q.Where("product_type = ?", productType)
	}
	if location != "" {
		q = q.Where("location = ?", location)
	}
	err := q.Order("check_date DESC, monitoring_round ASC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateMetalVerificationLog(ctx context.Context, log *domain.MetalDetectorVerificationLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteMetalVerificationLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.MetalDetectorVerificationLog{}, "id = ?", id).Error
}

// ─── Finished Goods Inspection (PJD — Form QC-004-PJD.05 Rev 05) ─────────────

func (r *qcRepository) CreateFinishedGoodsLog(ctx context.Context, log *domain.FinishedGoodsInspectionLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetFinishedGoodsLogByID(ctx context.Context, id uuid.UUID) (*domain.FinishedGoodsInspectionLog, error) {
	var item domain.FinishedGoodsInspectionLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListFinishedGoodsLogs(ctx context.Context, month string, productName string, batchNo string, decision string) ([]domain.FinishedGoodsInspectionLog, error) {
	var logs []domain.FinishedGoodsInspectionLog
	q := r.db.WithContext(ctx).Model(&domain.FinishedGoodsInspectionLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if productName != "" {
		q = q.Where("product_name LIKE ?", "%"+productName+"%")
	}
	if batchNo != "" {
		q = q.Where("batch_no LIKE ?", "%"+batchNo+"%")
	}
	if decision != "" {
		q = q.Where("inspection_decision = ?", decision)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateFinishedGoodsLog(ctx context.Context, log *domain.FinishedGoodsInspectionLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteFinishedGoodsLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.FinishedGoodsInspectionLog{}, "id = ?", id).Error
}

// ─── Packaging Label Check (PLK — Form QC-018-PLK) ───────────────────────────

func (r *qcRepository) CreatePackagingLabelCheck(ctx context.Context, log *domain.PackagingLabelCheckLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetPackagingLabelCheckByID(ctx context.Context, id uuid.UUID) (*domain.PackagingLabelCheckLog, error) {
	var item domain.PackagingLabelCheckLog
	if err := r.db.WithContext(ctx).Preload("Items").First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListPackagingLabelChecks(ctx context.Context, month string, batchNo string, decision string) ([]domain.PackagingLabelCheckLog, error) {
	var logs []domain.PackagingLabelCheckLog
	q := r.db.WithContext(ctx).Model(&domain.PackagingLabelCheckLog{}).Preload("Items")
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if batchNo != "" {
		q = q.Where("batch_no LIKE ? OR id IN (SELECT log_id FROM qc_packaging_label_check_items WHERE batch_no LIKE ?)", "%"+batchNo+"%", "%"+batchNo+"%")
	}
	if decision != "" {
		q = q.Where("overall_decision = ?", decision)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdatePackagingLabelCheck(ctx context.Context, log *domain.PackagingLabelCheckLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("log_id = ?", log.ID).Delete(&domain.PackagingLabelCheckItem{}).Error; err != nil {
			return err
		}
		return tx.Save(log).Error
	})
}

func (r *qcRepository) DeletePackagingLabelCheck(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("log_id = ?", id).Delete(&domain.PackagingLabelCheckItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(&domain.PackagingLabelCheckLog{}, "id = ?", id).Error
	})
}

// ─── Container Checklist (CCO — Form QC-008-CCO.02) ──────────────────────────

func (r *qcRepository) CreateContainerChecklist(ctx context.Context, log *domain.ContainerChecklistLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetContainerChecklistByID(ctx context.Context, id uuid.UUID) (*domain.ContainerChecklistLog, error) {
	var item domain.ContainerChecklistLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListContainerChecklists(ctx context.Context, month string, containerNo string, destination string, decision string) ([]domain.ContainerChecklistLog, error) {
	var logs []domain.ContainerChecklistLog
	q := r.db.WithContext(ctx).Model(&domain.ContainerChecklistLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if containerNo != "" {
		q = q.Where("container_number LIKE ?", "%"+containerNo+"%")
	}
	if destination != "" {
		q = q.Where("shipment_destination LIKE ?", "%"+destination+"%")
	}
	if decision != "" {
		q = q.Where("overall_decision = ?", decision)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateContainerChecklist(ctx context.Context, log *domain.ContainerChecklistLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteContainerChecklist(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.ContainerChecklistLog{}, "id = ?", id).Error
}

// ─── Brontol & Reject Return Reports (REP-013-KSB) ──────────────────────────

func (r *qcRepository) CreateBrontolReport(ctx context.Context, report *domain.BrontolReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

func (r *qcRepository) GetBrontolReportByID(ctx context.Context, id uuid.UUID) (*domain.BrontolReport, error) {
	var item domain.BrontolReport
	if err := r.db.WithContext(ctx).Preload("ReturnReport").First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) GetBrontolReportByNo(ctx context.Context, reportNo string) (*domain.BrontolReport, error) {
	var item domain.BrontolReport
	if err := r.db.WithContext(ctx).Preload("ReturnReport").First(&item, "report_no = ?", reportNo).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListBrontolReports(ctx context.Context, month, stage, action, status, search string) ([]domain.BrontolReport, error) {
	var reports []domain.BrontolReport
	q := r.db.WithContext(ctx).Model(&domain.BrontolReport{}).Preload("ReturnReport")

	if month != "" {
		q = q.Where("report_month = ?", month)
	}
	if stage != "" {
		q = q.Where("source_stage = ?", stage)
	}
	if action != "" {
		q = q.Where("action_disposition = ?", action)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if search != "" {
		pat := "%" + strings.ToLower(search) + "%"
		q = q.Where("LOWER(report_no) LIKE ? OR LOWER(batch_no) LIKE ? OR LOWER(product_name) LIKE ? OR LOWER(supplier_name) LIKE ?", pat, pat, pat, pat)
	}

	err := q.Order("report_date DESC, created_at DESC").Find(&reports).Error
	return reports, err
}

func (r *qcRepository) UpdateBrontolReport(ctx context.Context, report *domain.BrontolReport) error {
	return r.db.WithContext(ctx).Save(report).Error
}

func (r *qcRepository) DeleteBrontolReport(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.BrontolReport{}, "id = ?", id).Error
}

func (r *qcRepository) CreateRejectReturnReport(ctx context.Context, ret *domain.RejectReturnReport) error {
	return r.db.WithContext(ctx).Create(ret).Error
}

func (r *qcRepository) GetRejectReturnReportByID(ctx context.Context, id uuid.UUID) (*domain.RejectReturnReport, error) {
	var item domain.RejectReturnReport
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) GetRejectReturnReportByBrontolID(ctx context.Context, brontolReportID uuid.UUID) (*domain.RejectReturnReport, error) {
	var item domain.RejectReturnReport
	if err := r.db.WithContext(ctx).First(&item, "brontol_report_id = ?", brontolReportID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListRejectReturnReports(ctx context.Context, month, status, search string) ([]domain.RejectReturnReport, error) {
	var returns []domain.RejectReturnReport
	q := r.db.WithContext(ctx).Model(&domain.RejectReturnReport{})

	if month != "" {
		q = q.Where("return_date LIKE ?", month+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if search != "" {
		pat := "%" + strings.ToLower(search) + "%"
		q = q.Where("LOWER(return_no) LIKE ? OR LOWER(batch_no) LIKE ? OR LOWER(product_name) LIKE ? OR LOWER(supplier_name) LIKE ?", pat, pat, pat, pat)
	}

	err := q.Order("return_date DESC, created_at DESC").Find(&returns).Error
	return returns, err
}

func (r *qcRepository) UpdateRejectReturnReport(ctx context.Context, ret *domain.RejectReturnReport) error {
	return r.db.WithContext(ctx).Save(ret).Error
}

func (r *qcRepository) DeleteRejectReturnReport(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.RejectReturnReport{}, "id = ?", id).Error
}

func (r *qcRepository) GetBrontolStats(ctx context.Context) (totalCount int64, totalQuantityKg float64, returnCount int64, returnQtyKg float64, reworkCount int64, err error) {
	db := r.db.WithContext(ctx).Model(&domain.BrontolReport{})

	if err = db.Count(&totalCount).Error; err != nil {
		return
	}

	type sumResult struct {
		TotalQty float64
	}
	var sumRes sumResult
	_ = r.db.WithContext(ctx).Model(&domain.BrontolReport{}).Select("COALESCE(SUM(quantity), 0) as total_qty").Scan(&sumRes)
	totalQuantityKg = sumRes.TotalQty

	_ = r.db.WithContext(ctx).Model(&domain.BrontolReport{}).Where("action_disposition = ?", domain.ActionReturnToSupplier).Count(&returnCount)
	var retSum sumResult
	_ = r.db.WithContext(ctx).Model(&domain.BrontolReport{}).Where("action_disposition = ?", domain.ActionReturnToSupplier).Select("COALESCE(SUM(quantity), 0) as total_qty").Scan(&retSum)
	returnQtyKg = retSum.TotalQty

	_ = r.db.WithContext(ctx).Model(&domain.BrontolReport{}).Where("action_disposition IN (?, ?)", domain.ActionRework, domain.ActionReOven).Count(&reworkCount)

	return
}

// ─── Helper Prefix Counter ───────────────────────────────────────────────────

func (r *qcRepository) CountLogsByPrefix(ctx context.Context, tableName string, prefix string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(tableName).Where("document_no LIKE ?", prefix+"%").Count(&count).Error
	return count, err
}

