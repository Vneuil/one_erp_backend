package infrastructure

import (
	"context"
	"errors"
	"github.com/divinecoid/one-backend/internal/shared/inbox"
	"log/slog"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) domain.Repository { return &repository{db: db} }

func (r *repository) q(ctx context.Context, model any) *gorm.DB {
	return tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(model))
}

func getByID[T any](ctx context.Context, r *repository, id uuid.UUID) (*T, error) {
	var v T
	err := r.q(ctx, &v).Where("id = ?", id).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *repository) create(ctx context.Context, v any, tenant **uuid.UUID) error {
	tenantctx.SetTenantID(ctx, tenant)
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *repository) save(ctx context.Context, v any) error {
	return r.db.WithContext(ctx).Save(v).Error
}

// --- attendance corrections

func (r *repository) CreateCorrection(ctx context.Context, v *domain.AttendanceCorrection) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetCorrection(ctx context.Context, id uuid.UUID) (*domain.AttendanceCorrection, error) {
	return getByID[domain.AttendanceCorrection](ctx, r, id)
}
func (r *repository) UpdateCorrection(ctx context.Context, v *domain.AttendanceCorrection) error {
	return r.save(ctx, v)
}
func (r *repository) ListCorrections(ctx context.Context, status, nip string) ([]domain.AttendanceCorrection, error) {
	var out []domain.AttendanceCorrection
	db := r.q(ctx, &domain.AttendanceCorrection{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if nip != "" {
		db = db.Where(&domain.AttendanceCorrection{NIP: nip})
	}
	return out, db.Order("created_at desc").Find(&out).Error
}

// --- shifts

func (r *repository) CreateShift(ctx context.Context, v *domain.Shift) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetShift(ctx context.Context, id uuid.UUID) (*domain.Shift, error) {
	return getByID[domain.Shift](ctx, r, id)
}
func (r *repository) ListShifts(ctx context.Context) ([]domain.Shift, error) {
	var out []domain.Shift
	return out, r.q(ctx, &domain.Shift{}).Order("start_time asc, name asc").Find(&out).Error
}
func (r *repository) UpsertAssignment(ctx context.Context, v *domain.ShiftAssignment) error {
	tenantctx.SetTenantID(ctx, &v.TenantID)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "n_ip"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"shift_id", "updated_at"}),
	}).Create(v).Error
}
func (r *repository) GetAssignment(ctx context.Context, nip, date string) (*domain.ShiftAssignment, error) {
	var found []domain.ShiftAssignment
	err := r.q(ctx, &domain.ShiftAssignment{}).Where(&domain.ShiftAssignment{NIP: nip, Date: date}).Limit(1).Find(&found).Error
	if err != nil || len(found) == 0 {
		return nil, err
	}
	return &found[0], nil
}
func (r *repository) ListAssignments(ctx context.Context, period, nip string) ([]domain.ShiftAssignment, error) {
	var out []domain.ShiftAssignment
	db := r.q(ctx, &domain.ShiftAssignment{}).Where("date LIKE ?", period+"%")
	if nip != "" {
		db = db.Where(&domain.ShiftAssignment{NIP: nip})
	}
	return out, db.Order("date asc").Find(&out).Error
}
func (r *repository) CreateShiftChange(ctx context.Context, v *domain.ShiftChangeRequest) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetShiftChange(ctx context.Context, id uuid.UUID) (*domain.ShiftChangeRequest, error) {
	return getByID[domain.ShiftChangeRequest](ctx, r, id)
}
func (r *repository) UpdateShiftChange(ctx context.Context, v *domain.ShiftChangeRequest) error {
	return r.save(ctx, v)
}
func (r *repository) ListShiftChanges(ctx context.Context, status, nip string) ([]domain.ShiftChangeRequest, error) {
	var out []domain.ShiftChangeRequest
	db := r.q(ctx, &domain.ShiftChangeRequest{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if nip != "" {
		db = db.Where(&domain.ShiftChangeRequest{NIP: nip})
	}
	return out, db.Order("created_at desc").Find(&out).Error
}

// --- overtime

func (r *repository) CreateOvertime(ctx context.Context, v *domain.OvertimeRecord) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetOvertime(ctx context.Context, id uuid.UUID) (*domain.OvertimeRecord, error) {
	return getByID[domain.OvertimeRecord](ctx, r, id)
}
func (r *repository) UpdateOvertime(ctx context.Context, v *domain.OvertimeRecord) error {
	return r.save(ctx, v)
}
func (r *repository) ListOvertime(ctx context.Context, period, status, nip string) ([]domain.OvertimeRecord, error) {
	var out []domain.OvertimeRecord
	db := r.q(ctx, &domain.OvertimeRecord{})
	if period != "" {
		db = db.Where("date LIKE ?", period+"%")
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if nip != "" {
		db = db.Where(&domain.OvertimeRecord{NIP: nip})
	}
	return out, db.Order("date desc, created_at desc").Find(&out).Error
}
func (r *repository) OvertimeExists(ctx context.Context, nip, date, source string) (bool, error) {
	var n int64
	err := r.q(ctx, &domain.OvertimeRecord{}).Where(&domain.OvertimeRecord{NIP: nip, Date: date, Source: source}).Count(&n).Error
	return n > 0, err
}

// ApprovedOvertimeMinutes lists the minutes of each approved overtime record of
// an employee in a period (YYYY-MM). Payroll uses it to price overtime per day.
func (r *repository) ApprovedOvertimeMinutes(ctx context.Context, nip, period string) ([]int, error) {
	recs, err := r.ListOvertime(ctx, period, "approved", nip)
	if err != nil {
		return nil, err
	}
	out := make([]int, len(recs))
	for i, rec := range recs {
		out[i] = rec.Minutes
	}
	return out, nil
}

// ApprovedOvertimeDays lists the dates and minutes of an employee's approved
// overtime in a period, in matching order; payslips show them one by one.
func (r *repository) ApprovedOvertimeDays(ctx context.Context, nip, period string) ([]string, []int, error) {
	recs, err := r.ListOvertime(ctx, period, "approved", nip)
	if err != nil {
		return nil, nil, err
	}
	dates := make([]string, len(recs))
	mins := make([]int, len(recs))
	for i, rec := range recs {
		dates[i], mins[i] = rec.Date, rec.Minutes
	}
	return dates, mins, nil
}

// NewInboxSender lets other modules create notifications in this tenant's
// notification center.
func NewInboxSender(db *gorm.DB) inbox.Sender {
	r := &repository{db: db}
	return func(ctx context.Context, email, title, body, link string) {
		if strings.TrimSpace(email) == "" {
			return
		}
		if err := r.CreateNotifications(ctx, []domain.Notification{{RecipientEmail: email, Title: title, Body: body, Link: link}}); err != nil {
			slog.Warn("inbox: failed to create notification", "recipient", email, "error", err)
		}
	}
}

// --- documents

func (r *repository) CreateDocument(ctx context.Context, v *domain.EmployeeDocument) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) ListDocuments(ctx context.Context, employeeID uuid.UUID) ([]domain.EmployeeDocument, error) {
	var out []domain.EmployeeDocument
	return out, r.q(ctx, &domain.EmployeeDocument{}).Where("employee_id = ?", employeeID).Order("created_at desc").Find(&out).Error
}
func (r *repository) GetDocument(ctx context.Context, id uuid.UUID) (*domain.EmployeeDocument, error) {
	return getByID[domain.EmployeeDocument](ctx, r, id)
}
func (r *repository) ExpiringDocuments(ctx context.Context, onOrBefore string) ([]domain.EmployeeDocument, error) {
	var out []domain.EmployeeDocument
	return out, r.q(ctx, &domain.EmployeeDocument{}).Where("expires_on <> '' AND expires_on <= ?", onOrBefore).Order("expires_on asc").Find(&out).Error
}
func (r *repository) DeleteDocument(ctx context.Context, id uuid.UUID) (bool, error) {
	res := r.q(ctx, &domain.EmployeeDocument{}).Where("id = ?", id).Delete(&domain.EmployeeDocument{})
	return res.RowsAffected > 0, res.Error
}

// --- feedback

func (r *repository) CreateFeedback(ctx context.Context, v *domain.Feedback) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) ListFeedback(ctx context.Context, subjectID uuid.UUID, period string) ([]domain.Feedback, error) {
	var out []domain.Feedback
	db := r.q(ctx, &domain.Feedback{}).Where("subject_id = ?", subjectID)
	if period != "" {
		db = db.Where("period = ?", period)
	}
	return out, db.Order("created_at desc").Find(&out).Error
}
func (r *repository) FeedbackExists(ctx context.Context, subjectID uuid.UUID, reviewerEmail, period string) (bool, error) {
	var n int64
	err := r.q(ctx, &domain.Feedback{}).Where("subject_id = ? AND LOWER(reviewer_email) = LOWER(?) AND period = ?", subjectID, reviewerEmail, period).Count(&n).Error
	return n > 0, err
}

// --- announcements & notifications

func (r *repository) CreateAnnouncement(ctx context.Context, v *domain.Announcement) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) ListAnnouncements(ctx context.Context) ([]domain.Announcement, error) {
	var out []domain.Announcement
	return out, r.q(ctx, &domain.Announcement{}).Order("pinned desc, created_at desc").Find(&out).Error
}
func (r *repository) DeleteAnnouncement(ctx context.Context, id uuid.UUID) (bool, error) {
	res := r.q(ctx, &domain.Announcement{}).Where("id = ?", id).Delete(&domain.Announcement{})
	return res.RowsAffected > 0, res.Error
}
func (r *repository) CreateNotifications(ctx context.Context, v []domain.Notification) error {
	if len(v) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(&v, 200).Error
}
func (r *repository) ListNotifications(ctx context.Context, email string, unreadOnly bool) ([]domain.Notification, error) {
	var out []domain.Notification
	db := r.db.WithContext(ctx).Model(&domain.Notification{}).Where("LOWER(recipient_email) = LOWER(?)", email)
	if unreadOnly {
		db = db.Where("read_at IS NULL")
	}
	return out, db.Order("created_at desc").Limit(100).Find(&out).Error
}
func (r *repository) MarkNotificationRead(ctx context.Context, email string, id uuid.UUID) (bool, error) {
	res := r.db.WithContext(ctx).Model(&domain.Notification{}).
		Where("id = ? AND LOWER(recipient_email) = LOWER(?) AND read_at IS NULL", id, email).
		Update("read_at", time.Now())
	return res.RowsAffected > 0, res.Error
}
func (r *repository) MarkAllNotificationsRead(ctx context.Context, email string) error {
	return r.db.WithContext(ctx).Model(&domain.Notification{}).
		Where("LOWER(recipient_email) = LOWER(?) AND read_at IS NULL", email).Update("read_at", time.Now()).Error
}
func (r *repository) CountUnread(ctx context.Context, email string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Notification{}).Where("LOWER(recipient_email) = LOWER(?) AND read_at IS NULL", email).Count(&n).Error
	return n, err
}

