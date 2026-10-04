package application

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrletters/domain"
	hrmDomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// Employees is the part of the employee records HR letters read and change.
type Employees interface {
	GetEmployeeByID(ctx context.Context, id uuid.UUID) (*hrmDomain.Employee, error)
	UpdateEmployee(ctx context.Context, emp *hrmDomain.Employee) error
}

type UseCase interface {
	Create(ctx context.Context, in LetterInput) (*domain.Letter, error)
	Update(ctx context.Context, id uuid.UUID, in LetterInput) (*domain.Letter, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Letter, error)
	List(ctx context.Context, f domain.Filter) ([]domain.Letter, error)
	Issue(ctx context.Context, id uuid.UUID) (*domain.Letter, error)
	Cancel(ctx context.Context, id uuid.UUID, reason string) (*domain.Letter, error)
	Acknowledge(ctx context.Context, id uuid.UUID) (*domain.Letter, error)
	Apply(ctx context.Context, id uuid.UUID) (*domain.Letter, error)
	Preview(ctx context.Context, in LetterInput) (string, error)
	EmployeeSummary(ctx context.Context, employeeID uuid.UUID) (*EmployeeSummaryDTO, error)
	ExpiringContracts(ctx context.Context, days int) ([]ExpiringContract, error)
}

type useCase struct {
	repo      domain.Repository
	employees Employees
	now       func() time.Time
}

func NewUseCase(repo domain.Repository, employees Employees) UseCase {
	return &useCase{repo: repo, employees: employees, now: time.Now}
}

func (uc *useCase) today() string { return uc.now().Format("2006-01-02") }

func validDate(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }

func validClock(s string) bool { _, err := time.Parse("15:04", s); return err == nil }

type LetterInput struct {
	Type          string            `json:"type"`
	Date          string            `json:"date"`
	Subject       string            `json:"subject"`
	Body          string            `json:"body"`
	CompanyName   string            `json:"companyName"`
	City          string            `json:"city"`
	SignerName    string            `json:"signerName"`
	SignerTitle   string            `json:"signerTitle"`
	EmployeeID    *uuid.UUID        `json:"employeeId"`
	EffectiveDate string            `json:"effectiveDate"`
	EndDate       string            `json:"endDate"`
	Level         int               `json:"level"`
	Data          domain.LetterData `json:"data"`
	RecipientIDs  []uuid.UUID       `json:"recipientIds"`
}

// needsEmployee lists the letter types that are about one employee.
func needsEmployee(t string) bool {
	switch t {
	case domain.TypeContract, domain.TypeSummons, domain.TypeReprimand, domain.TypeWarning, domain.TypeTermination, domain.TypeMutation:
		return true
	}
	return false
}

// build validates the input and produces the letter to save: employee snapshot,
// dates derived from the type's data, recipients and the rendered body. existing
// (the draft being edited) keeps its identity and audit fields.
func (uc *useCase) build(ctx context.Context, in LetterInput, existing *domain.Letter) (*domain.Letter, error) {
	k, ok := kinds[in.Type]
	if !ok {
		return nil, apperrors.NewBadRequest("Unknown letter type")
	}
	date := in.Date
	if date == "" {
		date = uc.today()
	}
	if !validDate(date) {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	l := &domain.Letter{Type: in.Type, Status: domain.StatusDraft, Date: date, CompanyName: strings.TrimSpace(in.CompanyName), City: strings.TrimSpace(in.City),
		SignerName: strings.TrimSpace(in.SignerName), SignerTitle: strings.TrimSpace(in.SignerTitle), Level: in.Level, Data: in.Data,
		EffectiveDate: in.EffectiveDate, EndDate: in.EndDate, CreatedBy: actor.EmailFrom(ctx)}
	if existing != nil {
		l.BaseEntity, l.TenantID, l.CreatedBy = existing.BaseEntity, existing.TenantID, existing.CreatedBy
	}

	if needsEmployee(in.Type) && in.EmployeeID == nil {
		return nil, apperrors.NewBadRequest("An employee is required for a " + strings.ToLower(k.label))
	}
	if in.EmployeeID != nil {
		emp, err := uc.employees.GetEmployeeByID(ctx, *in.EmployeeID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to load the employee")
		}
		if emp == nil {
			return nil, apperrors.NewNotFound("Employee not found")
		}
		id := emp.ID
		l.EmployeeID, l.EmployeeName, l.NIP, l.Department, l.Role = &id, emp.Name, emp.NIP, emp.Department, emp.Role
		if in.Type == domain.TypeMutation {
			l.Data.FromDepartment, l.Data.FromRole = emp.Department, emp.Role
		}
	}

	// Recipients: who a memo or overtime order is addressed to.
	for _, rid := range in.RecipientIDs {
		emp, err := uc.employees.GetEmployeeByID(ctx, rid)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to load a recipient")
		}
		if emp == nil {
			return nil, apperrors.NewNotFound("A recipient was not found")
		}
		l.Recipients = append(l.Recipients, domain.Recipient{EmployeeID: emp.ID, Name: emp.Name, NIP: emp.NIP, Department: emp.Department})
	}

	if err := uc.validateAndDerive(l); err != nil {
		return nil, err
	}
	l.Subject = strings.TrimSpace(in.Subject)
	if l.Subject == "" {
		l.Subject = defaultSubject(l)
	}
	l.Body = strings.TrimSpace(in.Body)
	if l.Body == "" {
		if l.Type == domain.TypeMemo {
			return nil, apperrors.NewBadRequest("A memo needs a body")
		}
		l.Body = RenderBody(l)
	}
	return l, nil
}

