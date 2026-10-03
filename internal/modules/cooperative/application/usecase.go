package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/cooperative/domain"
	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type CooperativeUseCase interface {
	CreateLoan(ctx context.Context, dto CreateLoanDTO) (*LoanResponseDTO, error)
	ListLoans(ctx context.Context, query types.PaginationQuery) ([]LoanResponseDTO, types.PaginationMeta, error)
	RecordPayment(ctx context.Context, id uuid.UUID) (*LoanResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type cooperativeUseCase struct {
	repo    domain.CooperativeRepository
	hrmRepo hrmdomain.HRMRepository
}

// NewCooperativeUseCase wires the cooperative repository together with the
// HRM repository, so loans can be linked to real employee records
// (EmployeeID) instead of free-typed names. hrmRepo may be nil, in which
// case loans fall back to the free-typed EmployeeName/Department fields.
func NewCooperativeUseCase(repo domain.CooperativeRepository, hrmRepo hrmdomain.HRMRepository) CooperativeUseCase {
	return &cooperativeUseCase{repo: repo, hrmRepo: hrmRepo}
}

// generateLoanNo builds a sequential, timestamp-seeded loan number of the
// form "KOP-<year>-<0001>". The sequence is derived from the current total
// loan count (+1), scoped to the caller's tenant DB, so numbers restart per
// tenant/company and stay roughly sequential even though this is not
// strictly race-safe under high concurrency.
func (uc *cooperativeUseCase) generateLoanNo(ctx context.Context) (string, error) {
	count, err := uc.repo.CountLoans(ctx)
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("KOP-%d-%04d", year, count+1), nil
}

func (uc *cooperativeUseCase) CreateLoan(ctx context.Context, dto CreateLoanDTO) (*LoanResponseDTO, error) {
	name := strings.TrimSpace(dto.EmployeeName)
	department := dto.Department

	if dto.EmployeeID != nil && uc.hrmRepo != nil {
		emp, err := uc.hrmRepo.GetEmployeeByID(ctx, *dto.EmployeeID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get employee")
		}
		if emp == nil {
			return nil, apperrors.NewNotFound("Employee not found")
		}
		// Name/Department are never trusted from the client when
		// EmployeeID is provided - always derived from the real record.
		name = emp.Name
		department = emp.Department
	}

	if name == "" {
		return nil, apperrors.NewBadRequest("Employee is required")
	}
	if dto.TotalLoan <= 0 {
		return nil, apperrors.NewBadRequest("Total Loan must be greater than zero")
	}
	if dto.TenureMonths <= 0 {
		return nil, apperrors.NewBadRequest("Tenure Months must be greater than zero")
	}

	monthlyDeduction := dto.MonthlyDeduction
	if monthlyDeduction <= 0 {
		monthlyDeduction = dto.TotalLoan / float64(dto.TenureMonths)
	}

	loanNo, err := uc.generateLoanNo(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate loan number")
	}

	loan := &domain.CooperativeLoan{
		LoanNo:           loanNo,
		EmployeeID:       dto.EmployeeID,
		EmployeeName:     name,
		Department:       department,
		TotalLoan:        dto.TotalLoan,
		MonthlyDeduction: monthlyDeduction,
		RemainingBalance: dto.TotalLoan,
		TenureMonths:     dto.TenureMonths,
		MonthsPaid:       0,
		Status:           "active",
	}

	if err := uc.repo.CreateLoan(ctx, loan); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create cooperative loan")
	}

	return ToLoanResponse(loan), nil
}

func (uc *cooperativeUseCase) ListLoans(ctx context.Context, query types.PaginationQuery) ([]LoanResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	loans, total, err := uc.repo.ListLoans(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list cooperative loans")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToLoanResponseList(loans), meta, nil
}

func (uc *cooperativeUseCase) RecordPayment(ctx context.Context, id uuid.UUID) (*LoanResponseDTO, error) {
	loan, err := uc.repo.GetLoanByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get loan")
	}
	if loan == nil {
		return nil, apperrors.NewNotFound("Cooperative loan not found")
	}

	if loan.Status != "active" {
		return nil, apperrors.NewBadRequest("Only active loans can receive a payment")
	}

	loan.MonthsPaid++
	loan.RemainingBalance -= loan.MonthlyDeduction
	if loan.RemainingBalance < 0 {
		loan.RemainingBalance = 0
	}
	if loan.MonthsPaid >= loan.TenureMonths || loan.RemainingBalance <= 0 {
		loan.Status = "completed"
	}

	if err := uc.repo.UpdateLoan(ctx, loan); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record payment")
	}

	return ToLoanResponse(loan), nil
}

func (uc *cooperativeUseCase) SeedInitialData(ctx context.Context) error {
	// No default seed data required for the cooperative module.
	return nil
}
