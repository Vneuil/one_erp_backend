package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/recruitment/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type recruitmentRepository struct {
	db *gorm.DB
}

func NewRecruitmentRepository(db *gorm.DB) domain.RecruitmentRepository {
	return &recruitmentRepository{db: db}
}

func (r *recruitmentRepository) CreateVacancy(ctx context.Context, v *domain.JobVacancy) error {
	tenantctx.SetTenantID(ctx, &v.TenantID)
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *recruitmentRepository) GetVacancyByID(ctx context.Context, id uuid.UUID) (*domain.JobVacancy, error) {
	var v domain.JobVacancy
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&v).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *recruitmentRepository) ListVacancies(ctx context.Context, query types.PaginationQuery) ([]domain.JobVacancy, int64, error) {
	var vacancies []domain.JobVacancy
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.JobVacancy{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ? OR department ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&vacancies).Error
	return vacancies, total, err
}

func (r *recruitmentRepository) UpdateVacancy(ctx context.Context, v *domain.JobVacancy) error {
	return r.db.WithContext(ctx).Save(v).Error
}

func (r *recruitmentRepository) CountVacancies(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.JobVacancy{}).Count(&total).Error
	return total, err
}

func (r *recruitmentRepository) CreateCandidate(ctx context.Context, c *domain.Candidate) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *recruitmentRepository) GetCandidateByID(ctx context.Context, id uuid.UUID) (*domain.Candidate, error) {
	var c domain.Candidate
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *recruitmentRepository) ListCandidates(ctx context.Context, query types.PaginationQuery) ([]domain.Candidate, int64, error) {
	var candidates []domain.Candidate
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Candidate{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ? OR email ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&candidates).Error
	return candidates, total, err
}

func (r *recruitmentRepository) UpdateCandidate(ctx context.Context, c *domain.Candidate) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *recruitmentRepository) CountCandidates(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Candidate{}).Count(&total).Error
	return total, err
}

func (r *recruitmentRepository) CreateInterview(ctx context.Context, i *domain.Interview) error {
	tenantctx.SetTenantID(ctx, &i.TenantID)
	return r.db.WithContext(ctx).Create(i).Error
}

func (r *recruitmentRepository) GetInterviewByID(ctx context.Context, id uuid.UUID) (*domain.Interview, error) {
	var i domain.Interview
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&i).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &i, nil
}

func (r *recruitmentRepository) ListInterviews(ctx context.Context, query types.PaginationQuery) ([]domain.Interview, int64, error) {
	var interviews []domain.Interview
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Interview{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("interviewer_name ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&interviews).Error
	return interviews, total, err
}

func (r *recruitmentRepository) UpdateInterview(ctx context.Context, i *domain.Interview) error {
	return r.db.WithContext(ctx).Save(i).Error
}