// validateAndDerive checks the type-specific data and fills the dates the type implies.
func (uc *useCase) validateAndDerive(l *domain.Letter) error {
	d := &l.Data
	bad := func(msg string) error { return apperrors.NewBadRequest(msg) }
	switch l.Type {
	case domain.TypeContract:
		switch d.ContractType {
		case "PKWT", "PKWTT", "Magang":
		default:
			return bad("contractType must be PKWT, PKWTT or Magang")
		}
		if !validDate(l.EffectiveDate) {
			return bad("The contract start date (effectiveDate) is required")
		}
		if d.Salary <= 0 || strings.TrimSpace(d.Position) == "" {
			return bad("A contract needs a position and a positive salary")
		}
		if d.ContractType == "PKWTT" {
			l.EndDate = ""
		} else {
			if !validDate(l.EndDate) || l.EndDate <= l.EffectiveDate {
				return bad("A PKWT or internship needs an end date after the start date")
			}
			start, _ := time.Parse("2006-01-02", l.EffectiveDate)
			if d.ContractType == "PKWT" && l.EndDate > start.AddDate(5, 0, -1).Format("2006-01-02") {
				return bad("A PKWT cannot run longer than 5 years in total")
			}
		}
		if d.ProbationMonth < 0 || d.ProbationMonth > 3 {
			return bad("The probation period can be at most 3 months")
		}
		d.ApplyToEmployee = true // a contract updates the employee's contract type
	case domain.TypeSummons:
		if !validDate(d.MeetingDate) || strings.TrimSpace(d.Place) == "" || strings.TrimSpace(d.Reason) == "" || !validClock(d.MeetingTime) {
			return bad("A summons needs a meeting date, time (HH:MM), place and reason")
		}
		l.EffectiveDate = d.MeetingDate
	case domain.TypeReprimand:
		if strings.TrimSpace(d.Violation) == "" {
			return bad("Describe the violation")
		}
	case domain.TypeWarning:
		if l.Level < 1 || l.Level > 3 {
			return bad("A warning level must be 1, 2 or 3")
		}
		if strings.TrimSpace(d.Violation) == "" {
			return bad("Describe the violation")
		}
		if d.ValidMonths == 0 {
			d.ValidMonths = 6
		}
		if d.ValidMonths < 1 || d.ValidMonths > 12 {
			return bad("A warning is valid for 1 to 12 months")
		}
		if l.EffectiveDate == "" {
			l.EffectiveDate = l.Date
		}
		if !validDate(l.EffectiveDate) {
			return bad("effectiveDate must be YYYY-MM-DD")
		}
		start, _ := time.Parse("2006-01-02", l.EffectiveDate)
		l.EndDate = start.AddDate(0, d.ValidMonths, 0).Format("2006-01-02")
	case domain.TypeTermination:
		if !validDate(d.LastWorkDay) || strings.TrimSpace(d.TerminationReason) == "" {
			return bad("A termination needs a reason and the last working day")
		}
		if d.Severance < 0 || d.ServiceAward < 0 || d.CompensationRights < 0 {
			return bad("Amounts cannot be negative")
		}
		l.EffectiveDate = d.LastWorkDay
		d.ApplyToEmployee = true // a termination updates the employee's status when the last working day comes
	case domain.TypeMemo:
		if len(l.Recipients) == 0 && !d.AudienceAll && strings.TrimSpace(d.AudienceDepartment) == "" {
			return bad("Address the memo to everyone, a department, or specific employees")
		}
	case domain.TypeOvertimeOrder:
		if len(l.Recipients) == 0 {
			return bad("An overtime order needs at least one employee")
		}
		if !validDate(d.WorkDate) || !validClock(d.StartTime) || !validClock(d.EndTime) || strings.TrimSpace(d.Tasks) == "" {
			return bad("An overtime order needs a work date, start and end time (HH:MM) and the tasks")
		}
		if d.EndTime <= d.StartTime {
			return bad("The end time must be after the start time")
		}
		l.EffectiveDate = d.WorkDate
	case domain.TypeMutation:
		if strings.TrimSpace(d.ToDepartment) == "" && strings.TrimSpace(d.ToRole) == "" && strings.TrimSpace(d.ToManagerID) == "" {
			return bad("A mutation needs a new department, position or supervisor")
		}
		if d.ToManagerID != "" {
			mid, err := uuid.Parse(d.ToManagerID)
			if err != nil {
				return bad("toManagerId is not a valid id")
			}
			if l.EmployeeID != nil && mid == *l.EmployeeID {
				return bad("An employee cannot be their own supervisor")
			}
		}
		if !validDate(l.EffectiveDate) {
			return bad("The mutation effective date is required")
		}
		d.ApplyToEmployee = true // a mutation updates the employee's department, position and supervisor
	}
	if l.EffectiveDate != "" && !validDate(l.EffectiveDate) {
		return bad("effectiveDate must be YYYY-MM-DD")
	}
	return nil
}

