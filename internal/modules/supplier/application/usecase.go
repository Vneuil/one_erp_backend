package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
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
	SetStatus(ctx context.Context, id uuid.UUID, status string) (*SupplierResponseDTO, error)
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
	vendorType, ok := normalizeVendorType(dto.VendorType)
	if !ok {
		return nil, apperrors.NewBadRequest("Vendor type must be 2-4 letters (e.g. PRD, PCK, ATK)")
	}
	if code == "" {
		if vendorType == "" {
			return nil, apperrors.NewBadRequest("Vendor type is required (PRD, PCK, ATK) to generate the vendor code")
		}
		generated, err := uc.nextCode(ctx, vendorType)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to generate supplier code")
		}
		code = generated
	} else {
		t, valid := vendorCodeType(code)
		if !valid {
			return nil, apperrors.NewBadRequest("Vendor code must follow VD-<TYPE>-<NNN>, e.g. VD-PRD-001")
		}
		vendorType = t
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
		BatchCode:     strings.ToUpper(strings.TrimSpace(dto.BatchCode)),
		Name:          name,
		ContactPerson: dto.ContactPerson,
		Email:         dto.Email,
		Phone:         dto.Phone,
		Address:       dto.Address,
		VendorType:    vendorType,
		City:          defaultCity(dto.City),
		Province:      dto.Province,
		PostalCode:    dto.PostalCode,
		PaymentTerm:   strings.TrimSpace(dto.PaymentTerm),
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

	if dto.VendorType != nil {
		t, ok := normalizeVendorType(*dto.VendorType)
		if !ok {
			return nil, apperrors.NewBadRequest("Vendor type must be 2-4 letters (e.g. PRD, PCK, ATK)")
		}
		if t != "" {
			supplier.VendorType = t
		}
	}
	if dto.Code != nil {
		newCode := strings.ToUpper(strings.TrimSpace(*dto.Code))
		if newCode != "" && newCode != supplier.Code {
			t, valid := vendorCodeType(newCode)
			if !valid {
				return nil, apperrors.NewBadRequest("Vendor code must follow VD-<TYPE>-<NNN>, e.g. VD-PRD-001")
			}
			existing, err := uc.repo.GetByCode(ctx, newCode)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to check existing code")
			}
			if existing != nil && existing.ID != supplier.ID {
				return nil, apperrors.NewConflict("Supplier with this code already exists")
			}
			supplier.Code = newCode
			supplier.VendorType = t
		} else if newCode != "" && supplier.VendorType == "" {
			if t, valid := vendorCodeType(newCode); valid {
				supplier.VendorType = t
			}
		}
	}
	if dto.City != nil {
		supplier.City = defaultCity(*dto.City)
	}
	if dto.Province != nil {
		supplier.Province = *dto.Province
	}
	if dto.PostalCode != nil {
		supplier.PostalCode = *dto.PostalCode
	}
	if dto.PaymentTerm != nil {
		supplier.PaymentTerm = strings.TrimSpace(*dto.PaymentTerm)
	}
	if dto.BatchCode != nil {
		supplier.BatchCode = strings.ToUpper(strings.TrimSpace(*dto.BatchCode))
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
		status, err := normalizeStatus(*dto.Status)
		if err != nil {
			return nil, err
		}
		if status != supplier.Status && !canChangeStatus(ctx) {
			return nil, apperrors.NewForbidden("Only Manager or Admin can change vendor status")
		}
		supplier.Status = status
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
			Code:          "VD-PRD-001",
			Name:          "PT Surya Perkasa Paper",
			ContactPerson: "Hendro Wijaya",
			Email:         "sales@suryaperkasa.com",
			Phone:         "+62 21 8976 5432",
			Address:       "Kawasan Industri Jababeka 1, Cikarang",
			Category:      "Raw Materials",
			Status:        "Active",
		},
		{
			Code:          "VD-PRD-002",
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

// SetStatus activates or deactivates a vendor. Vendors are never hard-deleted in
// day-to-day use; deactivating keeps every historical PO/PI intact.
func (uc *supplierUseCase) SetStatus(ctx context.Context, id uuid.UUID, status string) (*SupplierResponseDTO, error) {
	normalized, err := normalizeStatus(status)
	if err != nil {
		return nil, err
	}
	supplier, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get supplier")
	}
	if supplier == nil {
		return nil, apperrors.NewNotFound("Supplier not found")
	}
	supplier.Status = normalized
	if err := uc.repo.Update(ctx, supplier); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update supplier status")
	}
	return ToSupplierResponse(supplier), nil
}

func canChangeStatus(ctx context.Context) bool {
	roles := actor.RolesFrom(ctx)
	if len(roles) == 0 {
		return true
	}
	for _, r := range roles {
		if strings.EqualFold(r, "admin") || strings.EqualFold(r, "manager") {
			return true
		}
	}
	return false
}

func normalizeStatus(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "active":
		return "Active", nil
	case "inactive":
		return "Inactive", nil
	}
	return "", apperrors.NewBadRequest("Status must be Active or Inactive")
}

// defaultCity falls back to Semarang, Manna's home base, when no city is given.
func defaultCity(c string) string {
	if c = strings.TrimSpace(c); c != "" {
		return c
	}
	return "Semarang"
}

// nextCode returns the next free "VD-<TYPE>-NNN" code for the vendor type.
func (uc *supplierUseCase) nextCode(ctx context.Context, vendorType string) (string, error) {
	existing, err := uc.repo.CodesWithPrefix(ctx, vendorCodePrefix+vendorType+"-")
	if err != nil {
		return "", err
	}
	for seq := nextVendorSeq(vendorType, existing); seq < 1000000; seq++ {
		candidate := formatVendorCode(vendorType, seq)
		taken, err := uc.repo.GetByCode(ctx, candidate)
		if err != nil {
			return "", err
		}
		if taken == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no free supplier code found")
}
