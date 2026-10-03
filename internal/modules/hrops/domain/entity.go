// Package hrops holds day-to-day HR operations that sit around the core HRM
// records: attendance corrections, shift schedules, overtime, employee
// documents, 360-degree feedback and announcements/notifications.
package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// AttendanceCorrection is a request to fix (or backfill) an attendance day.
// Approving it writes the requested times into the attendance record.
type AttendanceCorrection struct {
	types.BaseEntity
	TenantID         *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	NIP              string     `gorm:"type:varchar(50);not null;index" json:"nip"`
	EmployeeName     string     `gorm:"type:varchar(255)" json:"employeeName"`
	Date             string     `gorm:"type:varchar(10);not null" json:"date"`
	ClockIn          string     `gorm:"type:varchar(5);not null" json:"clockIn"` // HH:MM
	ClockOut         string     `gorm:"type:varchar(5)" json:"clockOut"`         // HH:MM, optional
	Reason           string     `gorm:"type:text;not null" json:"reason"`
	Status           string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	RequestedByEmail string     `gorm:"type:varchar(255)" json:"requestedByEmail,omitempty"`
	DecidedBy        string     `gorm:"type:varchar(255)" json:"decidedBy,omitempty"`
}

func (AttendanceCorrection) TableName() string { return "hrops_attendance_corrections" }

// Shift is a named working-hours template.
type Shift struct {
	types.BaseEntity
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name      string     `gorm:"type:varchar(100);not null" json:"name"`
	StartTime string     `gorm:"type:varchar(5);not null" json:"startTime"` // HH:MM
	EndTime   string     `gorm:"type:varchar(5);not null" json:"endTime"`   // HH:MM
	IsActive  bool       `gorm:"not null;default:true" json:"isActive"`
}

func (Shift) TableName() string { return "hrops_shifts" }

// ShiftAssignment puts one employee on one shift for one date (the monthly schedule).
type ShiftAssignment struct {
	types.BaseEntity
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	NIP      string     `gorm:"type:varchar(50);not null;uniqueIndex:uq_shift_assignment" json:"nip"`
	Date     string     `gorm:"type:varchar(10);not null;uniqueIndex:uq_shift_assignment" json:"date"`
	ShiftID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"shiftId"`
}

func (ShiftAssignment) TableName() string { return "hrops_shift_assignments" }

// ShiftChangeRequest is an employee's request to move to another shift on a date.
type ShiftChangeRequest struct {
	types.BaseEntity
	TenantID         *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	NIP              string     `gorm:"type:varchar(50);not null;index" json:"nip"`
	EmployeeName     string     `gorm:"type:varchar(255)" json:"employeeName"`
	Date             string     `gorm:"type:varchar(10);not null" json:"date"`
	ToShiftID        uuid.UUID  `gorm:"type:uuid;not null" json:"toShiftId"`
	Reason           string     `gorm:"type:text" json:"reason"`
	Status           string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	RequestedByEmail string     `gorm:"type:varchar(255)" json:"requestedByEmail,omitempty"`
	DecidedBy        string     `gorm:"type:varchar(255)" json:"decidedBy,omitempty"`
}

func (ShiftChangeRequest) TableName() string { return "hrops_shift_change_requests" }

// OvertimeRecord is one overtime session. Several may exist per employee.
// Source is "manual" (claimed) or "auto" (detected from attendance).
type OvertimeRecord struct {
	types.BaseEntity
	TenantID         *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	NIP              string     `gorm:"type:varchar(50);not null;index:idx_ot_nip_date" json:"nip"`
	EmployeeName     string     `gorm:"type:varchar(255)" json:"employeeName"`
	Date             string     `gorm:"type:varchar(10);not null;index:idx_ot_nip_date" json:"date"`
	Minutes          int        `gorm:"not null" json:"minutes"`
	Source           string     `gorm:"type:varchar(10);not null;default:'manual'" json:"source"`
	Reason           string     `gorm:"type:text" json:"reason"`
	Status           string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	RequestedByEmail string     `gorm:"type:varchar(255)" json:"requestedByEmail,omitempty"`
	DecidedBy        string     `gorm:"type:varchar(255)" json:"decidedBy,omitempty"`
}

func (OvertimeRecord) TableName() string { return "hrops_overtime_records" }

// EmployeeDocument is a file reference (contract, ID card, certificate...) kept
// on an employee's record. FileRef is a Drive file id or URL.
type EmployeeDocument struct {
	types.BaseEntity
	TenantID   *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	EmployeeID uuid.UUID  `gorm:"type:uuid;not null;index" json:"employeeId"`
	Title      string     `gorm:"type:varchar(255);not null" json:"title"`
	DocType    string     `gorm:"type:varchar(50);not null" json:"docType"`
	FileRef    string     `gorm:"type:varchar(500)" json:"fileRef"`
	ExpiresOn  string     `gorm:"type:varchar(10)" json:"expiresOn,omitempty"`
	Notes      string     `gorm:"type:text" json:"notes,omitempty"`
	UploadedBy string     `gorm:"type:varchar(255)" json:"uploadedBy,omitempty"`
	// An uploaded file, when there is one (FileRef then stays free for an external link).
	FileName   string `gorm:"type:varchar(255)" json:"fileName,omitempty"`
	FileSize   int64  `json:"fileSize,omitempty"`
	StorageKey string `gorm:"type:varchar(500)" json:"-"`
}

func (EmployeeDocument) TableName() string { return "hrops_employee_documents" }

// Feedback is one 360-degree review of an employee. Ratings are 1-5.
type Feedback struct {
	types.BaseEntity
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	SubjectID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"subjectId"`
	ReviewerEmail string     `gorm:"type:varchar(255);not null" json:"-"` // never sent to the subject
	Relationship  string     `gorm:"type:varchar(20);not null" json:"relationship"`
	Period        string     `gorm:"type:varchar(20);not null;index" json:"period"`
	Communication int        `json:"communication"`
	Teamwork      int        `json:"teamwork"`
	Leadership    int        `json:"leadership"`
	Reliability   int        `json:"reliability"`
	Comment       string     `gorm:"type:text" json:"comment,omitempty"`
}

