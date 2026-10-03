package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/projecttask/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type projectTaskRepository struct {
	db *gorm.DB
}

func NewProjectTaskRepository(db *gorm.DB) domain.ProjectTaskRepository {
	return &projectTaskRepository{db: db}
}

func (r *projectTaskRepository) Create(ctx context.Context, task *domain.ProjectTask) error {
	tenantctx.SetTenantID(ctx, &task.TenantID)
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *projectTaskRepository) CreateWithChecklist(ctx context.Context, task *domain.ProjectTask, items []domain.TaskChecklistItem) error {
	tenantctx.SetTenantID(ctx, &task.TenantID)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TaskID = task.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *projectTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ProjectTask, error) {
	var task domain.ProjectTask
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

func (r *projectTaskRepository) List(ctx context.Context, query types.PaginationQuery, projectCode, status string) ([]domain.ProjectTask, int64, error) {
	var tasks []domain.ProjectTask
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ProjectTask{}))

	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ? OR assignee ILIKE ?", pattern, pattern)
	}
	if projectCode != "" {
		db = db.Where("project_code = ?", projectCode)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&tasks).Error
	return tasks, total, err
}

func (r *projectTaskRepository) Update(ctx context.Context, task *domain.ProjectTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *projectTaskRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.ProjectTask{}).Count(&total).Error
	return total, err
}

func (r *projectTaskRepository) CreateChecklistItem(ctx context.Context, item *domain.TaskChecklistItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *projectTaskRepository) GetChecklistItem(ctx context.Context, id uuid.UUID) (*domain.TaskChecklistItem, error) {
	var item domain.TaskChecklistItem
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *projectTaskRepository) UpdateChecklistItem(ctx context.Context, item *domain.TaskChecklistItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *projectTaskRepository) ListChecklistByTask(ctx context.Context, taskID uuid.UUID) ([]domain.TaskChecklistItem, error) {
	var items []domain.TaskChecklistItem
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("created_at asc").Find(&items).Error
	return items, err
}
