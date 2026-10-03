package http

import (
	"context"
	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	crminfra "github.com/divinecoid/one-backend/internal/modules/crm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/application"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/infrastructure"
	projectapp "github.com/divinecoid/one-backend/internal/modules/project/application"
	projectinfra "github.com/divinecoid/one-backend/internal/modules/project/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	registry domain.FormRegistry
	store    storage.Storage
}

// WithStorage enables direct file upload for documents.
func (h *Handler) WithStorage(st storage.Storage) *Handler {
	h.store = st
	return h
}

func NewHandler(registry domain.FormRegistry) *Handler { return &Handler{registry: registry} }

// serviceFor builds the service against one company's tenant database.
func serviceFor(db *gorm.DB) *application.Service {
	projects := func(ctx context.Context, name, customer string, budget float64) (uuid.UUID, string, error) {
		uc := projectapp.NewProjectUseCase(projectinfra.NewProjectRepository(db), nil, nil)
		p, err := uc.Create(ctx, projectapp.CreateProjectDTO{Name: name, Customer: customer, RABValue: budget})
		if err != nil {
			return uuid.Nil, "", err
		}
		return p.ID, p.Code, nil
	}
	return application.NewService(infrastructure.NewRepository(db), crminfra.NewCRMRepository(db), infrastructure.NewNotifier(db), projects)
}

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
	return serviceFor(db).WithStorage(h.store), application.Caller{Email: claims.Email, Privileged: role == "admin" || role == "manager"}, nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func idParam(c *fiber.Ctx) (uuid.UUID, error) {
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

func optUUID(c *fiber.Ctx, name string) (*uuid.UUID, error) {
	raw := c.Query(name)
	if raw == "" {
		return nil, nil
	}
	v, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperrors.NewBadRequest("Invalid " + name)
	}
	return &v, nil
}

// wrap runs fn with the resolved service and caller and writes an OK response.
func (h *Handler) wrap(msg string, status int, fn func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		s, caller, err := h.resolve(c)
		if err != nil {
			return err
		}
		data, err := fn(c, s, caller)
		if err != nil {
			return err
		}
		return response.Success(c, status, msg, data)
	}
}

// ---- interactions

func (h *Handler) LogInteraction() fiber.Handler {
	return h.wrap("Interaction logged", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		var in struct {
			ParentType string     `json:"parentType"`
			ParentID   uuid.UUID  `json:"parentId"`
			Kind       string     `json:"kind"`
			Summary    string     `json:"summary"`
			OccurredAt *time.Time `json:"occurredAt"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.LogInteraction(h.ctx(c), caller, application.InteractionInput(in))
	})
}

func (h *Handler) ListInteractions() fiber.Handler {
	return h.wrap("Interactions retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		pid, err := uuid.Parse(c.Query("parentId"))
		if err != nil {
			return nil, apperrors.NewBadRequest("parentId is required")
		}
		return s.ListInteractions(h.ctx(c), c.Query("parentType"), pid)
	})
}

// ---- tasks

func (h *Handler) CreateTask() fiber.Handler {
	return h.wrap("Task created", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		var in struct {
			Title         string     `json:"title"`
			Notes         string     `json:"notes"`
			DueDate       string     `json:"dueDate"`
			Priority      string     `json:"priority"`
			AssigneeEmail string     `json:"assigneeEmail"`
			ParentType    string     `json:"parentType"`
			ParentID      *uuid.UUID `json:"parentId"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.CreateTask(h.ctx(c), caller, application.TaskInput{Title: in.Title, Notes: in.Notes, DueDate: in.DueDate, Priority: in.Priority,
			AssigneeEmail: in.AssigneeEmail, ParentType: in.ParentType, ParentID: in.ParentID})
	})
}

func (h *Handler) ListTasks() fiber.Handler {
	return h.wrap("Tasks retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		pid, err := optUUID(c, "parentId")
		if err != nil {
			return nil, err
		}
		assignee := c.Query("assignee")
		if c.Query("mine") == "true" {
			assignee = caller.Email
		}
		return s.ListTasks(h.ctx(c), c.Query("status"), assignee, c.Query("parentType"), pid)
	})
}

