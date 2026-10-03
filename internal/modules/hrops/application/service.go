package application

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	reimbursementDomain "github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/divinecoid/one-backend/internal/shared/attachment"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Caller identifies who is acting, taken from the JWT by the handler.
type Caller struct {
	Email string
	// Privileged callers (admin/manager) may act on other employees' behalf.
	Privileged bool
}

type Service struct {
	repo domain.Repository
	hrm  hrmdomain.HRMRepository
	// claims is optional: reimbursement evidence needs it.
	claims reimbursementDomain.ReimbursementRepository
	// ledger is optional: approving a cash advance posts the payout when set.
	ledger financeApp.LedgerPoster
	// leaves is optional: line managers can decide leave requests through it.
	leaves LeaveBridge
	// store holds uploaded employee documents; nil or disabled means link-only documents.
	store storage.Storage
	now   func() time.Time
}

func NewService(repo domain.Repository, hrm hrmdomain.HRMRepository) *Service {
	return &Service{repo: repo, hrm: hrm, now: time.Now}
}

// resolveEmployee finds the employee a request is about. With no NIP it is the
// caller's own record (matched by login email); naming someone else's NIP needs
// a privileged caller.
func (s *Service) resolveEmployee(ctx context.Context, c Caller, nip string) (*hrmdomain.Employee, error) {
	own, err := s.hrm.FindEmployeeByEmail(ctx, c.Email)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to look up your employee record")
	}
	nip = strings.TrimSpace(nip)
	if nip == "" || (own != nil && own.NIP == nip) {
		if own == nil {
			return nil, apperrors.NewBadRequest("No employee record is linked to your account; ask HR to add one")
		}
		return own, nil
	}
	if !c.Privileged {
		return nil, apperrors.NewForbidden("You can only submit requests for yourself")
	}
	emp, err := s.findByNIP(ctx, nip)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, apperrors.NewNotFound("Employee not found")
	}
	return emp, nil
}

// OwnNIP returns the NIP of the caller's own employee record.
func (s *Service) OwnNIP(ctx context.Context, c Caller) (string, error) {
	own, err := s.hrm.FindEmployeeByEmail(ctx, c.Email)
	if err != nil {
		return "", apperrors.NewInternal(err, "Failed to look up your employee record")
	}
	if own == nil {
		return "", apperrors.NewBadRequest("No employee record is linked to your account; ask HR to add one")
	}
	return own.NIP, nil
}

func (s *Service) findByNIP(ctx context.Context, nip string) (*hrmdomain.Employee, error) {
	list, _, err := s.hrm.ListEmployees(ctx, types.PaginationQuery{Page: 1, PerPage: 1000})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to look up employee")
	}
	for i := range list {
		if list[i].NIP == nip {
			return &list[i], nil
		}
	}
	return nil, nil
}

// decided rejects a decision on a request that is no longer pending.
func decided(status string) error {
	if status != "pending" {
		return apperrors.NewConflict("This request has already been " + status)
	}
	return nil
}

// ---------------------------------------------------------------- corrections

type CorrectionInput struct {
	NIP, Date, ClockIn, ClockOut, Reason string
}

func (s *Service) RequestCorrection(ctx context.Context, c Caller, in CorrectionInput) (*domain.AttendanceCorrection, error) {
	if !ValidDate(in.Date) || !ValidClock(in.ClockIn) || (in.ClockOut != "" && !ValidClock(in.ClockOut)) {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD and times HH:MM")
	}
	if strings.TrimSpace(in.Reason) == "" {
		return nil, apperrors.NewBadRequest("A reason is required")
	}
	if in.ClockOut != "" && in.ClockOut <= in.ClockIn {
		return nil, apperrors.NewBadRequest("Clock-out must be after clock-in")
	}
	day, _ := time.ParseInLocation("2006-01-02", in.Date, zone)
	if day.After(s.now().In(zone)) {
		return nil, apperrors.NewBadRequest("Corrections cannot be dated in the future")
	}
	emp, err := s.resolveEmployee(ctx, c, in.NIP)
	if err != nil {
		return nil, err
	}
	v := &domain.AttendanceCorrection{
		NIP: emp.NIP, EmployeeName: emp.Name, Date: in.Date, ClockIn: in.ClockIn, ClockOut: in.ClockOut,
		Reason: strings.TrimSpace(in.Reason), Status: "pending", RequestedByEmail: c.Email,
	}
	if err := s.repo.CreateCorrection(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save correction request")
	}
	s.notifyManagerOf(ctx, emp, "Koreksi absensi baru", emp.Name+" mengajukan koreksi absensi "+in.Date)
	return v, nil
}

