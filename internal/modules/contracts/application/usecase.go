package application

import (
	"context"
	"fmt"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/contracts/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var validPartyTypes = map[string]bool{
	"customer": true,
	"vendor":   true,
}

var validStatuses = map[string]bool{
	"draft":      true,
	"active":     true,
	"expired":    true,
	"terminated": true,
	"renewed":    true,
}

type ContractsUseCase interface {
	CreateContract(ctx context.Context, dto CreateContractDTO) (*ContractResponseDTO, error)
	GetContractByID(ctx context.Context, id uuid.UUID) (*ContractResponseDTO, error)
	ListContracts(ctx context.Context, query types.PaginationQuery) ([]ContractResponseDTO, types.PaginationMeta, error)
	UpdateContract(ctx context.Context, id uuid.UUID, dto UpdateContractDTO) (*ContractResponseDTO, error)
	UpdateContractStatus(ctx context.Context, id uuid.UUID, dto UpdateContractStatusDTO) (*ContractResponseDTO, error)
	RenewContract(ctx context.Context, id uuid.UUID) (*ContractResponseDTO, error)
	ListExpiringSoon(ctx context.Context) ([]ContractResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type contractsUseCase struct {
	repo domain.ContractsRepository
}

func NewContractsUseCase(repo domain.ContractsRepository) ContractsUseCase {
	return &contractsUseCase{repo: repo}
}

func (uc *contractsUseCase) validateDates(startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return apperrors.NewBadRequest("Start date must be in YYYY-MM-DD format")
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return apperrors.NewBadRequest("End date must be in YYYY-MM-DD format")
	}
	if !end.After(start) {
		return apperrors.NewBadRequest("End date must be after start date")
	}
	return nil
}

func (uc *contractsUseCase) CreateContract(ctx context.Context, dto CreateContractDTO) (*ContractResponseDTO, error) {
	if dto.Title == "" {
		return nil, apperrors.NewBadRequest("Contract title is required")
	}
	if !validPartyTypes[dto.PartyType] {
		return nil, apperrors.NewBadRequest("Party type must be one of customer, vendor")
	}
	if dto.PartyName == "" {
		return nil, apperrors.NewBadRequest("Party name is required")
	}
	if dto.ContractValue < 0 {
		return nil, apperrors.NewBadRequest("Contract value cannot be negative")
	}
	if err := uc.validateDates(dto.StartDate, dto.EndDate); err != nil {
		return nil, err
	}

	contractNumber := dto.ContractNumber
	if contractNumber == "" {
		count, err := uc.repo.CountContracts(ctx)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to generate contract number")
		}
		contractNumber = generateContractNumber(count + 1)
	}

	c := &domain.Contract{
		CompanyID:      dto.CompanyID,
		ContractNumber: contractNumber,
		PartyType:      dto.PartyType,
		PartyID:        dto.PartyID,
		PartyName:      dto.PartyName,
		Title:          dto.Title,
		Description:    dto.Description,
		ContractValue:  dto.ContractValue,
		StartDate:      dto.StartDate,
		EndDate:        dto.EndDate,
		Status:         "draft",
		PaymentTerms:   dto.PaymentTerms,
		Attachments:    dto.Attachments,
	}

	if err := uc.repo.CreateContract(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create contract")
	}
	return ToContractResponse(c), nil
}

func generateContractNumber(seq int64) string {
	return fmt.Sprintf("CTR-%s-%04d", time.Now().Format("2006"), seq)
}

func (uc *contractsUseCase) GetContractByID(ctx context.Context, id uuid.UUID) (*ContractResponseDTO, error) {
	c, err := uc.repo.GetContractByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get contract")
	}
	if c == nil {
		return nil, apperrors.NewNotFound("Contract not found")
	}
	return ToContractResponse(c), nil
}

func (uc *contractsUseCase) ListContracts(ctx context.Context, query types.PaginationQuery) ([]ContractResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	contracts, total, err := uc.repo.ListContracts(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list contracts")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToContractResponseList(contracts), meta, nil
}

func (uc *contractsUseCase) UpdateContract(ctx context.Context, id uuid.UUID, dto UpdateContractDTO) (*ContractResponseDTO, error) {
	c, err := uc.repo.GetContractByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get contract")
	}
	if c == nil {
		return nil, apperrors.NewNotFound("Contract not found")
	}
	if dto.PartyType != "" {
		if !validPartyTypes[dto.PartyType] {
			return nil, apperrors.NewBadRequest("Party type must be one of customer, vendor")
		}
		c.PartyType = dto.PartyType
	}
	if dto.PartyID != nil {
		c.PartyID = dto.PartyID
	}
	if dto.PartyName != "" {
		c.PartyName = dto.PartyName
	}
	if dto.Title != "" {
		c.Title = dto.Title
	}
	if dto.Description != "" {
		c.Description = dto.Description
	}
	if dto.ContractValue != nil {
		if *dto.ContractValue < 0 {
			return nil, apperrors.NewBadRequest("Contract value cannot be negative")
		}
		c.ContractValue = *dto.ContractValue
	}
	if dto.StartDate != "" {
		c.StartDate = dto.StartDate
	}
	if dto.EndDate != "" {
		c.EndDate = dto.EndDate
	}
	if err := uc.validateDates(c.StartDate, c.EndDate); err != nil {
		return nil, err
	}
	if dto.PaymentTerms != "" {
		c.PaymentTerms = dto.PaymentTerms
	}
	if dto.Attachments != "" {
		c.Attachments = dto.Attachments
	}

	if err := uc.repo.UpdateContract(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update contract")
	}
	return ToContractResponse(c), nil
}