func (uc *useCase) Create(ctx context.Context, in LetterInput) (*domain.Letter, error) {
	l, err := uc.build(ctx, in, nil)
	if err != nil {
		return nil, err
	}
	l.ID = uuid.New()
	for i := range l.Recipients {
		l.Recipients[i].ID, l.Recipients[i].LetterID = uuid.New(), l.ID
	}
	if err := uc.repo.Create(ctx, l); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save the letter")
	}
	return l, nil
}

// Preview renders the default body for the given input without saving anything.
func (uc *useCase) Preview(ctx context.Context, in LetterInput) (string, error) {
	in.Body = ""
	if in.Type == domain.TypeMemo {
		return "", apperrors.NewBadRequest("A memo has no standard text")
	}
	l, err := uc.build(ctx, in, nil)
	if err != nil {
		return "", err
	}
	return l.Body, nil
}

func (uc *useCase) Get(ctx context.Context, id uuid.UUID) (*domain.Letter, error) {
	l, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the letter")
	}
	if l == nil {
		return nil, apperrors.NewNotFound("Letter not found")
	}
	return l, nil
}

func (uc *useCase) Update(ctx context.Context, id uuid.UUID, in LetterInput) (*domain.Letter, error) {
	cur, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status != domain.StatusDraft {
		return nil, apperrors.NewBadRequest("Only a draft letter can be edited")
	}
	in.Type = cur.Type
	l, err := uc.build(ctx, in, cur)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Update(ctx, l, true); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update the letter")
	}
	return l, nil
}

func (uc *useCase) List(ctx context.Context, f domain.Filter) ([]domain.Letter, error) {
	for _, d := range []string{f.From, f.To} {
		if d != "" && !validDate(d) {
			return nil, apperrors.NewBadRequest("from and to must be YYYY-MM-DD")
		}
	}
	if f.Type != "" {
		if _, ok := kinds[f.Type]; !ok {
			return nil, apperrors.NewBadRequest("Unknown letter type")
		}
	}
	out, err := uc.repo.List(ctx, f)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list letters")
	}
	return out, nil
}

// Issue finalises a draft: it gets its number and, when it changes the employee
// record (mutation, contract, termination) and is already in effect, that change
// is applied.
func (uc *useCase) Issue(ctx context.Context, id uuid.UUID) (*domain.Letter, error) {
	l, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if l.Status != domain.StatusDraft {
		return nil, apperrors.NewBadRequest("Only a draft letter can be issued")
	}
	if strings.TrimSpace(l.SignerName) == "" {
		return nil, apperrors.NewBadRequest("Fill in the signer before issuing the letter")
	}
	// If the letter takes effect now, make sure its change can be written before it is numbered.
	if uc.appliesToEmployee(l) && l.EffectiveDate <= uc.today() {
		emp, err := uc.loadEmployee(ctx, *l.EmployeeID)
		if err != nil {
			return nil, err
		}
		if err := changeEmployee(emp, l); err != nil {
			return nil, err
		}
	}
	year := 0
	if t, err := time.Parse("2006-01-02", l.Date); err == nil {
		year = t.Year()
	}
	seq, err := uc.repo.CountIssued(ctx, l.Type, year)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to number the letter")
	}
	now := uc.now()
	l.Number, l.Status, l.IssuedAt, l.IssuedBy = LetterNumber(l.Type, seq+1, l.Date), domain.StatusIssued, &now, actor.EmailFrom(ctx)
	if err := uc.repo.Update(ctx, l, false); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to issue the letter")
	}
	if uc.appliesToEmployee(l) && l.EffectiveDate <= uc.today() {
		if err := uc.apply(ctx, l); err != nil {
			return l, err
		}
	}
	return l, nil
}