// NewOvertimeSource exposes approved overtime to payroll without it depending
// on the rest of this module's repository contract.
func NewOvertimeSource(db *gorm.DB) *repository { return &repository{db: db} }

// --- reimbursement evidence

func (r *repository) CreateEvidence(ctx context.Context, v *domain.ReimbursementEvidence) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) ListEvidence(ctx context.Context, claimID uuid.UUID) ([]domain.ReimbursementEvidence, error) {
	var out []domain.ReimbursementEvidence
	return out, r.q(ctx, &domain.ReimbursementEvidence{}).Where("claim_id = ?", claimID).Order("created_at asc").Find(&out).Error
}

// --- canteen

func (r *repository) CreateCanteenItem(ctx context.Context, v *domain.CanteenItem) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetCanteenItem(ctx context.Context, id uuid.UUID) (*domain.CanteenItem, error) {
	return getByID[domain.CanteenItem](ctx, r, id)
}
func (r *repository) UpdateCanteenItem(ctx context.Context, v *domain.CanteenItem) error {
	return r.save(ctx, v)
}
func (r *repository) ListCanteenItems(ctx context.Context, activeOnly bool) ([]domain.CanteenItem, error) {
	var out []domain.CanteenItem
	db := r.q(ctx, &domain.CanteenItem{})
	if activeOnly {
		db = db.Where("is_active = ?", true)
	}
	return out, db.Order("name asc").Find(&out).Error
}
func (r *repository) CreateCanteenOrder(ctx context.Context, v *domain.CanteenOrder) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetCanteenOrder(ctx context.Context, id uuid.UUID) (*domain.CanteenOrder, error) {
	return getByID[domain.CanteenOrder](ctx, r, id)
}
func (r *repository) UpdateCanteenOrder(ctx context.Context, v *domain.CanteenOrder) error {
	return r.save(ctx, v)
}
func (r *repository) ListCanteenOrders(ctx context.Context, nip, period string) ([]domain.CanteenOrder, error) {
	var out []domain.CanteenOrder
	db := r.q(ctx, &domain.CanteenOrder{}).Where("date LIKE ?", period+"%")
	if nip != "" {
		db = db.Where(&domain.CanteenOrder{NIP: nip})
	}
	return out, db.Order("date desc, created_at desc").Find(&out).Error
}
func (r *repository) CanteenTotals(ctx context.Context, period string) ([]domain.CanteenEmployeeTotal, error) {
	var out []domain.CanteenEmployeeTotal
	err := r.q(ctx, &domain.CanteenOrder{}).
		Select("n_ip, MAX(employee_name) AS employee_name, COUNT(*) AS orders, COALESCE(SUM(amount),0) AS amount").
		Where("date LIKE ? AND status = ?", period+"%", "ordered").Group("n_ip").Order("amount desc").Scan(&out).Error
	return out, err
}

