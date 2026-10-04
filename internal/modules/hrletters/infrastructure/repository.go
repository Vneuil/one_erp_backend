package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/hrletters/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository { return &repository{db: db} }

func (r *repository) Create(ctx context.Context, l *domain.Letter) error {
	tenantctx.SetTenantID(ctx, &l.TenantID)
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *repository) Get(ctx context.Context, id uuid.UUID) (*domain.Letter, error) {
	var l domain.Letter
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Recipients").Where("id = ?", id).First(&l).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

func (r *repository) Update(ctx context.Context, l *domain.Letter, replaceRecipients bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		recipients := l.Recipients
		l.Recipients = nil
		defer func() { l.Recipients = recipients }()
		if err := tx.Save(l).Error; err != nil {
			return err
		}
		if !replaceRecipients {
			return nil
		}
		if err := tx.Where("letter_id = ?", l.ID).Delete(&domain.Recipient{}).Error; err != nil {
			return err
		}
		for i := range recipients {
			recipients[i].LetterID = l.ID
			if recipients[i].ID == uuid.Nil {
				recipients[i].ID = uuid.New()
			}
			if err := tx.Create(&recipients[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) List(ctx context.Context, f domain.Filter) ([]domain.Letter, error) {
	var out []domain.Letter
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Recipients")
	if f.Type != "" {
		db = db.Where("type = ?", f.Type)
	}
	if f.Status != "" {
		db = db.Where("status = ?", f.Status)
	}
	if f.From != "" {
		db = db.Where("date >= ?", f.From)
	}
	if f.To != "" {
		db = db.Where("date <= ?", f.To)
	}
	if f.EmployeeID != nil {
		db = db.Where("employee_id = ?", *f.EmployeeID)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := "%" + s + "%"
		db = db.Where("number ILIKE ? OR subject ILIKE ? OR employee_name ILIKE ? OR nip ILIKE ?", p, p, p, p)
	}
	err := db.Order("date desc, created_at desc").Limit(500).Find(&out).Error
	return out, err
}

func (r *repository) CountIssued(ctx context.Context, letterType string, year int) (int64, error) {
	var n int64
	start, end := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	// Unscoped: a cancelled (or deleted) letter keeps its number.
	err := r.db.WithContext(ctx).Unscoped().Model(&domain.Letter{}).
		Where("type = ? AND number <> '' AND date >= ? AND date < ?", letterType, start.Format("2006-01-02"), end.Format("2006-01-02")).Count(&n).Error
	return n, err
}

func (r *repository) ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]domain.Letter, error) {
	var out []domain.Letter
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("employee_id = ? AND status <> ?", employeeID, domain.StatusCancelled).
		Order("date desc, created_at desc").Find(&out).Error
	return out, err
}

func (r *repository) ListIssuedContracts(ctx context.Context) ([]domain.Letter, error) {
	var out []domain.Letter
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("type = ? AND status = ?", domain.TypeContract, domain.StatusIssued).
		Order("effective_date asc").Find(&out).Error
	return out, err
}
