// Package tenantctx threads the active Tenant ID (see modules/workspace)
// through context.Context, the same way request-scoped values like
// request IDs are usually carried - so tenant-aware repositories can read
// it without every usecase method needing an extra parameter threaded
// through its whole call chain. A nil ID (the normal case for a company
// that never created more than its default Tenant) means "no filter".
package tenantctx

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contextKey struct{}

var tenantIDKey = contextKey{}

// WithTenantID returns a new context carrying the active tenant ID.
func WithTenantID(ctx context.Context, tenantID *uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// FromContext returns the active tenant ID, or nil if none is set.
func FromContext(ctx context.Context) *uuid.UUID {
	v, ok := ctx.Value(tenantIDKey).(*uuid.UUID)
	if !ok {
		return nil
	}
	return v
}

// Scope filters a query to the active tenant when the request carries one.
// A nil tenant ID (companies that never created more than their seeded
// default Tenant) applies no filter, so single-tenant companies see every
// row exactly as before this feature existed. The target model must have a
// `tenant_id` column (every business entity does, via TenantID *uuid.UUID).
func Scope(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tid := FromContext(ctx); tid != nil {
		return db.Where("tenant_id = ?", *tid)
	}
	return db
}

// SetTenantID assigns the active tenant ID to entity.TenantID if it isn't
// already set explicitly by the caller. Call this right before creating any
// tenant-aware entity.
func SetTenantID(ctx context.Context, tenantID **uuid.UUID) {
	if *tenantID == nil {
		*tenantID = FromContext(ctx)
	}
}
