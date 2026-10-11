package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/fingerprint/domain"
	hrmDomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) domain.Repository { return &repository{db: db} }

func (r *repository) CreateDevice(ctx context.Context, d *domain.Device) error {
	tenantctx.SetTenantID(ctx, &d.TenantID)
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *repository) GetDevice(ctx context.Context, serial string) (*domain.Device, error) {
	var d domain.Device
	if err := r.db.WithContext(ctx).Where("serial = ?", serial).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *repository) ListDevices(ctx context.Context) ([]domain.Device, error) {
	var out []domain.Device
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Order("name asc").Find(&out).Error
	return out, err
}

func (r *repository) SaveDevice(ctx context.Context, d *domain.Device) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *repository) DeleteDevice(ctx context.Context, serial string) error {
	return r.db.WithContext(ctx).Unscoped().Where("serial = ?", serial).Delete(&domain.Device{}).Error
}

func (r *repository) TouchDevice(ctx context.Context, serial, ip string, lastPunch *time.Time) error {
	now := time.Now()
	upd := map[string]any{"last_seen_at": now}
	if ip != "" {
		upd["last_ip"] = ip
	}
	if lastPunch != nil {
		upd["last_punch_at"] = *lastPunch
	}
	return r.db.WithContext(ctx).Model(&domain.Device{}).Where("serial = ?", serial).Updates(upd).Error
}

func (r *repository) SavePinMap(ctx context.Context, m *domain.PinMap) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *repository) GetPinMap(ctx context.Context, pin string) (*domain.PinMap, error) {
	var m domain.PinMap
	if err := r.db.WithContext(ctx).Where("pin = ?", pin).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *repository) ListPinMaps(ctx context.Context) ([]domain.PinMap, error) {
	var out []domain.PinMap
	err := r.db.WithContext(ctx).Order("pin asc").Find(&out).Error
	return out, err
}

func (r *repository) DeletePinMap(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&domain.PinMap{}).Error
}

func (r *repository) InsertPunches(ctx context.Context, ps []domain.Punch) (int, error) {
	if len(ps) == 0 {
		return 0, nil
	}
	stored := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range ps {
			res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ps[i])
			if res.Error != nil {
				return res.Error
			}
			stored += int(res.RowsAffected)
		}
		return nil
	})
	return stored, err
}

func (r *repository) ListPunches(ctx context.Context, unresolvedOnly bool, from, to string, limit int) ([]domain.Punch, error) {
	var out []domain.Punch
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	if unresolvedOnly {
		db = db.Where("resolved = ?", false)
	}
	if from != "" {
		db = db.Where("date >= ?", from)
	}
	if to != "" {
		db = db.Where("date <= ?", to)
	}
	err := db.Order("punched_at desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *repository) PunchesForDay(ctx context.Context, nip, date string) ([]domain.Punch, error) {
	var out []domain.Punch
	err := r.db.WithContext(ctx).Where("nip = ? AND date = ?", nip, date).Order("punched_at asc").Find(&out).Error
	return out, err
}

func (r *repository) UnresolvedPunches(ctx context.Context) ([]domain.Punch, error) {
	var out []domain.Punch
	err := r.db.WithContext(ctx).Where("resolved = ?", false).Order("punched_at asc").Find(&out).Error
	return out, err
}

func (r *repository) UpdatePunch(ctx context.Context, p *domain.Punch) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *repository) EmployeeName(ctx context.Context, nip string) (string, bool, error) {
	var found []hrmDomain.Employee
	if err := r.db.WithContext(ctx).Where(&hrmDomain.Employee{NIP: nip}).Limit(1).Find(&found).Error; err != nil {
		return "", false, err
	}
	if len(found) == 0 {
		return "", false, nil
	}
	return found[0].Name, true, nil
}

func (r *repository) FindAttendance(ctx context.Context, nip, date string) (*hrmDomain.Attendance, error) {
	var found []hrmDomain.Attendance
	if err := r.db.WithContext(ctx).Where(&hrmDomain.Attendance{NIP: nip, Date: date}).Order("created_at desc").Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, nil
	}
	return &found[0], nil
}

func (r *repository) SaveAttendance(ctx context.Context, a *hrmDomain.Attendance) error {
	if a.ID == uuid.Nil {
		tenantctx.SetTenantID(ctx, &a.TenantID)
		return r.db.WithContext(ctx).Create(a).Error
	}
	return r.db.WithContext(ctx).Save(a).Error
}

// registry (control-plane database)

type registryRepository struct{ db *gorm.DB }

func NewRegistryRepository(db *gorm.DB) domain.RegistryRepository { return &registryRepository{db: db} }

func (r *registryRepository) Get(ctx context.Context, serial string) (*domain.Registry, error) {
	var reg domain.Registry
	if err := r.db.WithContext(ctx).Where("serial = ?", serial).First(&reg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reg, nil
}

func (r *registryRepository) Save(ctx context.Context, reg *domain.Registry) error {
	return r.db.WithContext(ctx).Save(reg).Error
}

func (r *registryRepository) Delete(ctx context.Context, serial string) error {
	return r.db.WithContext(ctx).Where("serial = ?", serial).Delete(&domain.Registry{}).Error
}
