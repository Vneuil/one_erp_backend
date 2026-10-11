package application

import (
	"context"
	"fmt"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrletters/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

// OvertimeOrdered is overtime authorised by an issued Surat Perintah Lembur.
type OvertimeOrdered struct {
	NIP, Name, Date string
	Minutes         int
	// Reference ties the record to the letter (its number) so it can be withdrawn with the letter.
	Reference, Reason, By string
}

// OvertimeSink files and withdraws the overtime records an order authorises.
type OvertimeSink interface {
	// Record creates an approved overtime record, or skips it when the employee already has overtime that day.
	Record(ctx context.Context, o OvertimeOrdered) (created bool, err error)
	// Revoke rejects the records filed for a letter reference that are still approved.
	Revoke(ctx context.Context, nip, date, reference string) error
}

// orderMinutes is the length of the ordered overtime; an end time at or before the start runs past midnight.
func orderMinutes(start, end string) (int, error) {
	s, err := time.Parse("15:04", start)
	if err != nil {
		return 0, err
	}
	e, err := time.Parse("15:04", end)
	if err != nil {
		return 0, err
	}
	if !e.After(s) {
		e = e.Add(24 * time.Hour)
	}
	return int(e.Sub(s).Minutes()), nil
}

// fileOvertime turns an issued overtime order into approved overtime records,
// one per recipient: the signed order is the authorisation, so payroll can
// price the hours without a separate claim.
func (uc *useCase) fileOvertime(ctx context.Context, l *domain.Letter) error {
	if l.Type != domain.TypeOvertimeOrder || uc.overtime == nil {
		return nil
	}
	mins, err := orderMinutes(l.Data.StartTime, l.Data.EndTime)
	if err != nil || mins <= 0 {
		return apperrors.NewBadRequest("The overtime order has an invalid start or end time")
	}
	for _, r := range l.Recipients {
		_, err := uc.overtime.Record(ctx, OvertimeOrdered{
			NIP: r.NIP, Name: r.Name, Date: l.Data.WorkDate, Minutes: mins, Reference: l.Number,
			Reason: fmt.Sprintf("Surat Perintah Lembur %s: %s", l.Number, l.Data.Tasks), By: actor.EmailFrom(ctx),
		})
		if err != nil {
			return apperrors.NewInternal(err, "Letter issued but overtime could not be filed for "+r.Name)
		}
	}
	return nil
}

func (uc *useCase) revokeOvertime(ctx context.Context, l *domain.Letter) error {
	if l.Type != domain.TypeOvertimeOrder || uc.overtime == nil {
		return nil
	}
	for _, r := range l.Recipients {
		if err := uc.overtime.Revoke(ctx, r.NIP, l.Data.WorkDate, l.Number); err != nil {
			return apperrors.NewInternal(err, "Letter cancelled but overtime could not be withdrawn for "+r.Name)
		}
	}
	return nil
}
