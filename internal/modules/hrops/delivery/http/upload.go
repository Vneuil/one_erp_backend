package http

import (
	"io"
	"mime/multipart"

	"github.com/divinecoid/one-backend/internal/shared/attachment"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

// readUpload reads a multipart file, refusing anything over the size limit
// before it is held in memory.
func readUpload(fh *multipart.FileHeader) ([]byte, error) {
	if fh.Size > attachment.MaxBytes {
		return nil, apperrors.NewBadRequest("The file is larger than 10 MB")
	}
	f, err := fh.Open()
	if err != nil {
		return nil, apperrors.NewBadRequest("Failed to read the uploaded file")
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, attachment.MaxBytes+1))
}

// DownloadDocument streams an employee document's uploaded file.
func (h *Handler) DownloadDocument(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	did, err := id(c)
	if err != nil {
		return err
	}
	doc, data, err := svc.DocumentFile(h.ctx(c), caller, did)
	if err != nil {
		return err
	}
	f, _ := attachment.Check(doc.FileName, []byte{0})
	c.Set("Content-Type", f.ContentType)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Disposition", "attachment; filename=\""+f.Name+"\"")
	return c.Send(data)
}
