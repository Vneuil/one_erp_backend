package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Letter types.
const (
	TypeContract      = "contract"       // Surat Kontrak Kerja
	TypeSummons       = "summons"        // Surat Panggilan
	TypeReprimand     = "reprimand"      // Surat Teguran
	TypeWarning       = "warning"        // Surat Peringatan (SP1-SP3)
	TypeTermination   = "termination"    // Surat PHK
	TypeMemo          = "memo"           // Memo Internal
	TypeOvertimeOrder = "overtime_order" // Surat Perintah Lembur
	TypeMutation      = "mutation"       // Mutasi Karyawan
)

// Letter statuses.
const (
	StatusDraft     = "draft"
	StatusIssued    = "issued"
	StatusCancelled = "cancelled"
)

// LetterData holds the fields that only some letter types use.
type LetterData struct {
	// Contract
	ContractType   string  `json:"contractType,omitempty"` // PKWT | PKWTT | Magang
	Position       string  `json:"position,omitempty"`
	Salary         float64 `json:"salary,omitempty"`
	Workplace      string  `json:"workplace,omitempty"`
	ProbationMonth int     `json:"probationMonths,omitempty"`
	// Summons
	MeetingDate string `json:"meetingDate,omitempty"`
	MeetingTime string `json:"meetingTime,omitempty"`
	Place       string `json:"place,omitempty"`
	Reason      string `json:"reason,omitempty"`
	// Reprimand / warning
	Violation   string `json:"violation,omitempty"`
	ValidMonths int    `json:"validMonths,omitempty"`
	Consequence string `json:"consequence,omitempty"`
	// Termination
	TerminationReason  string  `json:"terminationReason,omitempty"`
	LastWorkDay        string  `json:"lastWorkDay,omitempty"`
	Severance          float64 `json:"severance,omitempty"`
	ServiceAward       float64 `json:"serviceAward,omitempty"`
	CompensationRights float64 `json:"compensationRights,omitempty"`
	Notes              string  `json:"notes,omitempty"`
	// Overtime order
	WorkDate  string `json:"workDate,omitempty"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
	Tasks     string `json:"tasks,omitempty"`
	// Memo audience: everyone, one department, or just the recipients.
	AudienceAll        bool   `json:"audienceAll,omitempty"`
	AudienceDepartment string `json:"audienceDepartment,omitempty"`
	// Mutation: the employee's position before (taken from the record when drafted) and after.
	FromDepartment string `json:"fromDepartment,omitempty"`
	FromRole       string `json:"fromRole,omitempty"`
	ToDepartment   string `json:"toDepartment,omitempty"`
	ToRole         string `json:"toRole,omitempty"`
	ToManagerID    string `json:"toManagerId,omitempty"`
	// ApplyToEmployee writes the change onto the employee record once the letter is issued and in effect.
	ApplyToEmployee bool `json:"applyToEmployee,omitempty"`
}

// Letter is one HR document. Employee fields are a snapshot taken when drafted.
type Letter struct {
	types.BaseEntity
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`

	Type   string `gorm:"type:varchar(30);not null;index" json:"type"`
	Status string `gorm:"type:varchar(20);not null;default:'draft';index" json:"status"`
	// Number is assigned when the letter is issued, so drafts never leave gaps in the sequence.
	Number string `gorm:"type:varchar(60);index" json:"number"`
	Date   string `gorm:"type:varchar(10);not null;index" json:"date"`

	Subject     string `gorm:"type:varchar(255);not null" json:"subject"`
	Body        string `gorm:"type:text;not null" json:"body"`
	CompanyName string `gorm:"type:varchar(255)" json:"companyName"`
	City        string `gorm:"type:varchar(100)" json:"city"`
	SignerName  string `gorm:"type:varchar(150)" json:"signerName"`
	SignerTitle string `gorm:"type:varchar(150)" json:"signerTitle"`

	EmployeeID   *uuid.UUID `gorm:"type:uuid;index" json:"employeeId,omitempty"`
	EmployeeName string     `gorm:"type:varchar(255)" json:"employeeName"`
	NIP          string     `gorm:"column:nip;type:varchar(50);index" json:"nip"`
	Department   string     `gorm:"type:varchar(100)" json:"department"`
	Role         string     `gorm:"type:varchar(100)" json:"role"`

	EffectiveDate string `gorm:"type:varchar(10)" json:"effectiveDate,omitempty"`
	EndDate       string `gorm:"type:varchar(10);index" json:"endDate,omitempty"`
	// Level is the warning level (1-3) of a Surat Peringatan.
	Level int        `gorm:"default:0" json:"level"`
	Data  LetterData `gorm:"type:text;serializer:json" json:"data"`

	Recipients []Recipient `gorm:"foreignKey:LetterID" json:"recipients,omitempty"`

	// Applied is set once the letter's effect (mutation, contract type, termination) was written to the employee record.
	Applied        bool       `gorm:"not null;default:false" json:"applied"`
	AppliedAt      *time.Time `json:"appliedAt,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
	IssuedAt       *time.Time `json:"issuedAt,omitempty"`
	IssuedBy       string     `gorm:"type:varchar(255)" json:"issuedBy,omitempty"`
	CancelReason   string     `gorm:"type:varchar(500)" json:"cancelReason,omitempty"`
	CreatedBy      string     `gorm:"type:varchar(255)" json:"createdBy,omitempty"`
}

func (Letter) TableName() string { return "hrletters_letters" }

// Recipient is an employee a memo or overtime order is addressed to.
type Recipient struct {
	types.BaseEntity
	LetterID   uuid.UUID `gorm:"type:uuid;not null;index" json:"letterId"`
	EmployeeID uuid.UUID `gorm:"type:uuid;not null;index" json:"employeeId"`
	Name       string    `gorm:"type:varchar(255)" json:"name"`
	NIP        string    `gorm:"column:nip;type:varchar(50)" json:"nip"`
	Department string    `gorm:"type:varchar(100)" json:"department"`
}

func (Recipient) TableName() string { return "hrletters_recipients" }

// Filter narrows a letter listing; empty fields are ignored.
type Filter struct {
	Type, Status, From, To, Search string
	EmployeeID                     *uuid.UUID
}

type Repository interface {
	Create(ctx context.Context, l *Letter) error
	Get(ctx context.Context, id uuid.UUID) (*Letter, error)
	// Update saves the letter; recipients are replaced when replaceRecipients is set.
	Update(ctx context.Context, l *Letter, replaceRecipients bool) error
	List(ctx context.Context, f Filter) ([]Letter, error)
	// CountIssued counts the letters of a type issued in a calendar year (for numbering).
	CountIssued(ctx context.Context, letterType string, year int) (int64, error)
	// ListByEmployee returns an employee's non-cancelled letters, newest first.
	ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]Letter, error)
	// ListIssuedContracts returns issued contract letters, for expiry tracking.
	ListIssuedContracts(ctx context.Context) ([]Letter, error)
}
