package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	"github.com/google/uuid"
)

type CreatePayrollEntryDTO struct {
	Period string `json:"period"`
	// EmployeeID should reference a real employee (see modules/hrm). When
	// set, NIP/EmployeeName/Department are derived server-side, BaseSalary
	// is auto-filled from the employee's HRM record (still overridable by
	// passing a non-zero value), and DeductionCoop is auto-filled from the
	// sum of the employee's active cooperative loan MonthlyDeduction (see
	// modules/cooperative), also overridable.
	EmployeeID    *uuid.UUID `json:"employeeId,omitempty"`
	NIP           string     `json:"nip"`
	EmployeeName  string     `json:"employeeName"`
	Department    string     `json:"department"`
	BaseSalary    float64    `json:"baseSalary"`
	Allowance     float64    `json:"allowance"`
	OvertimePay   float64    `json:"overtimePay"`
	DeductionTax  float64    `json:"deductionTax"`
	DeductionCoop float64    `json:"deductionCoop"`
	PaymentBank   string     `json:"paymentBank"`
	BankAccount   string     `json:"bankAccount"`
	// TaxMethod is "manual" (default: use DeductionTax as given) or "auto"
	// (estimate PPh 21; DeductionTax is then ignored).
	TaxMethod string `json:"taxMethod"`
}

type CalculatePeriodDTO struct {
	Period string `json:"period"`
}

type UpdateStatusDTO struct {
	Status string `json:"status"`
}

type PayrollEntryResponseDTO struct {
	ID               uuid.UUID  `json:"id"`
	Period           string     `json:"period"`
	EmployeeID       *uuid.UUID `json:"employeeId,omitempty"`
	NIP              string     `json:"nip"`
	EmployeeName     string     `json:"employeeName"`
	Department       string     `json:"department"`
	BaseSalary       float64    `json:"baseSalary"`
	Allowance        float64    `json:"allowance"`
	OvertimePay      float64    `json:"overtimePay"`
	DeductionTax     float64    `json:"deductionTax"`
	DeductionCoop    float64    `json:"deductionCoop"`
	DeductionAbsence float64    `json:"deductionAbsence"`
	DeductionCanteen float64    `json:"deductionCanteen"`
	DeductionAdvance float64    `json:"deductionAdvance"`
	DeductionLate    float64    `json:"deductionLate"`
	UnpaidDays       int        `json:"unpaidDays"`
	LateCount        int        `json:"lateCount"`
	TakeHomePay      float64    `json:"takeHomePay"`
	Status           string     `json:"status"`
	PaymentBank      string     `json:"paymentBank"`
	BankAccount      string     `json:"bankAccount"`
	TaxMethod        string     `json:"taxMethod"`
	PTKPStatus       string     `json:"ptkpStatus,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

func ToPayrollEntryResponse(e *domain.PayrollEntry) *PayrollEntryResponseDTO {
	if e == nil {
		return nil
	}
	return &PayrollEntryResponseDTO{
		ID:               e.ID,
		Period:           e.Period,
		EmployeeID:       e.EmployeeID,
		NIP:              e.NIP,
		EmployeeName:     e.EmployeeName,
		Department:       e.Department,
		BaseSalary:       e.BaseSalary,
		Allowance:        e.Allowance,
		OvertimePay:      e.OvertimePay,
		DeductionTax:     e.DeductionTax,
		DeductionCoop:    e.DeductionCoop,
		DeductionAbsence: e.DeductionAbsence,
		DeductionCanteen: e.DeductionCanteen,
		DeductionAdvance: e.DeductionAdvance,
		DeductionLate:    e.DeductionLate,
		UnpaidDays:       e.UnpaidDays,
		LateCount:        e.LateCount,
		TakeHomePay:      e.TakeHomePay,
		Status:           e.Status,
		PaymentBank:      e.PaymentBank,
		BankAccount:      e.BankAccount,
		TaxMethod:        e.TaxMethod,
		PTKPStatus:       e.PTKPStatus,
		CreatedAt:        e.CreatedAt,
	}
}

func ToPayrollEntryResponseList(entries []domain.PayrollEntry) []PayrollEntryResponseDTO {
	result := make([]PayrollEntryResponseDTO, len(entries))
	for i, e := range entries {
		result[i] = *ToPayrollEntryResponse(&e)
	}
	return result
}

type UpdatePolicyDTO struct {
	WorkDaysPerMonth       int     `json:"workDaysPerMonth"`
	LatePenaltyPerIncident float64 `json:"latePenaltyPerIncident"`
	DeductUnpaidLeave      bool    `json:"deductUnpaidLeave"`
}
