package application

import (
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

func TestBudgetPeriodAndSlots(t *testing.T) {
	for p, want := range map[string]bool{"2026-10": true, "2026-13": false, "2026-1": false, "oct-2026": false, "": false} {
		if ValidBudgetPeriod(p) != want {
			t.Errorf("%q", p)
		}
	}
	acc := uuid.New()
	a := domain.Budget{Department: "Ops", AccountCategory: "Transport", Period: "2026-10", AccountID: &acc}
	same := domain.Budget{Department: " ops ", AccountCategory: "Other", Period: "2026-10", AccountID: &acc}
	otherMonth := a
	otherMonth.Period = "2026-11"
	otherAcc := uuid.New()
	diffAcc := domain.Budget{Department: "Ops", Period: "2026-10", AccountID: &otherAcc}
	noAcc1 := domain.Budget{Department: "Ops", AccountCategory: "Transport", Period: "2026-10"}
	noAcc2 := domain.Budget{Department: "Ops", AccountCategory: "transport", Period: "2026-10"}
	if !SameBudgetSlot(a, same) || SameBudgetSlot(a, otherMonth) || SameBudgetSlot(a, diffAcc) || SameBudgetSlot(a, noAcc1) || !SameBudgetSlot(noAcc1, noAcc2) {
		t.Error("slot comparison is wrong")
	}
}
