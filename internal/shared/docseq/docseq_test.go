package docseq

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/google/uuid"
)

type dummyRepo struct{ id int }

func TestFormatAndMonthlyReset(t *testing.T) {
	ctx := WithMemory(context.Background(), NewMemorySequencer())
	repo := &dummyRepo{id: 1}

	n1, err := Next(ctx, repo, "po", "2026-11-24")
	if err != nil || n1 != "PO-001-1126" {
		t.Fatalf("got %q, err=%v", n1, err)
	}
	n2, err := Next(ctx, repo, "PO", "2026-11-25")
	if err != nil || n2 != "PO-002-1126" {
		t.Fatalf("got %q, err=%v", n2, err)
	}
	// Monthly reset in December 2026
	nDec, err := Next(ctx, repo, "PO", "2026-12-01")
	if err != nil || nDec != "PO-001-1226" {
		t.Fatalf("expected reset in 1226, got %q, err=%v", nDec, err)
	}
	// Separate prefix maintains independent sequence
	so1, err := Next(ctx, repo, "SO", "2026-11-24")
	if err != nil || so1 != "SO-001-1126" {
		t.Fatalf("got %q, err=%v", so1, err)
	}
}

func TestFormatExpandsPast999(t *testing.T) {
	ts := time.Date(2026, 11, 24, 10, 0, 0, 0, jakartaLoc)
	if got := Format("LHP", 23, ts); got != "LHP-023-1126" {
		t.Fatalf("got %q", got)
	}
	if got := Format("LHP", 1000, ts); got != "LHP-1000-1126" {
		t.Fatalf("got %q", got)
	}
}

func TestTenantIsolationAndConcurrency(t *testing.T) {
	mem := NewMemorySequencer()
	t1 := uuid.New()
	t2 := uuid.New()
	ctx1 := tenantctx.WithTenantID(WithMemory(context.Background(), mem), &t1)
	ctx2 := tenantctx.WithTenantID(WithMemory(context.Background(), mem), &t2)

	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := Next(ctx1, nil, "SO", "2026-11-24")
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			if seen[n] {
				t.Errorf("duplicate number %s", n)
			}
			seen[n] = true
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(seen) != 50 {
		t.Fatalf("expected 50 unique numbers, got %d", len(seen))
	}

	// Second tenant starts at 001 independently
	nT2, err := Next(ctx2, nil, "SO", "2026-11-24")
	if err != nil || nT2 != "SO-001-1126" {
		t.Fatalf("expected SO-001-1126 for tenant 2, got %q", nT2)
	}
}
