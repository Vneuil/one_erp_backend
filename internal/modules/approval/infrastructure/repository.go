package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/approval/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type approvalRepository struct {
	db *gorm.DB
}

func NewApprovalRepository(db *gorm.DB) domain.ApprovalRepository {
	return &approvalRepository{db: db}
}

func (r *approvalRepository) CreateWorkflow(ctx context.Context, w *domain.ApprovalWorkflow) error {
	tenantctx.SetTenantID(ctx, &w.TenantID)
	return r.db.WithContext(ctx).Create(w).Error
}

func (r *approvalRepository) GetWorkflowByID(ctx context.Context, id uuid.UUID) (*domain.ApprovalWorkflow, error) {
	var w domain.ApprovalWorkflow
	err := r.db.WithContext(ctx).Preload("Levels", func(db *gorm.DB) *gorm.DB {
		return db.Order("level_order asc")
	}).Where("id = ?", id).First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

func (r *approvalRepository) ListWorkflows(ctx context.Context, documentType string) ([]domain.ApprovalWorkflow, error) {
	var items []domain.ApprovalWorkflow
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ApprovalWorkflow{}))
	if documentType != "" {
		db = db.Where("document_type = ?", documentType)
	}
	err := db.Preload("Levels", func(db *gorm.DB) *gorm.DB {
		return db.Order("level_order asc")
	}).Order("document_type asc, min_amount asc").Find(&items).Error
	return items, err
}

func (r *approvalRepository) ListMatchingWorkflows(ctx context.Context, documentType string, amount float64) ([]domain.ApprovalWorkflow, error) {
	var items []domain.ApprovalWorkflow
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ApprovalWorkflow{}))
	err := db.Where("document_type = ? AND is_active = ? AND min_amount <= ?", documentType, true, amount).
		Preload("Levels", func(db *gorm.DB) *gorm.DB {
			return db.Order("level_order asc")
		}).
		Order("min_amount desc").
		Find(&items).Error
	return items, err
}

func (r *approvalRepository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workflow_id = ?", id).Delete(&domain.ApprovalWorkflowLevel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&domain.ApprovalWorkflow{}, id).Error
	})
}

func (r *approvalRepository) CreateRequest(ctx context.Context, req *domain.ApprovalRequest) error {
	tenantctx.SetTenantID(ctx, &req.TenantID)
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *approvalRepository) GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.ApprovalRequest, error) {
	var req domain.ApprovalRequest
	err := r.db.WithContext(ctx).Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("level_order asc")
	}).Where("id = ?", id).First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *approvalRepository) GetActiveRequestForDocument(ctx context.Context, documentType string, documentID uuid.UUID) (*domain.ApprovalRequest, error) {
	var req domain.ApprovalRequest
	err := r.db.WithContext(ctx).Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("level_order asc")
	}).Where("document_type = ? AND document_id = ?", documentType, documentID).
		Order("created_at desc").First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *approvalRepository) ListRequests(ctx context.Context, query types.PaginationQuery, documentType string) ([]domain.ApprovalRequest, int64, error) {
	var items []domain.ApprovalRequest
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ApprovalRequest{}))
	if documentType != "" {
		db = db.Where("document_type = ?", documentType)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("level_order asc")
	}).Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *approvalRepository) UpdateRequest(ctx context.Context, req *domain.ApprovalRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *approvalRepository) UpdateStep(ctx context.Context, s *domain.ApprovalStep) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *approvalRepository) ListPendingStepsForRole(ctx context.Context, role string) ([]domain.ApprovalStep, error) {
	var steps []domain.ApprovalStep
	// A step is actionable only when it's pending AND its parent request's
	// CurrentLevel matches this step's LevelOrder (earlier/later levels
	// aren't up for action yet).
	err := r.db.WithContext(ctx).
		Joins("JOIN approval_requests ON approval_requests.id = approval_steps.request_id").
		Where("approval_steps.status = ? AND approval_steps.approver_role = ? AND approval_requests.current_level = approval_steps.level_order AND approval_requests.status = ?",
			domain.StatusPending, role, domain.StatusPending).
		Order("approval_steps.created_at asc").
		Find(&steps).Error
	return steps, err
}
