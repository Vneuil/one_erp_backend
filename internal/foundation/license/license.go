// Package license enforces that a company's ONE ERP subscription is
// actually active before letting its users touch tenant-scoped business
// data. Without this, the entire platform/billing module built earlier is
// purely cosmetic - a trial or paid period lapsing would never actually
// stop anyone from using the product.
package license

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// PlatformSubscription represents a company's subscription plan and validity
// in the control-plane database.
type PlatformSubscription struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CompanyID          uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"companyId"`
	Status             string     `gorm:"size:50;not null;default:'active'" json:"status"`
	PlanDurationMonths int        `gorm:"default:1" json:"planDurationMonths"`
	PricePerMonth      float64    `gorm:"type:decimal(15,2);default:0" json:"pricePerMonth"`
	TotalPrice         float64    `gorm:"type:decimal(15,2);default:0" json:"totalPrice"`
	TrialEndsAt        *time.Time `json:"trialEndsAt"`
	StartsAt           *time.Time `json:"startsAt"`
	EndsAt             *time.Time `json:"endsAt"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func (PlatformSubscription) TableName() string {
	return "platform_subscriptions"
}

// AutoMigrate ensures the platform_subscriptions table exists in the control-plane database.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PlatformSubscription{})
}

type subscriptionRow struct {
	Status      string
	TrialEndsAt *time.Time
	EndsAt      *time.Time
}

func isTableMissingError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "42p01") ||
		strings.Contains(msg, "no such table")
}

// IsActive reports whether companyID's subscription currently permits
// access. Reads the platform_subscriptions table directly (raw table name,
// not through the platform module's repository) to avoid a foundation ->
// module dependency; the two are kept in sync only by the column names,
// which are stable since they are the platform module's own migration.
//
// A missing subscription row or missing table returns true - this is the
// pre-platform-module state (seeded/demo companies, or ones created before
// this system existed), and denying them access would be a regression, not
// enforcement.
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

	// Use silent logger session for this probe query so that a missing table
	// or record not found does not write spurious ERROR logs to console.
	silentDB := db.Session(&gorm.Session{
		Logger: db.Logger.LogMode(gormlogger.Silent),
	})

	err := silentDB.WithContext(ctx).
		Table("platform_subscriptions").
		Select("status, trial_ends_at, ends_at").
		Where("company_id = ?", companyID).
		Take(&row).Error

	if err == gorm.ErrRecordNotFound || isTableMissingError(err) {
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
