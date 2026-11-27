package infrastructure

import (
	"context"

	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/google/uuid"
)

// ─── MSS: Sterilization Temperature Log (Form QC-057-MSS) ───────────────────

func (r *qcRepository) CreateSterilizationTempLog(ctx context.Context, log *domain.SterilizationTempLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetSterilizationTempLogByID(ctx context.Context, id uuid.UUID) (*domain.SterilizationTempLog, error) {
	var item domain.SterilizationTempLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListSterilizationTempLogs(ctx context.Context, month string, stage string, batchNo string) ([]domain.SterilizationTempLog, error) {
	var logs []domain.SterilizationTempLog
	q := r.db.WithContext(ctx).Model(&domain.SterilizationTempLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if stage != "" {
		q = q.Where("stage = ?", stage)
	}
	if batchNo != "" {
		q = q.Where("batch_no = ?", batchNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateSterilizationTempLog(ctx context.Context, log *domain.SterilizationTempLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteSterilizationTempLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.SterilizationTempLog{}, "id = ?", id).Error
}

// ─── PVK: Liquid Packaging Visual Check (Form QC-041-PVK Rev. 01) ───────────

func (r *qcRepository) CreateLiquidPackagingVisualCheck(ctx context.Context, log *domain.LiquidPackagingVisualCheck) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetLiquidPackagingVisualCheckByID(ctx context.Context, id uuid.UUID) (*domain.LiquidPackagingVisualCheck, error) {
	var item domain.LiquidPackagingVisualCheck
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListLiquidPackagingVisualChecks(ctx context.Context, month string, batchNo string) ([]domain.LiquidPackagingVisualCheck, error) {
	var logs []domain.LiquidPackagingVisualCheck
	q := r.db.WithContext(ctx).Model(&domain.LiquidPackagingVisualCheck{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if batchNo != "" {
		q = q.Where("batch_no = ?", batchNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateLiquidPackagingVisualCheck(ctx context.Context, log *domain.LiquidPackagingVisualCheck) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteLiquidPackagingVisualCheck(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.LiquidPackagingVisualCheck{}, "id = ?", id).Error
}

// ─── MNC: Liquid Sugar Processing Log (Form QC-014-MNC) ─────────────────────

func (r *qcRepository) CreateLiquidSugarProcessingLog(ctx context.Context, log *domain.LiquidSugarProcessingLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetLiquidSugarProcessingLogByID(ctx context.Context, id uuid.UUID) (*domain.LiquidSugarProcessingLog, error) {
	var item domain.LiquidSugarProcessingLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListLiquidSugarProcessingLogs(ctx context.Context, month string, batchNo string, productCode string) ([]domain.LiquidSugarProcessingLog, error) {
	var logs []domain.LiquidSugarProcessingLog
	q := r.db.WithContext(ctx).Model(&domain.LiquidSugarProcessingLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if batchNo != "" {
		q = q.Where("batch_no = ?", batchNo)
	}
	if productCode != "" {
		q = q.Where("product_code LIKE ?", "%"+productCode+"%")
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateLiquidSugarProcessingLog(ctx context.Context, log *domain.LiquidSugarProcessingLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteLiquidSugarProcessingLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.LiquidSugarProcessingLog{}, "id = ?", id).Error
}

// ─── PJC: Liquid Sugar Finished Goods Log (Form QC-015-PJC Rev. 02) ─────────

func (r *qcRepository) CreateLiquidSugarFinishedGoodsLog(ctx context.Context, log *domain.LiquidSugarFinishedGoodsLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetLiquidSugarFinishedGoodsLogByID(ctx context.Context, id uuid.UUID) (*domain.LiquidSugarFinishedGoodsLog, error) {
	var item domain.LiquidSugarFinishedGoodsLog
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListLiquidSugarFinishedGoodsLogs(ctx context.Context, month string, batchNo string, decision string) ([]domain.LiquidSugarFinishedGoodsLog, error) {
	var logs []domain.LiquidSugarFinishedGoodsLog
	q := r.db.WithContext(ctx).Model(&domain.LiquidSugarFinishedGoodsLog{})
	if month != "" {
		q = q.Where("check_month = ?", month)
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

func (r *qcRepository) UpdateLiquidSugarFinishedGoodsLog(ctx context.Context, log *domain.LiquidSugarFinishedGoodsLog) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteLiquidSugarFinishedGoodsLog(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.LiquidSugarFinishedGoodsLog{}, "id = ?", id).Error
}

// ─── VHC: Liquid Sugar Wash Verification (Form QC-055-VHC Rev. 00) ──────────

func (r *qcRepository) CreateLiquidSugarWashVerification(ctx context.Context, log *domain.LiquidSugarWashVerification) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *qcRepository) GetLiquidSugarWashVerificationByID(ctx context.Context, id uuid.UUID) (*domain.LiquidSugarWashVerification, error) {
	var item domain.LiquidSugarWashVerification
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *qcRepository) ListLiquidSugarWashVerifications(ctx context.Context, month string, batchNo string) ([]domain.LiquidSugarWashVerification, error) {
	var logs []domain.LiquidSugarWashVerification
	q := r.db.WithContext(ctx).Model(&domain.LiquidSugarWashVerification{})
	if month != "" {
		q = q.Where("check_month = ?", month)
	}
	if batchNo != "" {
		q = q.Where("batch_no = ?", batchNo)
	}
	err := q.Order("check_date DESC, created_at DESC").Find(&logs).Error
	return logs, err
}

func (r *qcRepository) UpdateLiquidSugarWashVerification(ctx context.Context, log *domain.LiquidSugarWashVerification) error {
	return r.db.WithContext(ctx).Save(log).Error
}

func (r *qcRepository) DeleteLiquidSugarWashVerification(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.LiquidSugarWashVerification{}, "id = ?", id).Error
}
