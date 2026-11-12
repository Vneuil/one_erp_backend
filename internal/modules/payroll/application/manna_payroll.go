package application

import (
	"math"
	"time"
)

// CalculateLateMinutes calculates how many minutes actual arrival is after scheduled arrival.
// Format is "15:04" or similar time strings.
func CalculateLateMinutes(scheduledTime, actualTime string) (int, bool) {
	sTime, err1 := time.Parse("15:04", scheduledTime)
	aTime, err2 := time.Parse("15:04", actualTime)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	diff := aTime.Sub(sTime)
	minutes := int(diff.Minutes())
	if minutes > 0 {
		return minutes, true
	}
	return 0, false
}

// CalculateLatePenalty computes the penalty deduction for late arrivals.
// If isLateExempt is true (e.g. Sales Kanvaser), late penalty is overridden to 0.
func CalculateLatePenalty(minutesLate int, penaltyPerIncident float64, isLateExempt bool) float64 {
	if isLateExempt || minutesLate <= 0 {
		return 0
	}
	return penaltyPerIncident
}

// CalculateProportionalAllowance computes allowance proportionally to attended days:
// (attendedDays / totalWorkDays) * baseAllowance.
func CalculateProportionalAllowance(baseAllowance float64, attendedDays, totalWorkDays int) float64 {
	if totalWorkDays <= 0 || attendedDays <= 0 {
		return 0
	}
	ratio := float64(attendedDays) / float64(totalWorkDays)
	return math.Round(baseAllowance * ratio)
}

// CalculateDailyTransportAllowance calculates total transport/fuel allowance:
// daysFieldDuty * dailyRate.
func CalculateDailyTransportAllowance(dailyRate float64, daysFieldDuty int) float64 {
	if daysFieldDuty <= 0 || dailyRate <= 0 {
		return 0
	}
	return math.Round(dailyRate * float64(daysFieldDuty))
}

// Container loading rates for Manna ERP (TC-HR-009)
const (
	Bonus40ftContainer = 200_000.0 // Rp 200,000 per 40ft container
	Bonus20ftContainer = 100_000.0 // Rp 100,000 per 20ft container
)

// CalculateContainerBonus calculates loading team bonus for container shipments.
func CalculateContainerBonus(count40ft, count20ft int) float64 {
	total := (float64(count40ft) * Bonus40ftContainer) + (float64(count20ft) * Bonus20ftContainer)
	return total
}

// Quarterly Attendance Reward (TC-HR-010)
const (
	QuarterlyAttendanceBonus = 50_000.0 // Rp 50,000
	TeladanBadge             = "Teladan"
)

// CalculateAttendanceReward determines if the employee receives the quarterly attendance bonus & badge.
func CalculateAttendanceReward(isTop3 bool) (float64, string) {
	if isTop3 {
		return QuarterlyAttendanceBonus, TeladanBadge
	}
	return 0, ""
}
