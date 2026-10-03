package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/companyctx"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type exportsRepository struct {
	db *gorm.DB
}

func NewExportsRepository(db *gorm.DB) domain.ExportsRepository {
	return &exportsRepository{db: db}
}

func (r *exportsRepository) CreateJob(ctx context.Context, j *domain.ExportJob) error {
	companyctx.SetCompanyID(ctx, &j.CompanyID)
	tenantctx.SetTenantID(ctx, &j.TenantID)
	return r.db.WithContext(ctx).Create(j).Error
}

func (r *exportsRepository) GetJobByID(ctx context.Context, id uuid.UUID) (*domain.ExportJob, error) {
	var j domain.ExportJob
	err := companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&j).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &j, nil
}

func (r *exportsRepository) ListJobs(ctx context.Context) ([]domain.ExportJob, error) {
	var items []domain.ExportJob
	db := companyctx.Scope(ctx, tenantctx.Scope(ctx, r.db.WithContext(ctx)))
	err := db.Order("created_at desc").Find(&items).Error
	return items, err
}

func (r *exportsRepository) UpdateJob(ctx context.Context, j *domain.ExportJob) error {
	return r.db.WithContext(ctx).Save(j).Error
}

func (r *exportsRepository) CountJobs(ctx context.Context) (int64, error) {
	var total int64
	err := companyctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ExportJob{})).Count(&total).Error
	return total, err
}
