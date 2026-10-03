package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/shipping/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type ShippingMethodUseCase interface {
	Create(ctx context.Context, dto CreateShippingMethodDTO) (*ShippingMethodResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ShippingMethodResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]ShippingMethodResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateShippingMethodDTO) (*ShippingMethodResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SeedInitialData(ctx context.Context) error
}

type shippingMethodUseCase struct {
	repo domain.ShippingMethodRepository
}

func NewShippingMethodUseCase(repo domain.ShippingMethodRepository) ShippingMethodUseCase {
	return &shippingMethodUseCase{repo: repo}
}

func (uc *shippingMethodUseCase) Create(ctx context.Context, dto CreateShippingMethodDTO) (*ShippingMethodResponseDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(dto.Code))
	name := strings.TrimSpace(dto.Name)

	if code == "" || name == "" {
		return nil, apperrors.NewBadRequest("Shipping method Code and Name are required")
	}

	existing, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing code")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("Shipping method with this code already exists")
	}

	status := dto.Status
	if status == "" {
		status = "Active"
	}

	method := &domain.ShippingMethod{
		Code:          code,
		Name:          name,
		Carrier:       dto.Carrier,
		EstimatedDays: dto.EstimatedDays,
		Cost:          dto.Cost,
		Status:        status,
	}

	if err := uc.repo.Create(ctx, method); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create shipping method")
	}

	return ToShippingMethodResponse(method), nil
}

func (uc *shippingMethodUseCase) GetByID(ctx context.Context, id uuid.UUID) (*ShippingMethodResponseDTO, error) {
	method, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve shipping method")
	}
	if method == nil {
		return nil, apperrors.NewNotFound("Shipping method not found")
	}
	return ToShippingMethodResponse(method), nil
}

func (uc *shippingMethodUseCase) List(ctx context.Context, query types.PaginationQuery) ([]ShippingMethodResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list shipping methods")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToShippingMethodResponseList(items), meta, nil
}

func (uc *shippingMethodUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateShippingMethodDTO) (*ShippingMethodResponseDTO, error) {
	method, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get shipping method")
	}
	if method == nil {
		return nil, apperrors.NewNotFound("Shipping method not found")
	}

	if dto.Name != nil {
		method.Name = *dto.Name
	}
	if dto.Carrier != nil {
		method.Carrier = *dto.Carrier
	}
	if dto.EstimatedDays != nil {
		method.EstimatedDays = *dto.EstimatedDays
	}
	if dto.Cost != nil {
		method.Cost = *dto.Cost
	}
	if dto.Status != nil {
		method.Status = *dto.Status
	}

	if err := uc.repo.Update(ctx, method); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update shipping method")
	}

	return ToShippingMethodResponse(method), nil
}

func (uc *shippingMethodUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *shippingMethodUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	initial := []CreateShippingMethodDTO{
		{Code: "REG", Name: "Reguler", Carrier: "JNE", EstimatedDays: 3, Cost: 15000, Status: "Active"},
		{Code: "EXP", Name: "Express", Carrier: "JNE", EstimatedDays: 1, Cost: 35000, Status: "Active"},
		{Code: "OWN", Name: "Armada Sendiri", Carrier: "Internal Fleet", EstimatedDays: 2, Cost: 0, Status: "Active"},
	}

	for _, item := range initial {
		_, _ = uc.Create(ctx, item)
	}
	return nil
}
