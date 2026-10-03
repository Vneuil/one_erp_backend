package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/auth/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type passwordResetTokenRepository struct {
	db *gorm.DB
}

func NewPasswordResetTokenRepository(db *gorm.DB) domain.PasswordResetTokenRepository {
	return &passwordResetTokenRepository{db: db}
}

func (r *passwordResetTokenRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *passwordResetTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	var token domain.PasswordResetToken
	err := r.db.WithContext(ctx).First(&token, "token_hash = ?", tokenHash).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *passwordResetTokenRepository) InvalidateActiveForUser(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL AND expires_at > ?", userID, now).
		Update("used_at", now).Error
}

func (r *passwordResetTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.PasswordResetToken{}).
		Where("id = ?", id).
		Update("used_at", usedAt).Error
}