func (s *Service) ListCorrections(ctx context.Context, status, nip string) ([]domain.AttendanceCorrection, error) {
	out, err := s.repo.ListCorrections(ctx, status, nip)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list corrections")
	}
	return out, nil
}

// DecideCorrection approves or rejects a request. Approving writes the times
// into the day's attendance record (creating it when the employee never clocked
// in) and marks it as a correction.
func (s *Service) DecideCorrection(ctx context.Context, id uuid.UUID, approve bool) (*domain.AttendanceCorrection, error) {
	v, err := s.repo.GetCorrection(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load correction")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Correction request not found")
	}
	if err := decided(v.Status); err != nil {
		return nil, err
	}
	if approve {
		if err := sod.ForbidSelfApproval(ctx, v.RequestedByEmail); err != nil {
			return nil, err
		}
		if err := s.applyCorrection(ctx, v); err != nil {
			return nil, err
		}
		v.Status = "approved"
	} else {
		v.Status = "rejected"
	}
	v.DecidedBy = actor.EmailFrom(ctx)
	if err := s.repo.UpdateCorrection(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update correction")
	}
	s.notifyOutcome(ctx, v.NIP, v.RequestedByEmail, "Koreksi absensi", approve)
	return v, nil
}

func (s *Service) applyCorrection(ctx context.Context, v *domain.AttendanceCorrection) error {
	in, err := parseClock(v.Date, v.ClockIn)
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}
	shiftStart := ""
	if a, _ := s.repo.GetAssignment(ctx, v.NIP, v.Date); a != nil {
		if sh, _ := s.repo.GetShift(ctx, a.ShiftID); sh != nil {
			shiftStart = sh.StartTime
		}
	}
	status, err := clockStatus(in, shiftStart)
	if err != nil {
		return apperrors.NewBadRequest(err.Error())
	}
	att, err := s.hrm.FindAttendanceByNIPAndDate(ctx, v.NIP, v.Date)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load attendance")
	}
	isNew := att == nil
	if isNew {
		att = &hrmdomain.Attendance{NIP: v.NIP, EmployeeName: v.EmployeeName, Date: v.Date, Location: "Correction"}
	}
	att.ClockIn, att.ClockInAt, att.Status, att.Method = v.ClockIn+" WIB", &in, status, "Correction"
	att.ClockOut, att.ClockOutAt, att.WorkMinutes = "--:--", nil, 0
	if v.ClockOut != "" {
		out, err := parseClock(v.Date, v.ClockOut)
		if err != nil {
			return apperrors.NewBadRequest(err.Error())
		}
		att.ClockOut, att.ClockOutAt = v.ClockOut+" WIB", &out
		att.WorkMinutes = int(out.Sub(in).Minutes())
	}
	if isNew {
		err = s.hrm.RecordAttendance(ctx, att)
	} else {
		err = s.hrm.UpdateAttendance(ctx, att)
	}
	if err != nil {
		return apperrors.NewInternal(err, "Failed to write corrected attendance")
	}
	return nil
}

// -------------------------------------------------------------------- summary

// AttendanceSummaryRow is one employee's month.
type AttendanceSummaryRow struct {
	NIP          string `json:"nip"`
	EmployeeName string `json:"employeeName"`
	DaysPresent  int    `json:"daysPresent"`
	LateCount    int    `json:"lateCount"`
	WorkMinutes  int    `json:"workMinutes"`
	Corrections  int    `json:"corrections"`
}

