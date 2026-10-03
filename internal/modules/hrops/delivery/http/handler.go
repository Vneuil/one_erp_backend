package http

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/hrops/application"
	"github.com/divinecoid/one-backend/internal/modules/hrops/infrastructure"
	reimbursementinfra "github.com/divinecoid/one-backend/internal/modules/reimbursement/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ store storage.Storage }

func NewHandler(store storage.Storage) *Handler { return &Handler{store: store} }

func (h *Handler) resolve(c *fiber.Ctx) (*application.Service, application.Caller, error) {
	db := middleware.TenantDB(c)
	if db == nil {
		return nil, application.Caller{}, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return nil, application.Caller{}, apperrors.NewForbidden("Not authenticated")
	}
	role := strings.ToLower(claims.Role)
	caller := application.Caller{Email: claims.Email, Privileged: role == "admin" || role == "manager"}
	svc := application.NewService(infrastructure.NewRepository(db), hrminfra.NewHRMRepository(db)).WithClaims(reimbursementinfra.NewReimbursementRepository(db)).
		WithLedger(financeApp.NewLedgerPoster(financeInfra.NewFinanceRepository(db))).
		WithLeave(newLeaveBridge(db)).WithStorage(h.store)
	return svc, caller, nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func id(c *fiber.Ctx) (uuid.UUID, error) {
	v, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid id format")
	}
	return v, nil
}

func body(c *fiber.Ctx, v any) error {
	if err := c.BodyParser(v); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	return nil
}

// ownNIP resolves the caller's own employee NIP so self-service lists show only their records.
func (h *Handler) ownNIP(c *fiber.Ctx, svc *application.Service, caller application.Caller) (string, error) {
	return svc.OwnNIP(h.ctx(c), caller)
}

// ---- corrections

func (h *Handler) RequestCorrection(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in application.CorrectionInput
	var raw struct{ NIP, Date, ClockIn, ClockOut, Reason string }
	if err := body(c, &raw); err != nil {
		return err
	}
	in = application.CorrectionInput(raw)
	v, err := svc.RequestCorrection(h.ctx(c), caller, in)
	if err != nil {
		return err
	}
	return response.Created(c, "Correction request submitted", v)
}

func (h *Handler) ListOwnCorrections(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	nip, err := h.ownNIP(c, svc, caller)
	if err != nil {
		return err
	}
	v, err := svc.ListCorrections(h.ctx(c), c.Query("status"), nip)
	if err != nil {
		return err
	}
	return response.OK(c, "Corrections retrieved", v)
}

func (h *Handler) ListCorrections(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.ListCorrections(h.ctx(c), c.Query("status"), c.Query("nip"))
	if err != nil {
		return err
	}
	return response.OK(c, "Corrections retrieved", v)
}

func (h *Handler) decideCorrection(approve bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, _, err := h.resolve(c)
		if err != nil {
			return err
		}
		cid, err := id(c)
		if err != nil {
			return err
		}
		v, err := svc.DecideCorrection(h.ctx(c), cid, approve)
		if err != nil {
			return err
		}
		return response.OK(c, "Correction "+v.Status, v)
	}
}

func (h *Handler) AttendanceSummary(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	period := c.Query("period")
	if c.Query("format") == "csv" {
		data, err := svc.AttendanceSummaryCSV(h.ctx(c), period)
		if err != nil {
			return err
		}
		c.Set("Content-Type", "text/csv; charset=utf-8")
		c.Set("Content-Disposition", `attachment; filename="attendance-summary-`+period+`.csv"`)
		return c.Send(data)
	}
	rows, err := svc.AttendanceSummary(h.ctx(c), period)
	if err != nil {
		return err
	}
	return response.OK(c, "Attendance summary retrieved", rows)
}

// ---- shifts

func (h *Handler) CreateShift(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct{ Name, StartTime, EndTime string }
	if err := body(c, &in); err != nil {
		return err
	}
	v, err := svc.CreateShift(h.ctx(c), application.ShiftInput(in))
	if err != nil {
		return err
	}
	return response.Created(c, "Shift created", v)
}

func (h *Handler) ListShifts(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.ListShifts(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Shifts retrieved", v)
}

func (h *Handler) AssignShifts(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		ShiftID uuid.UUID `json:"shiftId"`
		NIPs    []string  `json:"nips"`
		Dates   []string  `json:"dates"`
	}
	if err := body(c, &in); err != nil {
		return err
	}
	n, err := svc.AssignShifts(h.ctx(c), in.ShiftID, in.NIPs, in.Dates)
	if err != nil {
		return err
	}
	return response.OK(c, "Shifts assigned", fiber.Map{"assigned": n})
}

