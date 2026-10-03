package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"github.com/divinecoid/one-backend/internal/shared/attachment"
	"net/mail"
	"regexp"
	"strings"
	"time"

	crmapp "github.com/divinecoid/one-backend/internal/modules/crm/application"
	crmdomain "github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var zone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}()

// Caller identifies who is acting, taken from the JWT by the handler.
type Caller struct {
	Email      string
	Privileged bool // admin or manager
}

// Notifier delivers an in-app notification to one user.
type Notifier interface {
	Notify(ctx context.Context, email, title, body, link string) error
}

// ProjectCreator starts a project from a won deal; it returns the project id and code.
type ProjectCreator func(ctx context.Context, name, customer string, budget float64) (uuid.UUID, string, error)

type Service struct {
	repo     domain.Repository
	crm      crmdomain.CRMRepository
	notifier Notifier        // optional
	projects ProjectCreator  // optional
	store    storage.Storage // optional: holds uploaded documents
	now      func() time.Time
}

func NewService(repo domain.Repository, crm crmdomain.CRMRepository, notifier Notifier, projects ProjectCreator) *Service {
	return &Service{repo: repo, crm: crm, notifier: notifier, projects: projects, now: time.Now}
}

func (s *Service) today() string { return s.now().In(zone).Format("2006-01-02") }

func validParent(t string) bool {
	return t == domain.ParentLead || t == domain.ParentDeal || t == domain.ParentCustomer
}

// checkParent confirms a lead or deal exists. Customers live in another
// module, so only the id shape is checked for them.
func (s *Service) checkParent(ctx context.Context, parentType string, id uuid.UUID) error {
	if !validParent(parentType) {
		return apperrors.NewBadRequest("parentType must be lead, deal or customer")
	}
	switch parentType {
	case domain.ParentLead:
		l, err := s.crm.GetByID(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to look up lead")
		}
		if l == nil {
			return apperrors.NewNotFound("Lead not found")
		}
	case domain.ParentDeal:
		d, err := s.crm.GetDealByID(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to look up deal")
		}
		if d == nil {
			return apperrors.NewNotFound("Deal not found")
		}
	}
	return nil
}

// ---------------------------------------------------------------- interactions

var interactionKinds = map[string]bool{"call": true, "meeting": true, "email": true, "whatsapp": true, "visit": true, "note": true}

type InteractionInput struct {
	ParentType string
	ParentID   uuid.UUID
	Kind       string
	Summary    string
	OccurredAt *time.Time
}

func (s *Service) LogInteraction(ctx context.Context, c Caller, in InteractionInput) (*domain.Interaction, error) {
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	if !interactionKinds[in.Kind] {
		return nil, apperrors.NewBadRequest("kind must be call, meeting, email, whatsapp, visit or note")
	}
	if strings.TrimSpace(in.Summary) == "" {
		return nil, apperrors.NewBadRequest("summary is required")
	}
	if err := s.checkParent(ctx, in.ParentType, in.ParentID); err != nil {
		return nil, err
	}
	at := s.now()
	if in.OccurredAt != nil {
		if in.OccurredAt.After(s.now().Add(5 * time.Minute)) {
			return nil, apperrors.NewBadRequest("occurredAt cannot be in the future")
		}
		at = *in.OccurredAt
	}
	v := &domain.Interaction{ParentType: in.ParentType, ParentID: in.ParentID, Kind: in.Kind, Summary: strings.TrimSpace(in.Summary), OccurredAt: at, CreatedByEmail: c.Email}
	if err := s.repo.CreateInteraction(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save interaction")
	}
	return v, nil
}

func (s *Service) ListInteractions(ctx context.Context, parentType string, parentID uuid.UUID) ([]domain.Interaction, error) {
	if !validParent(parentType) {
		return nil, apperrors.NewBadRequest("parentType must be lead, deal or customer")
	}
	out, err := s.repo.ListInteractions(ctx, parentType, parentID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list interactions")
	}
	return out, nil
}

// ---------------------------------------------------------------------- tasks

type TaskInput struct {
	Title, Notes, DueDate, Priority, AssigneeEmail, ParentType string
	ParentID                                                   *uuid.UUID
}

var priorities = map[string]bool{"low": true, "normal": true, "high": true}