// AttendanceSummary rolls a month (YYYY-MM) of attendance up per employee.
func (s *Service) AttendanceSummary(ctx context.Context, period string) ([]AttendanceSummaryRow, error) {
	if _, err := time.Parse("2006-01", period); err != nil {
		return nil, apperrors.NewBadRequest("period must be YYYY-MM")
	}
	recs, err := s.hrm.ListAttendanceByNIPAndPeriod(ctx, "", period)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load attendance")
	}
	idx := map[string]*AttendanceSummaryRow{}
	var order []string
	for _, a := range recs {
		row := idx[a.NIP]
		if row == nil {
			row = &AttendanceSummaryRow{NIP: a.NIP, EmployeeName: a.EmployeeName}
			idx[a.NIP] = row
			order = append(order, a.NIP)
		}
		row.DaysPresent++
		row.WorkMinutes += a.WorkMinutes
		if strings.EqualFold(a.Status, "Late") {
			row.LateCount++
		}
		if a.Method == "Correction" {
			row.Corrections++
		}
	}
	out := make([]AttendanceSummaryRow, 0, len(order))
	for _, nip := range order {
		out = append(out, *idx[nip])
	}
	return out, nil
}

// AttendanceSummaryCSV renders AttendanceSummary as CSV bytes.
func (s *Service) AttendanceSummaryCSV(ctx context.Context, period string) ([]byte, error) {
	rows, err := s.AttendanceSummary(ctx, period)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"NIP", "Employee", "Days Present", "Late", "Work Hours", "Corrections"})
	for _, r := range rows {
		_ = w.Write([]string{r.NIP, r.EmployeeName, strconv.Itoa(r.DaysPresent), strconv.Itoa(r.LateCount),
			fmt.Sprintf("%.1f", float64(r.WorkMinutes)/60), strconv.Itoa(r.Corrections)})
	}
	w.Flush()
	return []byte(b.String()), w.Error()
}

// --------------------------------------------------------------------- shifts

type ShiftInput struct{ Name, StartTime, EndTime string }

func (s *Service) CreateShift(ctx context.Context, in ShiftInput) (*domain.Shift, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || !ValidClock(in.StartTime) || !ValidClock(in.EndTime) {
		return nil, apperrors.NewBadRequest("name, startTime and endTime (HH:MM) are required")
	}
	v := &domain.Shift{Name: name, StartTime: in.StartTime, EndTime: in.EndTime, IsActive: true}
	if err := s.repo.CreateShift(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create shift")
	}
	return v, nil
}

func (s *Service) ListShifts(ctx context.Context) ([]domain.Shift, error) {
	out, err := s.repo.ListShifts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list shifts")
	}
	return out, nil
}

// AssignShifts schedules employees onto a shift for a set of dates (the monthly
// schedule). Existing assignments for the same employee and date are replaced.
func (s *Service) AssignShifts(ctx context.Context, shiftID uuid.UUID, nips, dates []string) (int, error) {
	sh, err := s.repo.GetShift(ctx, shiftID)
	if err != nil {
		return 0, apperrors.NewInternal(err, "Failed to load shift")
	}
	if sh == nil {
		return 0, apperrors.NewNotFound("Shift not found")
	}
	if len(nips) == 0 || len(dates) == 0 {
		return 0, apperrors.NewBadRequest("nips and dates are required")
	}
	for _, d := range dates {
		if !ValidDate(d) {
			return 0, apperrors.NewBadRequest("dates must be YYYY-MM-DD")
		}
	}
	n := 0
	for _, nip := range nips {
		for _, d := range dates {
			if err := s.repo.UpsertAssignment(ctx, &domain.ShiftAssignment{NIP: strings.TrimSpace(nip), Date: d, ShiftID: shiftID}); err != nil {
				return n, apperrors.NewInternal(err, "Failed to assign shift")
			}
			n++
		}
	}
	return n, nil
}

func (s *Service) ListAssignments(ctx context.Context, period, nip string) ([]domain.ShiftAssignment, error) {
	if _, err := time.Parse("2006-01", period); err != nil {
		return nil, apperrors.NewBadRequest("period must be YYYY-MM")
	}
	out, err := s.repo.ListAssignments(ctx, period, nip)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list schedule")
	}
	return out, nil
}

func (s *Service) RequestShiftChange(ctx context.Context, c Caller, nip, date string, toShift uuid.UUID, reason string) (*domain.ShiftChangeRequest, error) {
	if !ValidDate(date) {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	sh, err := s.repo.GetShift(ctx, toShift)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load shift")
	}
	if sh == nil || !sh.IsActive {
		return nil, apperrors.NewNotFound("Shift not found")
	}
	emp, err := s.resolveEmployee(ctx, c, nip)
	if err != nil {
		return nil, err
	}
	v := &domain.ShiftChangeRequest{NIP: emp.NIP, EmployeeName: emp.Name, Date: date, ToShiftID: toShift,
		Reason: strings.TrimSpace(reason), Status: "pending", RequestedByEmail: c.Email}
	if err := s.repo.CreateShiftChange(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save shift change request")
	}
	s.notifyManagerOf(ctx, emp, "Tukar shift baru", emp.Name+" mengajukan tukar shift "+date)
	return v, nil
}

