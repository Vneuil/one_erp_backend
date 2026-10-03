package application

import (
	"testing"
	"time"

	crmdomain "github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/google/uuid"
)

func at(s string) *time.Time {
	t, _ := time.ParseInLocation("2006-01-02", s, zone)
	return &t
}

func deal(stage, pic string, value float64, prob int, created, closed string, reason string) crmdomain.Deal {
	d := crmdomain.Deal{Title: "D", Stage: stage, PIC: pic, Value: value, Probability: prob, LostReason: reason}
	d.ID = uuid.New()
	d.CreatedAt = *at(created)
	if closed != "" {
		d.ClosedAt = at(closed)
	}
	return d
}

func TestBuildAnalytics(t *testing.T) {
	stages := DefaultStages()
	deals := []crmdomain.Deal{
		deal("discovery", "ani@x.com", 100, 30, "2026-09-25", "", ""),
		deal("negotiation", "ani@x.com", 1000, 80, "2026-08-01", "", ""),
		deal("won", "ani@x.com", 500, 100, "2026-09-01", "2026-09-11", ""),
		deal("won", "bob@x.com", 300, 100, "2026-09-01", "2026-09-21", ""),
		deal("lost", "bob@x.com", 200, 0, "2026-09-01", "2026-09-05", "Harga terlalu tinggi"),
		deal("lost", "bob@x.com", 50, 0, "2026-09-01", "2026-09-06", " harga terlalu tinggi "),
		deal("lost", "ani@x.com", 70, 0, "2026-09-01", "2026-09-07", ""),
		deal("won", "ani@x.com", 999, 100, "2026-01-01", "2026-02-01", ""), // closed outside the range
	}
	leads := []crmdomain.Lead{{Source: "Web", EstimatedValue: 10}, {Source: "web", EstimatedValue: 5}, {Source: "referral", EstimatedValue: 1}, {Source: ""}}
	tasks := []domain.SalesTask{
		{Status: "open", DueDate: "2026-09-29"}, {Status: "open", DueDate: "2026-09-30"}, {Status: "open", DueDate: "2026-10-05"}, {Status: "done", DueDate: "2026-09-01"},
	}
	a := BuildAnalytics(AnalyticsInput{Deals: deals, Leads: leads, Stages: stages, Tasks: tasks, From: "2026-09-01", To: "2026-09-30",
		Today: "2026-09-30", Now: *at("2026-09-30"), StaleAfterDays: 14})

	tot := a.Totals
	if tot.Open != 2 || tot.OpenValue != 1100 || tot.Won != 2 || tot.WonValue != 800 || tot.Lost != 3 {
		t.Fatalf("totals: %+v", tot)
	}
	if tot.WinRate != 0.4 { // 2 won / (2 won + 3 lost)
		t.Fatalf("win rate %v", tot.WinRate)
	}
	if tot.Weighted != 100*0.3+1000*0.8 {
		t.Fatalf("weighted pipeline %v", tot.Weighted)
	}
	if tot.AvgWonValue != 400 || tot.AvgCycleDays != 15 { // cycles 10 and 20 days
		t.Fatalf("avg value %v cycle %v", tot.AvgWonValue, tot.AvgCycleDays)
	}
	// Lost reasons merge case/whitespace variants and fall back to "Tanpa alasan".
	if len(a.LostReasons) != 2 || a.LostReasons[0].Count != 2 || a.LostReasons[0].Value != 250 || a.LostReasons[1].Reason != "Tanpa alasan" {
		t.Fatalf("reasons: %+v", a.LostReasons)
	}
	if len(a.Funnel) != 5 || a.Funnel[0].Key != "discovery" || a.Funnel[3].Count != 2 {
		t.Fatalf("funnel: %+v", a.Funnel)
	}
	if a.Owners[0].Owner != "bob@x.com" && a.Owners[0].WonValue != 500 {
		// ani won 500, bob won 300: ani leads
	}
	if a.Owners[0].Owner != "ani@x.com" || a.Owners[0].WonValue != 500 || a.Owners[1].Lost != 2 {
		t.Fatalf("owners: %+v", a.Owners)
	}
	if a.LeadSources[0].Source != "web" || a.LeadSources[0].Leads != 2 || a.LeadSources[0].EstimatedValue != 15 {
		t.Fatalf("sources: %+v", a.LeadSources)
	}
	if a.Tasks.Open != 3 || a.Tasks.Overdue != 1 || a.Tasks.DueToday != 1 {
		t.Fatalf("tasks: %+v", a.Tasks)
	}
	// The negotiation deal (created Aug 1, no interaction) is stale; the fresh discovery one is not.
	if len(a.StaleDeals) != 1 || a.StaleDeals[0].Value != 1000 {
		t.Fatalf("stale: %+v", a.StaleDeals)
	}
}

func TestAnalyticsTreatsRecentInteractionAsActivity(t *testing.T) {
	d := deal("negotiation", "a", 10, 80, "2026-07-01", "", "")
	a := BuildAnalytics(AnalyticsInput{Deals: []crmdomain.Deal{d}, Stages: DefaultStages(), LastInteraction: map[uuid.UUID]time.Time{d.ID: *at("2026-09-28")},
		Today: "2026-09-30", Now: *at("2026-09-30"), StaleAfterDays: 14})
	if len(a.StaleDeals) != 0 {
		t.Fatalf("a deal with a recent interaction is not stale: %+v", a.StaleDeals)
	}
}

func TestAnalyticsWithNoDataIsEmptyNotNull(t *testing.T) {
	a := BuildAnalytics(AnalyticsInput{Stages: DefaultStages(), Today: "2026-09-30", Now: time.Now()})
	if a.Totals.WinRate != 0 || a.LostReasons == nil || a.Owners == nil || a.StaleDeals == nil || a.LeadSources == nil {
		t.Fatalf("empty analytics must serialise arrays, got %+v", a)
	}
}
