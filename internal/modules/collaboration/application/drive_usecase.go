package application

import (
	"context"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

type DriveUseCase interface {
	CreateFolder(ctx context.Context, dto CreateFolderDTO) (*FolderResponseDTO, error)
	ListFolders(ctx context.Context) ([]FolderResponseDTO, error)
	UploadFile(ctx context.Context, dto UploadFileDTO) (*FileResponseDTO, error)
	ListFiles(ctx context.Context, folderID *uuid.UUID) ([]FileResponseDTO, error)
	DownloadFile(ctx context.Context, id uuid.UUID) (*FileDownloadDTO, error)
	DeleteFile(ctx context.Context, id uuid.UUID) error
	DeleteFolder(ctx context.Context, id uuid.UUID) error

	SeedInitialData(ctx context.Context) error
}

type driveUseCase struct {
	repo    domain.DriveRepository
	storage storage.Storage
}

func NewDriveUseCase(repo domain.DriveRepository, store storage.Storage) DriveUseCase {
	return &driveUseCase{repo: repo, storage: store}
}

func (uc *driveUseCase) CreateFolder(ctx context.Context, dto CreateFolderDTO) (*FolderResponseDTO, error) {
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("Folder name is required")
	}
	if dto.ParentFolderID != nil {
		parent, err := uc.repo.GetFolderByID(ctx, *dto.ParentFolderID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get parent folder")
		}
		if parent == nil {
			return nil, apperrors.NewNotFound("Parent folder not found")
		}
	}

	f := &domain.Folder{
		Name:           dto.Name,
		ParentFolderID: dto.ParentFolderID,
	}
	if err := uc.repo.CreateFolder(ctx, f); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create folder")
	}
	return ToFolderResponse(f), nil
}

func (uc *driveUseCase) ListFolders(ctx context.Context) ([]FolderResponseDTO, error) {
	items, err := uc.repo.ListFolders(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list folders")
	}
	return ToFolderResponseList(items), nil
}

func (uc *driveUseCase) UploadFile(ctx context.Context, dto UploadFileDTO) (*FileResponseDTO, error) {
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("File name is required")
	}
	if dto.UploadedBy == "" {
		return nil, apperrors.NewBadRequest("Uploader is required")
	}
	if dto.FolderID != nil {
		folder, err := uc.repo.GetFolderByID(ctx, *dto.FolderID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get folder")
		}
		if folder == nil {
			return nil, apperrors.NewNotFound("Folder not found")
		}
	}

	if len(dto.Content) > 0 && !uc.storage.Enabled() {
		return nil, apperrors.NewServiceUnavailable("File storage is not configured. Please set R2 credentials in the backend .env file.")
	}

	f := &domain.File{
		FolderID:   dto.FolderID,
		Name:       dto.Name,
		SizeBytes:  dto.SizeBytes,
		MimeType:   dto.MimeType,
		UploadedBy: dto.UploadedBy,
	}

	if len(dto.Content) > 0 {
		key := fmt.Sprintf("drive/%s/%s", uuid.New().String(), dto.Name)
		if _, err := uc.storage.Upload(ctx, key, dto.Content, dto.MimeType); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to upload file to storage")
		}
		f.StorageKey = key
		if f.SizeBytes == 0 {
			f.SizeBytes = int64(len(dto.Content))
		}
	}

	if err := uc.repo.CreateFile(ctx, f); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to upload file")
	}
	return ToFileResponse(f), nil
}

func (uc *driveUseCase) ListFiles(ctx context.Context, folderID *uuid.UUID) ([]FileResponseDTO, error) {
	items, err := uc.repo.ListFiles(ctx, folderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list files")
	}
	return ToFileResponseList(items), nil
}

