package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/taxid"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type CustomerUseCase interface {
	Create(ctx context.Context, dto CreateCustomerDTO) (*CustomerResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*CustomerResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]CustomerResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateCustomerDTO) (*CustomerResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SetStatus(ctx context.Context, id uuid.UUID, status string) (*CustomerResponseDTO, error)
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
	ctype, err := normalizeCustomerType(dto.CustomerType, code)
	if err != nil {
		return nil, err
	}
	// The code is generated when the client does not supply one, so the UI never
	// has to guess the next number (and collide with an existing record).
	if code == "" {
		generated, err := uc.nextCode(ctx, ctype)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to generate customer code")
		}
		code = generated
	} else {
		prefixType, valid := customerCodeType(code)
		if !valid {
			return nil, apperrors.NewBadRequest("Customer code must follow B2B-NNNN or B2C-NNNN, e.g. B2B-0001")
		}
		ctype = prefixType
	}
	if err := validateB2BContact(ctype, dto.ContactPerson, dto.ContactPhone, dto.ContactMobile, dto.ContactWhatsapp); err != nil {
		return nil, err
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
		if ctype == CustomerTypeB2C {
			segment = "Retail Individual"
		}
	}

	npwp, err := taxid.NPWP(dto.NPWP)
	if err != nil {
		return nil, err
	}
	nik, err := taxid.NIK(dto.NIK)
	if err != nil {
		return nil, err
	}

	customer := &domain.Customer{
		Code:          code,
		CustomerType:  ctype,
		Name:          name,
		Email:         dto.Email,
		Phone:         dto.Phone,
		Address:       dto.Address,
		City:          defaultCity(dto.City),
		Province:      dto.Province,
		PostalCode:    dto.PostalCode,
		NPWP:          npwp,
		NIK:           nik,
		Segment:       segment,
		Status:        status,
		ContactPerson: strings.TrimSpace(dto.ContactPerson),
		ContactPhone:  strings.TrimSpace(dto.ContactPhone),
		ContactMobile: strings.TrimSpace(dto.ContactMobile),
		ContactWA:     strings.TrimSpace(dto.ContactWhatsapp),
	}

	if err := uc.repo.Create(ctx, customer); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create customer")
	}
	if len(dto.ShippingAddresses) > 0 {
		if err := uc.repo.ReplaceShippingAddresses(ctx, customer.ID, toAddressEntities(dto.ShippingAddresses)); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to save shipping addresses")
		}
		if reloaded, _ := uc.repo.GetByID(ctx, customer.ID); reloaded != nil {
			customer = reloaded
		}
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
	if dto.CustomerType != nil {
		t, err := normalizeCustomerType(*dto.CustomerType, "")
		if err != nil {
			return nil, err
		}
		customer.CustomerType = t
	}
	if dto.Code != nil {
		newCode := strings.ToUpper(strings.TrimSpace(*dto.Code))
		if newCode != "" && newCode != customer.Code {
			prefixType, valid := customerCodeType(newCode)
			if !valid {
				return nil, apperrors.NewBadRequest("Customer code must follow B2B-NNNN or B2C-NNNN, e.g. B2B-0001")
			}
			existing, err := uc.repo.GetByCode(ctx, newCode)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to check existing code")
			}
			if existing != nil && existing.ID != customer.ID {
				return nil, apperrors.NewConflict("Customer with this code already exists")
			}
			customer.Code = newCode
			customer.CustomerType = prefixType
		} else if newCode != "" && customer.CustomerType == "" {
			if prefixType, valid := customerCodeType(newCode); valid {
				customer.CustomerType = prefixType
			}
		}
	}
	if dto.City != nil {
		customer.City = defaultCity(*dto.City)
	}
	if dto.Province != nil {
		customer.Province = *dto.Province
	}
	if dto.PostalCode != nil {
		customer.PostalCode = *dto.PostalCode
	}
	if dto.ContactPerson != nil {
		customer.ContactPerson = strings.TrimSpace(*dto.ContactPerson)
	}
	if dto.ContactPhone != nil {
		customer.ContactPhone = strings.TrimSpace(*dto.ContactPhone)
	}
	if dto.ContactMobile != nil {
		customer.ContactMobile = strings.TrimSpace(*dto.ContactMobile)
	}
	if dto.ContactWhatsapp != nil {
		customer.ContactWA = strings.TrimSpace(*dto.ContactWhatsapp)
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
	if dto.NPWP != nil {
		npwp, err := taxid.NPWP(*dto.NPWP)
		if err != nil {
			return nil, err
		}
		customer.NPWP = npwp
	}
	if dto.NIK != nil {
		nik, err := taxid.NIK(*dto.NIK)
		if err != nil {
			return nil, err
		}
		customer.NIK = nik
	}
	if dto.Segment != nil {
		customer.Segment = *dto.Segment
	}
	if dto.Status != nil {
		status, err := normalizeStatus(*dto.Status)
		if err != nil {
			return nil, err
		}
		if status != customer.Status && !canChangeStatus(ctx) {
			return nil, apperrors.NewForbidden("Only Manager or Admin can change customer status")
		}
		customer.Status = status
	}

	// Only explicit B2B records are validated; legacy rows without a type stay editable.
	if err := validateB2BContact(customer.CustomerType, customer.ContactPerson, customer.ContactPhone, customer.ContactMobile, customer.ContactWA); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, customer); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update customer")
	}
	if dto.ShippingAddresses != nil {
		if err := uc.repo.ReplaceShippingAddresses(ctx, customer.ID, toAddressEntities(*dto.ShippingAddresses)); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to save shipping addresses")
		}
		if reloaded, _ := uc.repo.GetByID(ctx, customer.ID); reloaded != nil {
			customer = reloaded
		}
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
			Code:          "B2B-0001",
			Name:          "PT Graha Konstruksi Nusantara",
			Email:         "procurement@grahakonstruksi.co.id",
			Phone:         "+62 21 8901 2345",
			Address:       "Kawasan Industri MM2100 Blok C, Cikarang Barat",
			Segment:       "Enterprise B2B",
			Status:        "Active",
			ContactPerson: "Bagian Pengadaan", ContactPhone: "+62 21 8901 2345",
		},
		{
			Code:          "B2B-0002",
			Name:          "CV Surya Makmur Logistik",
			Email:         "order@suryamakmur.id",
			Phone:         "+62 31 5566 7788",
			Address:       "Jl. Rungkut Industri Raya No. 45, Surabaya",
			Segment:       "Wholesale & Distributor",
			Status:        "Active",
			ContactPerson: "Bagian Order", ContactPhone: "+62 31 5566 7788",
		},
		{
			Code:          "B2B-0003",
			Name:          "PT Sentosa Jaya Abadi",
			Email:         "finance@sentosajaya.com",
			Phone:         "+62 21 5544 3322",
			Address:       "Jl. Daan Mogot Km 12, Jakarta Barat",
			Segment:       "Retail Network",
			Status:        "Active",
			ContactPerson: "Bagian Keuangan", ContactPhone: "+62 21 5544 3322",
		},
	}

	for _, item := range initial {
		_, _ = uc.Create(ctx, item)
	}
	return nil
}

