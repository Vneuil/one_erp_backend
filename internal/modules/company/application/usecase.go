package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/company/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type CompanyUseCase interface {
	Create(ctx context.Context, dto CreateCompanyDTO) (*domain.Company, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error)
	List(ctx context.Context, query types.PaginationQuery) ([]domain.Company, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateCompanyDTO) (*domain.Company, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type companyUseCase struct {
	repo domain.CompanyRepository
}

func NewCompanyUseCase(repo domain.CompanyRepository) CompanyUseCase {
	return &companyUseCase{repo: repo}
}

func (uc *companyUseCase) Create(ctx context.Context, dto CreateCompanyDTO) (*domain.Company, error) {
	code := strings.TrimSpace(strings.ToUpper(dto.Code))
	name := strings.TrimSpace(dto.Name)

	validationErrors := make(map[string]string)
	if code == "" {
		validationErrors["code"] = "Company code is required"
	}
	if name == "" {
		validationErrors["name"] = "Company name is required"
	}
	if len(validationErrors) > 0 {
		return nil, apperrors.NewValidation(validationErrors)
	}

	// Check if company code already exists
	existing, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing company code")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("Company with this code already exists")
	}

	currency := strings.TrimSpace(strings.ToUpper(dto.Currency))
	if currency == "" {
		currency = "IDR"
	}

	company := &domain.Company{
		Code:     code,
		Name:     name,
		Email:    strings.TrimSpace(dto.Email),
		Phone:    strings.TrimSpace(dto.Phone),
		Address:  strings.TrimSpace(dto.Address),
		TaxID:    strings.TrimSpace(dto.TaxID),
		Currency: currency,
		IsActive: true,
	}

	if err := uc.repo.Create(ctx, company); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create company")
	}

	return company, nil
}

func (uc *companyUseCase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	company, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve company")
	}
	if company == nil {
		return nil, apperrors.NewNotFound("Company not found")
	}
	return company, nil
}

func (uc *companyUseCase) List(ctx context.Context, query types.PaginationQuery) ([]domain.Company, types.PaginationMeta, error) {
	query.SetDefaults()

	companies, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list companies")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return companies, meta, nil
}

func (uc *companyUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateCompanyDTO) (*domain.Company, error) {
	company, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve company")
	}
	if company == nil {
		return nil, apperrors.NewNotFound("Company not found")
	}

	if dto.Name != nil {
		name := strings.TrimSpace(*dto.Name)
		if name == "" {
			return nil, apperrors.NewBadRequest("Company name cannot be empty")
		}
		company.Name = name
	}
	if dto.Email != nil {
		company.Email = strings.TrimSpace(*dto.Email)
	}
	if dto.Phone != nil {
		company.Phone = strings.TrimSpace(*dto.Phone)
	}
	if dto.Address != nil {
		company.Address = strings.TrimSpace(*dto.Address)
	}
	if dto.TaxID != nil {
		company.TaxID = strings.TrimSpace(*dto.TaxID)
	}
	if dto.Currency != nil {
		curr := strings.TrimSpace(strings.ToUpper(*dto.Currency))
		if curr != "" {
			company.Currency = curr
		}
	}
	if dto.IsActive != nil {
		company.IsActive = *dto.IsActive
	}

	if err := uc.repo.Update(ctx, company); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update company")
	}

	return company, nil
}

func (uc *companyUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	company, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to retrieve company")
	}
	if company == nil {
		return apperrors.NewNotFound("Company not found")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete company")
	}

	return nil
}
