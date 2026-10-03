package infrastructure

import (
	"context"

	hropsdomain "github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	"gorm.io/gorm"
)

// Notifier writes into the same per-user notification inbox HR uses
// (hrops_notifications), so a CRM task shows up in the one notification center.
type Notifier struct{ db *gorm.DB }

func NewNotifier(db *gorm.DB) *Notifier { return &Notifier{db: db} }

func (n *Notifier) Notify(ctx context.Context, email, title, body, link string) error {
	return n.db.WithContext(ctx).Create(&hropsdomain.Notification{RecipientEmail: email, Title: title, Body: body, Link: link}).Error
}
