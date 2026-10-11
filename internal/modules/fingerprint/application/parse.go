package application

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Zone is the timezone machine clocks are assumed to be set to.
var Zone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}()

// RawPunch is one event as read from a machine or file.
type RawPunch struct {
	PIN    string
	At     time.Time
	Verify int
	State  int
}

// ParseTime reads a machine timestamp: RFC 3339, or "YYYY-MM-DD HH:MM[:SS]"
// (also with a "T") in the machine's local (WIB) time.
func ParseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04", "02/01/2006 15:04:05", "02/01/2006 15:04"} {
		if t, err := time.ParseInLocation(layout, s, Zone); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time %q", s)
}

// ParseATTLOG parses the body of an ADMS "table=ATTLOG" upload: one punch per
// line, tab separated — PIN, time, status, verify mode, ... Lines it cannot
// read are reported, not fatal.
func ParseATTLOG(body string) (punches []RawPunch, problems []string) {
	for i, line := range strings.Split(strings.ReplaceAll(body, "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 2 {
			f = strings.Fields(line)
		}
		if len(f) < 2 {
			problems = append(problems, fmt.Sprintf("line %d: too few fields", i+1))
			continue
		}
		stamp, rest := f[1], f[2:]
		// Some firmware splits date and time into two fields.
		if len(rest) > 0 && len(stamp) == 10 && strings.Count(rest[0], ":") >= 1 {
			stamp, rest = stamp+" "+rest[0], rest[1:]
		}
		at, err := ParseTime(stamp)
		if err != nil {
			problems = append(problems, fmt.Sprintf("line %d: %v", i+1, err))
			continue
		}
		p := RawPunch{PIN: strings.TrimSpace(f[0]), At: at}
		if len(rest) > 0 {
			p.State, _ = strconv.Atoi(strings.TrimSpace(rest[0]))
		}
		if len(rest) > 1 {
			p.Verify, _ = strconv.Atoi(strings.TrimSpace(rest[1]))
		}
		if p.PIN == "" {
			problems = append(problems, fmt.Sprintf("line %d: empty PIN", i+1))
			continue
		}
		punches = append(punches, p)
	}
	return punches, problems
}