func validDate(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }

func (s *Service) CreateTask(ctx context.Context, c Caller, in TaskInput) (*domain.SalesTask, error) {
	if strings.TrimSpace(in.Title) == "" || !validDate(in.DueDate) {
		return nil, apperrors.NewBadRequest("title and dueDate (YYYY-MM-DD) are required")
	}
	pr := strings.ToLower(strings.TrimSpace(in.Priority))
	if pr == "" {
		pr = "normal"
	}
	if !priorities[pr] {
		return nil, apperrors.NewBadRequest("priority must be low, normal or high")
	}
	if (in.ParentType != "") != (in.ParentID != nil) {
		return nil, apperrors.NewBadRequest("parentType and parentId go together")
	}
	if in.ParentID != nil {
		if err := s.checkParent(ctx, in.ParentType, *in.ParentID); err != nil {
			return nil, err
		}
	}
	assignee := strings.TrimSpace(in.AssigneeEmail)
	if assignee == "" {
		assignee = c.Email
	}
	if _, err := mail.ParseAddress(assignee); err != nil {
		return nil, apperrors.NewBadRequest("assigneeEmail must be a valid email address")
	}
	v := &domain.SalesTask{Title: strings.TrimSpace(in.Title), Notes: strings.TrimSpace(in.Notes), DueDate: in.DueDate, Priority: pr, Status: "open",
		AssigneeEmail: assignee, ParentType: in.ParentType, ParentID: in.ParentID, CreatedByEmail: c.Email}
	if err := s.repo.CreateTask(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save task")
	}
	// Tell the assignee when someone else gives them work.
	if s.notifier != nil && !strings.EqualFold(assignee, c.Email) {
		_ = s.notifier.Notify(ctx, assignee, "Tugas baru: "+v.Title, "Jatuh tempo "+v.DueDate+" · ditugaskan oleh "+c.Email, "/crm/tasks")
	}
	return v, nil
}

// TaskBuckets groups open tasks by urgency for the reminder view.
type TaskBuckets struct {
	Overdue  []domain.SalesTask `json:"overdue"`
	Today    []domain.SalesTask `json:"today"`
	Upcoming []domain.SalesTask `json:"upcoming"`
}

// Bucketize splits open tasks around today (YYYY-MM-DD); done tasks are dropped.
func Bucketize(tasks []domain.SalesTask, today string) TaskBuckets {
	b := TaskBuckets{Overdue: []domain.SalesTask{}, Today: []domain.SalesTask{}, Upcoming: []domain.SalesTask{}}
	for _, t := range tasks {
		if t.Status != "open" {
			continue
		}
		switch {
		case t.DueDate < today:
			b.Overdue = append(b.Overdue, t)
		case t.DueDate == today:
			b.Today = append(b.Today, t)
		default:
			b.Upcoming = append(b.Upcoming, t)
		}
	}
	return b
}

func (s *Service) ListTasks(ctx context.Context, status, assignee, parentType string, parentID *uuid.UUID) ([]domain.SalesTask, error) {
	out, err := s.repo.ListTasks(ctx, status, assignee, parentType, parentID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list tasks")
	}
	return out, nil
}

func (s *Service) TaskReminders(ctx context.Context, assignee string) (*TaskBuckets, error) {
	tasks, err := s.ListTasks(ctx, "open", assignee, "", nil)
	if err != nil {
		return nil, err
	}
	b := Bucketize(tasks, s.today())
	return &b, nil
}

// SetTaskDone completes or reopens a task. Only its assignee, its creator or a
// privileged user may do so.
func (s *Service) SetTaskDone(ctx context.Context, c Caller, id uuid.UUID, done bool) (*domain.SalesTask, error) {
	t, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load task")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Task not found")
	}
	if !c.Privileged && !strings.EqualFold(t.AssigneeEmail, c.Email) && !strings.EqualFold(t.CreatedByEmail, c.Email) {
		return nil, apperrors.NewForbidden("Only the assignee or the creator can change this task")
	}
	if done {
		now := s.now()
		t.Status, t.CompletedAt = "done", &now
	} else {
		t.Status, t.CompletedAt = "open", nil
	}
	if err := s.repo.UpdateTask(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update task")
	}
	return t, nil
}

