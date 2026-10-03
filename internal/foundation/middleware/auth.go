package middleware

import (
	"log/slog"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/license"
	tenant "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const ContextKeyUser = "currentUser"
const ContextKeyTenantDB = "tenantDB"

// parseAndSetClaims validates the Bearer JWT and stores its claims in
// context, without advancing Fiber's handler chain itself. Both Protected
// and ProtectedConsultant call this and then call c.Next() exactly once
// each - calling a fiber.Handler that itself calls c.Next() as a plain
// nested Go function call (rather than letting Fiber dispatch it) double-
// advances Fiber's internal handler cursor and corrupts routing for the
// rest of the chain, so this logic must not call c.Next() on its own.
func parseAndSetClaims(c *fiber.Ctx, jwtSecret string) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return errors.NewUnauthorized("Missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return errors.NewUnauthorized("Invalid Authorization header format. Expected Bearer <token>")
	}

	claims, err := utils.ParseJWT(parts[1], jwtSecret)
	if err != nil {
		return errors.NewUnauthorized("Invalid or expired authentication token")
	}

	c.Locals(ContextKeyUser, claims)
	return nil
}

// Protected verifies the Bearer JWT token and stores the claims in context
func Protected(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := parseAndSetClaims(c, jwtSecret); err != nil {
			return err
		}
		return c.Next()
	}
}

// ProtectedConsultant is like Protected, but additionally requires the JWT
// to be an internal consultant token (see utils.TokenTypeConsultant) rather
// than a company user token - so a company admin's token can never be used
// to call consultant-only platform endpoints, and vice versa.
func ProtectedConsultant(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := parseAndSetClaims(c, jwtSecret); err != nil {
			return err
		}
		claims := CurrentUser(c)
		if claims == nil || !claims.IsConsultant() {
			return errors.NewForbidden("This action requires a consultant account")
		}
		return c.Next()
	}
}

// RequireRoles restricts access to users with one of the required roles
func RequireRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(ContextKeyUser).(*utils.JWTClaims)
		if !ok || claims == nil {
			return errors.NewUnauthorized("Authentication required")
		}

		for _, role := range allowedRoles {
			if strings.EqualFold(claims.Role, role) {
				return c.Next()
			}
		}

		return errors.NewForbidden("You do not have permission to perform this action")
	}
}

// CurrentUser retrieves JWT claims from Fiber context
func CurrentUser(c *fiber.Ctx) *utils.JWTClaims {
	claims, ok := c.Locals(ContextKeyUser).(*utils.JWTClaims)
	if !ok {
		return nil
	}
	return claims
}

// CurrentTenantID returns the active Tenant (business unit) ID from the
// caller's JWT, or nil if the session isn't scoped to one - which is the
// normal state for a company that has never created more than its default
// Tenant. Repositories that filter by tenant should treat a nil result as
// "no filter" so single-tenant companies are unaffected.
func CurrentTenantID(c *fiber.Ctx) *uuid.UUID {
	claims := CurrentUser(c)
	if claims == nil {
		return nil
	}
	return claims.TenantID
}

// TenantContext resolves the caller's tenant (per-company) database
// connection and stores it in context. Must run after Protected, since it
// reads CompanyID from the already-parsed JWT claims. If the caller has no
// active company (claims.CompanyID is nil) it skips silently rather than
// erroring - some routes run before any tenant exists (e.g. company
// creation). Handlers that require a tenant DB should check TenantDB(c) for
// nil and return their own "No active company" error.
func TenantContext(manager *tenant.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := CurrentUser(c)
		if claims == nil || claims.CompanyID == nil {
			return c.Next()
		}

		db, err := manager.GetDB(c.UserContext(), *claims.CompanyID)
		if err != nil {
			slog.Error("failed to resolve tenant database", "companyId", *claims.CompanyID, "error", err)
			return c.Next()
		}

		c.Locals(ContextKeyTenantDB, db)
		return c.Next()
	}
}

