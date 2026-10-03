package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	"github.com/google/uuid"
)

type CreateFolderDTO struct {
	Name           string     `json:"name"`
	ParentFolderID *uuid.UUID `json:"parentFolderId,omitempty"`
}

type UploadFileDTO struct {
	FolderID   *uuid.UUID `json:"folderId,omitempty"`
	Name       string     `json:"name"`
	SizeBytes  int64      `json:"sizeBytes"`
	MimeType   string     `json:"mimeType"`
	UploadedBy string     `json:"uploadedBy"`
	Content    []byte     `json:"-"`
}

type FolderResponseDTO struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	ParentFolderID *uuid.UUID `json:"parentFolderId,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type FileResponseDTO struct {
	ID         uuid.UUID  `json:"id"`
	FolderID   *uuid.UUID `json:"folderId,omitempty"`
	Name       string     `json:"name"`
	SizeBytes  int64      `json:"sizeBytes"`
	MimeType   string     `json:"mimeType"`
	UploadedBy string     `json:"uploadedBy"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type FileDownloadDTO struct {
	FileResponseDTO
	Content     string `json:"content"`
	ContentType string `json:"-"`
	Binary      []byte `json:"-"`
}

func ToFolderResponse(f *domain.Folder) *FolderResponseDTO {
	if f == nil {
		return nil
	}
	return &FolderResponseDTO{
		ID:             f.ID,
		Name:           f.Name,
		ParentFolderID: f.ParentFolderID,
		CreatedAt:      f.CreatedAt,
	}
}

func ToFolderResponseList(items []domain.Folder) []FolderResponseDTO {
	result := make([]FolderResponseDTO, len(items))
	for i, f := range items {
		result[i] = *ToFolderResponse(&f)
	}
	return result
}

func ToFileResponse(f *domain.File) *FileResponseDTO {
	if f == nil {
		return nil
	}
	return &FileResponseDTO{
		ID:         f.ID,
		FolderID:   f.FolderID,
		Name:       f.Name,
		SizeBytes:  f.SizeBytes,
		MimeType:   f.MimeType,
		UploadedBy: f.UploadedBy,
		CreatedAt:  f.CreatedAt,
	}
}

func ToFileResponseList(items []domain.File) []FileResponseDTO {
	result := make([]FileResponseDTO, len(items))
	for i, f := range items {
		result[i] = *ToFileResponse(&f)
	}
	return result
}
