package application

import "testing"

func TestPTKPAnnual(t *testing.T) {
	cases := map[string]float64{
		"TK/0": 54_000_000, "TK/1": 58_500_000, "TK/3": 67_500_000,
		"K/0": 58_500_000, "K/1": 63_000_000, "K/3": 72_000_000,
		"k/2": 67_500_000, // case-insensitive
		"":    54_000_000, // unknown falls back to TK/0
		"K/9": 54_000_000,
	}
	for status, want := range cases {
		if got := PTKPAnnual(status); got != want {
			t.Errorf("PTKPAnnual(%q) = %.0f, want %.0f", status, got, want)
		}
	}
}

func TestEstimateMonthlyPPh21(t *testing.T) {
	cases := []struct {
		name    string
		gross   float64
		status  string
		hasNPWP bool
		want    float64
	}{
		{"below PTKP pays nothing", 4_000_000, "TK/0", true, 0},
		// 60M - 3M job expense - 54M PTKP = 3M PKP -> 150k/yr
		{"just above PTKP", 5_000_000, "TK/0", true, 12_500},
		// 120M - 6M - 54M = 60M PKP -> 5% = 3M/yr
		{"10M single", 10_000_000, "TK/0", true, 250_000},
		// 120M - 6M - 63M = 51M PKP -> 2.55M/yr
		{"10M married with one dependent", 10_000_000, "K/1", true, 212_500},
		// 360M - 6M - 54M = 300M PKP -> 3M + 28.5M + 12.5M = 44M/yr
		{"30M crosses three brackets", 30_000_000, "TK/0", true, 3_666_667},
		{"no NPWP pays 120%", 10_000_000, "TK/0", false, 300_000},
		{"zero income", 0, "TK/0", true, 0},
	}
	for _, c := range cases {
		if got := EstimateMonthlyPPh21(c.gross, c.status, c.hasNPWP); got != c.want {
			t.Errorf("%s: got %.0f, want %.0f", c.name, got, c.want)
		}
	}
}

func TestAnnualTaxTopBracket(t *testing.T) {
	// 6B PKP: 60M*5% + 190M*15% + 250M*25% + 4.5B*30% + 1B*35%
	want := 3_000_000.0 + 28_500_000 + 62_500_000 + 1_350_000_000 + 350_000_000
	if got := annualTax(6_000_000_000); got != want {
		t.Errorf("annualTax(6B) = %.0f, want %.0f", got, want)
	}
}
