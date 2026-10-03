package domain

import (
	"math"
	"time"
)

func cents(v float64) float64 { return math.Round(v*100) / 100 }

// monthsSince is how many months period (YYYY-MM) is after start; negative
// when period is earlier. ok is false for malformed input.
func monthsSince(start, period string) (int, bool) {
	s, err1 := time.Parse("2006-01", start)
	p, err2 := time.Parse("2006-01", period)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return (p.Year()-s.Year())*12 + int(p.Month()) - int(s.Month()), true
}

// InstallmentFor is what an approved advance withholds in period: equal
// installments, with the last one absorbing any rounding remainder. Zero
// before the start, after the last installment, and for advances not approved.
func (a CashAdvance) InstallmentFor(period string) float64 {
	if a.Status != "approved" || a.Installments < 1 {
		return 0
	}
	k, ok := monthsSince(a.StartPeriod, period)
	if !ok || k < 0 || k >= a.Installments {
		return 0
	}
	monthly := math.Floor(a.Amount/float64(a.Installments)*100) / 100
	if k == a.Installments-1 {
		return cents(a.Amount - monthly*float64(a.Installments-1))
	}
	return monthly
}

// OutstandingAfter is the balance left once period's installment is withheld.
func (a CashAdvance) OutstandingAfter(period string) float64 {
	if a.Status != "approved" {
		return 0
	}
	k, ok := monthsSince(a.StartPeriod, period)
	if !ok || k < 0 {
		return cents(a.Amount)
	}
	if k >= a.Installments-1 {
		return 0
	}
	monthly := math.Floor(a.Amount/float64(a.Installments)*100) / 100
	return cents(a.Amount - monthly*float64(k+1))
}
