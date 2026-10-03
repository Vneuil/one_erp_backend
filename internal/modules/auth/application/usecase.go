package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/notify"
	authDomain "github.com/divinecoid/one-backend/internal/modules/auth/domain"
	companyDomain "github.com/divinecoid/one-backend/internal/modules/company/domain"
	tenantDomain "github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	userApp "github.com/divinecoid/one-backend/internal/modules/user/application"
	userDomain "github.com/divinecoid/one-backend/internal/modules/user/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/google/uuid"
)

// passwordResetTokenTTL is how long a forgot-password link stays redeemable.
const passwordResetTokenTTL = 30 * time.Minute

type AuthUseCase interface {
	Register(ctx context.Context, dto RegisterDTO) (*TokenResponseDTO, error)
	Login(ctx context.Context, dto LoginDTO) (*TokenResponseDTO, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*MeResponseDTO, error)
	ListCompanies(ctx context.Context, userID uuid.UUID) ([]CompanyMembershipDTO, error)
	SwitchCompany(ctx context.Context, userID uuid.UUID, companyID uuid.UUID) (*TokenResponseDTO, error)
	ForgotPassword(ctx context.Context, dto ForgotPasswordDTO) error
	ResetPassword(ctx context.Context, dto ResetPasswordDTO) error
}

type authUseCase struct {
	userRepo        userDomain.UserRepository
	companyRepo     companyDomain.CompanyRepository
	membershipRepo  tenantDomain.CompanyMembershipRepository
	resetTokenRepo  authDomain.PasswordResetTokenRepository
	notifier        notify.Notifier
	frontendBaseURL string
	jwtSecret       string
	jwtExpiry       time.Duration
}

func NewAuthUseCase(
	userRepo userDomain.UserRepository,
	companyRepo companyDomain.CompanyRepository,
	membershipRepo tenantDomain.CompanyMembershipRepository,
	resetTokenRepo authDomain.PasswordResetTokenRepository,
	notifier notify.Notifier,
	frontendBaseURL string,
	jwtSecret string,
	jwtExpiry time.Duration,
) AuthUseCase {
	return &authUseCase{
		userRepo:        userRepo,
		companyRepo:     companyRepo,
		membershipRepo:  membershipRepo,
		resetTokenRepo:  resetTokenRepo,
		notifier:        notifier,
		frontendBaseURL: frontendBaseURL,
		jwtSecret:       jwtSecret,
		jwtExpiry:       jwtExpiry,
	}
}

func (uc *authUseCase) Register(ctx context.Context, dto RegisterDTO) (*TokenResponseDTO, error) {
	email := strings.ToLower(strings.TrimSpace(dto.Email))
	name := strings.TrimSpace(dto.Name)
	companyName := strings.TrimSpace(dto.CompanyName)
	companyCode := strings.ToUpper(strings.TrimSpace(dto.CompanyCode))

	validationErrors := make(map[string]string)
	if email == "" {
		validationErrors["email"] = "Email is required"
	}
	if name == "" {
		validationErrors["name"] = "Name is required"
	}
	if len(dto.Password) < 6 {
		validationErrors["password"] = "Password must be at least 6 characters"
	}
	if len(validationErrors) > 0 {
		return nil, apperrors.NewValidation(validationErrors)
	}

	// Check existing user
	existingUser, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing email")
	}
	if existingUser != nil {
		return nil, apperrors.NewConflict("User with this email already exists")
	}

	var companyID *uuid.UUID

	// If company details provided, create company
	if companyName != "" && companyCode != "" {
		existingCompany, err := uc.companyRepo.GetByCode(ctx, companyCode)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check company code")
		}
		if existingCompany != nil {
			return nil, apperrors.NewConflict("Company with this code already exists")
		}

		newCompany := &companyDomain.Company{
			Code:     companyCode,
			Name:     companyName,
			Currency: "IDR",
			IsActive: true,
		}
		if err := uc.companyRepo.Create(ctx, newCompany); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to create company")
		}
		companyID = &newCompany.ID
	}

	// Hash password
	passwordHash, err := utils.HashPassword(dto.Password)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to hash password")
	}

	// First user registered with company is admin
	role := userDomain.RoleAdmin
	if companyID == nil {
		role = userDomain.RoleStaff
	}

	newUser := &userDomain.User{
		CompanyID:    companyID,
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		IsActive:     true,
	}

	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create user account")
	}

	if companyID != nil {
		membership := &tenantDomain.CompanyMembership{
			UserID:    newUser.ID,
			CompanyID: *companyID,
			Role:      string(role),
			Status:    tenantDomain.MembershipStatusActive,
		}
		if err := uc.membershipRepo.Create(ctx, membership); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to create company membership")
		}
	}

	// Generate JWT
	token, err := utils.GenerateJWT(
		newUser.ID,
		newUser.CompanyID,
		newUser.Email,
		string(newUser.Role),
		uc.jwtSecret,
		uc.jwtExpiry,
	)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate authentication token")
	}

	return &TokenResponseDTO{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(uc.jwtExpiry.Seconds()),
		User:        userApp.ToUserResponse(newUser),
		Company:     uc.loadCompanySummary(ctx, newUser.CompanyID),
	}, nil
}

