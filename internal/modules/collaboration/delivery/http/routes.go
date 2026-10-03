package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string) {
	protected := middleware.Protected(jwtSecret)
	collaboration := router.Group("/collaboration", protected)

	conversations := collaboration.Group("/conversations")
	conversations.Post("/", h.CreateConversation)
	conversations.Get("/", h.ListConversations)
	conversations.Post("/:id/messages", h.SendMessage)
	conversations.Get("/:id/messages", h.ListMessages)

	folders := collaboration.Group("/folders")
	folders.Post("/", h.CreateFolder)
	folders.Get("/", h.ListFolders)
	folders.Delete("/:id", h.DeleteFolder)

	files := collaboration.Group("/files")
	files.Post("/", h.UploadFile)
	files.Get("/", h.ListFiles)
	files.Get("/:id/download", h.DownloadFile)
	files.Delete("/:id", h.DeleteFile)

	exportJobs := collaboration.Group("/export-jobs")
	exportJobs.Post("/", h.CreateExportJob)
	exportJobs.Get("/", h.ListExportJobs)
	exportJobs.Post("/:id/run", h.RunExportJob)
	exportJobs.Get("/:id/download", h.DownloadExportJob)
}