// ------------------------------------------------------------------ documents

type DocumentInput struct {
	ParentType, Title, DocType, FileRef, Notes string
	ParentID                                   uuid.UUID
}

func (s *Service) AddDocument(ctx context.Context, c Caller, in DocumentInput) (*domain.Document, error) {
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.DocType) == "" {
		return nil, apperrors.NewBadRequest("title and docType are required")
	}
	if err := s.checkParent(ctx, in.ParentType, in.ParentID); err != nil {
		return nil, err
	}
	v := &domain.Document{ParentType: in.ParentType, ParentID: in.ParentID, Title: strings.TrimSpace(in.Title), DocType: strings.TrimSpace(in.DocType),
		FileRef: strings.TrimSpace(in.FileRef), Notes: in.Notes, UploadedBy: c.Email}
	if err := s.repo.CreateDocument(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save document")
	}
	return v, nil
}

func (s *Service) ListDocuments(ctx context.Context, parentType string, parentID uuid.UUID) ([]domain.Document, error) {
	if !validParent(parentType) {
		return nil, apperrors.NewBadRequest("parentType must be lead, deal or customer")
	}
	out, err := s.repo.ListDocuments(ctx, parentType, parentID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list documents")
	}
	return out, nil
}

// WithStorage enables direct file upload for documents.
func (s *Service) WithStorage(st storage.Storage) *Service {
	s.store = st
	return s
}

// AttachDocument adds a document whose file is uploaded and kept in storage.
func (s *Service) AttachDocument(ctx context.Context, c Caller, in DocumentInput, filename string, content []byte) (*domain.Document, error) {
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.DocType) == "" {
		return nil, apperrors.NewBadRequest("title and docType are required")
	}
	if err := s.checkParent(ctx, in.ParentType, in.ParentID); err != nil {
		return nil, err
	}
	saved, err := attachment.Save(ctx, s.store, "crm-documents", filename, content)
	if err != nil {
		return nil, err
	}
	v := &domain.Document{ParentType: in.ParentType, ParentID: in.ParentID, Title: strings.TrimSpace(in.Title), DocType: strings.TrimSpace(in.DocType),
		Notes: in.Notes, UploadedBy: c.Email, FileName: saved.Name, FileSize: saved.Size, StorageKey: saved.Key}
	if err := s.repo.CreateDocument(ctx, v); err != nil {
		attachment.Discard(ctx, s.store, saved.Key)
		return nil, apperrors.NewInternal(err, "Failed to save document")
	}
	return v, nil
}

// DocumentFile returns the document and its file content.
func (s *Service) DocumentFile(ctx context.Context, id uuid.UUID) (*domain.Document, []byte, error) {
	v, err := s.repo.GetDocument(ctx, id)
	if err != nil {
		return nil, nil, apperrors.NewInternal(err, "Failed to load document")
	}
	if v == nil {
		return nil, nil, apperrors.NewNotFound("Document not found")
	}
	if v.StorageKey == "" {
		return nil, nil, apperrors.NewNotFound("This document has no uploaded file")
	}
	data, err := attachment.Load(ctx, s.store, v.StorageKey)
	return v, data, err
}

