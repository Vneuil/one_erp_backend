package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/cooperative/domain"
	"github.com/google/uuid"
)

type CreateLoanDTO struct {
	// EmployeeID should reference a real employee (see modules/hrm). When
	// set, EmployeeName/Department are derived server-side from that
	// employee record rather than trusted from the client.
	EmployeeID       *uuid.UUID `json:"employeeId,omitempty"`
	EmployeeName     string     `json:"employeeName"`
	Department       string     `json:"department"`
	TotalLoan        float64    `json:"totalLoan"`
	MonthlyDeduction float64    `json:"monthlyDeduction"`
	TenureMonths     int        `json:"tenureMonths"`
}

type LoanResponseDTO struct {
	ID               uuid.UUID  `json:"id"`
	LoanNo           string     `json:"loanNo"`
	EmployeeID       *uuid.UUID `json:"employeeId,omitempty"`
	EmployeeName     string     `json:"employeeName"`
	Department       string     `json:"department"`
	TotalLoan        float64    `json:"totalLoan"`
	MonthlyDeduction float64    `json:"monthlyDeduction"`
	RemainingBalance float64    `json:"remainingBalance"`
	TenureMonths     int        `json:"tenureMonths"`
	MonthsPaid       int        `json:"monthsPaid"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
}

func ToLoanResponse(l *domain.CooperativeLoan) *LoanResponseDTO {
	if l == nil {
		return nil
	}
	return &LoanResponseDTO{
		ID:               l.ID,
		LoanNo:           l.LoanNo,
		EmployeeID:       l.EmployeeID,
		EmployeeName:     l.EmployeeName,
		Department:       l.Department,
		TotalLoan:        l.TotalLoan,
		MonthlyDeduction: l.MonthlyDeduction,
		RemainingBalance: l.RemainingBalance,
		TenureMonths:     l.TenureMonths,
		MonthsPaid:       l.MonthsPaid,
		Status:           l.Status,
		CreatedAt:        l.CreatedAt,
	}
}

func ToLoanResponseList(loans []domain.CooperativeLoan) []LoanResponseDTO {
	result := make([]LoanResponseDTO, len(loans))
	for i, l := range loans {
		result[i] = *ToLoanResponse(&l)
	}
	return result
}
