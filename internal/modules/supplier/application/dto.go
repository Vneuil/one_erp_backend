package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	"github.com/google/uuid"
)

type CreateSupplierDTO struct {
	Code          string `json:"code"`
	BatchCode     string `json:"batchCode,omitempty"`
	Name          string `json:"name"`
	ContactPerson string `json:"contactPerson"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Address       string `json:"address"`
	VendorType    string `json:"vendorType,omitempty"`
	City          string `json:"city,omitempty"`
	Province      string `json:"province,omitempty"`
	PostalCode    string `json:"postalCode,omitempty"`
	PaymentTerm   string `json:"paymentTerm,omitempty"`
	NPWP          string `json:"npwp"`
	NIK           string `json:"nik"`
	Category      string `json:"category"`
	Status        string `json:"status"`
}

type UpdateSupplierDTO struct {
	Code          *string `json:"code,omitempty"`
	Name          *string `json:"name,omitempty"`
	VendorType    *string `json:"vendorType,omitempty"`
	City          *string `json:"city,omitempty"`
	Province      *string `json:"province,omitempty"`
	PostalCode    *string `json:"postalCode,omitempty"`
	PaymentTerm   *string `json:"paymentTerm,omitempty"`
	BatchCode     *string `json:"batchCode,omitempty"`
	ContactPerson *string `json:"contactPerson,omitempty"`
	Email         *string `json:"email,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Address       *string `json:"address,omitempty"`
	NPWP          *string `json:"npwp,omitempty"`
	NIK           *string `json:"nik,omitempty"`
	Category      *string `json:"category,omitempty"`
	Status        *string `json:"status,omitempty"`
}

type SupplierResponseDTO struct {
	ID            uuid.UUID                 `json:"id"`
	Code          string                    `json:"code"`
	BatchCode     string                    `json:"batchCode,omitempty"`
	Name          string                    `json:"name"`
	ContactPerson string                    `json:"contactPerson"`
	Email         string                    `json:"email"`
	Phone         string                    `json:"phone"`
	Address       string                    `json:"address"`
	VendorType    string                    `json:"vendorType,omitempty"`
	City          string                    `json:"city"`
	Province      string                    `json:"province,omitempty"`
	PostalCode    string                    `json:"postalCode,omitempty"`
	PaymentTerm   string                    `json:"paymentTerm,omitempty"`
	NPWP          string                    `json:"npwp"`
	NIK           string                    `json:"nik"`
	Category      string                    `json:"category"`
	Status        string                    `json:"status"`
	Codes         []SupplierCodeResponseDTO `json:"codes"`
	CreatedAt     time.Time                 `json:"createdAt"`
	UpdatedAt     time.Time                 `json:"updatedAt"`
}

type CreateSupplierCodeDTO struct {
	SupplierID  uuid.UUID `json:"supplierId"`
	Code        string    `json:"code,omitempty"`
	VendorType  string    `json:"vendorType"`
	BatchCode   string    `json:"batchCode,omitempty"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status,omitempty"`
}

type UpdateSupplierCodeDTO struct {
	Code        *string `json:"code,omitempty"`
	VendorType  *string `json:"vendorType,omitempty"`
	BatchCode   *string `json:"batchCode,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type SupplierCodeResponseDTO struct {
	ID           uuid.UUID `json:"id"`
	SupplierID   uuid.UUID `json:"supplierId"`
	SupplierName string    `json:"supplierName,omitempty"`
	Code         string    `json:"code"`
	VendorType   string    `json:"vendorType"`
	BatchCode    string    `json:"batchCode,omitempty"`
	Description  string    `json:"description,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func ToSupplierCodeResponse(sc *domain.SupplierCode) *SupplierCodeResponseDTO {
	if sc == nil {
		return nil
	}
	return &SupplierCodeResponseDTO{
		ID:           sc.ID,
		SupplierID:   sc.SupplierID,
		SupplierName: sc.SupplierName,
		Code:         sc.Code,
		VendorType:   sc.VendorType,
		BatchCode:    sc.BatchCode,
		Description:  sc.Description,
		Status:       sc.Status,
		CreatedAt:    sc.CreatedAt,
		UpdatedAt:    sc.UpdatedAt,
	}
}

func ToSupplierCodeResponseList(items []domain.SupplierCode) []SupplierCodeResponseDTO {
	result := make([]SupplierCodeResponseDTO, len(items))
	for i, sc := range items {
		result[i] = *ToSupplierCodeResponse(&sc)
	}
	return result
}

func ToSupplierResponse(s *domain.Supplier) *SupplierResponseDTO {
	if s == nil {
		return nil
	}
	codes := make([]SupplierCodeResponseDTO, 0, len(s.Codes))
	for _, c := range s.Codes {
		codes = append(codes, *ToSupplierCodeResponse(&c))
	}
	// Fallback to legacy supplier code if codes array is empty but code is set
	if len(codes) == 0 && s.Code != "" {
		codes = append(codes, SupplierCodeResponseDTO{
			ID:           s.ID,
			SupplierID:   s.ID,
			SupplierName: s.Name,
			Code:         s.Code,
			VendorType:   s.VendorType,
			BatchCode:    s.BatchCode,
			Status:       s.Status,
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
		})
	}

	return &SupplierResponseDTO{
		ID:            s.ID,
		Code:          s.Code,
		BatchCode:     s.BatchCode,
		Name:          s.Name,
		ContactPerson: s.ContactPerson,
		Email:         s.Email,
		Phone:         s.Phone,
		Address:       s.Address,
		VendorType:    s.VendorType,
		City:          s.City,
		Province:      s.Province,
		PostalCode:    s.PostalCode,
		PaymentTerm:   s.PaymentTerm,
		NPWP:          s.NPWP,
		NIK:           s.NIK,
		Category:      s.Category,
		Status:        s.Status,
		Codes:         codes,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func ToSupplierResponseList(suppliers []domain.Supplier) []SupplierResponseDTO {
	result := make([]SupplierResponseDTO, len(suppliers))
	for i, s := range suppliers {
		result[i] = *ToSupplierResponse(&s)
	}
	return result
}

type SetStatusDTO struct {
	Status string `json:"status"`
}
