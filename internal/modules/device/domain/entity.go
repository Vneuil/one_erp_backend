package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Device represents a business terminal such as an attendance terminal, NFC reader, or POS device
type Device struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this device to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant.
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DeviceSerial  string     `gorm:"type:varchar(100);not null;uniqueIndex" json:"deviceSerial"`
	Name          string     `gorm:"type:varchar(255);not null" json:"name"`
	DeviceType    string     `gorm:"type:varchar(50);not null" json:"deviceType"`
	Location      string     `gorm:"type:varchar(255)" json:"location"`
	IPAddress     string     `gorm:"type:varchar(50)" json:"ipAddress"`
	LastHeartbeat *time.Time `gorm:"type:timestamptz" json:"lastHeartbeat,omitempty"`
	Status        string     `gorm:"type:varchar(20);not null;default:'offline'" json:"status"`
}

func (Device) TableName() string {
	return "device_devices"
}

type DeviceRepository interface {
	CreateDevice(ctx context.Context, d *Device) error
	GetDeviceByID(ctx context.Context, id uuid.UUID) (*Device, error)
	GetDeviceBySerial(ctx context.Context, serial string) (*Device, error)
	ListDevices(ctx context.Context, query types.PaginationQuery) ([]Device, int64, error)
	UpdateDevice(ctx context.Context, d *Device) error
	CountDevices(ctx context.Context) (int64, error)
}
