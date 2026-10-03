package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/companyctx"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type driveRepository struct {
	db *gorm.DB
}

func NewDriveRepository(db *gorm.DB) domain.DriveRepository {
	return &driveRepository{db: db}
}

func (r *driveRepository) CreateFolder(ctx context.Context, f *domain.Folder) error {
	companyctx.SetCompanyID(ctx, &f.CompanyID)
	tenantctx.SetTenantID(ctx, &f.TenantID)
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *driveRepository) GetFolderByID(ctx context.Context, id uuid.UUID) (*domain.Folder, error) {
	var f domain.Folder
	err := companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *driveRepository) ListFolders(ctx context.Context) ([]domain.Folder, error) {
	var items []domain.Folder
	db := companyctx.Scope(ctx, r.db.WithContext(ctx))
	err := tenantctx.Scope(ctx, db).Order("created_at asc").Find(&items).Error
	return items, err
}

func (r *driveRepository) DeleteFolder(ctx context.Context, id uuid.UUID) error {
	return companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).Delete(&domain.Folder{}).Error
}

func (r *driveRepository) CountFolders(ctx context.Context) (int64, error) {
	var total int64
	err := companyctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Folder{})).Count(&total).Error
	return total, err
}

func (r *driveRepository) CreateFile(ctx context.Context, f *domain.File) error {
	companyctx.SetCompanyID(ctx, &f.CompanyID)
	tenantctx.SetTenantID(ctx, &f.TenantID)
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *driveRepository) GetFileByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	var f domain.File
	err := companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *driveRepository) ListFiles(ctx context.Context, folderID *uuid.UUID) ([]domain.File, error) {
	var items []domain.File
	db := companyctx.Scope(ctx, tenantctx.Scope(ctx, r.db.WithContext(ctx)))
	if folderID != nil {
		db = db.Where("folder_id = ?", *folderID)
	}
	err := db.Order("created_at desc").Find(&items).Error
	return items, err
}

func (r *driveRepository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	return companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).Delete(&domain.File{}).Error
}

func (r *driveRepository) CountFiles(ctx context.Context) (int64, error) {
	var total int64
	err := companyctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.File{})).Count(&total).Error
	return total, err
}