func (s *Service) ListShiftChanges(ctx context.Context, status, nip string) ([]domain.ShiftChangeRequest, error) {
	out, err := s.repo.ListShiftChanges(ctx, status, nip)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list shift changes")
	}
	return out, nil
}

func (s *Service) DecideShiftChange(ctx context.Context, id uuid.UUID, approve bool) (*domain.ShiftChangeRequest, error) {
	v, err := s.repo.GetShiftChange(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load request")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Shift change request not found")
	}
	if err := decided(v.Status); err != nil {
		return nil, err
	}
	if approve {
		if err := sod.ForbidSelfApproval(ctx, v.RequestedByEmail); err != nil {
			return nil, err
		}
		if err := s.repo.UpsertAssignment(ctx, &domain.ShiftAssignment{NIP: v.NIP, Date: v.Date, ShiftID: v.ToShiftID}); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to apply shift change")
		}
		v.Status = "approved"
	} else {
		v.Status = "rejected"
	}
	v.DecidedBy = actor.EmailFrom(ctx)
	if err := s.repo.UpdateShiftChange(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update request")
	}
	s.notifyOutcome(ctx, v.NIP, v.RequestedByEmail, "Tukar shift", approve)
	return v, nil
}

// ------------------------------------------------------------------- overtime

func (s *Service) RequestOvertime(ctx context.Context, c Caller, nip, date string, minutes int, reason string) (*domain.OvertimeRecord, error) {
	if !ValidDate(date) || minutes <= 0 || minutes > 12*60 {
		return nil, apperrors.NewBadRequest("date (YYYY-MM-DD) and minutes (1-720) are required")
	}
	emp, err := s.resolveEmployee(ctx, c, nip)
	if err != nil {
		return nil, err
	}
	v := &domain.OvertimeRecord{NIP: emp.NIP, EmployeeName: emp.Name, Date: date, Minutes: minutes, Source: "manual",
		Reason: strings.TrimSpace(reason), Status: "pending", RequestedByEmail: c.Email}
	if err := s.repo.CreateOvertime(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save overtime")
	}
	s.notifyManagerOf(ctx, emp, "Lembur baru", emp.Name+" mengajukan lembur "+date)
	return v, nil
}

// DetectOvertime scans a month of attendance and files a pending "auto"
// overtime record for each day worked beyond the standard day. Days that
// already have an auto record are skipped, so it can be re-run safely.
func (s *Service) DetectOvertime(ctx context.Context, period string) (int, error) {
	if _, err := time.Parse("2006-01", period); err != nil {
		return 0, apperrors.NewBadRequest("period must be YYYY-MM")
	}
	recs, err := s.hrm.ListAttendanceByNIPAndPeriod(ctx, "", period)
	if err != nil {
		return 0, apperrors.NewInternal(err, "Failed to load attendance")
	}
	created := 0
	for _, a := range recs {
		mins := DetectOvertimeMinutes(a.WorkMinutes)
		if mins == 0 {
			continue
		}
		exists, err := s.repo.OvertimeExists(ctx, a.NIP, a.Date, "auto")
		if err != nil {
			return created, apperrors.NewInternal(err, "Failed to check existing overtime")
		}
		if exists {
			continue
		}
		v := &domain.OvertimeRecord{NIP: a.NIP, EmployeeName: a.EmployeeName, Date: a.Date, Minutes: mins, Source: "auto",
			Reason: "Detected from attendance", Status: "pending"}
		if err := s.repo.CreateOvertime(ctx, v); err != nil {
			return created, apperrors.NewInternal(err, "Failed to save overtime")
		}
		created++
	}
	return created, nil
}

func (s *Service) ListOvertime(ctx context.Context, period, status, nip string) ([]domain.OvertimeRecord, error) {
	out, err := s.repo.ListOvertime(ctx, period, status, nip)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list overtime")
	}
	return out, nil
}