func (h *Handler) Reminders() fiber.Handler {
	return h.wrap("Reminders retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		return s.TaskReminders(h.ctx(c), caller.Email)
	})
}

func (h *Handler) SetTaskDone(done bool) fiber.Handler {
	return h.wrap("Task updated", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		return s.SetTaskDone(h.ctx(c), caller, id, done)
	})
}

// ---- documents

func (h *Handler) AddDocument() fiber.Handler {
	return h.wrap("Document added", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		if fh, ferr := c.FormFile("file"); ferr == nil && fh != nil {
			pid, err := uuid.Parse(c.FormValue("parentId"))
			if err != nil {
				return nil, apperrors.NewBadRequest("parentId is required")
			}
			content, err := readUpload(fh)
			if err != nil {
				return nil, err
			}
			return s.AttachDocument(h.ctx(c), caller, application.DocumentInput{ParentType: c.FormValue("parentType"), ParentID: pid, Title: c.FormValue("title"),
				DocType: c.FormValue("docType"), Notes: c.FormValue("notes")}, fh.Filename, content)
		}
		var in struct {
			ParentType string    `json:"parentType"`
			ParentID   uuid.UUID `json:"parentId"`
			Title      string    `json:"title"`
			DocType    string    `json:"docType"`
			FileRef    string    `json:"fileRef"`
			Notes      string    `json:"notes"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.AddDocument(h.ctx(c), caller, application.DocumentInput{ParentType: in.ParentType, ParentID: in.ParentID, Title: in.Title, DocType: in.DocType, FileRef: in.FileRef, Notes: in.Notes})
	})
}

func (h *Handler) ListDocuments() fiber.Handler {
	return h.wrap("Documents retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		pid, err := uuid.Parse(c.Query("parentId"))
		if err != nil {
			return nil, apperrors.NewBadRequest("parentId is required")
		}
		return s.ListDocuments(h.ctx(c), c.Query("parentType"), pid)
	})
}

func (h *Handler) DeleteDocument() fiber.Handler {
	return h.wrap("Document deleted", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		return nil, s.DeleteDocument(h.ctx(c), id)
	})
}

// ---- tags

func (h *Handler) CreateTag() fiber.Handler {
	return h.wrap("Tag created", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		var in struct{ Name, Color string }
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.CreateTag(h.ctx(c), in.Name, in.Color)
	})
}

func (h *Handler) ListTags() fiber.Handler {
	return h.wrap("Tags retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		return s.ListTags(h.ctx(c))
	})
}

func (h *Handler) DeleteTag() fiber.Handler {
	return h.wrap("Tag deleted", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		return nil, s.DeleteTag(h.ctx(c), id)
	})
}

func (h *Handler) SetTag(on bool) fiber.Handler {
	return h.wrap("Tag updated", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		var in struct {
			ParentType string    `json:"parentType"`
			ParentID   uuid.UUID `json:"parentId"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return nil, s.SetTag(h.ctx(c), id, in.ParentType, in.ParentID, on)
	})
}

func (h *Handler) TagsFor() fiber.Handler {
	return h.wrap("Tags retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		pid, err := uuid.Parse(c.Query("parentId"))
		if err != nil {
			return nil, apperrors.NewBadRequest("parentId is required")
		}
		return s.TagsFor(h.ctx(c), c.Query("parentType"), pid)
	})
}

func (h *Handler) ParentsWithTag() fiber.Handler {
	return h.wrap("Tagged records retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		return s.ParentsWithTag(h.ctx(c), id, c.Query("parentType"))
	})
}

// ---- contacts

