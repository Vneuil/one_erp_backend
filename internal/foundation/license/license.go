// Package license enforces that a company's ONE ERP subscription is
// actually active before letting its users touch tenant-scoped business
// data. Without this, the entire platform/billing module built earlier is
// purely cosmetic - a trial or paid period lapsing would never actually
// stop anyone from using the product.
package license

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type subscriptionRow struct {
	Status      string
	TrialEndsAt *time.Time
	EndsAt      *time.Time
}

// IsActive reports whether companyID's subscription currently permits
// access. Reads the platform_subscriptions table directly (raw table name,
// not through the platform module's repository) to avoid a foundation ->
// module dependency; the two are kept in sync only by the column names,
// which are stable since they are the platform module's own migration.
//
// A missing subscription row returns true - this is the pre-platform-module
// state (seeded/demo companies, or ones created before this system
// existed), and denying them access would be a regression, not enforcement.
//
// Status handling:
//   - "expired": always inactive.
//   - "trial": active only until TrialEndsAt.
//   - "active": active only until EndsAt.
//   - anything else ("pending_activation", "pending_payment"): not enforced
//     here - pending_payment still has the concurrent trial clock covering
//     it, and pending_activation has no tenant DB yet to protect.
func IsActive(ctx context.Context, db *gorm.DB, companyID uuid.UUID) (bool, error) {
	var row subscriptionRow
	err := db.WithContext(ctx).
		Table("platform_subscriptions").
		Select("status, trial_ends_at, ends_at").
		Where("company_id = ?", companyID).
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	now := time.Now()
	switch row.Status {
	case "expired":
		return false, nil
	case "trial":
		return row.TrialEndsAt == nil || now.Before(*row.TrialEndsAt), nil
	case "active":
		return row.EndsAt == nil || now.Before(*row.EndsAt), nil
	default:
		return true, nil
	}
}
