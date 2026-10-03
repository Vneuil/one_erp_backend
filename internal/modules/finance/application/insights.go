package application

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

// Insight severities, most serious first.
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// Insight is one finding from the automatic review of the books.
type Insight struct {
	Rule     string   `json:"rule"`
	Severity string   `json:"severity"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	Refs     []string `json:"refs,omitempty"` // entry numbers or invoice numbers to look at
}

// InsightInput is what the rules review; BuildInsights does no I/O.
type InsightInput struct {
	Entries     []domain.JournalEntry
	Receivables []domain.Receivable
	Today       time.Time
}

func entryTotal(e domain.JournalEntry) (debit, credit float64) {
	for _, l := range e.Lines {
		debit += l.Debit
		credit += l.Credit
	}
	return
}

func isManual(e domain.JournalEntry) bool { return strings.TrimSpace(e.SourceDoc) == "" }

func normMemo(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

func parseDay(s string) (time.Time, bool) {
	if len(s) >= 10 {
		s = s[:10]
	}
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

const (
	duplicateWindowDays = 7
	backdatedAfterDays  = 30
	largeRoundEntry     = 10_000_000
	outlierMinSample    = 20
	outlierSigmas       = 3.0
	overdueShareWarn    = 0.30
	overdueBucketDays   = 90
	maxRefsPerInsight   = 8
)

// BuildInsights applies the review rules and returns findings, worst first.
//
//   - unbalanced: a posted entry whose debits and credits differ (books are broken).
//   - duplicate_entry: the same memo and amount posted twice within a week.
//   - backdated_entry: an entry dated more than 30 days before it was created.
//   - large_round_manual: a large manual entry in whole millions (estimates and plugs).
//   - amount_outlier: an entry far above the usual size (mean + 3 standard deviations).
//   - overdue_receivables: a large share of open receivables more than 90 days late.
func BuildInsights(in InsightInput) []Insight {
	var out []Insight
	add := func(i Insight) { out = append(out, i) }
	capRefs := func(r []string) []string {
		if len(r) > maxRefsPerInsight {
			return append(r[:maxRefsPerInsight:maxRefsPerInsight], fmt.Sprintf("+%d lainnya", len(r)-maxRefsPerInsight))
		}
		return r
	}

	// Unbalanced entries.
	var bad []string
	for _, e := range in.Entries {
		if d, c := entryTotal(e); math.Abs(d-c) > 0.005 {
			bad = append(bad, e.EntryNumber)
		}
	}
	if len(bad) > 0 {
		add(Insight{Rule: "unbalanced", Severity: SeverityCritical, Title: "Jurnal tidak seimbang",
			Detail: fmt.Sprintf("%d jurnal terposting memiliki total debit dan kredit berbeda. Buku besar tidak dapat dipercaya sebelum ini diperbaiki.", len(bad)), Refs: capRefs(bad)})
	}

	// Duplicates: group by memo+amount, look for two within the window.
	type key struct {
		memo  string
		total float64
	}
	groups := map[key][]domain.JournalEntry{}
	for _, e := range in.Entries {
		d, _ := entryTotal(e)
		if normMemo(e.Memo) == "" || d <= 0 {
			continue
		}
		k := key{normMemo(e.Memo), round2(d)}
		groups[k] = append(groups[k], e)
	}
	var dupRefs []string
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool { return g[i].Date < g[j].Date })
		for i := 1; i < len(g); i++ {
			a, ok1 := parseDay(g[i-1].Date)
			b, ok2 := parseDay(g[i].Date)
			// Two entries from the same source document are the system's own re-run, not a
			// keying mistake; only manual repeats or repeats from different sources count.
			sameSource := g[i-1].SourceDoc != "" && g[i-1].SourceDoc == g[i].SourceDoc
			if ok1 && ok2 && b.Sub(a) <= duplicateWindowDays*24*time.Hour && !sameSource {
				dupRefs = append(dupRefs, g[i-1].EntryNumber+" ≈ "+g[i].EntryNumber)
				break
			}
		}
	}
	if len(dupRefs) > 0 {
		sort.Strings(dupRefs)
		add(Insight{Rule: "duplicate_entry", Severity: SeverityWarning, Title: "Kemungkinan jurnal ganda",
			Detail: fmt.Sprintf("%d pasang jurnal punya keterangan dan nominal sama dalam %d hari. Pastikan bukan input ganda.", len(dupRefs), duplicateWindowDays), Refs: capRefs(dupRefs)})
	}

	// Backdated entries.
	var back []string
	for _, e := range in.Entries {
		day, ok := parseDay(e.Date)
		if !ok || e.CreatedAt.IsZero() {
			continue
		}
		if e.CreatedAt.Sub(day) > backdatedAfterDays*24*time.Hour {
			back = append(back, e.EntryNumber)
		}
	}
	if len(back) > 0 {
		add(Insight{Rule: "backdated_entry", Severity: SeverityWarning, Title: "Jurnal bertanggal mundur",
			Detail: fmt.Sprintf("%d jurnal dicatat lebih dari %d hari setelah tanggalnya. Periksa apakah periode yang sudah ditutup terpengaruh.", len(back), backdatedAfterDays), Refs: capRefs(back)})
	}

	// Large round manual entries.
	var round []string
	for _, e := range in.Entries {
		d, _ := entryTotal(e)
		if isManual(e) && d >= largeRoundEntry && math.Mod(d, 1_000_000) == 0 {
			round = append(round, e.EntryNumber)
		}
	}
	if len(round) > 0 {
		add(Insight{Rule: "large_round_manual", Severity: SeverityInfo, Title: "Jurnal manual besar bernominal bulat",
			Detail: "Nominal bulat jutaan pada jurnal manual sering berupa estimasi atau penyesuaian. Pastikan ada dokumen pendukung.", Refs: capRefs(round)})
	}

	// Outliers by entry size.
	var totals []float64
	for _, e := range in.Entries {
		if d, _ := entryTotal(e); d > 0 {
			totals = append(totals, d)
		}
	}
	if len(totals) >= outlierMinSample {
		var sum float64
		for _, v := range totals {
			sum += v
		}
		mean := sum / float64(len(totals))
		var sq float64
		for _, v := range totals {
			sq += (v - mean) * (v - mean)
		}
		sd := math.Sqrt(sq / float64(len(totals)))
		limit := mean + outlierSigmas*sd
		var big []string
		for _, e := range in.Entries {
			if d, _ := entryTotal(e); sd > 0 && d > limit {
				big = append(big, fmt.Sprintf("%s (%.0f)", e.EntryNumber, d))
			}
		}
		if len(big) > 0 {
			add(Insight{Rule: "amount_outlier", Severity: SeverityWarning, Title: "Jurnal jauh di atas ukuran biasa",
				Detail: fmt.Sprintf("Nominal melebihi rata-rata %.0f ditambah %.0f simpangan baku (batas %.0f).", mean, outlierSigmas, limit), Refs: capRefs(big)})
		}
	}

	// Overdue receivables concentration.
	var open, late float64
	var lateRefs []string
	for _, r := range in.Receivables {
		out := r.TotalInvoice - r.PaidAmount
		if out <= 0.005 {
			continue
		}
		open += out
		if due, ok := parseDay(r.DueDate); ok && in.Today.Sub(due) > overdueBucketDays*24*time.Hour {
			late += out
			lateRefs = append(lateRefs, r.InvoiceNo)
		}
	}
	if open > 0 && late/open >= overdueShareWarn {
		sev := SeverityWarning
		if late/open >= 0.5 {
			sev = SeverityCritical
		}
		add(Insight{Rule: "overdue_receivables", Severity: sev, Title: "Piutang macet menumpuk",
			Detail: fmt.Sprintf("%.0f%% piutang terbuka (Rp %.0f dari Rp %.0f) sudah lewat jatuh tempo lebih dari %d hari.", late/open*100, late, open, overdueBucketDays), Refs: capRefs(lateRefs)})
	}

	rank := map[string]int{SeverityCritical: 0, SeverityWarning: 1, SeverityInfo: 2}
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].Severity] < rank[out[j].Severity] })
	if out == nil {
		out = []Insight{}
	}
	return out
}

// Insights reviews the posted journal from `from` (default: 90 days ago) and open receivables.
func (uc *financeUseCase) Insights(ctx context.Context, from string) ([]Insight, error) {
	now := time.Now()
	if from == "" {
		from = now.AddDate(0, 0, -90).Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", from); err != nil {
		return nil, apperrors.NewBadRequest("from must be YYYY-MM-DD")
	}
	entries, err := uc.repo.ListPostedJournalEntries(ctx, from, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load journal entries")
	}
	recv, err := uc.repo.ListAllReceivables(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load receivables")
	}
	return BuildInsights(InsightInput{Entries: entries, Receivables: recv, Today: now}), nil
}
