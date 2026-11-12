package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeTrace struct{ withUsage bool }

func (f fakeTrace) Receipts(_ context.Context, lot string) ([]TraceReceipt, error) {
	if lot != "LOT-1" {
		return nil, nil
	}
	s := uuid.New()
	return []TraceReceipt{{ReceiptNo: "GR-1", LotNumber: lot, SupplierID: &s}}, nil
}
func (f fakeTrace) Inspections(_ context.Context, b []string) ([]TraceInspection, error) {
	if b[0] == "LOT-1" {
		return []TraceInspection{{InspectionNo: "QC-1", BatchNo: "LOT-1"}}, nil
	}
	return nil, nil
}
func (f fakeTrace) Usage(context.Context, string) ([]TraceUsage, error) {
	if f.withUsage {
		return []TraceUsage{{ChildBatchNo: "FG-9"}, {ChildBatchNo: "FG-9"}}, nil
	}
	return nil, nil
}
func (f fakeTrace) SupplierCerts(context.Context, []uuid.UUID) ([]CertificateResponse, error) {
	return nil, nil
}

func TestTraceLotFollowsForwardAndReportsGaps(t *testing.T) {
	tr, err := TraceLot(context.Background(), fakeTrace{withUsage: true}, " lot-1 ")
	if err != nil || !tr.Found || len(tr.ReceiptQC) != 1 {
		t.Fatalf("receipt and QC must be linked: %v %+v", err, tr)
	}
	if len(tr.FinishedBatches) != 1 || tr.FinishedBatches[0] != "FG-9" {
		t.Fatalf("finished batches must be de-duplicated: %+v", tr.FinishedBatches)
	}
	hasCOIGap := false
	for _, g := range tr.Gaps {
		if g == "Finished batches have no QC inspection yet, so no COI can be issued" {
			hasCOIGap = true
		}
	}
	if !hasCOIGap {
		t.Fatalf("missing COI gap: %v", tr.Gaps)
	}
	if nf, _ := TraceLot(context.Background(), fakeTrace{}, "nope"); nf.Found {
		t.Fatal("unknown lot must not be found")
	}
	if _, err := TraceLot(context.Background(), fakeTrace{}, "  "); err == nil {
		t.Fatal("empty lot is a bad request")
	}
}
