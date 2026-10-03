package domain

import (
	"github.com/divinecoid/one-backend/internal/shared/types"
)

// Company represents an organization/company entity in the ERP system
type Company struct {
	types.BaseEntity
	Code     string `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name     string `gorm:"type:varchar(255);not null" json:"name"`
	Email    string `gorm:"type:varchar(255)" json:"email,omitempty"`
	Phone    string `gorm:"type:varchar(50)" json:"phone,omitempty"`
	Address  string `gorm:"type:text" json:"address,omitempty"`
	TaxID    string `gorm:"type:varchar(50)" json:"taxId,omitempty"` // NPWP / Tax Registration
	Currency string `gorm:"type:varchar(10);default:'IDR'" json:"currency"`
	IsActive bool   `gorm:"default:true" json:"isActive"`
}

func (Company) TableName() string {
	return "companies"
}
