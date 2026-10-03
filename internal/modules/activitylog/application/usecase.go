package application

import (
	"context"

	"github.com/divinecoid/one-backend/internal/modules/activitylog/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
)

type ActivityLogUseCase interface {
	List(ctx context.Context, query types.PaginationQuery) ([]ActivityLogResponseDTO, types.PaginationMeta, error)
}

type activityLogUseCase struct {
	repo domain.ActivityLogRepository
}

func NewActivityLogUseCase(repo domain.ActivityLogRepository) ActivityLogUseCase {
	return &activityLogUseCase{repo: repo}
}

func (uc *activityLogUseCase) List(ctx context.Context, query types.PaginationQuery) ([]ActivityLogResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list activity logs")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToActivityLogResponseList(items), meta, nil
}
