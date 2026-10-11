package infrastructure

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/hrletters/application"
	hropsDomain "github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	hropsInfra "github.com/divinecoid/one-backend/internal/modules/hrops/infrastructure"
	"gorm.io/gorm"
)

type overtimeSink struct{ repo hropsDomain.Repository }

// NewOvertimeSink files overtime into the HR operations overtime register.
func NewOvertimeSink(db *gorm.DB) application.OvertimeSink {
	return &overtimeSink{repo: hropsInfra.NewRepository(db)}
}

func (s *overtimeSink) Record(ctx context.Context, o application.OvertimeOrdered) (bool, error) {
	existing, err := s.repo.ListOvertime(ctx, o.Date[:7], "", o.NIP)
	if err != nil {
		return false, err
	}
	for _, e := range existing {
		// A claim or detected overtime for the same day is already in the register; do not pay the day twice.
		if e.Date == o.Date && e.Status != "rejected" {
			return false, nil
		}
	}
	return true, s.repo.CreateOvertime(ctx, &hropsDomain.OvertimeRecord{
		NIP: o.NIP, EmployeeName: o.Name, Date: o.Date, Minutes: o.Minutes, Source: "order",
		Reason: o.Reason, Status: "approved", DecidedBy: o.By,
	})
}

func (s *overtimeSink) Revoke(ctx context.Context, nip, date, reference string) error {
	recs, err := s.repo.ListOvertime(ctx, date[:7], "approved", nip)
	if err != nil {
		return err
	}
	for i := range recs {
		r := &recs[i]
		if r.Date == date && r.Source == "order" && strings.Contains(r.Reason, reference) {
			r.Status = "rejected"
			if err := s.repo.UpdateOvertime(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}