func (uc *useCase) appliesToEmployee(l *domain.Letter) bool {
	switch l.Type {
	case domain.TypeMutation, domain.TypeContract, domain.TypeTermination:
		return l.Data.ApplyToEmployee && l.EmployeeID != nil && !l.Applied
	}
	return false
}

// Apply writes an issued letter's effect onto the employee record once its
// effective date has come (for letters dated in the future when issued).
func (uc *useCase) Apply(ctx context.Context, id uuid.UUID) (*domain.Letter, error) {
	l, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if l.Status != domain.StatusIssued {
		return nil, apperrors.NewBadRequest("Only an issued letter can be applied")
	}
	if !uc.appliesToEmployee(l) {
		return nil, apperrors.NewBadRequest("This letter has nothing to apply, or was already applied")
	}
	if l.EffectiveDate > uc.today() {
		return nil, apperrors.NewBadRequest("The letter is not in effect yet (" + idDate(l.EffectiveDate) + ")")
	}
	if err := uc.apply(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

var contractLabel = map[string]string{"PKWT": "PKWT", "PKWTT": "PKWTT Tetap", "Magang": "Magang"}

// changeEmployee writes a letter's effect onto an employee record in memory.
func changeEmployee(emp *hrmDomain.Employee, l *domain.Letter) error {
	switch l.Type {
	case domain.TypeMutation:
		if v := strings.TrimSpace(l.Data.ToDepartment); v != "" {
			emp.Department = v
		}
		if v := strings.TrimSpace(l.Data.ToRole); v != "" {
			emp.Role = v
		}
		if l.Data.ToManagerID != "" {
			mid, _ := uuid.Parse(l.Data.ToManagerID)
			if mid == emp.ID {
				return apperrors.NewBadRequest("An employee cannot be their own supervisor")
			}
			emp.ManagerID = &mid
		}
	case domain.TypeContract:
		emp.ContractType = contractLabel[l.Data.ContractType]
	case domain.TypeTermination:
		emp.Status = "Terminated"
	}
	return nil
}

func (uc *useCase) loadEmployee(ctx context.Context, id uuid.UUID) (*hrmDomain.Employee, error) {
	emp, err := uc.employees.GetEmployeeByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the employee")
	}
	if emp == nil {
		return nil, apperrors.NewNotFound("Employee not found")
	}
	return emp, nil
}

func (uc *useCase) apply(ctx context.Context, l *domain.Letter) error {
	emp, err := uc.loadEmployee(ctx, *l.EmployeeID)
	if err != nil {
		return err
	}
	if err := changeEmployee(emp, l); err != nil {
		return err
	}
	if err := uc.employees.UpdateEmployee(ctx, emp); err != nil {
		return apperrors.NewInternal(err, "Failed to update the employee record")
	}
	now := uc.now()
	l.Applied, l.AppliedAt = true, &now
	if err := uc.repo.Update(ctx, l, false); err != nil {
		return apperrors.NewInternal(err, "The employee record was updated but the letter could not be marked as applied")
	}
	return nil
}

func (uc *useCase) Cancel(ctx context.Context, id uuid.UUID, reason string) (*domain.Letter, error) {
	l, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if l.Status == domain.StatusCancelled {
		return nil, apperrors.NewBadRequest("The letter is already cancelled")
	}
	if l.Applied {
		return nil, apperrors.NewBadRequest("This letter was already applied to the employee record; issue a correcting letter instead")
	}
	reason = strings.TrimSpace(reason)
	if l.Status == domain.StatusIssued && reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required to cancel an issued letter")
	}
	l.Status, l.CancelReason = domain.StatusCancelled, reason
	if err := uc.repo.Update(ctx, l, false); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to cancel the letter")
	}
	return l, nil
}

