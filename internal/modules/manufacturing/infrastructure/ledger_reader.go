package infrastructure

import (
	"context"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeDomain "github.com/divinecoid/one-backend/internal/modules/finance/domain"
)

// LedgerWIPReader reads the Work in Process balance (account 1250) from the general ledger.
type LedgerWIPReader struct {
	Repo financeDomain.FinanceRepository
}

func (r LedgerWIPReader) WIPBalance(ctx context.Context) (float64, error) {
	accounts, err := r.Repo.ListAllAccounts(ctx)
	if err != nil {
		return 0, err
	}
	sums, err := r.Repo.SumPostedByAccount(ctx, "", "")
	if err != nil {
		return 0, err
	}
	for _, a := range accounts {
		if a.Code != financeApp.AccountWIP {
			continue
		}
		for _, s := range sums {
			if s.AccountID == a.ID {
				return s.TotalDebit - s.TotalCredit, nil
			}
		}
	}
	return 0, nil
}
