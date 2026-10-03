package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/device/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type deviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) domain.DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) CreateDevice(ctx context.Context, d *domain.Device) error {
	tenantctx.SetTenantID(ctx, &d.TenantID)
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *deviceRepository) GetDeviceByID(ctx context.Context, id uuid.UUID) (*domain.Device, error) {
	var d domain.Device
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *deviceRepository) GetDeviceBySerial(ctx context.Context, serial string) (*domain.Device, error) {
	var d domain.Device
	err := r.db.WithContext(ctx).Where("device_serial = ?", serial).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *deviceRepository) ListDevices(ctx context.Context, query types.PaginationQuery) ([]domain.Device, int64, error) {
	var devices []domain.Device
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Device{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("device_serial ILIKE ? OR name ILIKE ? OR location ILIKE ?", pattern, pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&devices).Error
	return devices, total, err
}

func (r *deviceRepository) UpdateDevice(ctx context.Context, d *domain.Device) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *deviceRepository) CountDevices(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Device{}).Count(&total).Error
	return total, err
}
