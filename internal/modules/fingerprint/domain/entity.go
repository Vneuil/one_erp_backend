package domain

import (
	"context"
	"time"

	hrmDomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

const (
	ProtocolADMS    = "adms"    // ZKTeco-style "iclock" push (also used by many compatible machines)
	ProtocolGeneric = "generic" // JSON push with a device token, for bridges and custom clients
)

// Device is an attendance machine registered in a company's database.
type Device struct {
	types.BaseEntity
	CompanyID   *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Serial      string     `gorm:"type:varchar(100);not null;uniqueIndex" json:"serial"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`
	Location    string     `gorm:"type:varchar(255)" json:"location"`
	Vendor      string     `gorm:"type:varchar(100)" json:"vendor"`
	Protocol    string     `gorm:"type:varchar(20);not null;default:'adms'" json:"protocol"`
	Active      bool       `gorm:"not null;default:true" json:"active"`
	AllowedIP   string     `gorm:"type:varchar(100)" json:"allowedIp"`
	LastSeenAt  *time.Time `json:"lastSeenAt,omitempty"`
	LastPunchAt *time.Time `json:"lastPunchAt,omitempty"`
	LastIP      string     `gorm:"type:varchar(60)" json:"lastIp"`
}

func (Device) TableName() string { return "fp_devices" }

// PinMap links the PIN enrolled on the machines to an employee NIP.
type PinMap struct {
	types.BaseEntity
	PIN          string `gorm:"column:pin;type:varchar(50);not null;uniqueIndex" json:"pin"`
	NIP          string `gorm:"column:nip;type:varchar(50);not null;index" json:"nip"`
	EmployeeName string `gorm:"type:varchar(255)" json:"employeeName"`
}

func (PinMap) TableName() string { return "fp_pin_maps" }

// Punch is one raw fingerprint event. (device, pin, time) is unique, so a
// machine re-sending its log never double counts.
type Punch struct {
	types.BaseEntity
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DeviceSerial string     `gorm:"type:varchar(100);not null;uniqueIndex:idx_fp_punch" json:"deviceSerial"`
	PIN          string     `gorm:"column:pin;type:varchar(50);not null;uniqueIndex:idx_fp_punch;index" json:"pin"`
	PunchedAt    time.Time  `gorm:"not null;uniqueIndex:idx_fp_punch" json:"punchedAt"`
	Date         string     `gorm:"type:varchar(10);not null;index" json:"date"` // local (WIB) date
	Verify       int        `json:"verify"`
	State        int        `json:"state"`
	NIP          string     `gorm:"column:nip;type:varchar(50);index" json:"nip"`
	EmployeeName string     `gorm:"type:varchar(255)" json:"employeeName"`
	Resolved     bool       `gorm:"not null;default:false;index" json:"resolved"`
}

func (Punch) TableName() string { return "fp_punches" }

// Registry lives in the main (control-plane) database: machines call in without
// a login, so the serial must be resolvable to a company before any tenant
// database is opened.
type Registry struct {
	Serial    string     `gorm:"primaryKey;type:varchar(100)" json:"serial"`
	CompanyID uuid.UUID  `gorm:"type:uuid;not null;index" json:"companyId"`
	TenantID  *uuid.UUID `gorm:"type:uuid" json:"tenantId,omitempty"`
	TokenHash string     `gorm:"type:varchar(64);not null" json:"-"`
	AllowedIP string     `gorm:"type:varchar(100)" json:"allowedIp"`
	Active    bool       `gorm:"not null;default:true" json:"active"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (Registry) TableName() string { return "fp_registry" }

type RegistryRepository interface {
	Get(ctx context.Context, serial string) (*Registry, error)
	Save(ctx context.Context, r *Registry) error
	Delete(ctx context.Context, serial string) error
}

type Repository interface {
	CreateDevice(ctx context.Context, d *Device) error
	GetDevice(ctx context.Context, serial string) (*Device, error)
	ListDevices(ctx context.Context) ([]Device, error)
	SaveDevice(ctx context.Context, d *Device) error
	DeleteDevice(ctx context.Context, serial string) error
	TouchDevice(ctx context.Context, serial, ip string, lastPunch *time.Time) error

	SavePinMap(ctx context.Context, m *PinMap) error
	GetPinMap(ctx context.Context, pin string) (*PinMap, error)
	ListPinMaps(ctx context.Context) ([]PinMap, error)
	DeletePinMap(ctx context.Context, id uuid.UUID) error

	// InsertPunches stores new punches and skips ones already known; it returns how many were new.
	InsertPunches(ctx context.Context, ps []Punch) (int, error)
	ListPunches(ctx context.Context, unresolvedOnly bool, from, to string, limit int) ([]Punch, error)
	PunchesForDay(ctx context.Context, nip, date string) ([]Punch, error)
	UnresolvedPunches(ctx context.Context) ([]Punch, error)
	UpdatePunch(ctx context.Context, p *Punch) error

	EmployeeName(ctx context.Context, nip string) (string, bool, error)
	FindAttendance(ctx context.Context, nip, date string) (*hrmDomain.Attendance, error)
	SaveAttendance(ctx context.Context, a *hrmDomain.Attendance) error
}
