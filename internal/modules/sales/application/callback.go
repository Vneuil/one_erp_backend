package application

import (
	"context"

	approvalApp "github.com/divinecoid/one-backend/internal/modules/approval/application"
	"github.com/google/uuid"
)

// salesOrderApprovalCallback implements approvalApp.DocumentStatusCallback
// for the "sales_order" document type. It is registered with the approval
// module (see module.go) so that a Sales Order approved or rejected from
// the generic Approval Center gets its status updated - and, on approval,
// its stock deducted - exactly the same way ApproveOrder/RejectOrder would
// do it, through the very same Finalize* methods, so there is exactly one
// place that mutates order status or touches inventory for this workflow.
type salesOrderApprovalCallback struct {
	uc SalesUseCase
}

// NewSalesOrderApprovalCallback is exported so module.go can construct it
// from a tenant-scoped SalesUseCase.
func NewSalesOrderApprovalCallback(uc SalesUseCase) approvalApp.DocumentStatusCallback {
	return &salesOrderApprovalCallback{uc: uc}
}

func (c *salesOrderApprovalCallback) OnApproved(ctx context.Context, documentID uuid.UUID) error {
	return c.uc.FinalizeOrderApproved(ctx, documentID)
}

func (c *salesOrderApprovalCallback) OnRejected(ctx context.Context, documentID uuid.UUID) error {
	return c.uc.FinalizeOrderRejected(ctx, documentID)
}

// SalesOrderDocumentType exposes salesOrderDocumentType for registration in
// module.go (RegisterCallbackFactory needs the exact string SubmitDocument
// used).
const SalesOrderDocumentType = salesOrderDocumentType
