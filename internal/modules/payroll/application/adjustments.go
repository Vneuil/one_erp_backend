package application

import (
	"context"
	"math"
	"strings"
	"time"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	leavedomain "github.com/divinecoid/one-backend/internal/modules/leave/domain"
	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
)

// isUnpaidLeaveType reports whether a leave type withholds pay. Leave types are
// free text, so match the wordings used in practice (English and Indonesian).
func isUnpaidLeaveType(t string) bool {
	t = strings.ToLower(t)
	for _, k := range []string{"unpaid", "tidak dibayar", "tanpa gaji", "tanpa upah"} {
		if strings.Contains(t, k) {
			return true
		}
	}
	return false
}

// weekdaysInPeriod counts Monday-Friday days of the approved range [start, end]
// (YYYY-MM-DD) that fall inside period (YYYY-MM).
func weekdaysInPeriod(start, end, period string) int {
	s, err1 := time.Parse("2006-01-02", start)
	e, err2 := time.Parse("2006-01-02", end)
	if err1 != nil || err2 != nil || e.Before(s) {
		return 0
	}
	n := 0
	for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
		if d.Format("2006-01") != period {
			continue
		}
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n++
		}
	}
	return n
}

// UnpaidLeaveDays totals the weekdays of approved unpaid leave inside period.
func UnpaidLeaveDays(leaves []leavedomain.LeaveRequest, period string) int {
	total := 0
	for _, l := range leaves {
		if l.Status != "approved" && !strings.EqualFold(l.Status, "Approved") {
			continue
		}
		if !isUnpaidLeaveType(l.Type) {
			continue
		}
		total += weekdaysInPeriod(l.StartDate, l.EndDate, period)
	}
	return total
}

// LateCount counts attendance records flagged as late.
func LateCount(records []hrmdomain.Attendance) int {
	n := 0
	for _, a := range records {
		if strings.EqualFold(a.Status, "Late") {
			n++
		}
	}
	return n
}

// AttendanceAdjustments returns the unpaid-leave withholding and late penalty
// for one entry. Withholding is base/workDays per unpaid day, never more than
// the base salary.
func AttendanceAdjustments(baseSalary float64, p domain.PayrollPolicy, unpaidDays, lateCount int) (absence, late float64) {
	workDays := p.WorkDaysPerMonth
	if workDays <= 0 {
		workDays = 22
	}
	if p.DeductUnpaidLeave && unpaidDays > 0 {
		absence = math.Min(baseSalary, math.Round(baseSalary/float64(workDays)*float64(unpaidDays)))
	}
	if lateCount > 0 && p.LatePenaltyPerIncident > 0 {
		late = p.LatePenaltyPerIncident * float64(lateCount)
	}
	return absence, late
}

// OvertimeSource lists, per approved overtime record of an employee in a period
// (YYYY-MM), the minutes worked. Implemented by the HR-operations repository.
type OvertimeSource interface {
	ApprovedOvertimeMinutes(ctx context.Context, nip, period string) ([]int, error)
}

// monthlyWorkHours is the statutory divisor for an hourly rate (Kepmenaker
// 102/2004: monthly wage / 173).
const monthlyWorkHours = 173

// OvertimePayFor prices overtime under the Indonesian weekday scheme: the first
// hour of each record at 1.5x the hourly rate and every further hour at 2x.
func OvertimePayFor(baseSalary float64, recordMinutes []int) float64 {
	hourly := baseSalary / monthlyWorkHours
	var total float64
	for _, m := range recordMinutes {
		if m <= 0 {
			continue
		}
		hours := float64(m) / 60
		first := math.Min(hours, 1)
		total += hourly * (first*1.5 + (hours-first)*2)
	}
	return math.Round(total)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
