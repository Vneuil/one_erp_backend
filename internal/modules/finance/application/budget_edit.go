package application

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// UpdateBudgetDTO changes only the fields that are present. The period and
// linked account are fixed: to budget another month or account, add a budget.
type UpdateBudgetDTO struct {
	Department      *string  `json:"department"`
	AccountCategory *string  `json:"accountCategory"`
	AllocatedBudget *float64 `json:"allocatedBudget"`
}

// ValidBudgetPeriod reports whether p is YYYY-MM.
func ValidBudgetPeriod(p string) bool {
	if len(p) != 7 {
		return false
	}
	_, err := time.Parse("2006-01", p)
	return err == nil
}

// SameBudgetSlot is true when two budgets cover the same department, account
// (or category when no account is linked) and month: only one may exist.
func SameBudgetSlot(a, b domain.Budget) bool {
	if !strings.EqualFold(strings.TrimSpace(a.Department), strings.TrimSpace(b.Department)) || a.Period != b.Period {
		return false
	}
	if a.AccountID != nil || b.AccountID != nil {
		return a.AccountID != nil && b.AccountID != nil && *a.AccountID == *b.AccountID
	}
	return strings.EqualFold(strings.TrimSpace(a.AccountCategory), strings.TrimSpace(b.AccountCategory))
}

func (uc *financeUseCase) budgetExists(ctx context.Context, cand domain.Budget, ignore uuid.UUID) (bool, error) {
	list, _, err := uc.repo.ListBudgets(ctx, types.PaginationQuery{Page: 1, PerPage: 5000})
	if err != nil {
		return false, err
	}
	for _, b := range list {
		if b.ID != ignore && SameBudgetSlot(b, cand) {
			return true, nil
		}
	}
	return false, nil
}

func (uc *financeUseCase) UpdateBudget(ctx context.Context, id uuid.UUID, dto UpdateBudgetDTO) (*BudgetResponseDTO, error) {
	b, err := uc.repo.GetBudgetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load budget")
	}
	if b == nil {
		return nil, apperrors.NewNotFound("Budget not found")
	}
	if dto.AllocatedBudget != nil {
		v := *dto.AllocatedBudget
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return nil, apperrors.NewBadRequest("Allocated budget must be positive")
		}
		b.AllocatedBudget = math.Round(v*100) / 100
	}
	if dto.Department != nil {
		if strings.TrimSpace(*dto.Department) == "" {
			return nil, apperrors.NewBadRequest("Department cannot be empty")
		}
		b.Department = strings.TrimSpace(*dto.Department)
	}
	if dto.AccountCategory != nil {
		if strings.TrimSpace(*dto.AccountCategory) == "" {
			return nil, apperrors.NewBadRequest("Account category cannot be empty")
		}
		b.AccountCategory = strings.TrimSpace(*dto.AccountCategory)
	}
	if dup, err := uc.budgetExists(ctx, *b, b.ID); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing budgets")
	} else if dup {
		return nil, apperrors.NewConflict("A budget for this department, account and month already exists")
	}
	if err := uc.repo.UpdateBudget(ctx, b); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update budget")
	}
	actual, err := uc.budgetActual(ctx, b)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute budget actuals")
	}
	return ToBudgetResponse(b, actual), nil
}

func (uc *financeUseCase) DeleteBudget(ctx context.Context, id uuid.UUID) error {
	ok, err := uc.repo.DeleteBudget(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete budget")
	}
	if !ok {
		return apperrors.NewNotFound("Budget not found")
	}
	return nil
}