// CanteenAmountFor sums a month's meal orders for one employee; payroll deducts it.
func (r *repository) CanteenAmountFor(ctx context.Context, nip, period string) (float64, error) {
	var total float64
	err := r.q(ctx, &domain.CanteenOrder{}).Where(&domain.CanteenOrder{NIP: nip}).Where("date LIKE ? AND status = ?", period+"%", "ordered").
		Select("COALESCE(SUM(amount),0)").Scan(&total).Error
	return total, err
}

// --- cash advances

func (r *repository) CreateAdvance(ctx context.Context, v *domain.CashAdvance) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetAdvance(ctx context.Context, id uuid.UUID) (*domain.CashAdvance, error) {
	return getByID[domain.CashAdvance](ctx, r, id)
}
func (r *repository) UpdateAdvance(ctx context.Context, v *domain.CashAdvance) error {
	return r.save(ctx, v)
}
func (r *repository) ListAdvances(ctx context.Context, nip, status string) ([]domain.CashAdvance, error) {
	var out []domain.CashAdvance
	db := r.q(ctx, &domain.CashAdvance{})
	if nip != "" {
		db = db.Where(&domain.CashAdvance{NIP: nip})
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	return out, db.Order("created_at desc").Find(&out).Error
}

// AdvanceAmountFor totals the installments of an employee's approved advances
// that fall due in period (YYYY-MM); payroll withholds it.
func (r *repository) AdvanceAmountFor(ctx context.Context, nip, period string) (float64, error) {
	list, err := r.ListAdvances(ctx, nip, "approved")
	if err != nil {
		return 0, err
	}
	var total float64
	for _, a := range list {
		total += a.InstallmentFor(period)
	}
	return total, nil
}

// --- visits

func (r *repository) CreateStops(ctx context.Context, v []domain.VisitStop) error {
	if len(v) == 0 {
		return nil
	}
	for i := range v {
		tenantctx.SetTenantID(ctx, &v[i].TenantID)
	}
	return r.db.WithContext(ctx).Create(&v).Error
}
func (r *repository) GetStop(ctx context.Context, id uuid.UUID) (*domain.VisitStop, error) {
	return getByID[domain.VisitStop](ctx, r, id)
}
func (r *repository) UpdateStops(ctx context.Context, v []domain.VisitStop) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range v {
			if err := tx.Save(&v[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *repository) ListStops(ctx context.Context, nip, date string) ([]domain.VisitStop, error) {
	var out []domain.VisitStop
	db := r.q(ctx, &domain.VisitStop{}).Where("date = ?", date)
	if nip != "" {
		db = db.Where(&domain.VisitStop{NIP: nip})
	}
	return out, db.Order("n_ip asc, seq asc").Find(&out).Error
}