func (Feedback) TableName() string { return "hrops_feedback" }

// Announcement is company news, optionally limited to one department.
type Announcement struct {
	types.BaseEntity
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Body           string     `gorm:"type:text;not null" json:"body"`
	Department     string     `gorm:"type:varchar(100)" json:"department,omitempty"` // empty = everyone
	Pinned         bool       `gorm:"not null;default:false" json:"pinned"`
	ExpiresOn      string     `gorm:"type:varchar(10)" json:"expiresOn,omitempty"`
	CreatedByEmail string     `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
}

func (Announcement) TableName() string { return "hrops_announcements" }

// Notification is one item in a user's notification center.
type Notification struct {
	types.BaseEntity
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	RecipientEmail string     `gorm:"type:varchar(255);not null;index" json:"-"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Body           string     `gorm:"type:text" json:"body"`
	Link           string     `gorm:"type:varchar(500)" json:"link,omitempty"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
}

func (Notification) TableName() string { return "hrops_notifications" }

// Repository is the persistence contract for every HR-operations entity.
type Repository interface {
	ExtrasRepository

	CreateCorrection(ctx context.Context, v *AttendanceCorrection) error
	GetCorrection(ctx context.Context, id uuid.UUID) (*AttendanceCorrection, error)
	UpdateCorrection(ctx context.Context, v *AttendanceCorrection) error
	ListCorrections(ctx context.Context, status, nip string) ([]AttendanceCorrection, error)

	CreateShift(ctx context.Context, v *Shift) error
	GetShift(ctx context.Context, id uuid.UUID) (*Shift, error)
	ListShifts(ctx context.Context) ([]Shift, error)
	UpsertAssignment(ctx context.Context, v *ShiftAssignment) error
	GetAssignment(ctx context.Context, nip, date string) (*ShiftAssignment, error)
	ListAssignments(ctx context.Context, period, nip string) ([]ShiftAssignment, error)
	CreateShiftChange(ctx context.Context, v *ShiftChangeRequest) error
	GetShiftChange(ctx context.Context, id uuid.UUID) (*ShiftChangeRequest, error)
	UpdateShiftChange(ctx context.Context, v *ShiftChangeRequest) error
	ListShiftChanges(ctx context.Context, status, nip string) ([]ShiftChangeRequest, error)

	CreateOvertime(ctx context.Context, v *OvertimeRecord) error
	GetOvertime(ctx context.Context, id uuid.UUID) (*OvertimeRecord, error)
	UpdateOvertime(ctx context.Context, v *OvertimeRecord) error
	ListOvertime(ctx context.Context, period, status, nip string) ([]OvertimeRecord, error)
	OvertimeExists(ctx context.Context, nip, date, source string) (bool, error)

	CreateDocument(ctx context.Context, v *EmployeeDocument) error
	ListDocuments(ctx context.Context, employeeID uuid.UUID) ([]EmployeeDocument, error)
	GetDocument(ctx context.Context, id uuid.UUID) (*EmployeeDocument, error)
	// ExpiringDocuments returns documents whose expiry date is on or before the given date (YYYY-MM-DD).
	ExpiringDocuments(ctx context.Context, onOrBefore string) ([]EmployeeDocument, error)
	DeleteDocument(ctx context.Context, id uuid.UUID) (bool, error)

	CreateFeedback(ctx context.Context, v *Feedback) error
	ListFeedback(ctx context.Context, subjectID uuid.UUID, period string) ([]Feedback, error)
	FeedbackExists(ctx context.Context, subjectID uuid.UUID, reviewerEmail, period string) (bool, error)

	CreateAnnouncement(ctx context.Context, v *Announcement) error
	ListAnnouncements(ctx context.Context) ([]Announcement, error)
	DeleteAnnouncement(ctx context.Context, id uuid.UUID) (bool, error)
	CreateNotifications(ctx context.Context, v []Notification) error
	ListNotifications(ctx context.Context, email string, unreadOnly bool) ([]Notification, error)
	MarkNotificationRead(ctx context.Context, email string, id uuid.UUID) (bool, error)
	MarkAllNotificationsRead(ctx context.Context, email string) error
	CountUnread(ctx context.Context, email string) (int64, error)
}

// ReimbursementEvidence is one receipt or proof attached to a reimbursement
// claim. FileRef is a Drive file id or URL.
type ReimbursementEvidence struct {
	types.BaseEntity
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ClaimID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"claimId"`
	Title           string     `gorm:"type:varchar(255);not null" json:"title"`
	FileRef         string     `gorm:"type:varchar(500);not null" json:"fileRef"`
	Amount          float64    `gorm:"type:decimal(15,2);default:0" json:"amount"`
	UploadedByEmail string     `gorm:"type:varchar(255)" json:"uploadedByEmail,omitempty"`
}

