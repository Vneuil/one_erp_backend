package application

import (
	"time"

	"github.com/google/uuid"
)

type TenantResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	IsActive  bool      `json:"isActive"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateTenantDTO struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type SwitchTenantResultDTO struct {
	AccessToken string            `json:"accessToken"`
	TokenType   string            `json:"tokenType"`
	ExpiresIn   int64             `json:"expiresIn"`
	Tenant      TenantResponseDTO `json:"tenant"`
}
