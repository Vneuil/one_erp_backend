package http

import (
	"context"
	"io"

	"github.com/divinecoid/one-backend/internal/foundation/companyctx"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/docflow/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	forms      application.FormsUseCase
	signatures application.SignaturesUseCase
	meetings   application.MeetingsUseCase
}

func NewHandler(forms application.FormsUseCase, signatures application.SignaturesUseCase, meetings application.MeetingsUseCase) *Handler {
	return &Handler{forms: forms, signatures: signatures, meetings: meetings}
}

func parseID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid " + param + " format")
	}
	return id, nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	ctx := tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
	claims := middleware.CurrentUser(c)
	var companyID *uuid.UUID
	if claims != nil {
		companyID = claims.CompanyID
	}
	return companyctx.WithCompanyID(ctx, companyID)
}

// Forms

func (h *Handler) CreateForm(c *fiber.Ctx) error {
	var dto application.CreateFormDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	f, err := h.forms.CreateForm(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Form created successfully", f)
}

func (h *Handler) ListForms(c *fiber.Ctx) error {
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := h.forms.ListForms(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Forms retrieved successfully", items, meta)
}

func (h *Handler) GetForm(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	f, err := h.forms.GetFormByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Form retrieved successfully", f)
}

func (h *Handler) PublishForm(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	f, err := h.forms.PublishForm(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Form published successfully", f)
}

func (h *Handler) SubmitResponse(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.SubmitResponseDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := h.forms.SubmitResponse(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Response submitted successfully", res)
}

func (h *Handler) ListResponses(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	items, err := h.forms.ListResponses(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Responses retrieved successfully", items)
}

// Signatures

func (h *Handler) UploadDocument(c *fiber.Ctx) error {
	var dto application.UploadDocumentDTO

	if fh, err := c.FormFile("file"); err == nil && fh != nil {
		f, err := fh.Open()
		if err != nil {
			return apperrors.NewBadRequest("Failed to read uploaded file")
		}
		defer f.Close()
		content, err := io.ReadAll(f)
		if err != nil {
			return apperrors.NewBadRequest("Failed to read uploaded file")
		}

		dto.Content = content
		dto.MimeType = fh.Header.Get("Content-Type")
		dto.FileName = fh.Filename
		dto.Title = c.FormValue("title")
		if companyID := c.FormValue("companyId"); companyID != "" {
			id, err := uuid.Parse(companyID)
			if err != nil {
				return apperrors.NewBadRequest("Invalid companyId format")
			}
			dto.CompanyID = &id
		}
	} else if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	// UploadedBy is never trusted from the client - it's always the
	// authenticated caller.
	if claims := middleware.CurrentUser(c); claims != nil {
		dto.UploadedBy = claims.Email
	}

	d, err := h.signatures.UploadDocument(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Document uploaded successfully", d)
}

func (h *Handler) DownloadDocument(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	res, err := h.signatures.DownloadDocument(h.ctx(c), id)
	if err != nil {
		return err
	}
	if len(res.Binary) > 0 {
		if res.ContentType != "" {
			c.Set("Content-Type", res.ContentType)
		}
		c.Set("Content-Disposition", "attachment; filename=\""+res.FileName+"\"")
		return c.Send(res.Binary)
	}
	return response.OK(c, "Document download prepared successfully", res)
}

func (h *Handler) ListDocuments(c *fiber.Ctx) error {
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := h.signatures.ListDocuments(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Documents retrieved successfully", items, meta)
}

func (h *Handler) GetDocument(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	d, err := h.signatures.GetDocumentByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Document retrieved successfully", d)
}

func (h *Handler) AddSigner(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.AddSignerDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	r, err := h.signatures.AddSigner(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Signer added successfully", r)
}

func (h *Handler) SignRequest(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var callerEmail string
	if claims := middleware.CurrentUser(c); claims != nil {
		callerEmail = claims.Email
	}
	r, err := h.signatures.SignRequest(h.ctx(c), id, callerEmail)
	if err != nil {
		return err
	}
	return response.OK(c, "Document signed successfully", r)
}

func (h *Handler) DeclineRequest(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	r, err := h.signatures.DeclineRequest(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Signature request declined", r)
}

// Meetings

func (h *Handler) CreateMeeting(c *fiber.Ctx) error {
	var dto application.CreateMeetingDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	m, err := h.meetings.CreateMeeting(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Meeting created successfully", m)
}

func (h *Handler) ListMeetings(c *fiber.Ctx) error {
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := h.meetings.ListMeetings(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Meetings retrieved successfully", items, meta)
}

func (h *Handler) GetMeeting(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	m, err := h.meetings.GetMeetingByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Meeting retrieved successfully", m)
}

func (h *Handler) UpdateMeetingStatus(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateMeetingStatusDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	m, err := h.meetings.UpdateStatus(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Meeting status updated successfully", m)
}

func (h *Handler) AddNote(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.AddNoteDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	n, err := h.meetings.AddNote(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Note added successfully", n)
}

func (h *Handler) GenerateAINote(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}

	var audioBytes []byte
	var audioFilename string
	if fileHeader, err := c.FormFile("audio"); err == nil && fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			return apperrors.NewBadRequest("Failed to read uploaded audio file")
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return apperrors.NewBadRequest("Failed to read uploaded audio file")
		}
		audioBytes = data
		audioFilename = fileHeader.Filename
	}

	n, err := h.meetings.GenerateAINote(h.ctx(c), id, audioBytes, audioFilename)
	if err != nil {
		return err
	}
	return response.Created(c, "AI note generated successfully", n)
}

func (h *Handler) ListNotes(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	items, err := h.meetings.ListNotes(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Notes retrieved successfully", items)
}
