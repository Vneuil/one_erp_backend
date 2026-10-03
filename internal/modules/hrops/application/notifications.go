package application

import (
	"context"
	"log/slog"
	"strings"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
)

// notify drops an item in someone's notification center. It never fails the
// action that caused it: a missed notification is not worth undoing an approval.
func (s *Service) notify(ctx context.Context, email, title, body, link string) {
	email = strings.TrimSpace(email)
	if email == "" {
		return
	}
	if err := s.repo.CreateNotifications(ctx, []domain.Notification{{RecipientEmail: email, Title: title, Body: body, Link: link}}); err != nil {
		slog.Warn("hrops: failed to create notification", "recipient", email, "error", err)
	}
}

// notifyManagerOf tells the employee's direct manager a request is waiting.
func (s *Service) notifyManagerOf(ctx context.Context, emp *hrmdomain.Employee, title, body string) {
	if emp == nil || emp.ManagerID == nil {
		return
	}
	mgr, err := s.hrm.GetEmployeeByID(ctx, *emp.ManagerID)
	if err != nil || mgr == nil {
		return
	}
	s.notify(ctx, mgr.Email, title, body, "/hrm/team-approvals")
}

// notifyOutcome tells the people behind a request how it was decided: the
// employee it concerns and, when different, whoever filed it for them.
func (s *Service) notifyOutcome(ctx context.Context, nip, requestedBy, what string, approved bool) {
	verdict := "ditolak"
	if approved {
		verdict = "disetujui"
	}
	title := what + " " + verdict
	sent := map[string]bool{}
	send := func(email string) {
		k := strings.ToLower(strings.TrimSpace(email))
		if k == "" || sent[k] {
			return
		}
		sent[k] = true
		s.notify(ctx, email, title, "Permintaan "+strings.ToLower(what)+" Anda telah "+verdict+".", "/hrm/requests")
	}
	if emp, err := s.findByNIP(ctx, nip); err == nil && emp != nil {
		send(emp.Email)
	}
	send(requestedBy)
}