func (uc *driveUseCase) DownloadFile(ctx context.Context, id uuid.UUID) (*FileDownloadDTO, error) {
	f, err := uc.repo.GetFileByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get file")
	}
	if f == nil {
		return nil, apperrors.NewNotFound("File not found")
	}

	if f.StorageKey != "" {
		if !uc.storage.Enabled() {
			return nil, apperrors.NewServiceUnavailable("File storage is not configured. Please set R2 credentials in the backend .env file.")
		}
		data, err := uc.storage.Download(ctx, f.StorageKey)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to download file from storage")
		}
		return &FileDownloadDTO{
			FileResponseDTO: *ToFileResponse(f),
			ContentType:     f.MimeType,
			Binary:          data,
		}, nil
	}

	content := fmt.Sprintf(
		"File: %s\nSize: %d bytes\nType: %s\nUploaded By: %s\nUploaded At: %s\n\n[Placeholder content - no binary storage configured for this module]",
		f.Name, f.SizeBytes, f.MimeType, f.UploadedBy, f.CreatedAt.Format("2006-01-02 15:04:05"),
	)
	return &FileDownloadDTO{
		FileResponseDTO: *ToFileResponse(f),
		Content:         content,
	}, nil
}

func (uc *driveUseCase) DeleteFile(ctx context.Context, id uuid.UUID) error {
	f, err := uc.repo.GetFileByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get file")
	}
	if f == nil {
		return apperrors.NewNotFound("File not found")
	}
	if f.StorageKey != "" && uc.storage.Enabled() {
		if err := uc.storage.Delete(ctx, f.StorageKey); err != nil {
			return apperrors.NewInternal(err, "Failed to delete file from storage")
		}
	}
	if err := uc.repo.DeleteFile(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete file")
	}
	return nil
}

// DeleteFolder removes an empty folder. A folder that still holds files or
// subfolders is refused so nothing is orphaned silently.
func (uc *driveUseCase) DeleteFolder(ctx context.Context, id uuid.UUID) error {
	f, err := uc.repo.GetFolderByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get folder")
	}
	if f == nil {
		return apperrors.NewNotFound("Folder not found")
	}
	folders, err := uc.repo.ListFolders(ctx)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check subfolders")
	}
	for _, c := range folders {
		if c.ParentFolderID != nil && *c.ParentFolderID == id {
			return apperrors.NewConflict("Folder still contains subfolders; delete or move them first")
		}
	}
	files, err := uc.repo.ListFiles(ctx, &id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check files")
	}
	if len(files) > 0 {
		return apperrors.NewConflict("Folder still contains files; delete them first")
	}
	if err := uc.repo.DeleteFolder(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete folder")
	}
	return nil
}

// SeedInitialData populates a couple of sample folders with files on first boot
func (uc *driveUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountFolders(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	folderSeeds := []struct {
		name  string
		files []UploadFileDTO
	}{
		{
			name: "Legal & Kontrak",
			files: []UploadFileDTO{
				{Name: "Akta-Pendirian-PT-Sentosa-Mandiri.pdf", SizeBytes: 4404019, MimeType: "application/pdf", UploadedBy: "Nicholas Tantra"},
			},
		},
		{
			name: "SOP & Regulasi",
			files: []UploadFileDTO{
				{Name: "SOP-Gudang-WMS-Inbound-Outbound-2026.docx", SizeBytes: 1887436, MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", UploadedBy: "Budi Santoso"},
			},
		},
		{
			name: "Laporan Keuangan",
			files: []UploadFileDTO{
				{Name: "Rekap-Laporan-Keuangan-Q2-Audited.xlsx", SizeBytes: 9017753, MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", UploadedBy: "Dewi Lestari"},
			},
		},
	}

	for _, seed := range folderSeeds {
		folder, err := uc.CreateFolder(ctx, CreateFolderDTO{Name: seed.name})
		if err != nil {
			continue
		}
		for _, fl := range seed.files {
			fl.FolderID = &folder.ID
			if _, err := uc.UploadFile(ctx, fl); err != nil {
				continue
			}
		}
	}
	return nil
}
