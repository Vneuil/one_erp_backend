package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PasswordResetTokenRepository interface {
	Create(ctx context.Context, token *PasswordResetToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	// InvalidateActiveForUser marks every currently-unused, unexpired token
	// belonging to the user as used (UsedAt = now) so a new reset request or
	// a successful reset can't leave older tokens redeemable.
	InvalidateActiveForUser(ctx context.Context, userID uuid.UUID, now time.Time) error
	MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error
}