func (s *Service) DecideOvertime(ctx context.Context, id uuid.UUID, approve bool) (*domain.OvertimeRecord, error) {
	v, err := s.repo.GetOvertime(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load overtime")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Overtime record not found")
	}
	if err := decided(v.Status); err != nil {
		return nil, err
	}
	if approve {
		// Auto-detected records have no requester, so anyone with approval rights may approve them.
		if err := sod.ForbidSelfApproval(ctx, v.RequestedByEmail); err != nil {
			return nil, err
		}
		v.Status = "approved"
	} else {
		v.Status = "rejected"
	}
	v.DecidedBy = actor.EmailFrom(ctx)
	if err := s.repo.UpdateOvertime(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update overtime")
	}
	s.notifyOutcome(ctx, v.NIP, v.RequestedByEmail, "Lembur", approve)
	return v, nil
}

// ------------------------------------------------------------------ documents

type DocumentInput struct {
	EmployeeID                                uuid.UUID
	Title, DocType, FileRef, ExpiresOn, Notes string
}

func (s *Service) AddDocument(ctx context.Context, c Caller, in DocumentInput) (*domain.EmployeeDocument, error) {
	if !c.Privileged {
		return nil, apperrors.NewForbidden("Only HR can add employee documents")
	}
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.DocType) == "" {
		return nil, apperrors.NewBadRequest("title and docType are required")
	}
	if in.ExpiresOn != "" && !ValidDate(in.ExpiresOn) {
		return nil, apperrors.NewBadRequest("expiresOn must be YYYY-MM-DD")
	}
	emp, err := s.hrm.GetEmployeeByID(ctx, in.EmployeeID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load employee")
	}
	if emp == nil {
		return nil, apperrors.NewNotFound("Employee not found")
	}
	v := &domain.EmployeeDocument{EmployeeID: in.EmployeeID, Title: strings.TrimSpace(in.Title), DocType: strings.TrimSpace(in.DocType),
		FileRef: strings.TrimSpace(in.FileRef), ExpiresOn: in.ExpiresOn, Notes: in.Notes, UploadedBy: c.Email}
	if err := s.repo.CreateDocument(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save document")
	}
	return v, nil
}

// mayUseDocuments: HR documents (contracts, ID cards) are for HR staff and for
// the employee they belong to, nobody else.
func (s *Service) mayUseDocuments(ctx context.Context, c Caller, employeeID uuid.UUID) error {
	if c.Privileged {
		return nil
	}
	own, err := s.hrm.FindEmployeeByEmail(ctx, c.Email)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to look up your employee record")
	}
	if own != nil && own.ID == employeeID {
		return nil
	}
	return apperrors.NewForbidden("Employee documents are only available to HR and to the employee concerned")
}

func (s *Service) ListDocuments(ctx context.Context, c Caller, employeeID uuid.UUID) ([]domain.EmployeeDocument, error) {
	if err := s.mayUseDocuments(ctx, c, employeeID); err != nil {
		return nil, err
	}
	out, err := s.repo.ListDocuments(ctx, employeeID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list documents")
	}
	return out, nil
}

// WithStorage enables direct file upload for employee documents.
func (s *Service) WithStorage(st storage.Storage) *Service {
	s.store = st
	return s
}

// AttachDocument adds a document whose file is uploaded and kept in storage.
func (s *Service) AttachDocument(ctx context.Context, c Caller, in DocumentInput, filename string, content []byte) (*domain.EmployeeDocument, error) {
	if !c.Privileged {
		return nil, apperrors.NewForbidden("Only HR can add employee documents")
	}
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.DocType) == "" {
		return nil, apperrors.NewBadRequest("title and docType are required")
	}
	if in.ExpiresOn != "" && !ValidDate(in.ExpiresOn) {
		return nil, apperrors.NewBadRequest("expiresOn must be YYYY-MM-DD")
	}
	emp, err := s.hrm.GetEmployeeByID(ctx, in.EmployeeID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load employee")
	}
	if emp == nil {
		return nil, apperrors.NewNotFound("Employee not found")
	}
	saved, err := attachment.Save(ctx, s.store, "hr-documents", filename, content)
	if err != nil {
		return nil, err
	}
	v := &domain.EmployeeDocument{EmployeeID: in.EmployeeID, Title: strings.TrimSpace(in.Title), DocType: strings.TrimSpace(in.DocType),
		ExpiresOn: in.ExpiresOn, Notes: in.Notes, UploadedBy: c.Email, FileName: saved.Name, FileSize: saved.Size, StorageKey: saved.Key}
	if err := s.repo.CreateDocument(ctx, v); err != nil {
		attachment.Discard(ctx, s.store, saved.Key) // do not leave an orphan behind
		return nil, apperrors.NewInternal(err, "Failed to save document")
	}
	return v, nil
}

