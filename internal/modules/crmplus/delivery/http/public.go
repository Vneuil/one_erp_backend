package http

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/response"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/application"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

// PublicHandler serves the unauthenticated lead-capture form. It carries no
// session, so the company is found through the control-plane form index, and
// only then is that company's own database opened.
type PublicHandler struct {
	registry domain.FormRegistry
	manager  *tenantMgr.Manager
}

func NewPublicHandler(registry domain.FormRegistry, manager *tenantMgr.Manager) *PublicHandler {
	return &PublicHandler{registry: registry, manager: manager}
}

// notFound is deliberately identical for unknown, inactive and broken forms, so
// probing keys reveals nothing.
func notFound() error { return apperrors.NewNotFound("Form not found") }

func (p *PublicHandler) load(c *fiber.Ctx) (*application.Service, *domain.LeadForm, error) {
	key := c.Params("key")
	idx, err := p.registry.Lookup(c.UserContext(), key)
	if err != nil {
		slog.Error("lead form lookup failed", "error", err)
		return nil, nil, notFound()
	}
	if idx == nil {
		return nil, nil, notFound()
	}
	db, err := p.manager.GetDB(c.UserContext(), idx.CompanyID)
	if err != nil {
		slog.Error("lead form: tenant database unavailable", "companyId", idx.CompanyID, "error", err)
		return nil, nil, notFound()
	}
	form, err := infrastructure.NewRepository(db).GetLeadFormByKey(c.UserContext(), key)
	if err != nil || form == nil || !form.IsActive {
		return nil, nil, notFound()
	}
	return serviceFor(db), form, nil
}

// Info returns what an embedding page needs to render the form.
func (p *PublicHandler) Info(c *fiber.Ctx) error {
	_, form, err := p.load(c)
	if err != nil {
		return err
	}
	return response.OK(c, "Form retrieved", fiber.Map{"name": form.Name, "successMessage": form.SuccessMessage})
}

// Submit accepts one submission.
func (p *PublicHandler) Submit(c *fiber.Ctx) error {
	svc, form, err := p.load(c)
	if err != nil {
		return err
	}
	var in struct {
		Name    string `json:"name"`
		Company string `json:"company"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
		Message string `json:"message"`
		Website string `json:"website"` // honeypot: real visitors never see or fill it
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if err := svc.AcceptSubmission(c.UserContext(), form, application.Submission{
		Name: in.Name, Company: in.Company, Email: in.Email, Phone: in.Phone, Message: in.Message, Honeypot: in.Website,
	}); err != nil {
		return err
	}
	return response.OK(c, "Submitted", fiber.Map{"message": form.SuccessMessage})
}