func (s *Service) DeleteDocument(ctx context.Context, id uuid.UUID) error {
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

// ----------------------------------------------------------------------- tags

var tagColors = map[string]bool{"slate": true, "red": true, "amber": true, "emerald": true, "blue": true, "violet": true, "pink": true}

func (s *Service) CreateTag(ctx context.Context, name, color string) (*domain.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 {
		return nil, apperrors.NewBadRequest("name is required (max 50 characters)")
	}
	color = strings.ToLower(strings.TrimSpace(color))
	if color == "" {
		color = "slate"
	}
	if !tagColors[color] {
		return nil, apperrors.NewBadRequest("color must be one of slate, red, amber, emerald, blue, violet, pink")
	}
	existing, err := s.repo.ListTags(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list tags")
	}
	for _, t := range existing {
		if strings.EqualFold(t.Name, name) {
			return nil, apperrors.NewConflict("A tag named " + t.Name + " already exists")
		}
	}
	v := &domain.Tag{Name: name, Color: color}
	if err := s.repo.CreateTag(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save tag")
	}
	return v, nil
}

func (s *Service) ListTags(ctx context.Context) ([]domain.Tag, error) {
	out, err := s.repo.ListTags(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list tags")
	}
	return out, nil
}

func (s *Service) DeleteTag(ctx context.Context, id uuid.UUID) error {
	ok, err := s.repo.DeleteTag(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete tag")
	}
	if !ok {
		return apperrors.NewNotFound("Tag not found")
	}
	return nil
}

func (s *Service) SetTag(ctx context.Context, tagID uuid.UUID, parentType string, parentID uuid.UUID, on bool) error {
	t, err := s.repo.GetTag(ctx, tagID)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load tag")
	}
	if t == nil {
		return apperrors.NewNotFound("Tag not found")
	}
	if err := s.checkParent(ctx, parentType, parentID); err != nil {
		return err
	}
	if on {
		err = s.repo.AssignTag(ctx, &domain.TagAssignment{TagID: tagID, ParentType: parentType, ParentID: parentID})
	} else {
		err = s.repo.UnassignTag(ctx, tagID, parentType, parentID)
	}
	if err != nil {
		return apperrors.NewInternal(err, "Failed to update tag assignment")
	}
	return nil
}

func (s *Service) TagsFor(ctx context.Context, parentType string, parentID uuid.UUID) ([]domain.Tag, error) {
	if !validParent(parentType) {
		return nil, apperrors.NewBadRequest("parentType must be lead, deal or customer")
	}
	out, err := s.repo.TagsFor(ctx, parentType, parentID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load tags")
	}
	return out, nil
}

func (s *Service) ParentsWithTag(ctx context.Context, tagID uuid.UUID, parentType string) ([]uuid.UUID, error) {
	if !validParent(parentType) {
		return nil, apperrors.NewBadRequest("parentType must be lead, deal or customer")
	}
	out, err := s.repo.ParentsWithTag(ctx, tagID, parentType)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load tagged records")
	}
	return out, nil
}

// ------------------------------------------------------------------- contacts

type ContactInput struct {
	Name, JobTitle, Email, Phone, CompanyName, Notes string
	LeadID                                           *uuid.UUID
	IsPrimary                                        bool
}

func (s *Service) validateContact(ctx context.Context, in *ContactInput) error {
	in.Name, in.CompanyName, in.Email = strings.TrimSpace(in.Name), strings.TrimSpace(in.CompanyName), strings.TrimSpace(in.Email)
	if in.Name == "" {
		return apperrors.NewBadRequest("name is required")
	}
	if in.Email != "" {
		if _, err := mail.ParseAddress(in.Email); err != nil {
			return apperrors.NewBadRequest("email is not valid")
		}
	}
	if in.LeadID != nil {
		l, err := s.crm.GetByID(ctx, *in.LeadID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to look up lead")
		}
		if l == nil {
			return apperrors.NewNotFound("Lead not found")
		}
		if in.CompanyName == "" {
			in.CompanyName = l.Company
		}
	}
	if in.CompanyName == "" {
		return apperrors.NewBadRequest("companyName (or a leadId) is required")
	}
	return nil
}

