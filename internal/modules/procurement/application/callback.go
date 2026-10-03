package application

import (
	"context"

	approvalApp "github.com/divinecoid/one-backend/internal/modules/approval/application"
	"github.com/google/uuid"
)

// purchaseOrderApprovalCallback implements approvalApp.DocumentStatusCallback
// for the "purchase_order" document type. It is registered with the
// approval module (see module.go) so that a Purchase Order approved or
// rejected from the generic Approval Center gets its status updated exactly
// the same way ApprovePurchaseOrder/RejectPurchaseOrder would do it -
// through the very same Finalize* methods, so there is exactly one place
// that mutates PO status.
type purchaseOrderApprovalCallback struct {
	uc ProcurementUseCase
}

// NewPurchaseOrderApprovalCallback is exported so module.go can construct
// it from a tenant-scoped ProcurementUseCase.
func NewPurchaseOrderApprovalCallback(uc ProcurementUseCase) approvalApp.DocumentStatusCallback {
	return &purchaseOrderApprovalCallback{uc: uc}
}

func (c *purchaseOrderApprovalCallback) OnApproved(ctx context.Context, documentID uuid.UUID) error {
	return c.uc.FinalizePurchaseOrderApproved(ctx, documentID)
}

func (c *purchaseOrderApprovalCallback) OnRejected(ctx context.Context, documentID uuid.UUID) error {
	return c.uc.FinalizePurchaseOrderRejected(ctx, documentID)
}

// PurchaseOrderDocumentType exposes purchaseOrderDocumentType for
// registration in module.go (RegisterCallbackFactory needs the exact
// string SubmitDocument used).
const PurchaseOrderDocumentType = purchaseOrderDocumentType