func (uc *authUseCase) Login(ctx context.Context, dto LoginDTO) (*TokenResponseDTO, error) {
	email := strings.ToLower(strings.TrimSpace(dto.Email))

	if email == "" || dto.Password == "" {
		return nil, apperrors.NewBadRequest("Email and password are required")
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to find user")
	}
	if user == nil {
		return nil, apperrors.NewUnauthorized("Invalid email or password")
	}

	if !user.IsActive {
		return nil, apperrors.NewForbidden("User account is inactive")
	}

	if !utils.CheckPasswordHash(dto.Password, user.PasswordHash) {
		return nil, apperrors.NewUnauthorized("Invalid email or password")
	}

	token, err := utils.GenerateJWT(
		user.ID,
		user.CompanyID,
		user.Email,
		string(user.Role),
		uc.jwtSecret,
		uc.jwtExpiry,
	)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate authentication token")
	}

	return &TokenResponseDTO{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(uc.jwtExpiry.Seconds()),
		User:        userApp.ToUserResponse(user),
		Company:     uc.loadCompanySummary(ctx, user.CompanyID),
	}, nil
}

// loadCompanySummary fetches the company for a token response, mirroring the
// lookup GetMe already does - Register/Login/SwitchCompany all mint a JWT
// scoped to a company and must return that same company so the frontend
// never has to fall back to a stale or hardcoded value.
func (uc *authUseCase) loadCompanySummary(ctx context.Context, companyID *uuid.UUID) *CompanySummaryDTO {
	if companyID == nil {
		return nil
	}
	comp, err := uc.companyRepo.GetByID(ctx, *companyID)
	if err != nil || comp == nil {
		return nil
	}
	return &CompanySummaryDTO{
		ID:       comp.ID,
		Code:     comp.Code,
		Name:     comp.Name,
		Currency: comp.Currency,
	}
}

func (uc *authUseCase) GetMe(ctx context.Context, userID uuid.UUID) (*MeResponseDTO, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve user profile")
	}
	if user == nil {
		return nil, apperrors.NewNotFound("User not found")
	}

	response := &MeResponseDTO{
		User: userApp.ToUserResponse(user),
	}

	if user.CompanyID != nil {
		comp, err := uc.companyRepo.GetByID(ctx, *user.CompanyID)
		if err == nil && comp != nil {
			response.Company = &CompanySummaryDTO{
				ID:       comp.ID,
				Code:     comp.Code,
				Name:     comp.Name,
				Currency: comp.Currency,
			}
		}
	}

	return response, nil
}

func (uc *authUseCase) ListCompanies(ctx context.Context, userID uuid.UUID) ([]CompanyMembershipDTO, error) {
	memberships, err := uc.membershipRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list company memberships")
	}

	dtos := make([]CompanyMembershipDTO, 0, len(memberships))
	for _, m := range memberships {
		comp, err := uc.companyRepo.GetByID(ctx, m.CompanyID)
		if err != nil || comp == nil {
			continue
		}
		dtos = append(dtos, CompanyMembershipDTO{
			ID:   comp.ID,
			Code: comp.Code,
			Name: comp.Name,
			Role: m.Role,
		})
	}
	return dtos, nil
}

// SwitchCompany issues a new JWT with companyID as the active company claim.
// The company_id from the request is never trusted directly - membership is
// always verified server-side first ("jangan percaya company_id dari frontend").
func (uc *authUseCase) SwitchCompany(ctx context.Context, userID uuid.UUID, companyID uuid.UUID) (*TokenResponseDTO, error) {
	membership, err := uc.membershipRepo.GetByUserAndCompany(ctx, userID, companyID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to verify company membership")
	}
	if membership == nil || membership.Status != tenantDomain.MembershipStatusActive {
		return nil, apperrors.NewForbidden("You are not a member of this company")
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load user")
	}
	if user == nil {
		return nil, apperrors.NewNotFound("User not found")
	}

	activeCompanyID := companyID
	token, err := utils.GenerateJWT(
		user.ID,
		&activeCompanyID,
		user.Email,
		string(user.Role),
		uc.jwtSecret,
		uc.jwtExpiry,
	)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate authentication token")
	}

	return &TokenResponseDTO{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(uc.jwtExpiry.Seconds()),
		User:        userApp.ToUserResponse(user),
		Company:     uc.loadCompanySummary(ctx, &activeCompanyID),
	}, nil
}