// demoteOtherPrimaries keeps at most one primary contact per company.
func (s *Service) demoteOtherPrimaries(ctx context.Context, keep uuid.UUID, company string) error {
	list, err := s.repo.ListContacts(ctx, company, nil)
	if err != nil {
		return err
	}
	for i := range list {
		if list[i].ID != keep && list[i].IsPrimary {
			list[i].IsPrimary = false
			if err := s.repo.UpdateContact(ctx, &list[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) CreateContact(ctx context.Context, in ContactInput) (*domain.Contact, error) {
	if err := s.validateContact(ctx, &in); err != nil {
		return nil, err
	}
	v := &domain.Contact{Name: in.Name, JobTitle: strings.TrimSpace(in.JobTitle), Email: in.Email, Phone: strings.TrimSpace(in.Phone),
		CompanyName: in.CompanyName, LeadID: in.LeadID, IsPrimary: in.IsPrimary, Notes: in.Notes}
	if err := s.repo.CreateContact(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save contact")
	}
	if v.IsPrimary {
		if err := s.demoteOtherPrimaries(ctx, v.ID, v.CompanyName); err != nil {
			return v, apperrors.NewInternal(err, "Contact saved but the previous primary contact could not be updated")
		}
	}
	return v, nil
}

func (s *Service) UpdateContact(ctx context.Context, id uuid.UUID, in ContactInput) (*domain.Contact, error) {
	v, err := s.repo.GetContact(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load contact")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Contact not found")
	}
	if err := s.validateContact(ctx, &in); err != nil {
		return nil, err
	}
	v.Name, v.JobTitle, v.Email, v.Phone, v.CompanyName, v.LeadID, v.IsPrimary, v.Notes =
		in.Name, strings.TrimSpace(in.JobTitle), in.Email, strings.TrimSpace(in.Phone), in.CompanyName, in.LeadID, in.IsPrimary, in.Notes
	if err := s.repo.UpdateContact(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update contact")
	}
	if v.IsPrimary {
		if err := s.demoteOtherPrimaries(ctx, v.ID, v.CompanyName); err != nil {
			return v, apperrors.NewInternal(err, "Contact saved but the previous primary contact could not be updated")
		}
	}
	return v, nil
}

func (s *Service) DeleteContact(ctx context.Context, id uuid.UUID) error {
	ok, err := s.repo.DeleteContact(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete contact")
	}
	if !ok {
		return apperrors.NewNotFound("Contact not found")
	}
	return nil
}

func (s *Service) ListContacts(ctx context.Context, companyName string, leadID *uuid.UUID) ([]domain.Contact, error) {
	out, err := s.repo.ListContacts(ctx, companyName, leadID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list contacts")
	}
	return out, nil
}

// --------------------------------------------------------------------- stages

// DefaultStages seeds a new company's pipeline (matches the original fixed one).
func DefaultStages() []domain.PipelineStage {
	return []domain.PipelineStage{
		{Key: "discovery", Name: "Discovery", Position: 1, Probability: 30, Kind: domain.StageOpen, IsActive: true},
		{Key: "quotation", Name: "Quotation", Position: 2, Probability: 60, Kind: domain.StageOpen, IsActive: true},
		{Key: "negotiation", Name: "Negotiation", Position: 3, Probability: 80, Kind: domain.StageOpen, IsActive: true},
		{Key: "won", Name: "Won", Position: 4, Probability: 100, Kind: domain.StageWon, IsActive: true},
		{Key: "lost", Name: "Lost", Position: 5, Probability: 0, Kind: domain.StageLost, IsActive: true},
	}
}

// SeedDefaultStages creates the default pipeline when none exists.
func (s *Service) SeedDefaultStages(ctx context.Context) error {
	n, err := s.repo.CountStages(ctx)
	if err != nil || n > 0 {
		return err
	}
	for _, st := range DefaultStages() {
		st := st
		if err := s.repo.CreateStage(ctx, &st); err != nil {
			return err
		}
	}
	return nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// StageKey turns a name into a stable machine key ("Site Survey" -> "site-survey").
func StageKey(name string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

type StageInput struct {
	Name        string
	Probability int
	Kind        string
	IsActive    *bool
}

func validKind(k string) bool {
	return k == domain.StageOpen || k == domain.StageWon || k == domain.StageLost
}

// ActiveStages implements the crm module's StageProvider.
func (s *Service) ActiveStages(ctx context.Context) ([]crmapp.StageInfo, error) {
	list, err := s.repo.ListStages(ctx, true)
	if err != nil {
		return nil, err
	}
	out := make([]crmapp.StageInfo, len(list))
	for i, st := range list {
		out[i] = crmapp.StageInfo{Key: st.Key, Probability: st.Probability, Kind: st.Kind}
	}
	return out, nil
}

func (s *Service) ListStages(ctx context.Context) ([]domain.PipelineStage, error) {
	out, err := s.repo.ListStages(ctx, false)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list stages")
	}
	return out, nil
}

// settledStillPresent checks the pipeline keeps a won and a lost stage after a
// change; without them a deal could never be closed.
func settledStillPresent(stages []domain.PipelineStage) bool {
	won, lost := false, false
	for _, st := range stages {
		if !st.IsActive {
			continue
		}
		won = won || st.Kind == domain.StageWon
		lost = lost || st.Kind == domain.StageLost
	}
	return won && lost
}

func (s *Service) CreateStage(ctx context.Context, in StageInput) (*domain.PipelineStage, error) {
	name := strings.TrimSpace(in.Name)
	key := StageKey(name)
	if name == "" || key == "" {
		return nil, apperrors.NewBadRequest("name is required")
	}
	if in.Kind == "" {
		in.Kind = domain.StageOpen
	}
	if !validKind(in.Kind) || in.Probability < 0 || in.Probability > 100 {
		return nil, apperrors.NewBadRequest("kind must be open, won or lost and probability 0-100")
	}
	if (in.Kind == domain.StageWon && in.Probability != 100) || (in.Kind == domain.StageLost && in.Probability != 0) {
		return nil, apperrors.NewBadRequest("a won stage has probability 100 and a lost stage 0")
	}
	all, err := s.repo.ListStages(ctx, false)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list stages")
	}
	pos := 1
	for _, st := range all {
		if st.Key == key {
			return nil, apperrors.NewConflict("A stage with this name already exists")
		}
		if st.Position >= pos {
			pos = st.Position + 1
		}
	}
	v := &domain.PipelineStage{Key: key, Name: name, Position: pos, Probability: in.Probability, Kind: in.Kind, IsActive: true}
	if err := s.repo.CreateStage(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save stage")
	}
	return v, nil
}

// UpdateStage renames a stage, changes its probability or activates/deactivates
// it. The key and kind never change, so existing deals keep their meaning.
func (s *Service) UpdateStage(ctx context.Context, id uuid.UUID, in StageInput, position *int) (*domain.PipelineStage, error) {
	v, err := s.repo.GetStage(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load stage")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Stage not found")
	}
	if strings.TrimSpace(in.Name) != "" {
		v.Name = strings.TrimSpace(in.Name)
	}
	if in.Probability != 0 || v.Kind == domain.StageLost {
		if in.Probability < 0 || in.Probability > 100 {
			return nil, apperrors.NewBadRequest("probability must be 0-100")
		}
		if (v.Kind == domain.StageWon && in.Probability != 100) || (v.Kind == domain.StageLost && in.Probability != 0) {
			return nil, apperrors.NewBadRequest("a won stage has probability 100 and a lost stage 0")
		}
		v.Probability = in.Probability
	}
	if position != nil && *position > 0 {
		v.Position = *position
	}
	if in.IsActive != nil {
		v.IsActive = *in.IsActive
		if !v.IsActive {
			all, err := s.repo.ListStages(ctx, false)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to list stages")
			}
			for i := range all {
				if all[i].ID == v.ID {
					all[i].IsActive = false
				}
			}
			if !settledStillPresent(all) {
				return nil, apperrors.NewBadRequest("The pipeline needs at least one active won stage and one active lost stage")
			}
		}
	}
	if err := s.repo.UpdateStage(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update stage")
	}
	return v, nil
}

// DeleteStage removes a stage no deal uses; a stage in use can only be deactivated.
func (s *Service) DeleteStage(ctx context.Context, id uuid.UUID) error {
	v, err := s.repo.GetStage(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load stage")
	}
	if v == nil {
		return apperrors.NewNotFound("Stage not found")
	}
	deals, _, err := s.crm.ListDeals(ctx, types.PaginationQuery{Page: 1, PerPage: 100000})
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check deals")
	}
	for _, d := range deals {
		if d.Stage == v.Key {
			return apperrors.NewConflict("Deals still use this stage; move them first or deactivate the stage instead")
		}
	}
	all, err := s.repo.ListStages(ctx, false)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to list stages")
	}
	rest := all[:0]
	for _, st := range all {
		if st.ID != id {
			rest = append(rest, st)
		}
	}
	if !settledStillPresent(rest) {
		return apperrors.NewBadRequest("The pipeline needs at least one active won stage and one active lost stage")
	}
	if err := s.repo.DeleteStage(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete stage")
	}
	return nil
}

