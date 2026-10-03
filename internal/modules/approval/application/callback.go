package application

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DocumentStatusCallback lets the module that owns a document type (e.g.
// procurement for "purchase_order", sales for "sales_order") react when an
// ApprovalRequest for one of its documents reaches a final state through
// *any* path - whether the module's own "/approve" endpoint drove the
// workflow to completion, or an admin approved/rejected it directly from
// the generic Approval Center ("/approval/requests/:id/approve").
//
// This is what prevents a document from getting stuck at
// "pending_approval" forever: no matter which endpoint fires the final
// approval/rejection, ApprovalUseCase.ApproveStep/RejectStep invokes the
// callback registered for that document's type, and the owning module's
// callback implementation is the single place that updates the document's
// own status (and, for Sales Orders, performs the deferred stock
// deduction).
type DocumentStatusCallback interface {
	// OnApproved is invoked once, exactly when an ApprovalRequest for
	// documentID transitions to its final "approved" state (i.e. the last
	// required approval level has signed off).
	OnApproved(ctx context.Context, documentID uuid.UUID) error
	// OnRejected is invoked once, exactly when an ApprovalRequest for
	// documentID transitions to its final "rejected" state.
	OnRejected(ctx context.Context, documentID uuid.UUID) error
}

// CallbackFactory builds a DocumentStatusCallback bound to a specific
// tenant's database. Callbacks are tenant-scoped (like every other
// repository/use case in this codebase), so they cannot be constructed
// once at process startup - instead, modules register a *factory* at
// startup, and the approval HTTP handler invokes every registered factory
// with the current request's tenant DB when it builds a request-scoped
// ApprovalUseCase (see delivery/http/handler.go resolve()).
type CallbackFactory func(tenantDB *gorm.DB) DocumentStatusCallback

var (
	factoryMu         sync.RWMutex
	callbackFactories = map[string]CallbackFactory{}
)

// RegisterCallbackFactory registers the callback factory for a document
// type. Called once per module at startup (see procurement/module.go and
// sales/module.go) - NOT per-request. Registering twice for the same
// document type overwrites the previous registration, which only matters
// for tests.
func RegisterCallbackFactory(documentType string, factory CallbackFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	callbackFactories[documentType] = factory
}

// BuildCallbacks invokes every registered factory with tenantDB and
// returns the resulting document-type -> callback map, ready to hand to
// an ApprovalUseCase via RegisterCallback.
func BuildCallbacks(tenantDB *gorm.DB) map[string]DocumentStatusCallback {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	out := make(map[string]DocumentStatusCallback, len(callbackFactories))
	for docType, factory := range callbackFactories {
		out[docType] = factory(tenantDB)
	}
	return out
}
