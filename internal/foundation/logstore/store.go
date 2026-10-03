// Package logstore keeps a bounded, in-memory ring buffer of recent server
// errors (500s and panics) so an admin can see "what just broke" from the
// app itself, without shell access to the server. It is deliberately not
// a general request logger and not backed by a database - see Store's doc
// comment for why, and the known limitations that follow from that.
package logstore

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Entry is one captured server error.
type Entry struct {
	ID        string     `json:"id"`
	Time      time.Time  `json:"time"`
	Status    int        `json:"status"`
	Method    string     `json:"method"`
	Path      string     `json:"path"`
	Message   string     `json:"message"`
	RequestID string     `json:"requestId,omitempty"`
	CompanyID *uuid.UUID `json:"-"` // never serialized - see List's filtering
	UserEmail string     `json:"userEmail,omitempty"`
}

const capacity = 500

// store is a fixed-capacity ring buffer. A plain mutex-guarded slice is
// used rather than a channel/lock-free structure - error volume is low
// enough (hopefully!) that a mutex is never a bottleneck, and this keeps
// List's "most recent first" slicing trivial.
type store struct {
	mu      sync.Mutex
	entries []Entry // oldest first
}

var global = &store{}

// Record appends one entry, evicting the oldest if at capacity. Safe for
// concurrent use from request-handling goroutines.
func Record(e Entry) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	global.mu.Lock()
	defer global.mu.Unlock()
	global.entries = append(global.entries, e)
	if len(global.entries) > capacity {
		global.entries = global.entries[len(global.entries)-capacity:]
	}
}

// List returns up to limit entries, most recent first.
//
// forCompanyID scopes the result the same way every tenant-facing query
// elsewhere in this codebase is scoped: a company only ever sees its own
// entries. Entries with no CompanyID (errors that happened before a JWT
// was parsed - a malformed Authorization header, a request to a public
// route) are platform-level, not any one company's business, and are
// excluded from every company-scoped view rather than shown to the first
// admin who happens to ask.
func List(forCompanyID uuid.UUID, limit int) []Entry {
	global.mu.Lock()
	defer global.mu.Unlock()

	out := make([]Entry, 0, limit)
	for i := len(global.entries) - 1; i >= 0 && len(out) < limit; i-- {
		e := global.entries[i]
		if e.CompanyID == nil || *e.CompanyID != forCompanyID {
			continue
		}
		out = append(out, e)
	}
	return out
}