// -------------------------------------------------------- deal -> project link

// StartProjectFromDeal creates a project from a won deal and links the two. A
// deal can start only one project.
func (s *Service) StartProjectFromDeal(ctx context.Context, dealID uuid.UUID) (uuid.UUID, string, error) {
	if s.projects == nil {
		return uuid.Nil, "", apperrors.NewBadRequest("Projects are not available")
	}
	d, err := s.crm.GetDealByID(ctx, dealID)
	if err != nil {
		return uuid.Nil, "", apperrors.NewInternal(err, "Failed to load deal")
	}
	if d == nil {
		return uuid.Nil, "", apperrors.NewNotFound("Deal not found")
	}
	stages, err := s.repo.ListStages(ctx, false)
	if err != nil {
		return uuid.Nil, "", apperrors.NewInternal(err, "Failed to load stages")
	}
	won := d.Stage == "won"
	for _, st := range stages {
		if st.Key == d.Stage {
			won = st.Kind == domain.StageWon
		}
	}
	if !won {
		return uuid.Nil, "", apperrors.NewConflict("Only a won deal can start a project")
	}
	if d.ProjectID != nil {
		return uuid.Nil, "", apperrors.NewConflict("This deal already started a project")
	}
	id, code, err := s.projects(ctx, d.Title, d.Customer, d.Value)
	if err != nil {
		return uuid.Nil, "", err
	}
	d.ProjectID = &id
	if err := s.crm.UpdateDeal(ctx, d); err != nil {
		return id, code, apperrors.NewInternal(err, "Project "+code+" was created but could not be linked to the deal")
	}
	return id, code, nil
}

