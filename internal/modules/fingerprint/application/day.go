package application

import (
	"sort"
	"time"

	hrmDomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
)

const (
	// workStartHour matches live clock-in: at or after 09:00 is "Late".
	workStartHour = 9
	// minWorkSpan is the shortest gap between the first and last punch that counts as a clock-out;
	// anything closer is a double tap on the machine.
	minWorkSpan = 5 * time.Minute
)

// MergeDay builds an employee's attendance for one day from all of that day's
// punches: the first is the clock-in and the last (if it is at least
// minWorkSpan later) the clock-out. An existing record, for example a GPS
// clock-in, is kept and only widened by the punches — never narrowed — so the
// result does not depend on the order punches arrive in. It reports whether
// the record changed.
func MergeDay(existing *hrmDomain.Attendance, punches []time.Time, nip, name, date, location string) (*hrmDomain.Attendance, bool) {
	if len(punches) == 0 {
		return existing, false
	}
	sort.Slice(punches, func(i, j int) bool { return punches[i].Before(punches[j]) })
	first, last := punches[0].In(Zone), punches[len(punches)-1].In(Zone)

	att := existing
	created := att == nil
	if created {
		att = &hrmDomain.Attendance{EmployeeName: name, NIP: nip, Date: date, ClockOut: "--:--", Location: location, Method: "Fingerprint"}
	}
	before := *att

	if att.ClockInAt == nil || first.Before(*att.ClockInAt) {
		in := first
		att.ClockInAt = &in
		att.ClockIn = in.Format("15:04 WIB")
		att.Status = "On Time"
		if in.Hour() >= workStartHour {
			att.Status = "Late"
		}
	}
	if last.Sub(*att.ClockInAt) >= minWorkSpan && (att.ClockOutAt == nil || last.After(*att.ClockOutAt)) {
		out := last
		att.ClockOutAt = &out
		att.ClockOut = out.Format("15:04 WIB")
	}
	if att.ClockOutAt != nil && att.ClockOutAt.After(*att.ClockInAt) {
		att.WorkMinutes = int(att.ClockOutAt.Sub(*att.ClockInAt).Minutes())
	}
	changed := created || att.ClockIn != before.ClockIn || att.ClockOut != before.ClockOut || att.WorkMinutes != before.WorkMinutes || att.Status != before.Status
	return att, changed
}
