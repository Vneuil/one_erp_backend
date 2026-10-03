package domain

import "testing"

func TestAdvanceSchedule(t *testing.T) {
	a := CashAdvance{Amount: 1_000_000, Installments: 3, StartPeriod: "2026-11", Status: "approved"}
	cases := map[string]float64{"2026-10": 0, "2026-11": 333_333.33, "2026-12": 333_333.33, "2027-01": 333_333.34, "2027-02": 0}
	var sum float64
	for p, want := range cases {
		got := a.InstallmentFor(p)
		if got != want {
			t.Errorf("%s: got %v want %v", p, got, want)
		}
		sum += got
	}
	if cents(sum) != 1_000_000 {
		t.Errorf("installments add up to %v, want the full amount", sum)
	}
	if a.OutstandingAfter("2026-10") != 1_000_000 || a.OutstandingAfter("2026-11") != 666_666.67 || a.OutstandingAfter("2027-01") != 0 {
		t.Errorf("outstanding: %v %v %v", a.OutstandingAfter("2026-10"), a.OutstandingAfter("2026-11"), a.OutstandingAfter("2027-01"))
	}
	pending := a
	pending.Status = "pending"
	if pending.InstallmentFor("2026-11") != 0 || pending.OutstandingAfter("2026-11") != 0 {
		t.Error("only approved advances withhold anything")
	}
	if (CashAdvance{Status: "approved", StartPeriod: "bad", Installments: 2, Amount: 10}).InstallmentFor("2026-01") != 0 {
		t.Error("malformed period must not withhold")
	}
}
