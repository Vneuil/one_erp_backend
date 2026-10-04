package application

import (
	"math"
	"sort"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"gorm.io/gorm"

	"context"
)

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func pct(part, whole float64) float64 {
	if whole == 0 {
		return 0
	}
	return round2(part / whole * 100)
}

// wib is the reporting time zone: movement and journal timestamps are stored in
// UTC but the business day is Western Indonesia Time.
var wib = time.FixedZone("WIB", 7*3600)

func today() string { return time.Now().In(wib).Format("2006-01-02") }

func monthStart() string { return today()[:8] + "01" }

func validDate(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }

// resolveRange validates from/to. Both empty means the current month to date; a
// missing end is today and a missing start is the first day of the end's month.
func resolveRange(from, to string) (string, string, error) {
	if from != "" && !validDate(from) {
		return "", "", apperrors.NewBadRequest("from must be YYYY-MM-DD")
	}
	if to != "" && !validDate(to) {
		return "", "", apperrors.NewBadRequest("to must be YYYY-MM-DD")
	}
	if from == "" && to == "" {
		return monthStart(), today(), nil
	}
	if to == "" {
		to = today()
	}
	if from == "" {
		from = to[:8] + "01"
	}
	if from > to {
		return "", "", apperrors.NewBadRequest("from cannot be after to")
	}
	return from, to, nil
}

// scope restricts a query to the active tenant using a qualified column, so it
// stays unambiguous when the query joins other tenant-aware tables.
func scope(ctx context.Context, db *gorm.DB, alias string) *gorm.DB {
	if tid := tenantctx.FromContext(ctx); tid != nil {
		return db.Where(alias+".tenant_id = ?", *tid)
	}
	return db
}

// daysInclusive counts the calendar days from..to inclusive.
func daysInclusive(from, to string) int {
	f, err1 := time.Parse("2006-01-02", from)
	t, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil || t.Before(f) {
		return 0
	}
	return int(t.Sub(f).Hours()/24) + 1
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// notCountedOrder lists sales-order statuses that are not revenue yet or any more.
var notCountedOrder = []string{"cancelled", "rejected", "pending_approval", "draft"}
