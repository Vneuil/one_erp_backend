package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/taxid"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type SupplierUseCase interface {
	Create(ctx context.Context, dto CreateSupplierDTO) (*SupplierResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*SupplierResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]SupplierResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateSupplierDTO) (*SupplierResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SeedInitialData(ctx context.Context) error
}

type supplierUseCase struct {
	repo domain.SupplierRepository
}

func NewSupplierUseCase(repo domain.SupplierRepository) SupplierUseCase {
	return &supplierUseCase{repo: repo}
}

func (uc *supplierUseCase) Create(ctx context.Context, dto CreateSupplierDTO) (*SupplierResponseDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(dto.Code))
	name := strings.TrimSpace(dto.Name)

	if name == "" {
		return nil, apperrors.NewBadRequest("Supplier Name is required")
	}
	// The code is generated when the client does not supply one, so the UI never
	// has to guess the next number (and collide with an existing record).
	if code == "" {
		generated, err := uc.nextCode(ctx)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to generate supplier code")
		}
		code = generated
	}

	existing, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing code")
	}
	if existing != nil {
		return nil, apperrors.NewConflict("Supplier with this code already exists")
	}

	status := dto.Status
	if status == "" {
		status = "Active"
	}
	category := dto.Category
	if category == "" {
		category = "Raw Materials"
	}

	npwp, err := taxid.NPWP(dto.NPWP)
	if err != nil {
		return nil, err
	}
	nik, err := taxid.NIK(dto.NIK)
	if err != nil {
		return nil, err
	}

	supplier := &domain.Supplier{
		Code:          code,
		Name:          name,
		ContactPerson: dto.ContactPerson,
		Email:         dto.Email,
		Phone:         dto.Phone,
		Address:       dto.Address,
		NPWP:          npwp,
		NIK:           nik,
		Category:      category,
		Status:        status,
	}

	if err := uc.repo.Create(ctx, supplier); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create supplier")
	}

	return ToSupplierResponse(supplier), nil
}

func (uc *supplierUseCase) GetByID(ctx context.Context, id uuid.UUID) (*SupplierResponseDTO, error) {
	supplier, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve supplier")
	}
	if supplier == nil {
		return nil, apperrors.NewNotFound("Supplier not found")
	}
	return ToSupplierResponse(supplier), nil
}

func (uc *supplierUseCase) List(ctx context.Context, query types.PaginationQuery) ([]SupplierResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	suppliers, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list suppliers")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToSupplierResponseList(suppliers), meta, nil
}

func (uc *supplierUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateSupplierDTO) (*SupplierResponseDTO, error) {
	supplier, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get supplier")
	}
	if supplier == nil {
		return nil, apperrors.NewNotFound("Supplier not found")
	}

	if dto.Name != nil {
		supplier.Name = *dto.Name
	}
	if dto.ContactPerson != nil {
		supplier.ContactPerson = *dto.ContactPerson
	}
	if dto.Email != nil {
		supplier.Email = *dto.Email
	}
	if dto.Phone != nil {
		supplier.Phone = *dto.Phone
	}
	if dto.Address != nil {
		supplier.Address = *dto.Address
	}
	if dto.NPWP != nil {
		npwp, err := taxid.NPWP(*dto.NPWP)
		if err != nil {
			return nil, err
		}
		supplier.NPWP = npwp
	}
	if dto.NIK != nil {
		nik, err := taxid.NIK(*dto.NIK)
		if err != nil {
			return nil, err
		}
		supplier.NIK = nik
	}
	if dto.Category != nil {
		supplier.Category = *dto.Category
	}
	if dto.Status != nil {
		supplier.Status = *dto.Status
	}

	if err := uc.repo.Update(ctx, supplier); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update supplier")
	}

	return ToSupplierResponse(supplier), nil
}

// Delete removes a supplier, but only if it is not still referenced by any
// Purchase Order or Purchase Invoice, mirroring the "block delete if
// referenced" convention used for warehouses/products/customers.
func (uc *supplierUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	referenced, err := uc.repo.HasReferences(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check supplier references")
	}
	if referenced {
		return apperrors.NewConflict("Cannot delete supplier: it is referenced by existing purchase orders or purchase invoices")
	}
	return uc.repo.Delete(ctx, id)
}

func (uc *supplierUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	initial := []CreateSupplierDTO{
		{
			Code:          "SUP-001",
			Name:          "PT Surya Perkasa Paper",
			ContactPerson: "Hendro Wijaya",
			Email:         "sales@suryaperkasa.com",
			Phone:         "+62 21 8976 5432",
			Address:       "Kawasan Industri Jababeka 1, Cikarang",
			Category:      "Raw Materials",
			Status:        "Active",
		},
		{
			Code:          "SUP-002",
			Name:          "PT Indo Steel Perkasa",
			ContactPerson: "Ir. Bambang",
			Email:         "order@indosteel.co.id",
			Phone:         "+62 21 8899 0011",
			Address:       "Kawasan Krakatau Industrial Estate Cilegon",
			Category:      "Raw Materials",
			Status:        "Active",
		},
	}

	for _, item := range initial {
		_, _ = uc.Create(ctx, item)
	}
	return nil
}

// nextCode returns the first unused "SUP-NNN" code, starting after the current record count.
func (uc *supplierUseCase) nextCode(ctx context.Context) (string, error) {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return "", err
	}
	for n := count + 1; n < count+10000; n++ {
		candidate := fmt.Sprintf("SUP-%03d", n)
		existing, err := uc.repo.GetByCode(ctx, candidate)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no free supplier code found")
}
