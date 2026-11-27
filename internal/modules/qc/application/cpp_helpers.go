package application

import (
	"strings"
	"time"
)

// Form PRO-007-CPP Revisi 01 — Catatan Pemakaian dan Pembersihan Alat (Metal Detector / X-Ray).

const (
	cppDocumentCode = "PRO-007-CPP"
	cppRevisionCode = "01"
)

// normalizeCPPEquipment maps free-text equipment names to the canonical "Nama Alat" on the form.
func normalizeCPPEquipment(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case n == "":
		return "Metal Detector"
	case strings.Contains(n, "x-ray") || strings.Contains(n, "xray") || strings.Contains(n, "x ray"):
		return "X-Ray"
	case strings.Contains(n, "metal"):
		return "Metal Detector"
	default:
		return strings.TrimSpace(name)
	}
}

// cppHasFinding returns true when the "Temuan" column contains an actual finding.
func cppHasFinding(finding string) bool {
	f := strings.ToLower(strings.TrimSpace(finding))
	switch f {
	case "", "-", "--", "tidak ada", "tidak ada temuan", "nihil", "none", "n/a", "na", "ok":
		return false
	}
	return true
}

// computeCPPHours returns the duration in hours between two HH:mm times (handles overnight usage).
func computeCPPHours(start, end string) float64 {
	s, err1 := time.Parse("15:04", strings.TrimSpace(start))
	e, err2 := time.Parse("15:04", strings.TrimSpace(end))
	if err1 != nil || err2 != nil {
		return 0
	}
	d := e.Sub(s)
	if d < 0 {
		d += 24 * time.Hour
	}
	return float64(int(d.Hours()*100+0.5)) / 100
}

// deriveCPPStatus derives the overall status from cleanliness, physical condition and findings.
func deriveCPPStatus(sensor, belt, condition, finding string) string {
	switch {
	case strings.EqualFold(sensor, "Perlu Dibersihkan") || strings.EqualFold(belt, "Perlu Dibersihkan"):
		return "needs_cleaning"
	case strings.EqualFold(condition, "Cacat") || strings.EqualFold(condition, "Kabel Terbuka"):
		return "maintenance"
	case cppHasFinding(finding):
		return "finding"
	default:
		return "passed"
	}
}
