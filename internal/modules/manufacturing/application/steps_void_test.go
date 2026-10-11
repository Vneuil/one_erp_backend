package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/google/uuid"
)

func (f *fixture) lastLog(o *domain.ProductionOrder, step int) uuid.UUID {
	for i := len(f.repo.logs) - 1; i >= 0; i-- {
		if f.repo.logs[i].StepID == o.Steps[step].ID && !f.repo.logs[i].Voided {
			return f.repo.logs[i].ID
		}
	}
	return uuid.Nil
}

func TestVoidStepLogRollsBackTheStepAndReversesTheCost(t *testing.T) {
	f := costedFixture(t)
	ctx := context.Background()
	o := f.newOrder(t, 10, "")
	f.release(o)
	if err := f.log(t, o, 0, 10); err != nil {
		t.Fatal(err)
	}
	if err := f.log(t, o, 1, 6); err != nil {
		t.Fatal(err)
	}
	jahit := f.lastLog(o, 1)

	if _, err := f.uc.VoidStepLog(ctx, o.ID, jahit, " "); err == nil {
		t.Fatal("a reason is required")
	}
	if _, err := f.uc.VoidStepLog(ctx, o.ID, uuid.New(), "x"); err == nil {
		t.Fatal("unknown log")
	}
	if _, err := f.uc.VoidStepLog(ctx, o.ID, jahit, "salah hitung"); err != nil {
		t.Fatal(err)
	}
	if o.Steps[1].QuantityDone != 0 || o.Steps[1].Status != domain.StepPending {
		t.Fatalf("Jahit should be back to pending with 0 done: %+v", o.Steps[1])
	}
	rev := f.ledger.entries[conversionVoidSourcePrefix+jahit.String()]
	orig := f.ledger.entries[conversionSourcePrefix+jahit.String()]
	if len(rev) != 3 || len(orig) != 3 {
		t.Fatalf("entries: orig %v rev %v", orig, rev)
	}
	for i := range orig {
		if rev[i].AccountCode != orig[i].AccountCode || rev[i].Debit != orig[i].Credit || rev[i].Credit != orig[i].Debit {
			t.Fatalf("line %d is not the mirror image: %+v vs %+v", i, rev[i], orig[i])
		}
	}
	if _, err := f.uc.VoidStepLog(ctx, o.ID, jahit, "lagi"); err == nil {
		t.Fatal("a log can only be voided once")
	}
	// A voided log no longer counts in the process report and can be entered again.
	logs, _ := f.repo.ListStepLogsAll()
	var live int
	for _, l := range logs {
		if !l.Voided {
			live += l.Quantity
		}
	}
	if live != 10 {
		t.Fatalf("live units %d", live)
	}
	if err := f.log(t, o, 1, 10); err != nil {
		t.Fatalf("after the void, Jahit can take all 10 again: %v", err)
	}
}

func TestVoidStepLogRefusedWhenLaterWorkDependsOnIt(t *testing.T) {
	f := newFixture(t) // Potong -> Jahit -> QC
	ctx := context.Background()
	o := f.newOrder(t, 10, "")
	f.release(o)
	if err := f.log(t, o, 0, 10); err != nil {
		t.Fatal(err)
	}
	potong := f.lastLog(o, 0)
	if err := f.log(t, o, 1, 8); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.VoidStepLog(ctx, o.ID, potong, "salah"); err == nil || !strings.Contains(err.Error(), "already taken") {
		t.Fatalf("Jahit took 8 units, so Potong cannot drop to 0: %v", err)
	}
	if o.Steps[0].QuantityDone != 10 {
		t.Fatal("a refused void must change nothing")
	}
	if err := f.log(t, o, 2, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 4}); err != nil {
		t.Fatal(err)
	}
	qc := f.lastLog(o, 2)
	if _, err := f.uc.VoidStepLog(ctx, o.ID, qc, "salah"); err == nil || !strings.Contains(err.Error(), "already completed") {
		t.Fatalf("4 units finished from the last step: %v", err)
	}
}

func TestVoidStepLogOnlyWhileTheOrderIsRunning(t *testing.T) {
	f := newFixture(t)
	o := f.newOrder(t, 5, "")
	f.release(o)
	if err := f.log(t, o, 0, 5); err != nil {
		t.Fatal(err)
	}
	l := f.lastLog(o, 0)
	o.Status = domain.OrderCancelled
	if _, err := f.uc.VoidStepLog(context.Background(), o.ID, l, "x"); err == nil {
		t.Fatal("a cancelled order's logs are frozen")
	}
}
