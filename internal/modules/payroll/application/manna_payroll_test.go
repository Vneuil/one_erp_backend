package application

import (
	"testing"
)

func TestTC_HR_003_DailyLateCalculation(t *testing.T) {
	// TC-HR-003: Scheduled 08:00, ClockIn 08:30 -> 30 mins late
	scheduled := "08:00"
	actual := "08:30"
	lateMinutes, isLate := CalculateLateMinutes(scheduled, actual)
	if !isLate {
		t.Fatalf("expected isLate to be true")
	}
	if lateMinutes != 30 {
		t.Fatalf("expected 30 minutes late, got %d", lateMinutes)
	}

	penalty := CalculateLatePenalty(lateMinutes, 25_000, false)
	if penalty != 25_000 {
		t.Fatalf("expected penalty 25000, got %f", penalty)
	}
}

func TestTC_HR_004_LateExemptionOverride(t *testing.T) {
	// TC-HR-004: Anton Sales Kanvaser arrives at 08:30 (30 mins late),
	// but Flag Bebas Potongan (isLateExempt) is active -> penalty must be 0.
	lateMinutes, isLate := CalculateLateMinutes("08:00", "08:30")
	if !isLate || lateMinutes != 30 {
		t.Fatalf("expected 30m late, got %d, isLate=%v", lateMinutes, isLate)
	}

	penalty := CalculateLatePenalty(lateMinutes, 25_000, true) // isLateExempt = true
	if penalty != 0 {
		t.Fatalf("expected penalty 0 due to exemption flag, got %f", penalty)
	}
}

func TestTC_HR_007_ProportionalFunctionalAllowance(t *testing.T) {
	// TC-HR-007: Tunjangan Rp 1,000,000. Total work days = 20, Attended = 19 days.
	// Expected = 19/20 * 1,000,000 = Rp 950,000
	baseAllowance := 1_000_000.0
	attendedDays := 19
	totalWorkDays := 20

	result := CalculateProportionalAllowance(baseAllowance, attendedDays, totalWorkDays)
	if result != 950_000.0 {
		t.Fatalf("expected 950000.0, got %f", result)
	}
}

func TestTC_HR_008_CanvaserFuelAllowance(t *testing.T) {
	// TC-HR-008: 15 days field duty, rate Rp 25,000/day -> Rp 375,000
	dailyRate := 25_000.0
	days := 15

	result := CalculateDailyTransportAllowance(dailyRate, days)
	if result != 375_000.0 {
		t.Fatalf("expected 375000.0, got %f", result)
	}
}

func TestTC_HR_009_ContainerTieredBonus(t *testing.T) {
	// TC-HR-009: 2 x 40ft (@ 200,000) + 3 x 20ft (@ 100,000) = Rp 700,000
	count40 := 2
	count20 := 3

	result := CalculateContainerBonus(count40, count20)
	expected := (2 * 200_000.0) + (3 * 100_000.0) // 400k + 300k = 700k
	if result != expected {
		t.Fatalf("expected %f, got %f", expected, result)
	}
}

func TestTC_HR_010_QuarterlyAttendanceReward(t *testing.T) {
	// TC-HR-010: Top 3 perfect attendance -> Rp 50,000 + "Teladan" badge
	bonus, badge := CalculateAttendanceReward(true)
	if bonus != 50_000.0 {
		t.Fatalf("expected bonus 50000, got %f", bonus)
	}
	if badge != "Teladan" {
		t.Fatalf("expected badge 'Teladan', got '%s'", badge)
	}

	// Non top 3
	bonusZero, badgeEmpty := CalculateAttendanceReward(false)
	if bonusZero != 0 || badgeEmpty != "" {
		t.Fatalf("expected 0 and empty badge for non top 3, got %f, '%s'", bonusZero, badgeEmpty)
	}
}

