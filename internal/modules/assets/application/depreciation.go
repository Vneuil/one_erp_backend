package application

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/assets/domain"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

const periodLayout = "2006-01"

type DepreciationPostedItemDTO struct {
	AssetID   uuid.UUID `json:"assetId"`
	AssetCode string    `json:"assetCode"`
	Name      string    `json:"name"`
	Period    string    `json:"period"`
	Amount    float64   `json:"amount"`
}

type DepreciationPostResultDTO struct {
	Period      string                      `json:"period"`
	PostedCount int                         `json:"postedCount"`
	TotalAmount float64                     `json:"totalAmount"`
	AlreadyDone int                         `json:"alreadyPosted"`
	Items       []DepreciationPostedItemDTO `json:"items"`
}

type DepreciationReportLineDTO struct {
	AssetID           uuid.UUID `json:"assetId"`
	AssetCode         string    `json:"assetCode"`
	Name              string    `json:"name"`
	Category          string    `json:"category"`
	PurchaseDate      string    `json:"purchaseDate"`
	Status            string    `json:"status"`
	Cost              float64   `json:"cost"`
	UsefulLifeYears   int       `json:"usefulLifeYears"`
	MonthlyDepr       float64   `json:"monthlyDepreciation"`
	PostedInPeriod    float64   `json:"postedInPeriod"`
	AccumulatedPosted float64   `json:"accumulatedPosted"`
	BookValue         float64   `json:"bookValue"`
	// Unposted counts months from acquisition up to the period that are not
	// posted to the ledger yet.
	UnpostedMonths int `json:"unpostedMonths"`
}

type DepreciationReportDTO struct {
	Period            string                      `json:"period"`
	Lines             []DepreciationReportLineDTO `json:"lines"`
	TotalCost         float64                     `json:"totalCost"`
	TotalPostedPeriod float64                     `json:"totalPostedInPeriod"`
	TotalAccumulated  float64                     `json:"totalAccumulated"`
	TotalBookValue    float64                     `json:"totalBookValue"`
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func resolvePeriod(period string) (string, error) {
	current := time.Now().Format(periodLayout)
	if period == "" {
		return current, nil
	}
	if _, err := time.Parse(periodLayout, period); err != nil {
		return "", apperrors.NewBadRequest("period must be YYYY-MM")
	}
	return period, nil
}

// monthsBetween returns the periods from start to end inclusive.
func monthsBetween(start, end time.Time) []string {
	var out []string
	for m := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); !m.After(end); m = m.AddDate(0, 1, 0) {
		out = append(out, m.Format(periodLayout))
	}
	return out
}

// depreciable reports whether an asset takes part in monthly depreciation and
// returns its acquisition month and monthly straight-line charge.
func depreciable(a domain.FixedAsset) (start time.Time, monthly float64, ok bool) {
	if a.Status == "Disposed" || a.UsefulLifeYears <= 0 || a.PurchasePrice <= 0 {
		return time.Time{}, 0, false
	}
	t, err := time.Parse("2006-01-02", a.PurchaseDate)
	if err != nil {
		return time.Time{}, 0, false
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), round2(a.PurchasePrice / float64(a.UsefulLifeYears*12)), true
}

type postingIndex struct {
	periods map[uuid.UUID]map[string]bool
	sum     map[uuid.UUID]float64
}

func indexPostings(ps []domain.DepreciationPosting) postingIndex {
	idx := postingIndex{periods: map[uuid.UUID]map[string]bool{}, sum: map[uuid.UUID]float64{}}
	for _, p := range ps {
		if idx.periods[p.AssetID] == nil {
			idx.periods[p.AssetID] = map[string]bool{}
		}
		idx.periods[p.AssetID][p.Period] = true
		idx.sum[p.AssetID] += p.Amount
	}
	return idx
}

// entryDate is the last day of the period, capped at today so a posting for
// the running month is not dated in the future.
func entryDate(period string) string {
	m, _ := time.Parse(periodLayout, period)
	end := m.AddDate(0, 1, -1).Format("2006-01-02")
	if today := time.Now().Format("2006-01-02"); end > today {
		return today
	}
	return end
}

