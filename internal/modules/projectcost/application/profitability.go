package application

import (
	"context"
	"math"
	"sort"

	projectDomain "github.com/divinecoid/one-backend/internal/modules/project/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
)

// ProfitRow is the profit-and-loss position of one project. Revenue is the
// RAB (the price agreed with the customer); cost is what was actually spent.
type ProfitRow struct {
	ProjectID   string  `json:"projectId"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Customer    string  `json:"customer"`
	Status      string  `json:"status"`
	Progress    int     `json:"progress"`
	Revenue     float64 `json:"revenue"`     // RAB
	PlannedCost float64 `json:"plannedCost"` // RAP
	ActualCost  float64 `json:"actualCost"`
	// EarnedRevenue is revenue recognised by progress; ProfitToDate = earned - actual.
	EarnedRevenue float64 `json:"earnedRevenue"`
	ProfitToDate  float64 `json:"profitToDate"`
	// ForecastProfit assumes the project finishes at its RAP (or at actual cost if already above it).
	ForecastProfit float64 `json:"forecastProfit"`
	ForecastPct    float64 `json:"forecastMarginPct"`
	LossMaking     bool    `json:"lossMaking"`
}

type ProfitReport struct {
	Rows           []ProfitRow `json:"rows"`
	Revenue        float64     `json:"revenue"`
	ActualCost     float64     `json:"actualCost"`
	ProfitToDate   float64     `json:"profitToDate"`
	ForecastProfit float64     `json:"forecastProfit"`
}

// BuildProfitReport is pure so the arithmetic can be tested without a database.
func BuildProfitReport(projects []projectDomain.Project) ProfitReport {
	var rep ProfitReport
	rep.Rows = make([]ProfitRow, 0, len(projects))
	for _, p := range projects {
		progress := p.Progress
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		earned := round2(p.RABValue * float64(progress) / 100)
		finalCost := math.Max(p.RAPValue, p.ActualCost)
		r := ProfitRow{
			ProjectID: p.ID.String(), Code: p.Code, Name: p.Name, Customer: p.Customer, Status: p.Status, Progress: progress,
			Revenue: round2(p.RABValue), PlannedCost: round2(p.RAPValue), ActualCost: round2(p.ActualCost),
			EarnedRevenue: earned, ProfitToDate: round2(earned - p.ActualCost),
			ForecastProfit: round2(p.RABValue - finalCost),
		}
		if p.RABValue > 0 {
			r.ForecastPct = math.Round(r.ForecastProfit/p.RABValue*1000) / 10
		}
		r.LossMaking = r.ForecastProfit < 0
		rep.Revenue += r.Revenue
		rep.ActualCost += r.ActualCost
		rep.ProfitToDate += r.ProfitToDate
		rep.ForecastProfit += r.ForecastProfit
		rep.Rows = append(rep.Rows, r)
	}
	sort.SliceStable(rep.Rows, func(i, j int) bool { return rep.Rows[i].ForecastProfit < rep.Rows[j].ForecastProfit })
	rep.Revenue, rep.ActualCost = round2(rep.Revenue), round2(rep.ActualCost)
	rep.ProfitToDate, rep.ForecastProfit = round2(rep.ProfitToDate), round2(rep.ForecastProfit)
	return rep
}

func (s *Service) Profitability(ctx context.Context) (*ProfitReport, error) {
	projects, _, err := s.projects.List(ctx, types.PaginationQuery{Page: 1, PerPage: 1000})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load projects")
	}
	rep := BuildProfitReport(projects)
	return &rep, nil
}
