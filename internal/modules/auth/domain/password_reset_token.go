package domain

import (
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// PasswordResetToken is a single-use, time-limited credential that lets a
// user set a new password without being logged in. Only a SHA-256 hash of
// the raw token is ever stored (mirroring how PasswordHash never stores a
// plaintext password) so a leaked database backup cannot be replayed - the
// raw token exists only in the emailed link and in the requester's memory.
type PasswordResetToken struct {
	types.BaseEntity
	UserID    uuid.UUID  `gorm:"type:uuid;index;not null" json:"userId"`
	TokenHash string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
}

func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

// IsValid reports whether the token can still be redeemed: not used and not
// expired.
func (t *PasswordResetToken) IsValid(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}
