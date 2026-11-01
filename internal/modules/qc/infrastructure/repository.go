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
