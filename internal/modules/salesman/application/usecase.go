package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/salesman/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type SalesmanUseCase interface {
	Create(ctx context.Context, dto CreateSalesmanDTO) (*SalesmanResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*SalesmanResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]SalesmanResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateSalesmanDTO) (*SalesmanResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SeedInitialData(ctx context.Context) error
}

type salesmanUseCase struct {
	repo domain.SalesmanRepository
}

func NewSalesmanUseCase(repo domain.SalesmanRepository) SalesmanUseCase {
	return &salesmanUseCase{repo: repo}
}

func (uc *salesmanUseCase) Create(ctx context.Context, dto CreateSalesmanDTO) (*SalesmanResponseDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(dto.Code))
	name := strings.TrimSpace(dto.Name)

	if code == "" || name == "" {
		return nil, apperrors.NewBadRequest("Salesman Code and Name are required")
	}

	existing, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing code")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("Salesman with this code already exists")
	}

	status := dto.Status
	if status == "" {
		status = "Active"
	}

	salesman := &domain.Salesman{
		Code:           code,
		Name:           name,
		Email:          dto.Email,
		Phone:          dto.Phone,
		Territory:      dto.Territory,
		CommissionRate: dto.CommissionRate,
		Status:         status,
	}

	if err := uc.repo.Create(ctx, salesman); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create salesman")
	}

	return ToSalesmanResponse(salesman), nil
}

func (uc *salesmanUseCase) GetByID(ctx context.Context, id uuid.UUID) (*SalesmanResponseDTO, error) {
	salesman, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve salesman")
	}
	if salesman == nil {
		return nil, apperrors.NewNotFound("Salesman not found")
	}
	return ToSalesmanResponse(salesman), nil
}

func (uc *salesmanUseCase) List(ctx context.Context, query types.PaginationQuery) ([]SalesmanResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	salesmen, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list salesmen")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToSalesmanResponseList(salesmen), meta, nil
}

func (uc *salesmanUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateSalesmanDTO) (*SalesmanResponseDTO, error) {
	salesman, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get salesman")
	}
	if salesman == nil {
		return nil, apperrors.NewNotFound("Salesman not found")
	}

	if dto.Name != nil {
		salesman.Name = *dto.Name
	}
	if dto.Email != nil {
		salesman.Email = *dto.Email
	}
	if dto.Phone != nil {
		salesman.Phone = *dto.Phone
	}
	if dto.Territory != nil {
		salesman.Territory = *dto.Territory
	}
	if dto.CommissionRate != nil {
		salesman.CommissionRate = *dto.CommissionRate
	}
	if dto.Status != nil {
		salesman.Status = *dto.Status
	}

	if err := uc.repo.Update(ctx, salesman); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update salesman")
	}

	return ToSalesmanResponse(salesman), nil
}

func (uc *salesmanUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *salesmanUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	initial := []CreateSalesmanDTO{
		{
			Code:           "SLM-001",
			Name:           "Budi Hartono",
			Email:          "budi.hartono@one-erp.com",
			Phone:          "+62 812 3456 7890",
			Territory:      "Jabodetabek",
			CommissionRate: 2.5,
			Status:         "Active",
		},
		{
			Code:           "SLM-002",
			Name:           "Siti Nurhaliza",
			Email:          "siti.nurhaliza@one-erp.com",
			Phone:          "+62 813 9876 5432",
			Territory:      "Jawa Timur",
			CommissionRate: 3.0,
			Status:         "Active",
		},
	}

	for _, item := range initial {
		_, _ = uc.Create(ctx, item)
	}
	return nil
}