func (uc *contractsUseCase) UpdateContractStatus(ctx context.Context, id uuid.UUID, dto UpdateContractStatusDTO) (*ContractResponseDTO, error) {
	if !validStatuses[dto.Status] {
		return nil, apperrors.NewBadRequest("Status must be one of draft, active, expired, terminated, renewed")
	}
	c, err := uc.repo.GetContractByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get contract")
	}
	if c == nil {
		return nil, apperrors.NewNotFound("Contract not found")
	}
	c.Status = dto.Status
	if err := uc.repo.UpdateContract(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update contract status")
	}
	return ToContractResponse(c), nil
}

func (uc *contractsUseCase) RenewContract(ctx context.Context, id uuid.UUID) (*ContractResponseDTO, error) {
	old, err := uc.repo.GetContractByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get contract")
	}
	if old == nil {
		return nil, apperrors.NewNotFound("Contract not found")
	}

	oldEnd, err := time.Parse("2006-01-02", old.EndDate)
	if err != nil {
		oldEnd = time.Now()
	}
	oldStart, err := time.Parse("2006-01-02", old.StartDate)
	if err != nil {
		oldStart = time.Now()
	}
	duration := oldEnd.Sub(oldStart)
	newStart := oldEnd.AddDate(0, 0, 1)
	newEnd := newStart.Add(duration)

	count, err := uc.repo.CountContracts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate contract number")
	}

	renewed := &domain.Contract{
		CompanyID:      old.CompanyID,
		ContractNumber: generateContractNumber(count + 1),
		PartyType:      old.PartyType,
		PartyID:        old.PartyID,
		PartyName:      old.PartyName,
		Title:          old.Title,
		Description:    old.Description,
		ContractValue:  old.ContractValue,
		StartDate:      newStart.Format("2006-01-02"),
		EndDate:        newEnd.Format("2006-01-02"),
		Status:         "active",
		PaymentTerms:   old.PaymentTerms,
		Attachments:    old.Attachments,
		RenewedFromID:  &old.ID,
	}

	if err := uc.repo.CreateContract(ctx, renewed); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create renewed contract")
	}

	old.Status = "renewed"
	if err := uc.repo.UpdateContract(ctx, old); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update original contract")
	}

	return ToContractResponse(renewed), nil
}

func (uc *contractsUseCase) ListExpiringSoon(ctx context.Context) ([]ContractResponseDTO, error) {
	contracts, err := uc.repo.ListAllContracts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list contracts")
	}
	var result []ContractResponseDTO
	for _, c := range contracts {
		dto := ToContractResponse(&c)
		if dto.NeedsAttention {
			result = append(result, *dto)
		}
	}
	return result, nil
}

// SeedInitialData populates a few sample contracts on first boot
func (uc *contractsUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountContracts(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()
	contracts := []struct {
		dto    CreateContractDTO
		status string
	}{
		{
			dto: CreateContractDTO{
				PartyType:     "customer",
				PartyName:     "PT Graha Konstruksi Nusantara",
				Title:         "Kontrak Pasokan Rutin Karton Box Corrugated Tahunan",
				ContractValue: 5400000000,
				StartDate:     now.AddDate(0, -8, 0).Format("2006-01-02"),
				EndDate:       now.AddDate(0, 4, 0).Format("2006-01-02"),
				PaymentTerms:  "Net 30",
			},
			status: "active",
		},
		{
			dto: CreateContractDTO{
				PartyType:     "vendor",
				PartyName:     "PT Surya Perkasa Paper",
				Title:         "Perjanjian Suplai Bahan Baku Kertas Kraft Roll",
				ContractValue: 2800000000,
				StartDate:     now.AddDate(0, -6, 0).Format("2006-01-02"),
				EndDate:       now.AddDate(0, 0, 20).Format("2006-01-02"),
				PaymentTerms:  "Net 45",
			},
			status: "active",
		},
		{
			dto: CreateContractDTO{
				PartyType:     "customer",
				PartyName:     "CV Surya Makmur Logistik",
				Title:         "Kontrak Distribusi Bahan Bangunan & Packaging Surabaya",
				ContractValue: 1850000000,
				StartDate:     now.AddDate(0, -1, 0).Format("2006-01-02"),
				EndDate:       now.AddDate(1, 0, 0).Format("2006-01-02"),
				PaymentTerms:  "Net 14",
			},
			status: "draft",
		},
	}
	for _, item := range contracts {
		res, err := uc.CreateContract(ctx, item.dto)
		if err != nil {
			continue
		}
		if item.status != "draft" {
			id, parseErr := uuid.Parse(res.ID.String())
			if parseErr == nil {
				_, _ = uc.UpdateContractStatus(ctx, id, UpdateContractStatusDTO{Status: item.status})
			}
		}
	}
	return nil
}
