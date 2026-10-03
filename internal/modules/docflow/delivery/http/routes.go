package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string) {
	protected := middleware.Protected(jwtSecret)
	docflow := router.Group("/docflow", protected)

	forms := docflow.Group("/forms")
	forms.Post("/", h.CreateForm)
	forms.Get("/", h.ListForms)
	forms.Get("/:id", h.GetForm)
	forms.Post("/:id/publish", h.PublishForm)
	forms.Post("/:id/responses", h.SubmitResponse)
	forms.Get("/:id/responses", h.ListResponses)

	documents := docflow.Group("/documents")
	documents.Post("/", h.UploadDocument)
	documents.Get("/", h.ListDocuments)
	documents.Get("/:id", h.GetDocument)
	documents.Get("/:id/download", h.DownloadDocument)
	documents.Post("/:id/signers", h.AddSigner)

	signatureRequests := docflow.Group("/signature-requests")
	signatureRequests.Post("/:id/sign", h.SignRequest)
	signatureRequests.Post("/:id/decline", h.DeclineRequest)

	meetings := docflow.Group("/meetings")
	meetings.Post("/", h.CreateMeeting)
	meetings.Get("/", h.ListMeetings)
	meetings.Get("/:id", h.GetMeeting)
	meetings.Put("/:id/status", h.UpdateMeetingStatus)
	meetings.Post("/:id/notes", h.AddNote)
	meetings.Get("/:id/notes", h.ListNotes)
	meetings.Post("/:id/notes/generate", h.GenerateAINote)
}