// Acknowledge records that the employee received the letter.
func (uc *useCase) Acknowledge(ctx context.Context, id uuid.UUID) (*domain.Letter, error) {
	l, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if l.Status != domain.StatusIssued {
		return nil, apperrors.NewBadRequest("Only an issued letter can be acknowledged")
	}
	if l.AcknowledgedAt == nil {
		now := uc.now()
		l.AcknowledgedAt = &now
		if err := uc.repo.Update(ctx, l, false); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to record the acknowledgement")
		}
	}
	return l, nil
}

// Employee summary and contract expiry

type EmployeeSummaryDTO struct {
	EmployeeID uuid.UUID `json:"employeeId"`
	// ActiveWarnings are issued warnings still within their validity period.
	ActiveWarnings []domain.Letter `json:"activeWarnings"`
	// HighestActiveLevel is the highest level among them (0 = none), so the next warning can follow the sequence.
	HighestActiveLevel int             `json:"highestActiveLevel"`
	Mutations          []domain.Letter `json:"mutations"`
	Letters            []domain.Letter `json:"letters"`
	Unacknowledged     int             `json:"unacknowledged"`
}

func (uc *useCase) EmployeeSummary(ctx context.Context, employeeID uuid.UUID) (*EmployeeSummaryDTO, error) {
	letters, err := uc.repo.ListByEmployee(ctx, employeeID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the employee's letters")
	}
	out := &EmployeeSummaryDTO{EmployeeID: employeeID, ActiveWarnings: []domain.Letter{}, Mutations: []domain.Letter{}, Letters: letters}
	today := uc.today()
	for _, l := range letters {
		if l.Status != domain.StatusIssued {
			continue
		}
		if l.AcknowledgedAt == nil && l.Type != domain.TypeContract && l.Type != domain.TypeMemo {
			out.Unacknowledged++
		}
		switch l.Type {
		case domain.TypeWarning:
			if l.EndDate >= today && l.EffectiveDate <= today {
				out.ActiveWarnings = append(out.ActiveWarnings, l)
				if l.Level > out.HighestActiveLevel {
					out.HighestActiveLevel = l.Level
				}
			}
		case domain.TypeMutation:
			out.Mutations = append(out.Mutations, l)
		}
	}
	return out, nil
}

type ExpiringContract struct {
	Letter       domain.Letter `json:"letter"`
	DaysLeft     int           `json:"daysLeft"` // negative = already expired
	Status       string        `json:"status"`   // expiring | expired
	EmployeeName string        `json:"employeeName"`
}

// ExpiringContracts lists fixed-term contracts that end within the next days
// (or ended in the last 30) and have no later contract letter for the same
// employee, so HR can renew or let them lapse in time.
func (uc *useCase) ExpiringContracts(ctx context.Context, days int) ([]ExpiringContract, error) {
	if days <= 0 || days > 365 {
		days = 60
	}
	letters, err := uc.repo.ListIssuedContracts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load contracts")
	}
	today := uc.now().Format("2006-01-02")
	horizon := uc.now().AddDate(0, 0, days).Format("2006-01-02")
	lapsedSince := uc.now().AddDate(0, 0, -30).Format("2006-01-02")
	latest := map[uuid.UUID]domain.Letter{}
	for _, l := range letters {
		if l.EmployeeID == nil {
			continue
		}
		if cur, ok := latest[*l.EmployeeID]; !ok || l.EffectiveDate > cur.EffectiveDate {
			latest[*l.EmployeeID] = l
		}
	}
	out := []ExpiringContract{}
	for _, l := range latest {
		if l.Data.ContractType == "PKWTT" || l.EndDate == "" || l.EndDate > horizon || l.EndDate < lapsedSince {
			continue
		}
		emp, err := uc.employees.GetEmployeeByID(ctx, *l.EmployeeID)
		if err != nil || emp == nil || emp.Status != "Active" {
			continue
		}
		end, _ := time.Parse("2006-01-02", l.EndDate)
		now, _ := time.Parse("2006-01-02", today)
		left := int(end.Sub(now).Hours() / 24)
		status := "expiring"
		if left < 0 {
			status = "expired"
		}
		out = append(out, ExpiringContract{Letter: l, DaysLeft: left, Status: status, EmployeeName: l.EmployeeName})
	}
	for i := 1; i < len(out); i++ { // oldest end date first
		for j := i; j > 0 && out[j].Letter.EndDate < out[j-1].Letter.EndDate; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}