// PostDepreciation posts straight-line monthly depreciation for every active
// asset (Dr Depreciation Expense / Cr Accumulated Depreciation), one ledger
// entry per asset and month. It is idempotent: months already posted are
// skipped. Without catchUp only the given month is posted; with catchUp every
// unposted month from each asset's acquisition month up to the period is.
// Depreciation starts in the acquisition month and never exceeds the cost.
func (uc *assetsUseCase) PostDepreciation(ctx context.Context, period string, catchUp bool) (*DepreciationPostResultDTO, error) {
	period, err := resolvePeriod(period)
	if err != nil {
		return nil, err
	}
	if period > time.Now().Format(periodLayout) {
		return nil, apperrors.NewBadRequest("Cannot post depreciation for a future period")
	}
	if uc.ledger == nil {
		return nil, apperrors.NewInternal(fmt.Errorf("assets: ledger not configured"), "Ledger posting is not available")
	}
	assets, err := uc.repo.ListAllAssets(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list assets")
	}
	existing, err := uc.repo.ListDepreciationPostings(ctx, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load depreciation postings")
	}
	idx := indexPostings(existing)
	end, _ := time.Parse(periodLayout, period)

	res := &DepreciationPostResultDTO{Period: period, Items: []DepreciationPostedItemDTO{}}
	sort.Slice(assets, func(i, j int) bool { return assets[i].AssetCode < assets[j].AssetCode })
	for _, a := range assets {
		start, monthly, ok := depreciable(a)
		if !ok || start.After(end) {
			continue
		}
		months := []string{period}
		if catchUp {
			months = monthsBetween(start, end)
		}
		for _, m := range months {
			if idx.periods[a.ID][m] {
				res.AlreadyDone++
				continue
			}
			remaining := round2(a.PurchasePrice - idx.sum[a.ID])
			if remaining <= 0.005 {
				break
			}
			amt := math.Min(monthly, remaining)
			doc := fmt.Sprintf("depreciation:%s:%s", a.ID, m)
			desc := a.AssetCode + " " + a.Name
			err := uc.ledger.PostEntryOn(ctx, entryDate(m), doc, "Depreciation "+m+" - "+desc, []financeApp.LedgerLine{
				{AccountCode: financeApp.AccountDepreciationExpense, Debit: amt, Description: desc},
				{AccountCode: financeApp.AccountAccumDepreciation, Credit: amt, Description: desc},
			})
			if err != nil {
				return res, apperrors.NewInternal(err, "Failed to post depreciation for "+a.AssetCode)
			}
			p := &domain.DepreciationPosting{AssetID: a.ID, Period: m, Amount: amt, JournalSourceDoc: doc}
			if err := uc.repo.CreateDepreciationPosting(ctx, p); err != nil {
				return res, apperrors.NewInternal(err, "Posted to the ledger but could not record depreciation for "+a.AssetCode)
			}
			if idx.periods[a.ID] == nil {
				idx.periods[a.ID] = map[string]bool{}
			}
			idx.periods[a.ID][m] = true
			idx.sum[a.ID] += amt
			res.PostedCount++
			res.TotalAmount += amt
			res.Items = append(res.Items, DepreciationPostedItemDTO{AssetID: a.ID, AssetCode: a.AssetCode, Name: a.Name, Period: m, Amount: amt})
		}
	}
	res.TotalAmount = round2(res.TotalAmount)
	return res, nil
}

// DepreciationReport (Laporan Penyusutan Aktiva) shows, per asset, the
// depreciation posted in the period, the accumulated amount posted through the
// end of it, and the resulting book value.
func (uc *assetsUseCase) DepreciationReport(ctx context.Context, period string) (*DepreciationReportDTO, error) {
	period, err := resolvePeriod(period)
	if err != nil {
		return nil, err
	}
	assets, err := uc.repo.ListAllAssets(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list assets")
	}
	postings, err := uc.repo.ListDepreciationPostings(ctx, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load depreciation postings")
	}
	type agg struct {
		inPeriod, accumulated float64
		months                int
	}
	byAsset := map[uuid.UUID]*agg{}
	for _, p := range postings {
		if p.Period > period {
			continue
		}
		a := byAsset[p.AssetID]
		if a == nil {
			a = &agg{}
			byAsset[p.AssetID] = a
		}
		a.accumulated += p.Amount
		a.months++
		if p.Period == period {
			a.inPeriod += p.Amount
		}
	}
	end, _ := time.Parse(periodLayout, period)

	out := &DepreciationReportDTO{Period: period, Lines: []DepreciationReportLineDTO{}}
	sort.Slice(assets, func(i, j int) bool { return assets[i].AssetCode < assets[j].AssetCode })
	for _, a := range assets {
		g := byAsset[a.ID]
		if g == nil {
			g = &agg{}
		}
		line := DepreciationReportLineDTO{
			AssetID: a.ID, AssetCode: a.AssetCode, Name: a.Name, Category: a.Category, PurchaseDate: a.PurchaseDate,
			Status: a.Status, Cost: a.PurchasePrice, UsefulLifeYears: a.UsefulLifeYears,
			PostedInPeriod: round2(g.inPeriod), AccumulatedPosted: round2(g.accumulated),
			BookValue: round2(a.PurchasePrice - g.accumulated),
		}
		if start, monthly, ok := depreciable(a); ok {
			line.MonthlyDepr = monthly
			if !start.After(end) {
				due := len(monthsBetween(start, end))
				if max := a.UsefulLifeYears * 12; due > max {
					due = max
				}
				if due > g.months {
					line.UnpostedMonths = due - g.months
				}
			}
		}
		out.Lines = append(out.Lines, line)
		out.TotalCost += line.Cost
		out.TotalPostedPeriod += line.PostedInPeriod
		out.TotalAccumulated += line.AccumulatedPosted
		out.TotalBookValue += line.BookValue
	}
	out.TotalCost, out.TotalPostedPeriod = round2(out.TotalCost), round2(out.TotalPostedPeriod)
	out.TotalAccumulated, out.TotalBookValue = round2(out.TotalAccumulated), round2(out.TotalBookValue)
	return out, nil
}
