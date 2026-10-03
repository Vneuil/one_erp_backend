package application

import (
	"fmt"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

func entry(no, date, memo, src string, debit, credit float64, created string) domain.JournalEntry {
	e := domain.JournalEntry{EntryNumber: no, Date: date, Memo: memo, SourceDoc: src, Status: "posted",
		Lines: []domain.JournalLine{{Debit: debit}, {Credit: credit}}}
	e.ID = uuid.New()
	if created != "" {
		e.CreatedAt, _ = time.Parse("2006-01-02", created)
	}
	return e
}

func rules(in []Insight) map[string]Insight {
	m := map[string]Insight{}
	for _, i := range in {
		m[i.Rule] = i
	}
	return m
}

func TestInsightsFlagBrokenDuplicateBackdatedAndRoundEntries(t *testing.T) {
	today, _ := time.Parse("2006-01-02", "2026-09-30")
	res := rules(BuildInsights(InsightInput{Today: today, Entries: []domain.JournalEntry{
		entry("JV-1", "2026-09-01", "Bayar sewa", "", 5_000_000, 5_000_000, "2026-09-01"),
		entry("JV-2", "2026-09-04", "  bayar   SEWA ", "", 5_000_000, 5_000_000, "2026-09-04"), // duplicate of JV-1
		entry("JV-3", "2026-06-01", "Koreksi", "", 1_000_000, 1_000_000, "2026-09-20"),         // backdated ~111 days
		entry("JV-4", "2026-09-10", "Penyesuaian", "", 25_000_000, 25_000_000, "2026-09-10"),   // round manual
		entry("JV-5", "2026-09-11", "Rusak", "", 1_000, 900, "2026-09-11"),                     // unbalanced
		entry("JV-6", "2026-09-12", "Penjualan", "sales-invoice:a", 3_500_000, 3_500_000, "2026-09-12"),
	}}))
	for _, want := range []string{"unbalanced", "duplicate_entry", "backdated_entry", "large_round_manual"} {
		if _, ok := res[want]; !ok {
			t.Errorf("missing insight %s: %+v", want, res)
		}
	}
	if res["unbalanced"].Severity != SeverityCritical || res["unbalanced"].Refs[0] != "JV-5" {
		t.Fatalf("unbalanced: %+v", res["unbalanced"])
	}
	if res["duplicate_entry"].Refs[0] != "JV-1 ≈ JV-2" || res["backdated_entry"].Refs[0] != "JV-3" || res["large_round_manual"].Refs[0] != "JV-4" {
		t.Fatalf("refs: %+v", res)
	}
	// The system entry and the plain small ones are not flagged as anything else.
	all := BuildInsights(InsightInput{Today: today, Entries: []domain.JournalEntry{entry("JV-6", "2026-09-12", "Penjualan", "sales-invoice:a", 3_500_000, 3_500_000, "2026-09-12")}})
	if len(all) != 0 {
		t.Fatalf("a normal entry must not raise findings: %+v", all)
	}
}

func TestSameMemoSameAmountFarApartOrSameSourceIsNotDuplicate(t *testing.T) {
	today, _ := time.Parse("2006-01-02", "2026-09-30")
	res := rules(BuildInsights(InsightInput{Today: today, Entries: []domain.JournalEntry{
		entry("A", "2026-08-01", "Gaji", "", 9_000_000, 9_000_000, "2026-08-01"),
		entry("B", "2026-09-01", "Gaji", "", 9_000_000, 9_000_000, "2026-09-01"), // a month later: normal
		entry("C", "2026-09-02", "Invoice", "sales-invoice:x", 1_000_000, 1_000_000, "2026-09-02"),
		entry("D", "2026-09-03", "Invoice", "sales-invoice:x", 1_000_000, 1_000_000, "2026-09-03"), // same source: system re-run, not a keying error
	}}))
	if _, ok := res["duplicate_entry"]; ok {
		t.Fatalf("false positive: %+v", res["duplicate_entry"])
	}
}

func TestOutlierNeedsEnoughHistory(t *testing.T) {
	today, _ := time.Parse("2006-01-02", "2026-09-30")
	var many []domain.JournalEntry
	for i := 0; i < 30; i++ {
		many = append(many, entry(fmt.Sprintf("N%d", i), "2026-09-10", fmt.Sprintf("Belanja %d", i), "s", 100_000, 100_000, "2026-09-10"))
	}
	many = append(many, entry("BIG", "2026-09-11", "Pembelian mesin", "s2", 90_000_000, 90_000_000, "2026-09-11"))
	res := rules(BuildInsights(InsightInput{Today: today, Entries: many}))
	if res["amount_outlier"].Refs[0] != "BIG (90000000)" {
		t.Fatalf("outlier: %+v", res["amount_outlier"])
	}
	// With little history there is nothing to compare against.
	if _, ok := rules(BuildInsights(InsightInput{Today: today, Entries: many[28:]}))["amount_outlier"]; ok {
		t.Fatal("outliers need a minimum sample")
	}
}

func TestOverdueReceivablesShareAndSeverity(t *testing.T) {
	today, _ := time.Parse("2006-01-02", "2026-09-30")
	mk := func(no, due string, total, paid float64) domain.Receivable {
		return domain.Receivable{InvoiceNo: no, DueDate: due, TotalInvoice: total, PaidAmount: paid}
	}
	res := rules(BuildInsights(InsightInput{Today: today, Receivables: []domain.Receivable{
		mk("INV-1", "2026-05-01", 4_000_000, 0),         // 152 days late
		mk("INV-2", "2026-09-20", 6_000_000, 0),         // current
		mk("INV-3", "2026-01-01", 5_000_000, 5_000_000), // fully paid: ignored
	}}))
	o := res["overdue_receivables"]
	if o.Severity != SeverityWarning || o.Refs[0] != "INV-1" { // 40% of 10m
		t.Fatalf("overdue: %+v", o)
	}
	res = rules(BuildInsights(InsightInput{Today: today, Receivables: []domain.Receivable{mk("INV-1", "2026-05-01", 6_000_000, 0), mk("INV-2", "2026-09-20", 4_000_000, 0)}}))
	if res["overdue_receivables"].Severity != SeverityCritical { // 60%
		t.Fatalf("above half is critical: %+v", res["overdue_receivables"])
	}
	if r := BuildInsights(InsightInput{Today: today, Receivables: []domain.Receivable{mk("X", "2026-09-25", 1_000_000, 0)}}); len(r) != 0 {
		t.Fatalf("recent overdue is not flagged: %+v", r)
	}
}

func TestInsightsAreSortedWorstFirstAndNeverNil(t *testing.T) {
	today, _ := time.Parse("2006-01-02", "2026-09-30")
	got := BuildInsights(InsightInput{Today: today, Entries: []domain.JournalEntry{
		entry("R", "2026-09-10", "Plug", "", 20_000_000, 20_000_000, "2026-09-10"),
		entry("U", "2026-09-11", "Rusak", "", 5, 1, "2026-09-11"),
	}})
	if len(got) != 2 || got[0].Severity != SeverityCritical || got[1].Severity != SeverityInfo {
		t.Fatalf("order: %+v", got)
	}
	if BuildInsights(InsightInput{Today: today}) == nil {
		t.Fatal("no findings must serialise as an empty array")
	}
}
