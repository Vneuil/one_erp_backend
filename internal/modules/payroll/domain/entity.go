package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type PayrollEntry struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this payroll entry to one business unit within the
	// company (see modules/workspace). Nil means the company's default tenant.
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Period   string     `gorm:"type:varchar(20);not null;index" json:"period"`
	// EmployeeID links this entry to a real HRM employee record (see
	// modules/hrm), so BaseSalary and DeductionCoop can be sourced from
	// real HRM/Cooperative data instead of manual entry.
	EmployeeID    *uuid.UUID `gorm:"type:uuid;index" json:"employeeId,omitempty"`
	NIP           string     `gorm:"type:varchar(50);not null;index" json:"nip"`
	EmployeeName  string     `gorm:"type:varchar(255);not null" json:"employeeName"`
	Department    string     `gorm:"type:varchar(100)" json:"department"`
	BaseSalary    float64    `gorm:"type:decimal(15,2);default:0" json:"baseSalary"`
	Allowance     float64    `gorm:"type:decimal(15,2);default:0" json:"allowance"`
	OvertimePay   float64    `gorm:"type:decimal(15,2);default:0" json:"overtimePay"`
	DeductionTax  float64    `gorm:"type:decimal(15,2);default:0" json:"deductionTax"`
	DeductionCoop float64    `gorm:"type:decimal(15,2);default:0" json:"deductionCoop"`
	TakeHomePay   float64    `gorm:"type:decimal(15,2);default:0" json:"takeHomePay"`
	Status        string     `gorm:"type:varchar(50);default:'draft'" json:"status"`
	PaymentBank   string     `gorm:"type:varchar(100)" json:"paymentBank"`
	BankAccount   string     `gorm:"type:varchar(100)" json:"bankAccount"`
	// TaxMethod is "manual" (DeductionTax is entered by hand, the original
	// behaviour) or "auto" (PPh 21 is estimated from gross income and the
	// employee's PTKP status, and refreshed when the period is calculated).
	TaxMethod string `gorm:"type:varchar(10);not null;default:'manual'" json:"taxMethod"`
	// PTKPStatus is the tax status used for the last automatic estimate.
	PTKPStatus string `gorm:"type:varchar(10)" json:"ptkpStatus,omitempty"`

	// DeductionAbsence is the pay withheld for unpaid leave days and
	// DeductionLate the penalty for late clock-ins, both derived from
	// leave/attendance records when the period is calculated (see PayrollPolicy).
	DeductionAbsence float64 `gorm:"type:decimal(15,2);default:0" json:"deductionAbsence"`
	DeductionLate    float64 `gorm:"type:decimal(15,2);default:0" json:"deductionLate"`
	// DeductionCanteen is the employee's meal orders for the month, withheld from pay.
	DeductionCanteen float64 `gorm:"type:decimal(15,2);default:0" json:"deductionCanteen"`
	// DeductionAdvance is this month's installment of the employee's cash advance (kasbon).
	DeductionAdvance float64 `gorm:"type:decimal(15,2);default:0" json:"deductionAdvance"`
	UnpaidDays       int     `gorm:"default:0" json:"unpaidDays"`
	LateCount        int     `gorm:"default:0" json:"lateCount"`
}

// PayrollPolicy holds the company's payroll rules. There is at most one row;
// when none exists the defaults from DefaultPayrollPolicy apply.
type PayrollPolicy struct {
	types.BaseEntity
	// WorkDaysPerMonth is the divisor turning monthly salary into a daily rate.
	WorkDaysPerMonth int `gorm:"not null;default:22" json:"workDaysPerMonth"`
	// LatePenaltyPerIncident is deducted for each "Late" attendance record.
	LatePenaltyPerIncident float64 `gorm:"type:decimal(15,2);default:0" json:"latePenaltyPerIncident"`
	// DeductUnpaidLeave withholds a day's pay for each weekday of approved unpaid leave.
	DeductUnpaidLeave bool `gorm:"default:true" json:"deductUnpaidLeave"`
}

func (PayrollPolicy) TableName() string { return "payroll_policies" }

// DefaultPayrollPolicy is used until a company saves its own.
func DefaultPayrollPolicy() PayrollPolicy {
	return PayrollPolicy{WorkDaysPerMonth: 22, DeductUnpaidLeave: true}
}

func (PayrollEntry) TableName() string {
	return "payroll_entries"
}

type PayrollRepository interface {
	CreateEntry(ctx context.Context, entry *PayrollEntry) error
	GetEntryByID(ctx context.Context, id uuid.UUID) (*PayrollEntry, error)
	ListEntries(ctx context.Context, query types.PaginationQuery, period string) ([]PayrollEntry, int64, error)
	ListEntriesByEmployee(ctx context.Context, employeeID uuid.UUID) ([]PayrollEntry, error)
	ListEntriesByPeriodAndStatus(ctx context.Context, period string, status string) ([]PayrollEntry, error)
	UpdateEntry(ctx context.Context, entry *PayrollEntry) error
	CountEntries(ctx context.Context) (int64, error)
	// GetPolicy returns the saved policy, or the defaults when none is saved.
	GetPolicy(ctx context.Context) (*PayrollPolicy, error)
	SavePolicy(ctx context.Context, p *PayrollPolicy) error
}