// SetStatus activates or deactivates a customer without touching its history.
func (uc *customerUseCase) SetStatus(ctx context.Context, id uuid.UUID, status string) (*CustomerResponseDTO, error) {
	normalized, err := normalizeStatus(status)
	if err != nil {
		return nil, err
	}
	customer, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get customer")
	}
	if customer == nil {
		return nil, apperrors.NewNotFound("Customer not found")
	}
	customer.Status = normalized
	if err := uc.repo.Update(ctx, customer); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update customer status")
	}
	return ToCustomerResponse(customer), nil
}

func canChangeStatus(ctx context.Context) bool {
	roles := actor.RolesFrom(ctx)
	if len(roles) == 0 {
		return true
	}
	for _, r := range roles {
		if strings.EqualFold(r, "admin") || strings.EqualFold(r, "superadmin") || strings.EqualFold(r, "owner") || strings.EqualFold(r, "manager") {
			return true
		}
	}
	return false
}

// nextCode returns the next free "B2B-NNNN" / "B2C-NNNN" code.
func (uc *customerUseCase) nextCode(ctx context.Context, ctype string) (string, error) {
	existing, err := uc.repo.CodesWithPrefix(ctx, ctype+"-")
	if err != nil {
		return "", err
	}
	for seq := nextCustomerSeq(ctype, existing); seq < 10000000; seq++ {
		candidate := formatCustomerCode(ctype, seq)
		taken, err := uc.repo.GetByCode(ctx, candidate)
		if err != nil {
			return "", err
		}
		if taken == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no free customer code found")
}
