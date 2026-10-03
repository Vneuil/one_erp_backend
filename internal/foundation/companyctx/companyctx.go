// Package companyctx threads the caller's Company ID through
// context.Context, for the small number of modules (collaboration,
// docflow) that store every company's data in the single shared
// control-plane database rather than a per-company tenant database (see
// foundation/tenant). Those modules have no database-level isolation
// between companies, so every query MUST filter by CompanyID - unlike
// tenantctx's TenantID (where a nil value safely means "no filter" because
// the surrounding database is already scoped to one company), a nil/missing
// CompanyID here must never be treated as "show everything": callers should
// reject the request instead.
package companyctx

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contextKey struct{}

var companyIDKey = contextKey{}

// WithCompanyID returns a new context carrying the caller's Company ID.
func WithCompanyID(ctx context.Context, companyID *uuid.UUID) context.Context {
	return context.WithValue(ctx, companyIDKey, companyID)
}

// FromContext returns the caller's Company ID, or nil if none is set.
func FromContext(ctx context.Context) *uuid.UUID {
	v, ok := ctx.Value(companyIDKey).(*uuid.UUID)
	if !ok {
		return nil
	}
	return v
}

// Scope filters a query to the caller's company. Unlike tenantctx.Scope, a
// missing Company ID does NOT mean "no filter" - it forces a query that
// matches nothing (company_id = a nil UUID), so a bug that forgets to set
// the company context fails closed (empty result) instead of open
// (every company's data).
func Scope(ctx context.Context, db *gorm.DB) *gorm.DB {
	id := FromContext(ctx)
	if id == nil {
		return db.Where("company_id = ?", uuid.Nil)
	}
	return db.Where("company_id = ?", *id)
}

// SetCompanyID assigns the caller's Company ID to entity.CompanyID if it
// isn't already set explicitly by the caller. Call this right before
// creating any company-scoped entity in modules that share one database
// across all companies.
func SetCompanyID(ctx context.Context, companyID **uuid.UUID) {
	if *companyID == nil {
		*companyID = FromContext(ctx)
	}
}
