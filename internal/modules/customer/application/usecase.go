package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type CustomerUseCase interface {
	Create(ctx context.Context, dto CreateCustomerDTO) (*CustomerResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*CustomerResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]CustomerResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateCustomerDTO) (*CustomerResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SeedInitialData(ctx context.Context) error
}

type customerUseCase struct {
	repo domain.CustomerRepository
}

func NewCustomerUseCase(repo domain.CustomerRepository) CustomerUseCase {
	return &customerUseCase{repo: repo}
}

func (uc *customerUseCase) Create(ctx context.Context, dto CreateCustomerDTO) (*CustomerResponseDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(dto.Code))
	name := strings.TrimSpace(dto.Name)

	if name == "" {
		return nil, apperrors.NewBadRequest("Customer Name is required")
	}
	// The code is generated when the client does not supply one, so the UI never
	// has to guess the next number (and collide with an existing record).
	if code == "" {
		generated, err := uc.nextCode(ctx)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to generate customer code")
		}
		code = generated
	}

	existing, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing code")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("Customer with this code already exists")
	}

	status := dto.Status
	if status == "" {
		status = "Active"
	}
	segment := dto.Segment
	if segment == "" {
		segment = "Enterprise B2B"
	}

	customer := &domain.Customer{
		Code:    code,
		Name:    name,
		Email:   dto.Email,
		Phone:   dto.Phone,
		Address: dto.Address,
		Segment: segment,
		Status:  status,
	}

	if err := uc.repo.Create(ctx, customer); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create customer")
	}

	return ToCustomerResponse(customer), nil
}

func (uc *customerUseCase) GetByID(ctx context.Context, id uuid.UUID) (*CustomerResponseDTO, error) {
	customer, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve customer")
	}
	if customer == nil {
		return nil, apperrors.NewNotFound("Customer not found")
	}
	return ToCustomerResponse(customer), nil
}

func (uc *customerUseCase) List(ctx context.Context, query types.PaginationQuery) ([]CustomerResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	customers, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list customers")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToCustomerResponseList(customers), meta, nil
}

func (uc *customerUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateCustomerDTO) (*CustomerResponseDTO, error) {
	customer, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get customer")
	}
	if customer == nil {
		return nil, apperrors.NewNotFound("Customer not found")
	}

	if dto.Name != nil {
		customer.Name = *dto.Name
	}
	if dto.Email != nil {
		customer.Email = *dto.Email
	}
	if dto.Phone != nil {
		customer.Phone = *dto.Phone
	}
	if dto.Address != nil {
		customer.Address = *dto.Address
	}
	if dto.Segment != nil {
		customer.Segment = *dto.Segment
	}
	if dto.Status != nil {
		customer.Status = *dto.Status
	}

	if err := uc.repo.Update(ctx, customer); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update customer")
	}

	return ToCustomerResponse(customer), nil
}

// Delete removes a customer, but only if it is not still referenced by any
// Sales Order, Invoice, Quotation, or CRM Deal, mirroring the "block delete
// if referenced" convention used for warehouses/products.
func (uc *customerUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	customer, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get customer")
	}
	if customer == nil {
		return apperrors.NewNotFound("Customer not found")
	}
	referenced, err := uc.repo.HasReferences(ctx, customer.Name)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check customer references")
	}
	if referenced {
		return apperrors.NewConflict("Cannot delete customer: it is referenced by existing sales orders, invoices, quotations, or deals")
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *customerUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	initial := []CreateCustomerDTO{
		{
			Code:    "CUST-001",
			Name:    "PT Graha Konstruksi Nusantara",
			Email:   "procurement@grahakonstruksi.co.id",
			Phone:   "+62 21 8901 2345",
			Address: "Kawasan Industri MM2100 Blok C, Cikarang Barat",
			Segment: "Enterprise B2B",
			Status:  "Active",
		},
		{
			Code:    "CUST-002",
			Name:    "CV Surya Makmur Logistik",
			Email:   "order@suryamakmur.id",
			Phone:   "+62 31 5566 7788",
			Address: "Jl. Rungkut Industri Raya No. 45, Surabaya",
			Segment: "Wholesale & Distributor",
			Status:  "Active",
		},
		{
			Code:    "CUST-003",
			Name:    "PT Sentosa Jaya Abadi",
			Email:   "finance@sentosajaya.com",
			Phone:   "+62 21 5544 3322",
			Address: "Jl. Daan Mogot Km 12, Jakarta Barat",
			Segment: "Retail Network",
			Status:  "Active",
		},
	}

	for _, item := range initial {
		_, _ = uc.Create(ctx, item)
	}
	return nil
}

// nextCode returns the first unused "CUST-NNN" code, starting after the current record count.
func (uc *customerUseCase) nextCode(ctx context.Context) (string, error) {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return "", err
	}
	for n := count + 1; n < count+10000; n++ {
		candidate := fmt.Sprintf("CUST-%03d", n)
		existing, err := uc.repo.GetByCode(ctx, candidate)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no free customer code found")
}
