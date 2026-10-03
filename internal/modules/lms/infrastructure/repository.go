package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/lms/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type lmsRepository struct {
	db *gorm.DB
}

func NewLMSRepository(db *gorm.DB) domain.LMSRepository {
	return &lmsRepository{db: db}
}

func (r *lmsRepository) CreateCourse(ctx context.Context, c *domain.Course) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *lmsRepository) GetCourseByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	var c domain.Course
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *lmsRepository) ListCourses(ctx context.Context, query types.PaginationQuery) ([]domain.Course, int64, error) {
	var courses []domain.Course
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Course{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ? OR category ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&courses).Error
	return courses, total, err
}

func (r *lmsRepository) UpdateCourse(ctx context.Context, c *domain.Course) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *lmsRepository) CountCourses(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Course{})).Count(&total).Error
	return total, err
}

func (r *lmsRepository) CreateEnrollment(ctx context.Context, e *domain.Enrollment) error {
	tenantctx.SetTenantID(ctx, &e.TenantID)
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *lmsRepository) GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	var e domain.Enrollment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *lmsRepository) ListEnrollments(ctx context.Context, query types.PaginationQuery) ([]domain.Enrollment, int64, error) {
	var enrollments []domain.Enrollment
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Enrollment{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("employee_name ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&enrollments).Error
	return enrollments, total, err
}

func (r *lmsRepository) UpdateEnrollment(ctx context.Context, e *domain.Enrollment) error {
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *lmsRepository) ListEnrollmentsByEmployeeID(ctx context.Context, employeeID uuid.UUID) ([]domain.Enrollment, error) {
	var enrollments []domain.Enrollment
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Enrollment{}))
	err := db.Where("employee_id = ?", employeeID).Order("created_at desc").Find(&enrollments).Error
	return enrollments, err
}

func (r *lmsRepository) CreateQuiz(ctx context.Context, q *domain.Quiz) error {
	tenantctx.SetTenantID(ctx, &q.TenantID)
	return r.db.WithContext(ctx).Create(q).Error
}

func (r *lmsRepository) GetQuizByID(ctx context.Context, id uuid.UUID) (*domain.Quiz, error) {
	var q domain.Quiz
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&q).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &q, nil
}

func (r *lmsRepository) ListQuizzes(ctx context.Context, query types.PaginationQuery) ([]domain.Quiz, int64, error) {
	var quizzes []domain.Quiz
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Quiz{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&quizzes).Error
	return quizzes, total, err
}

func (r *lmsRepository) CreateQuizAttempt(ctx context.Context, a *domain.QuizAttempt) error {
	tenantctx.SetTenantID(ctx, &a.TenantID)
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *lmsRepository) CreateCertificate(ctx context.Context, cert *domain.Certificate) error {
	tenantctx.SetTenantID(ctx, &cert.TenantID)
	return r.db.WithContext(ctx).Create(cert).Error
}

func (r *lmsRepository) ListCertificates(ctx context.Context, query types.PaginationQuery) ([]domain.Certificate, int64, error) {
	var certs []domain.Certificate
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Certificate{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("certificate_number ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&certs).Error
	return certs, total, err
}

func (r *lmsRepository) CountCertificates(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Certificate{})).Count(&total).Error
	return total, err
}
