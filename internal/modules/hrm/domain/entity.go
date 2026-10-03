package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type Employee struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this employee to one business unit within the company
	// (see modules/workspace). Nil means the company's default tenant.
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	NIP          string     `gorm:"type:varchar(50);not null;index" json:"nip"`
	Name         string     `gorm:"type:varchar(255);not null" json:"name"`
	Email        string     `gorm:"type:varchar(255)" json:"email"`
	Phone        string     `gorm:"type:varchar(50)" json:"phone"`
	Department   string     `gorm:"type:varchar(100);not null" json:"department"`
	Role         string     `gorm:"type:varchar(100);not null" json:"role"`
	ContractType string     `gorm:"type:varchar(50);default:'PKWTT Tetap'" json:"contractType"`
	JoinDate     string     `gorm:"type:varchar(50)" json:"joinDate"`
	Status       string     `gorm:"type:varchar(50);default:'Active'" json:"status"`
	BaseSalary   float64    `gorm:"type:decimal(15,2);default:0" json:"baseSalary"`
	// LeaveQuotaDays is the employee's annual leave allowance (default 12
	// days/year). LeaveUsedDays tracks how many of those days have already
	// been consumed by approved leave requests (see modules/leave).
	LeaveQuotaDays int `gorm:"not null;default:12" json:"leaveQuotaDays"`
	LeaveUsedDays  int `gorm:"not null;default:0" json:"leaveUsedDays"`

	// PTKPStatus is the employee's tax status for PPh 21 (TK/0..TK/3, K/0..K/3).
	// NoNPWP is stored inverted so the zero value means "has an NPWP", the common case.
	PTKPStatus string `gorm:"type:varchar(10);not null;default:'TK/0'" json:"ptkpStatus"`
	NoNPWP     bool   `gorm:"not null;default:false" json:"noNpwp"`

	// ManagerID is the employee's direct supervisor (the reporting line).
	ManagerID *uuid.UUID `gorm:"type:uuid;index" json:"managerId,omitempty"`
}

func (Employee) TableName() string {
	return "employees"
}

type Attendance struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this attendance record to one business unit within
	// the company (see modules/workspace). Nil means the default tenant.
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	EmployeeName string     `gorm:"type:varchar(255);not null" json:"employeeName"`
	NIP          string     `gorm:"type:varchar(50);not null" json:"nip"`
	Date         string     `gorm:"type:varchar(50);not null" json:"date"`
	ClockIn      string     `gorm:"type:varchar(50)" json:"clockIn"`
	ClockOut     string     `gorm:"type:varchar(50)" json:"clockOut"`
	Location     string     `gorm:"type:varchar(255)" json:"location"`
	Method       string     `gorm:"type:varchar(50);default:'GPS Mobile'" json:"method"`
	Status       string     `gorm:"type:varchar(50);default:'On Time'" json:"status"`

	// ClockInAt / ClockOutAt are the exact instants behind the display
	// strings above; WorkMinutes is set when the employee clocks out.
	ClockInAt   *time.Time `json:"clockInAt,omitempty"`
	ClockOutAt  *time.Time `json:"clockOutAt,omitempty"`
	WorkMinutes int        `gorm:"default:0" json:"workMinutes"`
	// Latitude/Longitude are the device coordinates at clock-in; LocationID is
	// the configured AttendanceLocation the coordinates matched, if any.
	Latitude   *float64   `json:"latitude,omitempty"`
	Longitude  *float64   `json:"longitude,omitempty"`
	LocationID *uuid.UUID `gorm:"type:uuid;index" json:"locationId,omitempty"`
	// PhotoRef references a verification photo (URL or file key) supplied by the client.
	PhotoRef string `gorm:"type:varchar(500)" json:"photoRef,omitempty"`
}

func (Attendance) TableName() string {
	return "attendances"
}

// AttendanceLocation is an allowed place to clock in: a centre point and a
// radius (geofence). When at least one active location exists, clock-in must
// fall inside one of them.
type AttendanceLocation struct {
	types.BaseEntity
	CompanyID    *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name         string     `gorm:"type:varchar(255);not null" json:"name"`
	Latitude     float64    `gorm:"not null" json:"latitude"`
	Longitude    float64    `gorm:"not null" json:"longitude"`
	RadiusMeters int        `gorm:"not null;default:100" json:"radiusMeters"`
	IsActive     bool       `gorm:"not null;default:true" json:"isActive"`
}

func (AttendanceLocation) TableName() string {
	return "attendance_locations"
}

type HRMRepository interface {
	CreateEmployee(ctx context.Context, emp *Employee) error
	GetEmployeeByID(ctx context.Context, id uuid.UUID) (*Employee, error)
	UpdateEmployee(ctx context.Context, emp *Employee) error
	// CreateEmployeeIfEmailAbsent creates emp unless an employee with the same
	// email (case-insensitive) already exists, and reports whether it created one.
	// When emp.NIP is empty the next free "EMP-NNN" is assigned. Safe against
	// concurrent callers for the same email.
	// FindEmployeeByEmail matches case-insensitively; nil when there is none.
	FindEmployeeByEmail(ctx context.Context, email string) (*Employee, error)
	DeleteEmployee(ctx context.Context, id uuid.UUID) (bool, error)
	CreateEmployeeIfEmailAbsent(ctx context.Context, emp *Employee) (bool, error)
	ListEmployees(ctx context.Context, query types.PaginationQuery) ([]Employee, int64, error)
	CountEmployees(ctx context.Context) (int64, error)

	RecordAttendance(ctx context.Context, att *Attendance) error
	ListAttendance(ctx context.Context, query types.PaginationQuery) ([]Attendance, int64, error)
	// FindAttendanceByNIPAndDate returns the employee's latest record for the
	// given date (YYYY-MM-DD), or nil when there is none.
	FindAttendanceByNIPAndDate(ctx context.Context, nip, date string) (*Attendance, error)
	UpdateAttendance(ctx context.Context, att *Attendance) error
	// ListAttendanceByNIPAndPeriod returns one employee's records whose date
	// starts with period (YYYY-MM). An empty nip returns everyone's.
	ListAttendanceByNIPAndPeriod(ctx context.Context, nip, period string) ([]Attendance, error)

	CreateAttendanceLocation(ctx context.Context, loc *AttendanceLocation) error
	ListAttendanceLocations(ctx context.Context, activeOnly bool) ([]AttendanceLocation, error)
	DeleteAttendanceLocation(ctx context.Context, id uuid.UUID) (bool, error)
}
