package application

import (
	"math"
	"sort"
	"strings"
	"time"

	crmdomain "github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/google/uuid"
)

// Analytics is the CRM intelligence report: funnel, win/loss, lost reasons,
// per-owner performance, lead sources and follow-up hygiene.
type Analytics struct {
	From        string        `json:"from,omitempty"`
	To          string        `json:"to,omitempty"`
	Funnel      []FunnelStage `json:"funnel"`
	Totals      DealTotals    `json:"totals"`
	LostReasons []ReasonRow   `json:"lostReasons"`
	Owners      []OwnerRow    `json:"owners"`
	LeadSources []SourceRow   `json:"leadSources"`
	Tasks       TaskHealth    `json:"tasks"`
	StaleDeals  []StaleDeal   `json:"staleDeals"`
}

type FunnelStage struct {
	Key           string  `json:"key"`
	Name          string  `json:"name"`
	Kind          string  `json:"kind"`
	Count         int     `json:"count"`
	Value         float64 `json:"value"`
	WeightedValue float64 `json:"weightedValue"`
}

type DealTotals struct {
	Open         int     `json:"open"`
	OpenValue    float64 `json:"openValue"`
	Weighted     float64 `json:"weightedPipeline"`
	Won          int     `json:"won"`
	WonValue     float64 `json:"wonValue"`
	Lost         int     `json:"lost"`
	LostValue    float64 `json:"lostValue"`
	WinRate      float64 `json:"winRate"` // won / (won + lost), 0-1
	AvgWonValue  float64 `json:"avgWonValue"`
	AvgCycleDays float64 `json:"avgCycleDays"` // creation to close, won deals
}

type ReasonRow struct {
	Reason string  `json:"reason"`
	Count  int     `json:"count"`
	Value  float64 `json:"value"`
}

type OwnerRow struct {
	Owner     string  `json:"owner"`
	Open      int     `json:"open"`
	OpenValue float64 `json:"openValue"`
	Won       int     `json:"won"`
	WonValue  float64 `json:"wonValue"`
	Lost      int     `json:"lost"`
	WinRate   float64 `json:"winRate"`
}

type SourceRow struct {
	Source         string  `json:"source"`
	Leads          int     `json:"leads"`
	EstimatedValue float64 `json:"estimatedValue"`
}

type TaskHealth struct {
	Open     int `json:"open"`
	Overdue  int `json:"overdue"`
	DueToday int `json:"dueToday"`
}

type StaleDeal struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Customer string    `json:"customer"`
	Value    float64   `json:"value"`
	Owner    string    `json:"owner"`
	DaysIdle int       `json:"daysIdle"`
}