// DocumentFile returns the document and its file content.
func (s *Service) DocumentFile(ctx context.Context, c Caller, id uuid.UUID) (*domain.EmployeeDocument, []byte, error) {
	v, err := s.repo.GetDocument(ctx, id)
	if err != nil {
		return nil, nil, apperrors.NewInternal(err, "Failed to load document")
	}
	if v == nil {
		return nil, nil, apperrors.NewNotFound("Document not found")
	}
	if err := s.mayUseDocuments(ctx, c, v.EmployeeID); err != nil {
		return nil, nil, err
	}
	if v.StorageKey == "" {
		return nil, nil, apperrors.NewNotFound("This document has no uploaded file")
	}
	data, err := attachment.Load(ctx, s.store, v.StorageKey)
	return v, data, err
}

func (s *Service) DeleteDocument(ctx context.Context, c Caller, id uuid.UUID) error {
	if !c.Privileged {
		return apperrors.NewForbidden("Only HR can delete employee documents")
	}
	doc, _ := s.repo.GetDocument(ctx, id)
	ok, err := s.repo.DeleteDocument(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete document")
	}
	if !ok {
		return apperrors.NewNotFound("Document not found")
	}
	if doc != nil {
		attachment.Discard(ctx, s.store, doc.StorageKey)
	}
	return nil
}

// ------------------------------------------------------------------- feedback

type FeedbackInput struct {
	SubjectID                                        uuid.UUID
	Relationship, Period, Comment                    string
	Communication, Teamwork, Leadership, Reliability int
}

var relationships = map[string]bool{"peer": true, "manager": true, "subordinate": true, "self": true}

func validRating(n int) bool { return n >= 1 && n <= 5 }

func (s *Service) SubmitFeedback(ctx context.Context, c Caller, in FeedbackInput) error {
	in.Relationship = strings.ToLower(strings.TrimSpace(in.Relationship))
	if !relationships[in.Relationship] {
		return apperrors.NewBadRequest("relationship must be peer, manager, subordinate or self")
	}
	if strings.TrimSpace(in.Period) == "" {
		return apperrors.NewBadRequest("period is required")
	}
	for _, r := range []int{in.Communication, in.Teamwork, in.Leadership, in.Reliability} {
		if !validRating(r) {
			return apperrors.NewBadRequest("every rating must be between 1 and 5")
		}
	}
	subject, err := s.hrm.GetEmployeeByID(ctx, in.SubjectID)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load employee")
	}
	if subject == nil {
		return apperrors.NewNotFound("Employee not found")
	}
	isSelf := sod.SamePerson(subject.Email, c.Email)
	if isSelf != (in.Relationship == "self") {
		return apperrors.NewBadRequest(`use relationship "self" only when reviewing yourself, and never for anyone else`)
	}
	dup, err := s.repo.FeedbackExists(ctx, in.SubjectID, c.Email, in.Period)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check existing feedback")
	}
	if dup {
		return apperrors.NewConflict("You have already reviewed this employee for this period")
	}
	v := &domain.Feedback{SubjectID: in.SubjectID, ReviewerEmail: c.Email, Relationship: in.Relationship, Period: in.Period,
		Communication: in.Communication, Teamwork: in.Teamwork, Leadership: in.Leadership, Reliability: in.Reliability, Comment: strings.TrimSpace(in.Comment)}
	if err := s.repo.CreateFeedback(ctx, v); err != nil {
		return apperrors.NewInternal(err, "Failed to save feedback")
	}
	return nil
}

// FeedbackSummaryFor aggregates feedback about an employee. Reviewers are never
// identified, and a summary is withheld until at least minReviewers reviews exist
// (self reviews do not count towards that), so a single reviewer cannot be inferred.
const minReviewers = 3

