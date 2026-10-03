package application

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// zone is the timezone attendance times are entered and judged in.
var zone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}()

const (
	defaultShiftStart = "09:00"
	// Overtime is detected once the worked time exceeds the standard day by more than the threshold.
	standardWorkMinutes      = 8 * 60
	overtimeThresholdMinutes = 30
)

// parseClock reads "HH:MM" on a YYYY-MM-DD date in the attendance timezone.
func parseClock(date, hhmm string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+strings.TrimSpace(hhmm), zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date/time %q %q", date, hhmm)
	}
	return t, nil
}

// clockStatus is "Late" when the clock-in is at or after the shift start
// (the same rule live clock-in uses), otherwise "On Time".
func clockStatus(clockIn time.Time, shiftStart string) (string, error) {
	if shiftStart == "" {
		shiftStart = defaultShiftStart
	}
	start, err := parseClock(clockIn.In(zone).Format("2006-01-02"), shiftStart)
	if err != nil {
		return "", err
	}
	if !clockIn.Before(start) {
		return "Late", nil
	}
	return "On Time", nil
}

// DetectOvertimeMinutes returns the overtime in a day worked workMinutes, or 0
// when the excess over a standard day is within the threshold.
func DetectOvertimeMinutes(workMinutes int) int {
	if workMinutes <= standardWorkMinutes+overtimeThresholdMinutes {
		return 0
	}
	return workMinutes - standardWorkMinutes
}

// ValidDate reports whether s is a YYYY-MM-DD date.
func ValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// ValidClock reports whether s is HH:MM (24h).
func ValidClock(s string) bool {
	_, err := time.Parse("15:04", s)
	return err == nil
}

// FeedbackSummary averages feedback per competency, overall and by relationship.
type FeedbackSummary struct {
	Count          int                           `json:"count"`
	Overall        map[string]float64            `json:"overall"`
	ByRelationship map[string]map[string]float64 `json:"byRelationship"`
	Comments       []string                      `json:"comments"`
}

type ratings struct {
	relationship                                     string
	communication, teamwork, leadership, reliability int
	comment                                          string
}

func avg(sum, n int) float64 {
	if n == 0 {
		return 0
	}
	return math.Round(float64(sum)/float64(n)*100) / 100
}

func summarizeFeedback(items []ratings) FeedbackSummary {
	type acc struct{ c, t, l, r, n int }
	total := acc{}
	by := map[string]*acc{}
	out := FeedbackSummary{Overall: map[string]float64{}, ByRelationship: map[string]map[string]float64{}, Comments: []string{}}
	for _, it := range items {
		total.c, total.t, total.l, total.r, total.n = total.c+it.communication, total.t+it.teamwork, total.l+it.leadership, total.r+it.reliability, total.n+1
		a := by[it.relationship]
		if a == nil {
			a = &acc{}
			by[it.relationship] = a
		}
		a.c, a.t, a.l, a.r, a.n = a.c+it.communication, a.t+it.teamwork, a.l+it.leadership, a.r+it.reliability, a.n+1
		if strings.TrimSpace(it.comment) != "" {
			out.Comments = append(out.Comments, it.comment)
		}
	}
	out.Count = total.n
	if total.n > 0 {
		out.Overall = map[string]float64{"communication": avg(total.c, total.n), "teamwork": avg(total.t, total.n), "leadership": avg(total.l, total.n), "reliability": avg(total.r, total.n)}
	}
	for rel, a := range by {
		out.ByRelationship[rel] = map[string]float64{"communication": avg(a.c, a.n), "teamwork": avg(a.t, a.n), "leadership": avg(a.l, a.n), "reliability": avg(a.r, a.n)}
	}
	return out
}
