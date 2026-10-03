package application

import (
	userApp "github.com/divinecoid/one-backend/internal/modules/user/application"
	"github.com/google/uuid"
)

type RegisterDTO struct {
	CompanyName string `json:"companyName"`
	CompanyCode string `json:"companyCode"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type LoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponseDTO struct {
	AccessToken string                  `json:"accessToken"`
	TokenType   string                  `json:"tokenType"`
	ExpiresIn   int64                   `json:"expiresIn"`
	User        userApp.UserResponseDTO `json:"user"`
	Company     *CompanySummaryDTO      `json:"company,omitempty"`
}

type MeResponseDTO struct {
	User    userApp.UserResponseDTO `json:"user"`
	Company *CompanySummaryDTO      `json:"company,omitempty"`
}

type CompanySummaryDTO struct {
	ID       uuid.UUID `json:"id"`
	Code     string    `json:"code"`
	Name     string    `json:"name"`
	Currency string    `json:"currency"`
}

type CompanyMembershipDTO struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
	Role string    `json:"role"`
}

type SwitchCompanyDTO struct {
	CompanyID uuid.UUID `json:"companyId"`
}

type ForgotPasswordDTO struct {
	Email string `json:"email"`
}

type ResetPasswordDTO struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}