func (s *Service) FeedbackSummaryFor(ctx context.Context, subjectID uuid.UUID, period string) (*FeedbackSummary, error) {
	items, err := s.repo.ListFeedback(ctx, subjectID, period)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load feedback")
	}
	rs := make([]ratings, 0, len(items))
	others := 0
	for _, f := range items {
		rs = append(rs, ratings{f.Relationship, f.Communication, f.Teamwork, f.Leadership, f.Reliability, f.Comment})
		if f.Relationship != "self" {
			others++
		}
	}
	if others < minReviewers {
		return &FeedbackSummary{Count: len(items), Overall: map[string]float64{}, ByRelationship: map[string]map[string]float64{}, Comments: []string{}}, nil
	}
	sum := summarizeFeedback(rs)
	return &sum, nil
}

// -------------------------------------------------- announcements & notifications

type AnnouncementInput struct {
	Title, Body, Department, ExpiresOn string
	Pinned                             bool
}

// PublishAnnouncement stores the announcement and drops a notification into the
// inbox of every employee it targets (everyone, or one department).
func (s *Service) PublishAnnouncement(ctx context.Context, c Caller, in AnnouncementInput) (*domain.Announcement, int, error) {
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Body) == "" {
		return nil, 0, apperrors.NewBadRequest("title and body are required")
	}
	if in.ExpiresOn != "" && !ValidDate(in.ExpiresOn) {
		return nil, 0, apperrors.NewBadRequest("expiresOn must be YYYY-MM-DD")
	}
	v := &domain.Announcement{Title: strings.TrimSpace(in.Title), Body: strings.TrimSpace(in.Body), Department: strings.TrimSpace(in.Department),
		Pinned: in.Pinned, ExpiresOn: in.ExpiresOn, CreatedByEmail: c.Email}
	if err := s.repo.CreateAnnouncement(ctx, v); err != nil {
		return nil, 0, apperrors.NewInternal(err, "Failed to save announcement")
	}
	emps, _, err := s.hrm.ListEmployees(ctx, types.PaginationQuery{Page: 1, PerPage: 10000})
	if err != nil {
		return v, 0, apperrors.NewInternal(err, "Announcement saved but recipients could not be loaded")
	}
	var notes []domain.Notification
	for _, e := range emps {
		if e.Email == "" || strings.EqualFold(e.Status, "Inactive") {
			continue
		}
		if v.Department != "" && !strings.EqualFold(e.Department, v.Department) {
			continue
		}
		notes = append(notes, domain.Notification{RecipientEmail: e.Email, Title: v.Title, Body: v.Body, Link: "/hrm/announcements"})
	}
	if err := s.repo.CreateNotifications(ctx, notes); err != nil {
		return v, 0, apperrors.NewInternal(err, "Announcement saved but notifications failed")
	}
	return v, len(notes), nil
}

// ListAnnouncements hides expired ones.
func (s *Service) ListAnnouncements(ctx context.Context) ([]domain.Announcement, error) {
	all, err := s.repo.ListAnnouncements(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list announcements")
	}
	today := s.now().In(zone).Format("2006-01-02")
	out := all[:0]
	for _, a := range all {
		if a.ExpiresOn == "" || a.ExpiresOn >= today {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s *Service) DeleteAnnouncement(ctx context.Context, id uuid.UUID) error {
	ok, err := s.repo.DeleteAnnouncement(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete announcement")
	}
	if !ok {
		return apperrors.NewNotFound("Announcement not found")
	}
	return nil
}

func (s *Service) ListNotifications(ctx context.Context, email string, unreadOnly bool) ([]domain.Notification, int64, error) {
	items, err := s.repo.ListNotifications(ctx, email, unreadOnly)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err, "Failed to list notifications")
	}
	n, err := s.repo.CountUnread(ctx, email)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err, "Failed to count notifications")
	}
	return items, n, nil
}

func (s *Service) MarkRead(ctx context.Context, email string, id uuid.UUID) error {
	ok, err := s.repo.MarkNotificationRead(ctx, email, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to update notification")
	}
	if !ok {
		return apperrors.NewNotFound("Notification not found or already read")
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, email string) error {
	if err := s.repo.MarkAllNotificationsRead(ctx, email); err != nil {
		return apperrors.NewInternal(err, "Failed to update notifications")
	}
	return nil
}
