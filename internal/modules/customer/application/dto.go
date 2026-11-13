package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	"github.com/google/uuid"
)

// AddressDTO is one delivery address (alamat kirim).
type AddressDTO struct {
	ID            *uuid.UUID `json:"id,omitempty"`
	Label         string     `json:"label"`
	RecipientName string     `json:"recipientName,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	Address       string     `json:"address"`
	City          string     `json:"city,omitempty"`
	Province      string     `json:"province,omitempty"`
	PostalCode    string     `json:"postalCode,omitempty"`
	IsDefault     bool       `json:"isDefault"`
}

type CreateCustomerDTO struct {
	Code              string       `json:"code"`
	CustomerType      string       `json:"customerType"` // B2B | B2C
	Name              string       `json:"name"`
	Email             string       `json:"email"`
	Phone             string       `json:"phone"`
	Address           string       `json:"address"`
	City              string       `json:"city,omitempty"`
	Province          string       `json:"province,omitempty"`
	PostalCode        string       `json:"postalCode,omitempty"`
	NPWP              string       `json:"npwp"`
	NIK               string       `json:"nik"`
	Segment           string       `json:"segment"`
	Status            string       `json:"status"`
	ContactPerson     string       `json:"contactPerson,omitempty"`
	ContactPhone      string       `json:"contactPhone,omitempty"`
	ContactMobile     string       `json:"contactMobile,omitempty"`
	ContactWhatsapp   string       `json:"contactWhatsapp,omitempty"`
	ShippingAddresses []AddressDTO `json:"shippingAddresses,omitempty"`
}

type UpdateCustomerDTO struct {
	Code            *string `json:"code,omitempty"`
	Name            *string `json:"name,omitempty"`
	CustomerType    *string `json:"customerType,omitempty"`
	Email           *string `json:"email,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	Address         *string `json:"address,omitempty"`
	City            *string `json:"city,omitempty"`
	Province        *string `json:"province,omitempty"`
	PostalCode      *string `json:"postalCode,omitempty"`
	NPWP            *string `json:"npwp,omitempty"`
	NIK             *string `json:"nik,omitempty"`
	Segment         *string `json:"segment,omitempty"`
	Status          *string `json:"status,omitempty"`
	ContactPerson   *string `json:"contactPerson,omitempty"`
	ContactPhone    *string `json:"contactPhone,omitempty"`
	ContactMobile   *string `json:"contactMobile,omitempty"`
	ContactWhatsapp *string `json:"contactWhatsapp,omitempty"`
	// ShippingAddresses, when present (even empty), replaces the whole list.
	ShippingAddresses *[]AddressDTO `json:"shippingAddresses,omitempty"`
}

type SetStatusDTO struct {
	Status string `json:"status"`
}

type CustomerResponseDTO struct {
	ID                uuid.UUID    `json:"id"`
	Code              string       `json:"code"`
	CustomerType      string       `json:"customerType"`
	Name              string       `json:"name"`
	Email             string       `json:"email"`
	Phone             string       `json:"phone"`
	Address           string       `json:"address"`
	City              string       `json:"city"`
	Province          string       `json:"province,omitempty"`
	PostalCode        string       `json:"postalCode,omitempty"`
	NPWP              string       `json:"npwp"`
	NIK               string       `json:"nik"`
	Segment           string       `json:"segment"`
	Status            string       `json:"status"`
	ContactPerson     string       `json:"contactPerson,omitempty"`
	ContactPhone      string       `json:"contactPhone,omitempty"`
	ContactMobile     string       `json:"contactMobile,omitempty"`
	ContactWhatsapp   string       `json:"contactWhatsapp,omitempty"`
	ShippingAddresses []AddressDTO `json:"shippingAddresses"`
	CreatedAt         time.Time    `json:"createdAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
}

func ToCustomerResponse(c *domain.Customer) *CustomerResponseDTO {
	if c == nil {
		return nil
	}
	addrs := make([]AddressDTO, 0, len(c.ShippingAddresses))
	for _, a := range c.ShippingAddresses {
		id := a.ID
		addrs = append(addrs, AddressDTO{
			ID: &id, Label: a.Label, RecipientName: a.RecipientName, Phone: a.Phone,
			Address: a.Address, City: a.City, Province: a.Province, PostalCode: a.PostalCode, IsDefault: a.IsDefault,
		})
	}
	// Legacy rows predate customerType; the old default segment was a B2B one.
	ctype := c.CustomerType
	if ctype == "" {
		ctype = CustomerTypeB2B
	}
	return &CustomerResponseDTO{
		ID: c.ID, Code: c.Code, CustomerType: ctype, Name: c.Name, Email: c.Email, Phone: c.Phone,
		Address: c.Address, City: c.City, Province: c.Province, PostalCode: c.PostalCode,
		NPWP: c.NPWP, NIK: c.NIK, Segment: c.Segment, Status: c.Status,
		ContactPerson: c.ContactPerson, ContactPhone: c.ContactPhone, ContactMobile: c.ContactMobile, ContactWhatsapp: c.ContactWA,
		ShippingAddresses: addrs, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func ToCustomerResponseList(customers []domain.Customer) []CustomerResponseDTO {
	result := make([]CustomerResponseDTO, len(customers))
	for i := range customers {
		result[i] = *ToCustomerResponse(&customers[i])
	}
	return result
}
