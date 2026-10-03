package application

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/workspace/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/google/uuid"
)

type TenantUseCase interface {
	ListTenants(ctx context.Context) ([]TenantResponseDTO, error)
	CreateTenant(ctx context.Context, dto CreateTenantDTO) (*TenantResponseDTO, error)
	SwitchTenant(ctx context.Context, claims *utils.JWTClaims, tenantID uuid.UUID) (*SwitchTenantResultDTO, error)
}

type tenantUseCase struct {
	repo      domain.TenantRepository
	jwtSecret string
	jwtExpiry time.Duration
}

func NewTenantUseCase(repo domain.TenantRepository, jwtSecret string, jwtExpiry time.Duration) TenantUseCase {
	return &tenantUseCase{repo: repo, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
}

func toTenantResponse(t *domain.Tenant) TenantResponseDTO {
	return TenantResponseDTO{
		ID: t.ID, Name: t.Name, Code: t.Code, IsActive: t.IsActive, IsDefault: t.IsDefault, CreatedAt: t.CreatedAt,
	}
}

func (uc *tenantUseCase) ListTenants(ctx context.Context) ([]TenantResponseDTO, error) {
	tenants, err := uc.repo.List(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list tenants")
	}
	dtos := make([]TenantResponseDTO, len(tenants))
	for i := range tenants {
		dtos[i] = toTenantResponse(&tenants[i])
	}
	return dtos, nil
}

func (uc *tenantUseCase) CreateTenant(ctx context.Context, dto CreateTenantDTO) (*TenantResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	code := strings.ToUpper(strings.TrimSpace(dto.Code))
	if name == "" || code == "" {
		return nil, apperrors.NewBadRequest("Tenant name and code are required")
	}

	t := &domain.Tenant{Name: name, Code: code, IsActive: true}
	if err := uc.repo.Create(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create tenant")
	}
	resp := toTenantResponse(t)
	return &resp, nil
}

// SwitchTenant mints a new JWT scoped to the given tenant (within the same
// company the caller is already authenticated against), mirroring how
// auth.SwitchCompany reissues a token scoped to a different company.
func (uc *tenantUseCase) SwitchTenant(ctx context.Context, claims *utils.JWTClaims, tenantID uuid.UUID) (*SwitchTenantResultDTO, error) {
	t, err := uc.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load tenant")
	}
	if t == nil || !t.IsActive {
		return nil, apperrors.NewNotFound("Tenant not found")
	}

	token, err := utils.GenerateJWTWithTenant(claims.UserID, claims.CompanyID, &t.ID, claims.Email, claims.Role, utils.TokenTypeUser, uc.jwtSecret, uc.jwtExpiry)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate authentication token")
	}

	return &SwitchTenantResultDTO{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(uc.jwtExpiry.Seconds()),
		Tenant:      toTenantResponse(t),
	}, nil
}