func (h *Handler) ListSchedule(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.ListAssignments(h.ctx(c), c.Query("period"), c.Query("nip"))
	if err != nil {
		return err
	}
	return response.OK(c, "Schedule retrieved", v)
}

func (h *Handler) ListOwnSchedule(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	nip, err := h.ownNIP(c, svc, caller)
	if err != nil {
		return err
	}
	v, err := svc.ListAssignments(h.ctx(c), c.Query("period"), nip)
	if err != nil {
		return err
	}
	return response.OK(c, "Schedule retrieved", v)
}

func (h *Handler) RequestShiftChange(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		NIP       string    `json:"nip"`
		Date      string    `json:"date"`
		ToShiftID uuid.UUID `json:"toShiftId"`
		Reason    string    `json:"reason"`
	}
	if err := body(c, &in); err != nil {
		return err
	}
	v, err := svc.RequestShiftChange(h.ctx(c), caller, in.NIP, in.Date, in.ToShiftID, in.Reason)
	if err != nil {
		return err
	}
	return response.Created(c, "Shift change requested", v)
}

func (h *Handler) ListOwnShiftChanges(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	nip, err := h.ownNIP(c, svc, caller)
	if err != nil {
		return err
	}
	v, err := svc.ListShiftChanges(h.ctx(c), c.Query("status"), nip)
	if err != nil {
		return err
	}
	return response.OK(c, "Shift changes retrieved", v)
}

func (h *Handler) ListShiftChanges(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.ListShiftChanges(h.ctx(c), c.Query("status"), c.Query("nip"))
	if err != nil {
		return err
	}
	return response.OK(c, "Shift changes retrieved", v)
}

func (h *Handler) decideShiftChange(approve bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, _, err := h.resolve(c)
		if err != nil {
			return err
		}
		rid, err := id(c)
		if err != nil {
			return err
		}
		v, err := svc.DecideShiftChange(h.ctx(c), rid, approve)
		if err != nil {
			return err
		}
		return response.OK(c, "Shift change "+v.Status, v)
	}
}

// ---- overtime

func (h *Handler) RequestOvertime(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		NIP     string `json:"nip"`
		Date    string `json:"date"`
		Minutes int    `json:"minutes"`
		Reason  string `json:"reason"`
	}
	if err := body(c, &in); err != nil {
		return err
	}
	v, err := svc.RequestOvertime(h.ctx(c), caller, in.NIP, in.Date, in.Minutes, in.Reason)
	if err != nil {
		return err
	}
	return response.Created(c, "Overtime submitted", v)
}

func (h *Handler) ListOwnOvertime(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	nip, err := h.ownNIP(c, svc, caller)
	if err != nil {
		return err
	}
	v, err := svc.ListOvertime(h.ctx(c), c.Query("period"), c.Query("status"), nip)
	if err != nil {
		return err
	}
	return response.OK(c, "Overtime retrieved", v)
}

func (h *Handler) ListOvertime(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.ListOvertime(h.ctx(c), c.Query("period"), c.Query("status"), c.Query("nip"))
	if err != nil {
		return err
	}
	return response.OK(c, "Overtime retrieved", v)
}

func (h *Handler) DetectOvertime(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		Period string `json:"period"`
	}
	if err := body(c, &in); err != nil {
		return err
	}
	n, err := svc.DetectOvertime(h.ctx(c), in.Period)
	if err != nil {
		return err
	}
	return response.OK(c, "Overtime detection finished", fiber.Map{"created": n})
}

func (h *Handler) decideOvertime(approve bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, _, err := h.resolve(c)
		if err != nil {
			return err
		}
		rid, err := id(c)
		if err != nil {
			return err
		}
		v, err := svc.DecideOvertime(h.ctx(c), rid, approve)
		if err != nil {
			return err
		}
		return response.OK(c, "Overtime "+v.Status, v)
	}
}

// ---- documents