// AnalyticsInput is everything BuildAnalytics needs; it does no I/O.
type AnalyticsInput struct {
	Deals           []crmdomain.Deal
	Leads           []crmdomain.Lead
	Stages          []domain.PipelineStage
	Tasks           []domain.SalesTask
	LastInteraction map[uuid.UUID]time.Time
	From, To        string // YYYY-MM-DD, inclusive; filter settled (won/lost) deals by close date
	Today           string // YYYY-MM-DD
	Now             time.Time
	StaleAfterDays  int
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func rate(won, lost int) float64 {
	if won+lost == 0 {
		return 0
	}
	return math.Round(float64(won)/float64(won+lost)*1000) / 1000
}

func inRange(t *time.Time, from, to string) bool {
	if from == "" && to == "" {
		return true
	}
	if t == nil {
		return false
	}
	d := t.In(zone).Format("2006-01-02")
	return (from == "" || d >= from) && (to == "" || d <= to)
}

// BuildAnalytics computes the CRM intelligence report. Open deals are always
// counted in full (they describe the pipeline as it is now); settled deals are
// limited to those closed inside [From, To] when a range is given.
func BuildAnalytics(in AnalyticsInput) Analytics {
	out := Analytics{From: in.From, To: in.To, Funnel: []FunnelStage{}, LostReasons: []ReasonRow{}, Owners: []OwnerRow{}, LeadSources: []SourceRow{}, StaleDeals: []StaleDeal{}}

	stageByKey := map[string]domain.PipelineStage{}
	for _, s := range in.Stages {
		stageByKey[s.Key] = s
	}
	kindOf := func(key string) string {
		if s, ok := stageByKey[key]; ok {
			return s.Kind
		}
		switch key {
		case "won":
			return domain.StageWon
		case "lost":
			return domain.StageLost
		}
		return domain.StageOpen
	}

	funnel := map[string]*FunnelStage{}
	for _, s := range in.Stages {
		funnel[s.Key] = &FunnelStage{Key: s.Key, Name: s.Name, Kind: s.Kind}
	}
	owners := map[string]*OwnerRow{}
	ownerOf := func(pic string) *OwnerRow {
		pic = strings.TrimSpace(pic)
		if pic == "" {
			pic = "Belum ditugaskan"
		}
		if owners[pic] == nil {
			owners[pic] = &OwnerRow{Owner: pic}
		}
		return owners[pic]
	}
	reasons := map[string]*ReasonRow{}
	var cycleSum float64
	var cycleN int

	for _, d := range in.Deals {
		kind := kindOf(d.Stage)
		if kind != domain.StageOpen && !inRange(d.ClosedAt, in.From, in.To) {
			continue
		}
		f := funnel[d.Stage]
		if f == nil {
			f = &FunnelStage{Key: d.Stage, Name: d.Stage, Kind: kind}
			funnel[d.Stage] = f
		}
		f.Count++
		f.Value += d.Value
		f.WeightedValue += d.Value * float64(d.Probability) / 100

		o := ownerOf(d.PIC)
		switch kind {
		case domain.StageOpen:
			out.Totals.Open++
			out.Totals.OpenValue += d.Value
			out.Totals.Weighted += d.Value * float64(d.Probability) / 100
			o.Open++
			o.OpenValue += d.Value
		case domain.StageWon:
			out.Totals.Won++
			out.Totals.WonValue += d.Value
			o.Won++
			o.WonValue += d.Value
			if d.ClosedAt != nil && d.ClosedAt.After(d.CreatedAt) {
				cycleSum += d.ClosedAt.Sub(d.CreatedAt).Hours() / 24
				cycleN++
			}
		case domain.StageLost:
			out.Totals.Lost++
			out.Totals.LostValue += d.Value
			o.Lost++
			reason := strings.TrimSpace(d.LostReason)
			if reason == "" {
				reason = "Tanpa alasan"
			}
			key := strings.ToLower(reason)
			if reasons[key] == nil {
				reasons[key] = &ReasonRow{Reason: reason}
			}
			reasons[key].Count++
			reasons[key].Value += d.Value
		}
	}
	out.Totals.WinRate = rate(out.Totals.Won, out.Totals.Lost)
	if out.Totals.Won > 0 {
		out.Totals.AvgWonValue = math.Round(out.Totals.WonValue / float64(out.Totals.Won))
	}
	if cycleN > 0 {
		out.Totals.AvgCycleDays = round1(cycleSum / float64(cycleN))
	}

	// Funnel in configured order; unknown legacy stages go last.
	seen := map[string]bool{}
	for _, s := range in.Stages {
		out.Funnel = append(out.Funnel, *funnel[s.Key])
		seen[s.Key] = true
	}
	var extra []string
	for k := range funnel {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		out.Funnel = append(out.Funnel, *funnel[k])
	}

	for _, r := range reasons {
		out.LostReasons = append(out.LostReasons, *r)
	}
	sort.Slice(out.LostReasons, func(i, j int) bool {
		if out.LostReasons[i].Count != out.LostReasons[j].Count {
			return out.LostReasons[i].Count > out.LostReasons[j].Count
		}
		return out.LostReasons[i].Reason < out.LostReasons[j].Reason
	})
	for _, o := range owners {
		o.WinRate = rate(o.Won, o.Lost)
		out.Owners = append(out.Owners, *o)
	}
	sort.Slice(out.Owners, func(i, j int) bool {
		if out.Owners[i].WonValue != out.Owners[j].WonValue {
			return out.Owners[i].WonValue > out.Owners[j].WonValue
		}
		return out.Owners[i].Owner < out.Owners[j].Owner
	})

	sources := map[string]*SourceRow{}
	for _, l := range in.Leads {
		src := strings.ToLower(strings.TrimSpace(l.Source))
		if src == "" {
			src = "unknown"
		}
		if sources[src] == nil {
			sources[src] = &SourceRow{Source: src}
		}
		sources[src].Leads++
		sources[src].EstimatedValue += l.EstimatedValue
	}
	for _, s := range sources {
		out.LeadSources = append(out.LeadSources, *s)
	}
	sort.Slice(out.LeadSources, func(i, j int) bool {
		if out.LeadSources[i].Leads != out.LeadSources[j].Leads {
			return out.LeadSources[i].Leads > out.LeadSources[j].Leads
		}
		return out.LeadSources[i].Source < out.LeadSources[j].Source
	})

	for _, t := range in.Tasks {
		if t.Status != "open" {
			continue
		}
		out.Tasks.Open++
		switch {
		case t.DueDate < in.Today:
			out.Tasks.Overdue++
		case t.DueDate == in.Today:
			out.Tasks.DueToday++
		}
	}

	// Stale deals: open deals with no logged interaction for a while (falling
	// back to when the deal was created).
	staleAfter := in.StaleAfterDays
	if staleAfter <= 0 {
		staleAfter = 14
	}
	for _, d := range in.Deals {
		if kindOf(d.Stage) != domain.StageOpen {
			continue
		}
		last := d.CreatedAt
		if t, ok := in.LastInteraction[d.ID]; ok && t.After(last) {
			last = t
		}
		idle := int(in.Now.Sub(last).Hours() / 24)
		if idle >= staleAfter {
			out.StaleDeals = append(out.StaleDeals, StaleDeal{ID: d.ID, Title: d.Title, Customer: d.Customer, Value: d.Value, Owner: d.PIC, DaysIdle: idle})
		}
	}
	sort.Slice(out.StaleDeals, func(i, j int) bool { return out.StaleDeals[i].DaysIdle > out.StaleDeals[j].DaysIdle })
	if len(out.StaleDeals) > 20 {
		out.StaleDeals = out.StaleDeals[:20]
	}
	return out
}
