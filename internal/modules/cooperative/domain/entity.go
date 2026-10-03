package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// CooperativeLoan is an employee cooperative loan (koperasi) with a fixed
// tenure and monthly deduction schedule.
type CooperativeLoan struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this loan to one business unit within the company
	// (see modules/workspace). Nil means the company's default tenant.
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	LoanNo   string     `gorm:"type:varchar(50);not null;index" json:"loanNo"`
	// EmployeeID links this loan to a real HRM employee record (see
	// modules/hrm), so payroll can look up an employee's active loans when
	// calculating deductions.
	EmployeeID       *uuid.UUID `gorm:"type:uuid;index" json:"employeeId,omitempty"`
	EmployeeName     string     `gorm:"type:varchar(255);not null" json:"employeeName"`
	Department       string     `gorm:"type:varchar(100)" json:"department"`
	TotalLoan        float64    `gorm:"type:decimal(15,2);not null" json:"totalLoan"`
	MonthlyDeduction float64    `gorm:"type:decimal(15,2);not null" json:"monthlyDeduction"`
	RemainingBalance float64    `gorm:"type:decimal(15,2);not null" json:"remainingBalance"`
	TenureMonths     int        `gorm:"not null" json:"tenureMonths"`
	MonthsPaid       int        `gorm:"default:0" json:"monthsPaid"`
	Status           string     `gorm:"type:varchar(50);default:'active'" json:"status"` // active | completed | pending_approval
}

func (CooperativeLoan) TableName() string {
	return "cooperative_loans"
}

type CooperativeRepository interface {
	CreateLoan(ctx context.Context, loan *CooperativeLoan) error
	GetLoanByID(ctx context.Context, id uuid.UUID) (*CooperativeLoan, error)
	UpdateLoan(ctx context.Context, loan *CooperativeLoan) error
	ListLoans(ctx context.Context, query types.PaginationQuery) ([]CooperativeLoan, int64, error)
	CountLoans(ctx context.Context) (int64, error)
	// ListActiveLoansByEmployeeID returns the employee's active cooperative
	// loans, used by payroll to auto-populate DeductionCoop (see
	// modules/payroll).
	ListActiveLoansByEmployeeID(ctx context.Context, employeeID uuid.UUID) ([]CooperativeLoan, error)
}
