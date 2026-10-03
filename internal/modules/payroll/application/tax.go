package application

import (
	"math"
	"strings"
)

// PPh 21 (Indonesian employee income tax) estimation.
//
// Method: annualised progressive tax under UU PPh Pasal 17 (rates as amended by
// UU HPP), i.e. the method used for the final period of the year. It is an
// ESTIMATE for regular monthly income: it projects the month's gross income to
// a full year, and does not model irregular income (THR, bonuses), pension
// contributions (JHT/JP) as deductible, or the monthly TER withholding tables
// (PMK 168/2023) used for January-November. The amounts can therefore differ
// slightly from a payslip produced with TER; confirm with your tax advisor.

const (
	jobExpenseRate      = 0.05      // biaya jabatan: 5% of gross income...
	jobExpenseMaxAnnual = 6_000_000 // ...capped at Rp500,000/month
	noNPWPSurcharge     = 1.2       // Pasal 21(5a): 120% of the tax when the employee has no NPWP
	ptkpBase            = 54_000_000
	ptkpPerDependent    = 4_500_000
	ptkpMarriedExtra    = 4_500_000
	maxDependents       = 3
)

// taxBracket is one Pasal 17 layer: income up to Upto (inclusive) is taxed at Rate.
type taxBracket struct {
	Upto float64
	Rate float64
}

var pasal17Brackets = []taxBracket{
	{60_000_000, 0.05},
	{250_000_000, 0.15},
	{500_000_000, 0.25},
	{5_000_000_000, 0.30},
	{math.MaxFloat64, 0.35},
}

// DefaultPTKPStatus is assumed when an employee has no (valid) PTKP status set.
const DefaultPTKPStatus = "TK/0"

// NormalizePTKPStatus upper-cases and trims a status like "k/2"; ok is false
// when it is not one of TK/0..TK/3, K/0..K/3.
func NormalizePTKPStatus(s string) (string, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) < 3 || s[len(s)-2] != '/' {
		return "", false
	}
	deps := int(s[len(s)-1] - '0')
	if deps < 0 || deps > maxDependents {
		return "", false
	}
	if s[:len(s)-2] != "TK" && s[:len(s)-2] != "K" {
		return "", false
	}
	return s, true
}

// PTKPAnnual is the annual non-taxable income (PTKP) for a status such as "K/1".
func PTKPAnnual(status string) float64 {
	s, ok := NormalizePTKPStatus(status)
	if !ok {
		s = DefaultPTKPStatus
	}
	amount := float64(ptkpBase + ptkpPerDependent*int(s[len(s)-1]-'0'))
	if s[0] == 'K' {
		amount += ptkpMarriedExtra
	}
	return amount
}

// annualTax applies the Pasal 17 progressive brackets to taxable income (PKP).
func annualTax(pkp float64) float64 {
	var tax, lower float64
	for _, b := range pasal17Brackets {
		if pkp <= lower {
			break
		}
		tax += (math.Min(pkp, b.Upto) - lower) * b.Rate
		lower = b.Upto
	}
	return tax
}

// EstimateMonthlyPPh21 returns the estimated monthly PPh 21 for a regular
// monthly gross income (base + fixed allowances + overtime), rounded to the
// nearest rupiah.
func EstimateMonthlyPPh21(monthlyGross float64, ptkpStatus string, hasNPWP bool) float64 {
	if monthlyGross <= 0 {
		return 0
	}
	annualGross := monthlyGross * 12
	jobExpense := math.Min(annualGross*jobExpenseRate, jobExpenseMaxAnnual)
	// PKP is rounded down to the nearest thousand rupiah.
	pkp := math.Floor((annualGross-jobExpense-PTKPAnnual(ptkpStatus))/1000) * 1000
	if pkp <= 0 {
		return 0
	}
	tax := annualTax(pkp)
	if !hasNPWP {
		tax *= noNPWPSurcharge
	}
	return math.Round(tax / 12)
}