// ----------------------------------------------------------------- lead forms

func newFormKey() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type LeadFormInput struct {
	Name, Source, DefaultPIC, SuccessMessage string
	IsActive                                 *bool
}

// CreateLeadForm makes a public form and registers its key in the control plane.
func (s *Service) CreateLeadForm(ctx context.Context, reg domain.FormRegistry, companyID uuid.UUID, tenantID *uuid.UUID, in LeadFormInput) (*domain.LeadForm, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, apperrors.NewBadRequest("name is required")
	}
	key, err := newFormKey()
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate form key")
	}
	source := strings.ToLower(strings.TrimSpace(in.Source))
	if source == "" {
		source = "web"
	}
	msg := strings.TrimSpace(in.SuccessMessage)
	if msg == "" {
		msg = "Terima kasih! Tim kami akan segera menghubungi Anda."
	}
	v := &domain.LeadForm{Key: key, Name: strings.TrimSpace(in.Name), Source: source, DefaultPIC: strings.TrimSpace(in.DefaultPIC), SuccessMessage: msg, IsActive: true}
	if err := s.repo.CreateLeadForm(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save form")
	}
	if err := reg.Upsert(ctx, key, companyID, tenantID); err != nil {
		_ = s.repo.DeleteLeadForm(ctx, v.ID)
		return nil, apperrors.NewInternal(err, "Failed to register the form")
	}
	return v, nil
}

func (s *Service) ListLeadForms(ctx context.Context) ([]domain.LeadForm, error) {
	out, err := s.repo.ListLeadForms(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list forms")
	}
	return out, nil
}

func (s *Service) UpdateLeadForm(ctx context.Context, id uuid.UUID, in LeadFormInput) (*domain.LeadForm, error) {
	v, err := s.repo.GetLeadForm(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load form")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Form not found")
	}
	if n := strings.TrimSpace(in.Name); n != "" {
		v.Name = n
	}
	if src := strings.ToLower(strings.TrimSpace(in.Source)); src != "" {
		v.Source = src
	}
	v.DefaultPIC = strings.TrimSpace(in.DefaultPIC)
	if m := strings.TrimSpace(in.SuccessMessage); m != "" {
		v.SuccessMessage = m
	}
	if in.IsActive != nil {
		v.IsActive = *in.IsActive
	}
	if err := s.repo.UpdateLeadForm(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update form")
	}
	return v, nil
}

func (s *Service) DeleteLeadForm(ctx context.Context, reg domain.FormRegistry, id uuid.UUID) error {
	v, err := s.repo.GetLeadForm(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load form")
	}
	if v == nil {
		return apperrors.NewNotFound("Form not found")
	}
	// Unregister first: once the key is gone the form can no longer receive submissions.
	if err := reg.Delete(ctx, v.Key); err != nil {
		return apperrors.NewInternal(err, "Failed to unregister the form")
	}
	if err := s.repo.DeleteLeadForm(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete form")
	}
	return nil
}

