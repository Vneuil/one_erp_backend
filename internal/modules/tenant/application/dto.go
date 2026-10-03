package application

import "github.com/google/uuid"

type TenantDatabaseDTO struct {
	CompanyID    uuid.UUID `json:"companyId"`
	DatabaseHost string    `json:"databaseHost"`
	DatabasePort int       `json:"databasePort"`
	DatabaseName string    `json:"databaseName"`
	Status       string    `json:"status"`
}