func (h *Handler) AddDocument(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	eid, err := id(c)
	if err != nil {
		return err
	}
	if fh, ferr := c.FormFile("file"); ferr == nil && fh != nil {
		content, err := readUpload(fh)
		if err != nil {
			return err
		}
		v, err := svc.AttachDocument(h.ctx(c), caller, application.DocumentInput{EmployeeID: eid, Title: c.FormValue("title"), DocType: c.FormValue("docType"),
			ExpiresOn: c.FormValue("expiresOn"), Notes: c.FormValue("notes")}, fh.Filename, content)
		if err != nil {
			return err
		}
		return response.Created(c, "Document uploaded", v)
	}
	var in struct{ Title, DocType, FileRef, ExpiresOn, Notes string }
	if err := body(c, &in); err != nil {
		return err
	}
	v, err := svc.AddDocument(h.ctx(c), caller, application.DocumentInput{EmployeeID: eid, Title: in.Title, DocType: in.DocType, FileRef: in.FileRef, ExpiresOn: in.ExpiresOn, Notes: in.Notes})
	if err != nil {
		return err
	}
	return response.Created(c, "Document added", v)
}

func (h *Handler) ListDocuments(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	eid, err := id(c)
	if err != nil {
		return err
	}
	v, err := svc.ListDocuments(h.ctx(c), caller, eid)
	if err != nil {
		return err
	}
	return response.OK(c, "Documents retrieved", v)
}

func (h *Handler) DeleteDocument(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	did, err := id(c)
	if err != nil {
		return err
	}
	if err := svc.DeleteDocument(h.ctx(c), caller, did); err != nil {
		return err
	}
	return response.OK(c, "Document deleted", nil)
}

// ---- feedback

func (h *Handler) SubmitFeedback(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		SubjectID     uuid.UUID `json:"subjectId"`
		Relationship  string    `json:"relationship"`
		Period        string    `json:"period"`
		Comment       string    `json:"comment"`
		Communication int       `json:"communication"`
		Teamwork      int       `json:"teamwork"`
		Leadership    int       `json:"leadership"`
		Reliability   int       `json:"reliability"`
	}
	if err := body(c, &in); err != nil {
		return err
	}
	if err := svc.SubmitFeedback(h.ctx(c), caller, application.FeedbackInput(in)); err != nil {
		return err
	}
	return response.Created(c, "Feedback submitted", nil)
}

func (h *Handler) FeedbackSummary(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	sid, err := id(c)
	if err != nil {
		return err
	}
	v, err := svc.FeedbackSummaryFor(h.ctx(c), sid, c.Query("period"))
	if err != nil {
		return err
	}
	return response.OK(c, "Feedback summary retrieved", v)
}

// ---- announcements & notifications

func (h *Handler) PublishAnnouncement(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		Title      string `json:"title"`
		Body       string `json:"body"`
		Department string `json:"department"`
		ExpiresOn  string `json:"expiresOn"`
		Pinned     bool   `json:"pinned"`
	}
	if err := body(c, &in); err != nil {
		return err
	}
	v, n, err := svc.PublishAnnouncement(h.ctx(c), caller, application.AnnouncementInput{Title: in.Title, Body: in.Body, Department: in.Department, ExpiresOn: in.ExpiresOn, Pinned: in.Pinned})
	if err != nil {
		return err
	}
	return response.Created(c, "Announcement published", fiber.Map{"announcement": v, "notified": n})
}

func (h *Handler) ListAnnouncements(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.ListAnnouncements(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Announcements retrieved", v)
}

func (h *Handler) DeleteAnnouncement(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	aid, err := id(c)
	if err != nil {
		return err
	}
	if err := svc.DeleteAnnouncement(h.ctx(c), aid); err != nil {
		return err
	}
	return response.OK(c, "Announcement deleted", nil)
}

func (h *Handler) ListNotifications(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	items, unread, err := svc.ListNotifications(h.ctx(c), caller.Email, c.Query("unread") == "true")
	if err != nil {
		return err
	}
	return response.OK(c, "Notifications retrieved", fiber.Map{"items": items, "unread": unread})
}

func (h *Handler) MarkNotificationRead(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	nid, err := id(c)
	if err != nil {
		return err
	}
	if err := svc.MarkRead(h.ctx(c), caller.Email, nid); err != nil {
		return err
	}
	return response.OK(c, "Notification marked as read", nil)
}

func (h *Handler) MarkAllNotificationsRead(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	if err := svc.MarkAllRead(h.ctx(c), caller.Email); err != nil {
		return err
	}
	return response.OK(c, "Notifications marked as read", nil)
}