func (ReimbursementEvidence) TableName() string { return "hrops_reimbursement_evidence" }

// CanteenItem is a menu item employees can order.
type CanteenItem struct {
	types.BaseEntity
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name     string     `gorm:"type:varchar(150);not null" json:"name"`
	Price    float64    `gorm:"type:decimal(15,2);not null" json:"price"`
	IsActive bool       `gorm:"not null;default:true" json:"isActive"`
}

func (CanteenItem) TableName() string { return "hrops_canteen_items" }

// CanteenOrder is one employee's meal order. The month's non-cancelled orders
// are deducted from salary when payroll is calculated.
type CanteenOrder struct {
	types.BaseEntity
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	NIP            string     `gorm:"type:varchar(50);not null;index:idx_canteen_nip_date" json:"nip"`
	EmployeeName   string     `gorm:"type:varchar(255)" json:"employeeName"`
	ItemID         uuid.UUID  `gorm:"type:uuid;not null" json:"itemId"`
	ItemName       string     `gorm:"type:varchar(150);not null" json:"itemName"`
	Quantity       int        `gorm:"not null" json:"quantity"`
	UnitPrice      float64    `gorm:"type:decimal(15,2);not null" json:"unitPrice"`
	Amount         float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	Date           string     `gorm:"type:varchar(10);not null;index:idx_canteen_nip_date" json:"date"`
	Status         string     `gorm:"type:varchar(15);not null;default:'ordered'" json:"status"` // ordered | cancelled
	CreatedByEmail string     `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
}

func (CanteenOrder) TableName() string { return "hrops_canteen_orders" }

// CanteenEmployeeTotal is one employee's canteen spend for a month.
type CanteenEmployeeTotal struct {
	NIP          string  `json:"nip"`
	EmployeeName string  `json:"employeeName"`
	Orders       int     `json:"orders"`
	Amount       float64 `json:"amount"`
}

// VisitStop is one planned customer visit on a field employee's route for a day.
type VisitStop struct {
	types.BaseEntity
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	NIP          string     `gorm:"type:varchar(50);not null;index:idx_visit_nip_date" json:"nip"`
	EmployeeName string     `gorm:"type:varchar(255)" json:"employeeName"`
	Date         string     `gorm:"type:varchar(10);not null;index:idx_visit_nip_date" json:"date"`
	Seq          int        `gorm:"not null" json:"seq"`
	CustomerName string     `gorm:"type:varchar(255);not null" json:"customerName"`
	Address      string     `gorm:"type:varchar(500)" json:"address,omitempty"`
	Latitude     float64    `gorm:"not null" json:"latitude"`
	Longitude    float64    `gorm:"not null" json:"longitude"`
	RadiusMeters int        `gorm:"not null;default:100" json:"radiusMeters"`
	Status       string     `gorm:"type:varchar(10);not null;default:'planned'" json:"status"` // planned | visited | skipped
	CheckInAt    *time.Time `json:"checkInAt,omitempty"`
	CheckInLat   *float64   `json:"checkInLat,omitempty"`
	CheckInLng   *float64   `json:"checkInLng,omitempty"`
	DistanceM    *float64   `json:"distanceM,omitempty"` // how far from the customer the check-in was
	Note         string     `gorm:"type:varchar(500)" json:"note,omitempty"`
}

func (VisitStop) TableName() string { return "hrops_visit_stops" }

// ExtrasRepository is the persistence contract for evidence, canteen and visits.
// CashAdvance (kasbon) is money lent to an employee and withheld from pay in
// equal monthly installments starting at StartPeriod (YYYY-MM). Repayment is
// derived from the schedule, so recalculating payroll never double-counts.
type CashAdvance struct {
	types.BaseEntity
	TenantID         *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	NIP              string     `gorm:"type:varchar(50);not null;index" json:"nip"`
	EmployeeName     string     `gorm:"type:varchar(255)" json:"employeeName"`
	EmployeeEmail    string     `gorm:"type:varchar(255)" json:"-"`
	Amount           float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	Installments     int        `gorm:"not null" json:"installments"`
	StartPeriod      string     `gorm:"type:varchar(7);not null" json:"startPeriod"`
	Reason           string     `gorm:"type:varchar(500)" json:"reason"`
	Status           string     `gorm:"type:varchar(15);not null;default:'pending'" json:"status"` // pending | approved | rejected
	RequestedByEmail string     `gorm:"type:varchar(255)" json:"requestedByEmail,omitempty"`
	DecidedBy        string     `gorm:"type:varchar(255)" json:"decidedBy,omitempty"`
}

func (CashAdvance) TableName() string { return "hr_cash_advances" }

type ExtrasRepository interface {
	CreateAdvance(ctx context.Context, v *CashAdvance) error
	GetAdvance(ctx context.Context, id uuid.UUID) (*CashAdvance, error)
	UpdateAdvance(ctx context.Context, v *CashAdvance) error
	ListAdvances(ctx context.Context, nip, status string) ([]CashAdvance, error)

	CreateEvidence(ctx context.Context, v *ReimbursementEvidence) error
	ListEvidence(ctx context.Context, claimID uuid.UUID) ([]ReimbursementEvidence, error)

	CreateCanteenItem(ctx context.Context, v *CanteenItem) error
	GetCanteenItem(ctx context.Context, id uuid.UUID) (*CanteenItem, error)
	UpdateCanteenItem(ctx context.Context, v *CanteenItem) error
	ListCanteenItems(ctx context.Context, activeOnly bool) ([]CanteenItem, error)
	CreateCanteenOrder(ctx context.Context, v *CanteenOrder) error
	GetCanteenOrder(ctx context.Context, id uuid.UUID) (*CanteenOrder, error)
	UpdateCanteenOrder(ctx context.Context, v *CanteenOrder) error
	ListCanteenOrders(ctx context.Context, nip, period string) ([]CanteenOrder, error)
	CanteenTotals(ctx context.Context, period string) ([]CanteenEmployeeTotal, error)

	CreateStops(ctx context.Context, v []VisitStop) error
	GetStop(ctx context.Context, id uuid.UUID) (*VisitStop, error)
	UpdateStops(ctx context.Context, v []VisitStop) error
	ListStops(ctx context.Context, nip, date string) ([]VisitStop, error)
}