// Submission is what a visitor sends to a public form.
type Submission struct {
	Name, Company, Email, Phone, Message string
	// Honeypot is a field real visitors never see; bots fill it in.
	Honeypot string
}

var phoneRe = regexp.MustCompile(`^[0-9+()\-\s]{6,30}$`)

// ValidateSubmission checks a public submission.
func ValidateSubmission(in *Submission) error {
	in.Name, in.Company, in.Email, in.Phone = strings.TrimSpace(in.Name), strings.TrimSpace(in.Company), strings.TrimSpace(in.Email), strings.TrimSpace(in.Phone)
	in.Message = strings.TrimSpace(in.Message)
	if in.Name == "" || len(in.Name) > 255 || len(in.Company) > 255 || len(in.Message) > 2000 {
		return apperrors.NewBadRequest("A name is required, and fields must not be too long")
	}
	if in.Email == "" && in.Phone == "" {
		return apperrors.NewBadRequest("Please give an email or a phone number so we can reach you")
	}
	if in.Email != "" {
		if _, err := mail.ParseAddress(in.Email); err != nil {
			return apperrors.NewBadRequest("The email address is not valid")
		}
	}
	if in.Phone != "" && !phoneRe.MatchString(in.Phone) {
		return apperrors.NewBadRequest("The phone number is not valid")
	}
	return nil
}

// AcceptSubmission turns a valid submission into a lead (plus a note carrying
// the visitor's message). A filled honeypot is silently accepted and dropped, so
// bots get no signal that they were detected.
func (s *Service) AcceptSubmission(ctx context.Context, form *domain.LeadForm, in Submission) error {
	if strings.TrimSpace(in.Honeypot) != "" {
		return nil
	}
	if !form.IsActive {
		return apperrors.NewNotFound("This form is not accepting submissions")
	}
	if err := ValidateSubmission(&in); err != nil {
		return err
	}
	company := in.Company
	if company == "" {
		company = in.Name
	}
	lead := &crmdomain.Lead{Name: in.Name, Company: company, Email: in.Email, Phone: in.Phone, Segment: "Web Inquiry", Source: form.Source, Status: "New", PIC: form.DefaultPIC}
	if err := s.crm.Create(ctx, lead); err != nil {
		return apperrors.NewInternal(err, "Failed to save your details")
	}
	if in.Message != "" {
		_ = s.repo.CreateInteraction(ctx, &domain.Interaction{ParentType: domain.ParentLead, ParentID: lead.ID, Kind: "note",
			Summary: "Pesan dari formulir \"" + form.Name + "\": " + in.Message, OccurredAt: s.now()})
	}
	_ = s.repo.IncrementSubmissions(ctx, form.ID)
	if s.notifier != nil && form.DefaultPIC != "" {
		if _, err := mail.ParseAddress(form.DefaultPIC); err == nil {
			_ = s.notifier.Notify(ctx, form.DefaultPIC, "Lead baru dari formulir web", fmt.Sprintf("%s (%s)", in.Name, company), "/crm/leads")
		}
	}
	return nil
}

// ------------------------------------------------------------------ analytics

func (s *Service) GetAnalytics(ctx context.Context, from, to string) (*Analytics, error) {
	for _, d := range []string{from, to} {
		if d != "" && !validDate(d) {
			return nil, apperrors.NewBadRequest("from and to must be YYYY-MM-DD")
		}
	}
	page := types.PaginationQuery{Page: 1, PerPage: 100000}
	deals, _, err := s.crm.ListDeals(ctx, page)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load deals")
	}
	leads, _, err := s.crm.List(ctx, page)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load leads")
	}
	stages, err := s.repo.ListStages(ctx, false)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load stages")
	}
	tasks, err := s.repo.ListTasks(ctx, "", "", "", nil)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load tasks")
	}
	last, err := s.repo.LastInteractionByParent(ctx, domain.ParentDeal)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load interactions")
	}
	a := BuildAnalytics(AnalyticsInput{Deals: deals, Leads: leads, Stages: stages, Tasks: tasks, LastInteraction: last,
		From: from, To: to, Today: s.today(), Now: s.now(), StaleAfterDays: 14})
	return &a, nil
}
