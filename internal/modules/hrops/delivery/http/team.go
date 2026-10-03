package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/response"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/hrops/application"
	hropsinfra "github.com/divinecoid/one-backend/internal/modules/hrops/infrastructure"
	leaveApp "github.com/divinecoid/one-backend/internal/modules/leave/application"
	leaveDomain "github.com/divinecoid/one-backend/internal/modules/leave/domain"
	leaveinfra "github.com/divinecoid/one-backend/internal/modules/leave/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// leaveBridge adapts the leave module to the manager flow.
type leaveBridge struct {
	repo leaveDomain.LeaveRepository
	uc   leaveApp.LeaveUseCase
}

func newLeaveBridge(db *gorm.DB) *leaveBridge {
	repo := leaveinfra.NewLeaveRepository(db)
	return &leaveBridge{repo: repo, uc: leaveApp.NewLeaveUseCase(repo, hrminfra.NewHRMRepository(db), leaveApp.WithInbox(hropsinfra.NewInboxSender(db)))}
}

func (b *leaveBridge) Pending(ctx context.Context) ([]application.TeamLeave, error) {
	list, _, err := b.uc.ListLeaveRequests(ctx, types.PaginationQuery{Page: 1, PerPage: 1000})
	if err != nil {
		return nil, err
	}
	var out []application.TeamLeave
	for _, l := range list {
		if l.Status == "pending" {
			out = append(out, application.TeamLeave{ID: l.ID, EmployeeID: l.EmployeeID, EmployeeName: l.EmployeeName, Type: l.Type,
				StartDate: l.StartDate, EndDate: l.EndDate, TotalDays: l.TotalDays, Reason: l.Reason})
		}
	}
	return out, nil
}

func (b *leaveBridge) Owner(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	r, err := b.repo.GetByID(ctx, id)
	if err != nil {
		return uuid.Nil, apperrors.NewInternal(err, "Failed to load leave request")
	}
	if r == nil {
		return uuid.Nil, apperrors.NewNotFound("Leave request not found")
	}
	return r.EmployeeID, nil
}

func (b *leaveBridge) Decide(ctx context.Context, id uuid.UUID, approve bool, by string) error {
	var err error
	if approve {
		_, err = b.uc.ApproveLeaveRequest(ctx, id, leaveApp.ApproveLeaveRequestDTO{ApprovedBy: by})
	} else {
		_, err = b.uc.RejectLeaveRequest(ctx, id, leaveApp.RejectLeaveRequestDTO{ApprovedBy: by})
	}
	return err
}

// ---- line-manager approval

func (h *Handler) TeamRequests(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.TeamRequests(h.ctx(c), caller)
	if err != nil {
		return err
	}
	return response.OK(c, "Team requests retrieved", v)
}

func (h *Handler) decideAsManager(approve bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, caller, err := h.resolve(c)
		if err != nil {
			return err
		}
		rid, err := id(c)
		if err != nil {
			return err
		}
		if err := svc.DecideAsManager(h.ctx(c), caller, c.Params("kind"), rid, approve); err != nil {
			return err
		}
		msg := "Request rejected"
		if approve {
			msg = "Request approved"
		}
		return response.OK(c, msg, nil)
	}
}
