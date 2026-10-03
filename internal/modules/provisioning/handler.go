package provisioning

import (
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/response"
	companyDomain "github.com/divinecoid/one-backend/internal/modules/company/domain"
	tenantApp "github.com/divinecoid/one-backend/internal/modules/tenant/application"
	tenantDomain "github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	userDomain "github.com/divinecoid/one-backend/internal/modules/user/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler exposes internal, service-to-service endpoints that let the
// standalone abc-backend (the Associate Business Consultant program, moved
// out of this repo) still provision real ONE ERP companies and tenants -
// that machinery (tenant DB creation) only exists here and cannot be moved.
// Every route here is guarded by requireInternalToken, never by a normal
// user JWT.
type Handler struct {
	companyRepo    companyDomain.CompanyRepository
	userRepo       userDomain.UserRepository
	membershipRepo tenantDomain.CompanyMembershipRepository
	tenantUseCase  tenantApp.TenantUseCase
	jwtSecret      string
	jwtExpiry      time.Duration
}

func NewHandler(
	companyRepo companyDomain.CompanyRepository,
	userRepo userDomain.UserRepository,
	membershipRepo tenantDomain.CompanyMembershipRepository,
	tenantUseCase tenantApp.TenantUseCase,
	jwtSecret string,
	jwtExpiry time.Duration,
) *Handler {
	return &Handler{
		companyRepo: companyRepo, userRepo: userRepo, membershipRepo: membershipRepo,
		tenantUseCase: tenantUseCase, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry,
	}
}

type createCompanyRequest struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone,omitempty"`
}

func (h *Handler) CreateCompany(c *fiber.Ctx) error {
	var req createCompanyRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	validationErrors := make(map[string]string)
	if req.Code == "" {
		validationErrors["code"] = "Company code is required"
	}
	if req.Name == "" {
		validationErrors["name"] = "Company name is required"
	}
	if req.Email == "" {
		validationErrors["email"] = "Company email is required"
	}
	if len(validationErrors) > 0 {
		return apperrors.NewValidation(validationErrors)
	}

	existing, err := h.companyRepo.GetByCode(c.UserContext(), req.Code)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check existing company code")
	}
	if existing != nil {
		return apperrors.NewConflict("Company with this code already exists")
	}

	company := &companyDomain.Company{
		Code: req.Code, Name: req.Name, Email: req.Email, Phone: req.Phone,
		Currency: "IDR", IsActive: true,
	}
	if err := h.companyRepo.Create(c.UserContext(), company); err != nil {
		return apperrors.NewInternal(err, "Failed to create company")
	}

	return response.Created(c, "Company created", fiber.Map{
		"id": company.ID, "code": company.Code, "name": company.Name,
		"email": company.Email, "phone": company.Phone, "currency": company.Currency,
	})
}

type activateCompanyRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
}

func (h *Handler) ActivateCompany(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Params("companyId"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid companyId format")
	}

	var req activateCompanyRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	if req.Name == "" || req.Email == "" || req.PasswordHash == "" {
		return apperrors.NewValidation(map[string]string{"_": "name, email and passwordHash are required"})
	}

	existingUser, err := h.userRepo.GetByEmail(c.UserContext(), req.Email)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check existing user")
	}
	if existingUser != nil {
		return apperrors.NewConflict("A user with this email already exists")
	}

	user := &userDomain.User{
		CompanyID: &companyID, Name: req.Name, Email: req.Email,
		PasswordHash: req.PasswordHash, Role: userDomain.RoleAdmin, IsActive: true,
	}
	if err := h.userRepo.Create(c.UserContext(), user); err != nil {
		return apperrors.NewInternal(err, "Failed to create user")
	}

	membership := &tenantDomain.CompanyMembership{
		UserID: user.ID, CompanyID: companyID, Role: "admin", Status: tenantDomain.MembershipStatusActive,
	}
	if err := h.membershipRepo.Create(c.UserContext(), membership); err != nil {
		return apperrors.NewInternal(err, "Failed to create company membership")
	}

	if _, err := h.tenantUseCase.GetOrCreateTenantDatabase(c.UserContext(), companyID); err != nil {
		return apperrors.NewInternal(err, "Failed to provision tenant database")
	}

	accessToken, err := utils.GenerateJWT(user.ID, &companyID, user.Email, string(user.Role), h.jwtSecret, h.jwtExpiry)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to generate access token")
	}

	company, err := h.companyRepo.GetByID(c.UserContext(), companyID)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load company")
	}
	var companySummary fiber.Map
	if company != nil {
		companySummary = fiber.Map{"id": company.ID, "code": company.Code, "name": company.Name, "currency": company.Currency}
	}

	return response.OK(c, "Account activated", fiber.Map{
		"accessToken": accessToken, "tokenType": "Bearer", "expiresIn": int64(h.jwtExpiry.Seconds()),
		"user": fiber.Map{
			"id": user.ID, "companyId": companyID, "name": user.Name, "email": user.Email,
			"role": string(user.Role), "isActive": user.IsActive,
		},
		"company": companySummary,
	})
}
