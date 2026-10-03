package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/project/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) domain.ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(ctx context.Context, project *domain.Project) error {
	tenantctx.SetTenantID(ctx, &project.TenantID)
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *projectRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	var project domain.Project
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &project, nil
}

func (r *projectRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.Project, int64, error) {
	var projects []domain.Project
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Project{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ? OR code ILIKE ? OR customer ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&projects).Error
	return projects, total, err
}

func (r *projectRepository) Update(ctx context.Context, project *domain.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Project{}, id).Error
}

func (r *projectRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Project{}).Count(&total).Error
	return total, err
}

type timeEntryRepository struct {
	db *gorm.DB
}

func NewTimeEntryRepository(db *gorm.DB) domain.TimeEntryRepository {
	return &timeEntryRepository{db: db}
}

func (r *timeEntryRepository) Create(ctx context.Context, entry *domain.TimeEntry) error {
	tenantctx.SetTenantID(ctx, &entry.TenantID)
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *timeEntryRepository) List(ctx context.Context, query types.PaginationQuery, projectCode string) ([]domain.TimeEntry, int64, error) {
	var entries []domain.TimeEntry
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.TimeEntry{}))
	if projectCode != "" {
		db = db.Where("project_code = ?", projectCode)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&entries).Error
	return entries, total, err
}

func (r *timeEntryRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.TimeEntry{}).Count(&total).Error
	return total, err
}

func (r *timeEntryRepository) ListAllByProjectCode(ctx context.Context, projectCode string) ([]domain.TimeEntry, error) {
	var entries []domain.TimeEntry
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.TimeEntry{}))
	err := db.Where("project_code = ?", projectCode).Find(&entries).Error
	return entries, err
}
