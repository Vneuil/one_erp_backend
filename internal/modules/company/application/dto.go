package application

// CreateCompanyDTO holds input data for creating a new company
type CreateCompanyDTO struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	TaxID    string `json:"taxId"`
	Currency string `json:"currency"`
}

// UpdateCompanyDTO holds input data for updating an existing company
type UpdateCompanyDTO struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Address  *string `json:"address"`
	TaxID    *string `json:"taxId"`
	Currency *string `json:"currency"`
	IsActive *bool   `json:"isActive"`
}
