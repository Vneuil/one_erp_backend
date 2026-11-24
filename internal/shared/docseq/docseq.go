// Package docseq provides centralized, atomic, per-tenant monthly document numbering
// in the standard format: <PREFIX>-<SEQ>-<MMYY> (e.g. PO-001-1126, SO-013-1126).
package docseq

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"gorm.io/gorm"
)

var jakartaLoc = loadJakarta()

func loadJakarta() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}

// DocumentSequence tracks the monotonically increasing sequence per (tenant, prefix, MMYY).
type DocumentSequence struct {
	TenantKey  string    `gorm:"type:varchar(36);primaryKey"`
	Prefix     string    `gorm:"type:varchar(20);primaryKey"`
	PeriodMMYY string    `gorm:"type:varchar(4);primaryKey"`
	LastSeq    int64     `gorm:"not null;default:0"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (DocumentSequence) TableName() string {
	return "document_sequences"
}

// AutoMigrate creates or updates the document_sequences table on db.
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(&DocumentSequence{})
}

type dbContextKey struct{}
type memContextKey struct{}

// WithDB attaches a GORM database handle to ctx so docseq.Next can persist sequences atomically.
func WithDB(ctx context.Context, db *gorm.DB) context.Context {
	if db == nil {
		return ctx
	}
	return context.WithValue(ctx, dbContextKey{}, db)
}

// DBFromContext retrieves the GORM database attached to ctx, or nil if absent.
func DBFromContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return nil
	}
	db, _ := ctx.Value(dbContextKey{}).(*gorm.DB)
	return db
}

// MemorySequencer provides an isolated in-memory sequence store for unit tests.
type MemorySequencer struct {
	mu   sync.Mutex
	seqs map[string]int64
}

// NewMemorySequencer creates a fresh in-memory sequence counter.
func NewMemorySequencer() *MemorySequencer {
	return &MemorySequencer{seqs: make(map[string]int64)}
}

func (m *MemorySequencer) next(tenantKey, prefix, mmyy string) int64 {
	k := tenantKey + "|" + prefix + "|" + mmyy
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seqs[k]++
	return m.seqs[k]
}

// WithMemory attaches an explicit in-memory sequencer to ctx (useful in tests).
func WithMemory(ctx context.Context, m *MemorySequencer) context.Context {
	return context.WithValue(ctx, memContextKey{}, m)
}

// MemoryFromContext returns the in-memory sequencer attached to ctx, if any.
func MemoryFromContext(ctx context.Context) *MemorySequencer {
	if ctx == nil {
		return nil
	}
	m, _ := ctx.Value(memContextKey{}).(*MemorySequencer)
	return m
}

// fallbackRepos isolates in-memory counters per mock repository pointer when no DB or MemorySequencer is attached.
var (
	fallbackMu    sync.Mutex
	fallbackStore = make(map[uintptr]*MemorySequencer)
	defaultMem    = NewMemorySequencer()
)

func sequencerForScope(scope any) *MemorySequencer {
	if scope == nil {
		return defaultMem
	}
	v := reflect.ValueOf(scope)
	if v.Kind() == reflect.Ptr && !v.IsNil() && v.Elem().Type().Size() > 0 {
		ptr := v.Pointer()
		fallbackMu.Lock()
		defer fallbackMu.Unlock()
		seq, ok := fallbackStore[ptr]
		if !ok {
			seq = NewMemorySequencer()
			fallbackStore[ptr] = seq
		}
		return seq
	}
	return defaultMem
}

// ParseDate parses YYYY-MM-DD, YYYY-MM, or RFC3339 timestamps into Asia/Jakarta time,
// falling back to the current time in Asia/Jakarta when empty or invalid.
func ParseDate(dateStr string) time.Time {
	s := strings.TrimSpace(dateStr)
	if s != "" {
		for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05", "2006-01"} {
			if t, err := time.ParseInLocation(layout, s, jakartaLoc); err == nil {
				return t
			}
		}
	}
	return time.Now().In(jakartaLoc)
}

// Format builds the canonical document number: <PREFIX>-<SEQ>-<MMYY>.
// Sequence numbers 1..999 are zero-padded to 3 digits; >=1000 expand naturally.
func Format(prefix string, seq int64, t time.Time) string {
	p := strings.ToUpper(strings.TrimSpace(prefix))
	mmyy := t.In(jakartaLoc).Format("0106")
	return fmt.Sprintf("%s-%03d-%s", p, seq, mmyy)
}

// NextSeq allocates the next sequence number for (tenant, prefix, MMYY) and returns
// both the raw integer sequence and the formatted <PREFIX>-<SEQ>-<MMYY> string.
func NextSeq(ctx context.Context, scope any, prefix string, dateStr string) (int64, string, error) {
	p := strings.ToUpper(strings.TrimSpace(prefix))
	if p == "" {
		return 0, "", fmt.Errorf("docseq: prefix is required")
	}
	t := ParseDate(dateStr)
	mmyy := t.In(jakartaLoc).Format("0106")

	tenantKey := "default"
	if tid := tenantctx.FromContext(ctx); tid != nil {
		tenantKey = tid.String()
	}

	var seq int64
	if db := DBFromContext(ctx); db != nil {
		err := db.WithContext(ctx).Raw(`
			INSERT INTO document_sequences (tenant_key, prefix, period_mmyy, last_seq, updated_at)
			VALUES (?, ?, ?, 1, NOW())
			ON CONFLICT (tenant_key, prefix, period_mmyy)
			DO UPDATE SET last_seq = document_sequences.last_seq + 1, updated_at = NOW()
			RETURNING last_seq
		`, tenantKey, p, mmyy).Scan(&seq).Error
		if err != nil {
			return 0, "", fmt.Errorf("docseq: allocate %s-%s: %w", p, mmyy, err)
		}
		return seq, fmt.Sprintf("%s-%03d-%s", p, seq, mmyy), nil
	}

	if mem := MemoryFromContext(ctx); mem != nil {
		seq = mem.next(tenantKey, p, mmyy)
		return seq, fmt.Sprintf("%s-%03d-%s", p, seq, mmyy), nil
	}

	seq = sequencerForScope(scope).next(tenantKey, p, mmyy)
	return seq, fmt.Sprintf("%s-%03d-%s", p, seq, mmyy), nil
}

// Next atomically allocates the next sequence number for (tenant, prefix, MMYY)
// and formats it as <PREFIX>-<SEQ>-<MMYY>.
//
// When ctx carries a *gorm.DB (via WithDB, set by TenantContext and seeders),
// it executes an atomic PostgreSQL UPSERT on document_sequences.
// Otherwise (unit tests with mock repositories), it falls back to an isolated
// in-memory counter scoped to ctx or the repository pointer.
func Next(ctx context.Context, scope any, prefix string, dateStr string) (string, error) {
	_, formatted, err := NextSeq(ctx, scope, prefix, dateStr)
	return formatted, err
}
