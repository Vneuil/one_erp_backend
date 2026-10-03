package application

import (
	"testing"

	projectDomain "github.com/divinecoid/one-backend/internal/modules/project/domain"
)

func TestBuildProfitReport(t *testing.T) {
	ok := projectDomain.Project{Code: "A", RABValue: 100_000_000, RAPValue: 80_000_000, ActualCost: 30_000_000, Progress: 50}
	over := projectDomain.Project{Code: "B", RABValue: 50_000_000, RAPValue: 40_000_000, ActualCost: 55_000_000, Progress: 120}
	rep := BuildProfitReport([]projectDomain.Project{ok, over})
	if len(rep.Rows) != 2 || rep.Rows[0].Code != "B" {
		t.Fatalf("loss-making project should sort first: %+v", rep.Rows)
	}
	b, a := rep.Rows[0], rep.Rows[1]
	if !b.LossMaking || b.ForecastProfit != -5_000_000 || b.Progress != 100 || b.EarnedRevenue != 50_000_000 {
		t.Fatalf("B = %+v", b)
	}
	if a.LossMaking || a.ProfitToDate != 20_000_000 || a.ForecastProfit != 20_000_000 || a.ForecastPct != 20 {
		t.Fatalf("A = %+v", a)
	}
	if rep.ForecastProfit != 15_000_000 || rep.Revenue != 150_000_000 {
		t.Fatalf("totals = %+v", rep)
	}
}