// exemptFromLicenseGate lists path prefixes that must stay reachable even
// after a company's license has lapsed - otherwise an expired customer
// could never reach the page that lets them pay to fix it, and platform/
// auth/tenant/company are central-db control-plane routes anyway, not
// tenant business data.
var exemptFromLicenseGate = []string{
	"/api/v1/platform",
	"/api/v1/auth",
	"/api/v1/tenant",
	"/api/v1/companies",
	// /checkout is the public pricing-page purchase flow - a prospective
	// customer paying for a plan has no account/license yet.
	"/api/v1/checkout",
}

// RequireActiveUser blocks any request whose JWT belongs to a user who has
// since been deactivated (Users.IsActive = false). JWTs are otherwise valid
// until natural expiry regardless of account state - Login already checks
// IsActive, but without this, a deactivated user keeps full API access with
// their existing token for the rest of its lifetime. Mounted globally next
// to RequireActiveLicense so it applies before any module-specific route
// runs, without touching every module's individual Protected() call site.
//
// Consultant tokens (claims.IsConsultant()) are skipped - they belong to
// internal platform staff tracked in a different table, not company Users.
// Fails open on a DB error, matching RequireActiveLicense's philosophy: a
// transient database hiccup must not lock out the whole platform.
func RequireActiveUser(db *gorm.DB, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		for _, prefix := range exemptFromLicenseGate {
			if strings.HasPrefix(path, prefix) {
				return c.Next()
			}
		}

		if err := parseAndSetClaims(c, jwtSecret); err != nil {
			return c.Next()
		}
		claims := CurrentUser(c)
		if claims == nil || claims.IsConsultant() {
			return c.Next()
		}

		var isActive bool
		err := db.Table("users").Select("is_active").Where("id = ?", claims.UserID).Scan(&isActive).Error
		if err != nil {
			slog.Error("failed to check user active status", "userId", claims.UserID, "error", err)
			return c.Next()
		}
		if !isActive {
			return errors.NewUnauthorized("This account has been deactivated")
		}
		return c.Next()
	}
}

// RequireActiveLicense blocks tenant business-data requests once a
// company's subscription has actually lapsed (trial past TrialEndsAt,
// paid period past EndsAt, or explicitly marked expired) - see
// foundation/license.IsActive for the exact rules.
//
// Mounted globally (once, before any module registers routes) rather than
// per-module like Protected/TenantContext, so it must parse the JWT itself
// instead of relying on Protected having already run first - a global
// app-level Use() executes before any route-specific middleware a module
// later attaches to its own group, so CurrentUser(c) would always be nil
// at this point otherwise. A missing/invalid token here is not an error:
// it just means there is nothing to gate, and the actual Protected
// middleware further down the chain will reject the request properly.
// Fails open on a DB error so a transient database hiccup can't lock out
// an entire company.
func RequireActiveLicense(db *gorm.DB, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		for _, prefix := range exemptFromLicenseGate {
			if strings.HasPrefix(path, prefix) {
				return c.Next()
			}
		}

		if err := parseAndSetClaims(c, jwtSecret); err != nil {
			return c.Next()
		}
		claims := CurrentUser(c)
		if claims == nil || claims.CompanyID == nil {
			return c.Next()
		}

		active, err := license.IsActive(c.UserContext(), db, *claims.CompanyID)
		if err != nil {
			slog.Error("failed to check license status", "companyId", *claims.CompanyID, "error", err)
			return c.Next()
		}
		if !active {
			return errors.NewPaymentRequired("Your ONE ERP license has expired. Please renew your subscription to continue.")
		}
		return c.Next()
	}
}

// TenantDB retrieves the resolved tenant database connection from context.
// Returns nil if TenantContext did not run or the caller has no active company.
func TenantDB(c *fiber.Ctx) *gorm.DB {
	db, ok := c.Locals(ContextKeyTenantDB).(*gorm.DB)
	if !ok {
		return nil
	}
	return db
}
