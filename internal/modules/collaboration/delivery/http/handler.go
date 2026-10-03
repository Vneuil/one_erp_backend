package http

import (
	"context"
	"io"

	"github.com/divinecoid/one-backend/internal/foundation/companyctx"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	chat    application.ChatUseCase
	drive   application.DriveUseCase
	exports application.ExportsUseCase
}

func NewHandler(chat application.ChatUseCase, drive application.DriveUseCase, exports application.ExportsUseCase) *Handler {
	return &Handler{chat: chat, drive: drive, exports: exports}
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	ctx := tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
	claims := middleware.CurrentUser(c)
	var companyID *uuid.UUID
	if claims != nil {
		companyID = claims.CompanyID
	}
	return companyctx.WithCompanyID(ctx, companyID)
}

func parseID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid " + param + " format")
	}
	return id, nil
}

// Chat

func (h *Handler) CreateConversation(c *fiber.Ctx) error {
	var dto application.CreateConversationDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := h.chat.CreateConversation(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Conversation created successfully", res)
}

func (h *Handler) ListConversations(c *fiber.Ctx) error {
	res, err := h.chat.ListConversations(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Conversations retrieved successfully", res)
}

func (h *Handler) SendMessage(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.SendMessageDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := h.chat.SendMessage(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Message sent successfully", res)
}

func (h *Handler) ListMessages(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	res, err := h.chat.ListMessages(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Messages retrieved successfully", res)
}

// Drive

func (h *Handler) CreateFolder(c *fiber.Ctx) error {
	var dto application.CreateFolderDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := h.drive.CreateFolder(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Folder created successfully", res)
}

func (h *Handler) ListFolders(c *fiber.Ctx) error {
	res, err := h.drive.ListFolders(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Folders retrieved successfully", res)
}

func (h *Handler) UploadFile(c *fiber.Ctx) error {
	var dto application.UploadFileDTO

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
		dto.Name = fh.Filename
		dto.SizeBytes = fh.Size
		dto.MimeType = fh.Header.Get("Content-Type")
		dto.UploadedBy = c.FormValue("uploadedBy")
		if folderID := c.FormValue("folderId"); folderID != "" {
			id, err := uuid.Parse(folderID)
			if err != nil {
				return apperrors.NewBadRequest("Invalid folderId format")
			}
			dto.FolderID = &id
		}
	} else if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	res, err := h.drive.UploadFile(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "File uploaded successfully", res)
}

func (h *Handler) ListFiles(c *fiber.Ctx) error {
	var folderID *uuid.UUID
	if q := c.Query("folderId"); q != "" {
		id, err := uuid.Parse(q)
		if err != nil {
			return apperrors.NewBadRequest("Invalid folderId format")
		}
		folderID = &id
	}
	res, err := h.drive.ListFiles(h.ctx(c), folderID)
	if err != nil {
		return err
	}
	return response.OK(c, "Files retrieved successfully", res)
}

func (h *Handler) DownloadFile(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	res, err := h.drive.DownloadFile(h.ctx(c), id)
	if err != nil {
		return err
	}
	if len(res.Binary) > 0 {
		if res.ContentType != "" {
			c.Set("Content-Type", res.ContentType)
		}
		c.Set("Content-Disposition", "attachment; filename=\""+res.Name+"\"")
		return c.Send(res.Binary)
	}
	return response.OK(c, "File download prepared successfully", res)
}

func (h *Handler) DeleteFile(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.drive.DeleteFile(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "File deleted successfully", nil)
}

func (h *Handler) DeleteFolder(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.drive.DeleteFolder(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Folder deleted successfully", nil)
}

// Exports

func (h *Handler) CreateExportJob(c *fiber.Ctx) error {
	var dto application.CreateExportJobDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := h.exports.CreateJob(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Export job created successfully", res)
}

func (h *Handler) ListExportJobs(c *fiber.Ctx) error {
	res, err := h.exports.ListJobs(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Export jobs retrieved successfully", res)
}

func (h *Handler) RunExportJob(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	res, err := h.exports.RunJob(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Export job run successfully", res)
}

func (h *Handler) DownloadExportJob(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	res, err := h.exports.DownloadJob(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Export job download prepared successfully", res)
}
