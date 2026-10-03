package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenTypeUser and TokenTypeConsultant distinguish a company-scoped user
// token (the default, empty Type for backward compatibility with tokens
// issued before this field existed) from an internal consultant token,
// which is never tied to a single company.
const (
	TokenTypeUser       = "user"
	TokenTypeConsultant = "consultant"
)

type JWTClaims struct {
	UserID    uuid.UUID  `json:"userId"`
	CompanyID *uuid.UUID `json:"companyId,omitempty"`
	// TenantID scopes the session to one business unit (branch/subsidiary)
	// within the Company's shared database - see modules/workspace. It is
	// optional: a nil TenantID means "no tenant filter", which keeps
	// existing single-tenant companies working exactly as before. Set via
	// SwitchTenant once a company creates more than one Tenant.
	TenantID *uuid.UUID `json:"tenantId,omitempty"`
	Email    string     `json:"email"`
	Role     string     `json:"role"`
	Type     string     `json:"type,omitempty"`
	jwt.RegisteredClaims
}

// IsConsultant reports whether these claims belong to an internal
// consultant token rather than a company user token.
func (c *JWTClaims) IsConsultant() bool {
	return c.Type == TokenTypeConsultant
}

// GenerateJWT creates a new signed JWT token with claims
func GenerateJWT(userID uuid.UUID, companyID *uuid.UUID, email, role, secret string, duration time.Duration) (string, error) {
	return GenerateJWTWithType(userID, companyID, email, role, TokenTypeUser, secret, duration)
}

// GenerateJWTWithType is like GenerateJWT but lets the caller set the token
// Type (e.g. TokenTypeConsultant for internal consultant accounts, which
// have no CompanyID).
func GenerateJWTWithType(userID uuid.UUID, companyID *uuid.UUID, email, role, tokenType, secret string, duration time.Duration) (string, error) {
	return GenerateJWTWithTenant(userID, companyID, nil, email, role, tokenType, secret, duration)
}

// GenerateJWTWithTenant is GenerateJWTWithType plus an optional TenantID,
// used by modules/workspace.SwitchTenant to reissue a token scoped to a
// specific business unit within the caller's company.
func GenerateJWTWithTenant(userID uuid.UUID, companyID *uuid.UUID, tenantID *uuid.UUID, email, role, tokenType, secret string, duration time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := JWTClaims{
		UserID:    userID,
		CompanyID: companyID,
		TenantID:  tenantID,
		Email:     email,
		Role:      role,
		Type:      tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseJWT validates a token string against secret and returns the claims
func ParseJWT(tokenStr, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid or expired token")
}
