package middleware

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantDBResolver hands out a company's tenant database (tenant.Manager does).
type TenantDBResolver interface {
	GetDB(ctx context.Context, companyID uuid.UUID) (*gorm.DB, error)
}

// onboardingRetryAfter is how long to wait before retrying a member whose
// onboarding failed (for example a tenant schema that is not migrated yet), so
// a persistent failure does not repeat on every request.
const onboardingRetryAfter = 5 * time.Minute

// EnsureUserOnboarded runs the registered onboarding hooks (see
// tenant.RegisterUserOnboarding) the first time each member makes an
// authenticated request for their company - which is what gives every member,
// including the company's creator, an HR employee record without manual setup.
//
// It runs before the route handler, so the record exists by the time that same
// request reads it. It never blocks a request: failures are logged and retried
// later. Successes are remembered per process; the hooks are idempotent, so a
// restart simply re-checks once.
//
//   - resolver gives the company's tenant database;
//   - nameOf looks up the member's display name (control-plane data);
//   - run executes the hooks (tenant.RunUserOnboarding in production).
func EnsureUserOnboarded(
	enabled bool,
	resolver TenantDBResolver,
	nameOf func(userID uuid.UUID) string,
	run func(ctx context.Context, tenantDB *gorm.DB, u tenant.OnboardingUser) error,
) fiber.Handler {
	type key struct{ user, company uuid.UUID }
	var (
		done   sync.Map // key -> struct{}
		failed sync.Map // key -> time.Time of the last failure
		locks  sync.Map // key -> *sync.Mutex
	)

	onboard := func(ctx context.Context, k key, claimsTenant *uuid.UUID, email, role string) {
		mu, _ := locks.LoadOrStore(k, &sync.Mutex{})
		mu.(*sync.Mutex).Lock()
		defer mu.(*sync.Mutex).Unlock()
		if _, ok := done.Load(k); ok {
			return
		}
		if at, ok := failed.Load(k); ok && time.Since(at.(time.Time)) < onboardingRetryAfter {
			return
		}
		tenantDB, err := resolver.GetDB(ctx, k.company)
		if err != nil || tenantDB == nil {
			return // no tenant database yet; try again on a later request
		}
		err = run(ctx, tenantDB, tenant.OnboardingUser{
			UserID: k.user, CompanyID: k.company, TenantID: claimsTenant,
			Name: nameOf(k.user), Email: email, Role: role,
		})
		if err != nil {
			slog.Warn("member onboarding failed", "userId", k.user, "companyId", k.company, "error", err)
			failed.Store(k, time.Now())
			return
		}
		failed.Delete(k)
		done.Store(k, struct{}{})
	}

	return func(c *fiber.Ctx) error {
		if !enabled {
			return c.Next()
		}
		path := c.Path()
		for _, prefix := range exemptFromLicenseGate {
			if strings.HasPrefix(path, prefix) {
				return c.Next()
			}
		}
		claims := CurrentUser(c)
		if claims == nil || claims.IsConsultant() || claims.CompanyID == nil || strings.TrimSpace(claims.Email) == "" {
			return c.Next()
		}
		k := key{user: claims.UserID, company: *claims.CompanyID}
		if _, ok := done.Load(k); !ok {
			onboard(c.UserContext(), k, claims.TenantID, claims.Email, claims.Role)
		}
		return c.Next()
	}
}