type contactBody struct {
	Name        string     `json:"name"`
	JobTitle    string     `json:"jobTitle"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	CompanyName string     `json:"companyName"`
	Notes       string     `json:"notes"`
	LeadID      *uuid.UUID `json:"leadId"`
	IsPrimary   bool       `json:"isPrimary"`
}

func (b contactBody) input() application.ContactInput {
	return application.ContactInput{Name: b.Name, JobTitle: b.JobTitle, Email: b.Email, Phone: b.Phone, CompanyName: b.CompanyName, Notes: b.Notes, LeadID: b.LeadID, IsPrimary: b.IsPrimary}
}

func (h *Handler) CreateContact() fiber.Handler {
	return h.wrap("Contact created", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		var in contactBody
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.CreateContact(h.ctx(c), in.input())
	})
}

func (h *Handler) UpdateContact() fiber.Handler {
	return h.wrap("Contact updated", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		var in contactBody
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.UpdateContact(h.ctx(c), id, in.input())
	})
}

func (h *Handler) DeleteContact() fiber.Handler {
	return h.wrap("Contact deleted", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		return nil, s.DeleteContact(h.ctx(c), id)
	})
}

func (h *Handler) ListContacts() fiber.Handler {
	return h.wrap("Contacts retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		lid, err := optUUID(c, "leadId")
		if err != nil {
			return nil, err
		}
		return s.ListContacts(h.ctx(c), c.Query("companyName"), lid)
	})
}

// ---- stages

type stageBody struct {
	Name        string `json:"name"`
	Probability int    `json:"probability"`
	Kind        string `json:"kind"`
	IsActive    *bool  `json:"isActive"`
	Position    *int   `json:"position"`
}

func (h *Handler) ListStages() fiber.Handler {
	return h.wrap("Stages retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		return s.ListStages(h.ctx(c))
	})
}

func (h *Handler) CreateStage() fiber.Handler {
	return h.wrap("Stage created", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		var in stageBody
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.CreateStage(h.ctx(c), application.StageInput{Name: in.Name, Probability: in.Probability, Kind: in.Kind})
	})
}

func (h *Handler) UpdateStage() fiber.Handler {
	return h.wrap("Stage updated", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		var in stageBody
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.UpdateStage(h.ctx(c), id, application.StageInput{Name: in.Name, Probability: in.Probability, IsActive: in.IsActive}, in.Position)
	})
}

func (h *Handler) DeleteStage() fiber.Handler {
	return h.wrap("Stage deleted", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		return nil, s.DeleteStage(h.ctx(c), id)
	})
}

// ---- deal -> project, analytics

func (h *Handler) StartProject() fiber.Handler {
	return h.wrap("Project created from deal", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		pid, code, err := s.StartProjectFromDeal(h.ctx(c), id)
		if err != nil {
			return nil, err
		}
		return fiber.Map{"projectId": pid, "projectCode": code}, nil
	})
}

func (h *Handler) Analytics() fiber.Handler {
	return h.wrap("Analytics retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		return s.GetAnalytics(h.ctx(c), c.Query("from"), c.Query("to"))
	})
}

// ---- lead forms (admin side)

func (h *Handler) ListLeadForms() fiber.Handler {
	return h.wrap("Forms retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		return s.ListLeadForms(h.ctx(c))
	})
}

type formBody struct {
	Name           string `json:"name"`
	Source         string `json:"source"`
	DefaultPIC     string `json:"defaultPic"`
	SuccessMessage string `json:"successMessage"`
	IsActive       *bool  `json:"isActive"`
}

func (b formBody) input() application.LeadFormInput {
	return application.LeadFormInput{Name: b.Name, Source: b.Source, DefaultPIC: b.DefaultPIC, SuccessMessage: b.SuccessMessage, IsActive: b.IsActive}
}

func (h *Handler) CreateLeadForm() fiber.Handler {
	return h.wrap("Form created", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		claims := middleware.CurrentUser(c)
		if claims == nil || claims.CompanyID == nil {
			return nil, apperrors.NewBadRequest("No active company")
		}
		var in formBody
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.CreateLeadForm(h.ctx(c), h.registry, *claims.CompanyID, middleware.CurrentTenantID(c), in.input())
	})
}

func (h *Handler) UpdateLeadForm() fiber.Handler {
	return h.wrap("Form updated", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		var in formBody
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.UpdateLeadForm(h.ctx(c), id, in.input())
	})
}

func (h *Handler) DeleteLeadForm() fiber.Handler {
	return h.wrap("Form deleted", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := idParam(c)
		if err != nil {
			return nil, err
		}
		return nil, s.DeleteLeadForm(h.ctx(c), h.registry, id)
	})
}