// ForgotPassword always returns nil (a generic "if that email exists"
// response is what the handler sends either way) so the caller can never
// distinguish "no such account" from "email sent" - the classic user
// enumeration hole. Internal failures (DB, SMTP) are logged, not returned,
// for the same reason.
func (uc *authUseCase) ForgotPassword(ctx context.Context, dto ForgotPasswordDTO) error {
	email := strings.ToLower(strings.TrimSpace(dto.Email))
	if email == "" {
		return nil
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		slog.Error("forgot-password: failed to look up user", "error", err)
		return nil
	}
	if user == nil || !user.IsActive {
		// Unknown or deactivated account: say nothing that would let a
		// caller distinguish this from "email sent".
		return nil
	}

	rawToken, err := generateResetToken()
	if err != nil {
		slog.Error("forgot-password: failed to generate token", "error", err)
		return nil
	}

	now := time.Now()

	// Invalidate any still-valid tokens from a previous request so tokens
	// can't pile up and only the newest link works. This also stands in for
	// proper per-email rate limiting, which this codebase has no existing
	// pattern for - see NOTE below.
	if err := uc.resetTokenRepo.InvalidateActiveForUser(ctx, user.ID, now); err != nil {
		slog.Error("forgot-password: failed to invalidate previous tokens", "error", err)
		return nil
	}

	resetToken := &authDomain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashResetToken(rawToken),
		ExpiresAt: now.Add(passwordResetTokenTTL),
	}
	if err := uc.resetTokenRepo.Create(ctx, resetToken); err != nil {
		slog.Error("forgot-password: failed to store token", "error", err)
		return nil
	}

	// NOTE: no rate limiting on this endpoint yet (no existing simple
	// rate-limit middleware/pattern found in this codebase to reuse -
	// building one from scratch was judged out of scope here). The
	// single-active-token invalidation above prevents unbounded token
	// pile-up, but does not stop repeated email sends; add real per-email/IP
	// rate limiting as a follow-up.

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(uc.frontendBaseURL, "/"), rawToken)
	subject := "Reset your ONE ERP password"
	body := fmt.Sprintf(
		`<p>Hi %s,</p><p>We received a request to reset your ONE ERP password. Click the link below to choose a new one:</p><p><a href="%s">%s</a></p><p>This link expires in 30 minutes. If you didn't request this, you can safely ignore this email.</p>`,
		user.Name, resetLink, resetLink,
	)
	if err := uc.notifier.SendEmail(ctx, user.Email, user.Name, subject, body); err != nil {
		slog.Error("forgot-password: failed to send reset email", "error", err)
	}

	return nil
}

// ResetPassword redeems a single-use reset token. Every failure path
// (unknown token, expired, already used) returns the same generic
// unauthorized error so a caller cannot probe for which tokens exist.
func (uc *authUseCase) ResetPassword(ctx context.Context, dto ResetPasswordDTO) error {
	genericErr := apperrors.NewUnauthorized("Invalid or expired reset link")

	rawToken := strings.TrimSpace(dto.Token)
	if rawToken == "" {
		return genericErr
	}
	if len(dto.NewPassword) < 6 {
		return apperrors.NewValidation(map[string]string{"newPassword": "Password must be at least 6 characters"})
	}

	tokenHash := hashResetToken(rawToken)
	resetToken, err := uc.resetTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to verify reset token")
	}
	if resetToken == nil || !resetToken.IsValid(time.Now()) {
		return genericErr
	}

	user, err := uc.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load user")
	}
	if user == nil || !user.IsActive {
		return genericErr
	}

	passwordHash, err := utils.HashPassword(dto.NewPassword)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to hash password")
	}
	user.PasswordHash = passwordHash
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return apperrors.NewInternal(err, "Failed to update password")
	}

	now := time.Now()
	if err := uc.resetTokenRepo.MarkUsed(ctx, resetToken.ID, now); err != nil {
		slog.Error("reset-password: failed to mark token used", "error", err)
	}
	// Single-use, and also invalidate any other still-valid tokens for this
	// user (e.g. several forgot-password requests in a row).
	if err := uc.resetTokenRepo.InvalidateActiveForUser(ctx, user.ID, now); err != nil {
		slog.Error("reset-password: failed to invalidate other tokens", "error", err)
	}

	return nil
}

// generateResetToken returns a random, high-entropy raw token (32 bytes of
// crypto/rand, hex-encoded). Only its SHA-256 hash is ever persisted.
func generateResetToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// hashResetToken hashes a raw reset token with SHA-256. Unlike a user
// password, a reset token is already 256 bits of random entropy generated
// server-side (not a low-entropy secret a human chose), so a fast
// general-purpose hash is the appropriate comparison here - the same
// reasoning that makes bcrypt/argon2 unnecessary for API keys or session
// tokens, only for human-chosen passwords.
func hashResetToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
