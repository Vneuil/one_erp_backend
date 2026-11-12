package application

import (
	"context"
	"strings"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

type TraceReceipt struct {
	ReceiptNo    string     `json:"receiptNo"`
	ReceivedDate string     `json:"receivedDate"`
	OrderNo      string     `json:"orderNo"`
	SupplierID   *uuid.UUID `json:"supplierId,omitempty"`
	SupplierName string     `json:"supplierName"`
	ProductID    uuid.UUID  `json:"productId"`
	Quantity     float64    `json:"quantity"`
	LotNumber    string     `json:"lotNumber"`
}

type TraceInspection struct {
	InspectionNo  string `json:"inspectionNo"`
	BatchNo       string `json:"batchNo"`
	TargetType    string `json:"targetType"`
	OverallStatus string `json:"overallStatus"`
	InspectedAt   string `json:"inspectedAt"`
	IsVoided      bool   `json:"isVoided"`
}

type TraceUsage struct {
	OrderNumber  string  `json:"orderNumber"`
	ChildBatchNo string  `json:"childBatchNo"`
	ChildProduct string  `json:"childProduct"`
	QuantityUsed float64 `json:"quantityUsed"`
	RecordedAt   string  `json:"recordedAt"`
}

type LotTrace struct {
	Lot             string                `json:"lot"`
	Found           bool                  `json:"found"`
	Receipts        []TraceReceipt        `json:"receipts"`
	ReceiptQC       []TraceInspection     `json:"receiptQc"`
	UsedIn          []TraceUsage          `json:"usedIn"`
	FinishedBatches []string              `json:"finishedBatches"`
	FinishedQC      []TraceInspection     `json:"finishedQc"` // basis of the COI per finished batch
	SupplierCerts   []CertificateResponse `json:"supplierCertificates"`
	Gaps            []string              `json:"gaps,omitempty"`
}

// TraceReader is the read side of the lot trace; it spans procurement, QC and
// manufacturing tables, so it is an interface the infrastructure layer implements.
type TraceReader interface {
	Receipts(ctx context.Context, lot string) ([]TraceReceipt, error)
	Inspections(ctx context.Context, batchNos []string) ([]TraceInspection, error)
	Usage(ctx context.Context, lot string) ([]TraceUsage, error)
	SupplierCerts(ctx context.Context, supplierIDs []uuid.UUID) ([]CertificateResponse, error)
}

// TraceLot walks a lot forward from the goods receipt to the QC records of the batches it
// ended up in. Steps that cannot be linked yet are listed in Gaps rather than guessed.
func TraceLot(ctx context.Context, r TraceReader, lot string) (*LotTrace, error) {
	lot = strings.ToUpper(strings.TrimSpace(lot))
	if lot == "" {
		return nil, apperrors.NewBadRequest("Lot number is required")
	}
	t := &LotTrace{Lot: lot, Receipts: []TraceReceipt{}, ReceiptQC: []TraceInspection{}, UsedIn: []TraceUsage{}, FinishedBatches: []string{}, FinishedQC: []TraceInspection{}, SupplierCerts: []CertificateResponse{}}

	var err error
	if t.Receipts, err = r.Receipts(ctx, lot); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to trace receipts")
	}
	t.Found = len(t.Receipts) > 0
	if t.ReceiptQC, err = r.Inspections(ctx, []string{lot}); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to trace receipt QC")
	}
	if t.UsedIn, err = r.Usage(ctx, lot); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to trace production usage")
	}
	seen := map[string]bool{}
	for _, u := range t.UsedIn {
		if !seen[u.ChildBatchNo] {
			seen[u.ChildBatchNo] = true
			t.FinishedBatches = append(t.FinishedBatches, u.ChildBatchNo)
		}
	}
	if len(t.FinishedBatches) > 0 {
		if t.FinishedQC, err = r.Inspections(ctx, t.FinishedBatches); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to trace finished batch QC")
		}
	}
	var sup []uuid.UUID
	for _, rc := range t.Receipts {
		if rc.SupplierID != nil {
			sup = append(sup, *rc.SupplierID)
		}
	}
	if len(sup) > 0 {
		if t.SupplierCerts, err = r.SupplierCerts(ctx, sup); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to load supplier certificates")
		}
	}

	if !t.Found {
		t.Gaps = append(t.Gaps, "No goods receipt carries this lot number")
	}
	if t.Found && len(t.ReceiptQC) == 0 {
		t.Gaps = append(t.Gaps, "No incoming QC inspection recorded for this lot")
	}
	if t.Found && len(t.UsedIn) == 0 {
		t.Gaps = append(t.Gaps, "Lot not yet consumed in production (no batch genealogy)")
	}
	if len(t.FinishedBatches) > 0 && len(t.FinishedQC) == 0 {
		t.Gaps = append(t.Gaps, "Finished batches have no QC inspection yet, so no COI can be issued")
	}
	t.Gaps = append(t.Gaps, "Customer shipment link is not recorded yet: deliveries do not carry batch numbers")
	return t, nil
}
