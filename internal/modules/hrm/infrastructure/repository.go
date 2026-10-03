package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type hrmRepository struct {
	db *gorm.DB
}

func NewHRMRepository(db *gorm.DB) domain.HRMRepository {
	return &hrmRepository{db: db}
}

func (r *hrmRepository) CreateEmployee(ctx context.Context, emp *domain.Employee) error {
	tenantctx.SetTenantID(ctx, &emp.TenantID)
	return r.db.WithContext(ctx).Create(emp).Error
}

func (r *hrmRepository) GetEmployeeByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	var emp domain.Employee
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&emp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &emp, nil
}

func (r *hrmRepository) UpdateEmployee(ctx context.Context, emp *domain.Employee) error {
	return r.db.WithContext(ctx).Save(emp).Error
}

func (r *hrmRepository) ListEmployees(ctx context.Context, query types.PaginationQuery) ([]domain.Employee, int64, error) {
	var employees []domain.Employee
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Employee{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ? OR n_ip ILIKE ? OR department ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&employees).Error
	return employees, total, err
}

func (r *hrmRepository) CountEmployees(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Employee{})).Count(&total).Error
	return total, err
}

func (r *hrmRepository) RecordAttendance(ctx context.Context, att *domain.Attendance) error {
	tenantctx.SetTenantID(ctx, &att.TenantID)
	return r.db.WithContext(ctx).Create(att).Error
}

func (r *hrmRepository) ListAttendance(ctx context.Context, query types.PaginationQuery) ([]domain.Attendance, int64, error) {
	var records []domain.Attendance
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Attendance{}))
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&records).Error
	return records, total, err
}

func (r *hrmRepository) FindAttendanceByNIPAndDate(ctx context.Context, nip, date string) (*domain.Attendance, error) {
	var found []domain.Attendance
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Attendance{}))
	if err := db.Where(&domain.Attendance{NIP: nip, Date: date}).Order("created_at desc").Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, nil
	}
	return &found[0], nil
}

func (r *hrmRepository) UpdateAttendance(ctx context.Context, att *domain.Attendance) error {
	return r.db.WithContext(ctx).Save(att).Error
}

func (r *hrmRepository) CreateAttendanceLocation(ctx context.Context, loc *domain.AttendanceLocation) error {
	tenantctx.SetTenantID(ctx, &loc.TenantID)
	return r.db.WithContext(ctx).Create(loc).Error
}

func (r *hrmRepository) ListAttendanceLocations(ctx context.Context, activeOnly bool) ([]domain.AttendanceLocation, error) {
	var locs []domain.AttendanceLocation
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.AttendanceLocation{}))
	if activeOnly {
		db = db.Where("is_active = ?", true)
	}
	err := db.Order("name asc").Find(&locs).Error
	return locs, err
}

func (r *hrmRepository) DeleteAttendanceLocation(ctx context.Context, id uuid.UUID) (bool, error) {
	res := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.AttendanceLocation{})).Where("id = ?", id).Delete(&domain.AttendanceLocation{})
	return res.RowsAffected > 0, res.Error
}

func (r *hrmRepository) CreateEmployeeIfEmailAbsent(ctx context.Context, emp *domain.Employee) (bool, error) {
	email := strings.ToLower(strings.TrimSpace(emp.Email))
	if email == "" {
		return false, fmt.Errorf("email is required")
	}
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialise concurrent onboarding of the same email (even across app instances).
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "employee-email:"+email).Error; err != nil {
			return err
		}
		var existing int64
		if err := tx.Model(&domain.Employee{}).Where("lower(email) = ?", email).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return nil
		}
		if emp.NIP == "" {
			// Different emails onboard concurrently and must not draw the same
			// number, so NIP assignment is serialised across the whole table. This
			// lock is always taken after the email lock, so they cannot deadlock.
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "employee-nip").Error; err != nil {
				return err
			}
			var total int64
			if err := tx.Unscoped().Model(&domain.Employee{}).Count(&total).Error; err != nil {
				return err
			}
			for n := total + 1; n <= total+10000; n++ {
				candidate := fmt.Sprintf("EMP-%03d", n)
				var taken int64
				if err := tx.Unscoped().Model(&domain.Employee{}).Where(&domain.Employee{NIP: candidate}).Count(&taken).Error; err != nil {
					return err
				}
				if taken == 0 {
					emp.NIP = candidate
					break
				}
			}
			if emp.NIP == "" {
				return fmt.Errorf("no free NIP found")
			}
		}
		tenantctx.SetTenantID(ctx, &emp.TenantID)
		if err := tx.Create(emp).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

func (r *hrmRepository) ListAttendanceByNIPAndPeriod(ctx context.Context, nip, period string) ([]domain.Attendance, error) {
	var records []domain.Attendance
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Attendance{})).Where("date LIKE ?", period+"%")
	if nip != "" {
		db = db.Where(&domain.Attendance{NIP: nip})
	}
	err := db.Order("date asc, created_at asc").Find(&records).Error
	return records, err
}

func (r *hrmRepository) FindEmployeeByEmail(ctx context.Context, email string) (*domain.Employee, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, nil
	}
	var found []domain.Employee
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Employee{}))
	if err := db.Where("LOWER(email) = ?", email).Order("created_at asc").Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, nil
	}
	return &found[0], nil
}

func (r *hrmRepository) DeleteEmployee(ctx context.Context, id uuid.UUID) (bool, error) {
	res := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Employee{})).Where("id = ?", id).Delete(&domain.Employee{})
	return res.RowsAffected > 0, res.Error
}
