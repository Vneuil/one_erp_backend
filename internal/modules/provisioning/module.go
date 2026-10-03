package provisioning

import (
	"log/slog"
	"time"

	companyDomain "github.com/divinecoid/one-backend/internal/modules/company/domain"
	tenantApp "github.com/divinecoid/one-backend/internal/modules/tenant/application"
	tenantDomain "github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	userDomain "github.com/divinecoid/one-backend/internal/modules/user/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

type Module struct {
	Handler *Handler
}

// requireInternalToken guards every route in this module against anything
// but the abc-backend service itself, mirroring the exact fail-closed
// pattern the old platform module used for the Xendit webhook: if no token
// is configured, refuse the route entirely rather than silently accepting
// unauthenticated calls.
func requireInternalToken(expectedToken string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if expectedToken == "" {
			return apperrors.NewServiceUnavailable("Internal provisioning is not configured (INTERNAL_PROVISIONING_TOKEN missing)")
		}
		if c.Get("X-Internal-Token") != expectedToken {
			return apperrors.NewUnauthorized("Invalid internal token")
		}
		return c.Next()
	}
}

// NewModule wires the internal provisioning endpoints that let the
// standalone abc-backend service (the Associate Business Consultant
// program) create companies and activate their tenant database here, since
// that machinery cannot leave this repo. router should be a group mounted
// OUTSIDE /api/v1 (e.g. on the raw *fiber.App) so it never passes through
// the license/RBAC/activity-log middlewares mounted on that group, which
// expect a normal user JWT this service-to-service call will never have.
func NewModule(
	router fiber.Router,
	companyRepo companyDomain.CompanyRepository,
	userRepo userDomain.UserRepository,
	membershipRepo tenantDomain.CompanyMembershipRepository,
	tenantUseCase tenantApp.TenantUseCase,
	jwtSecret string,
	jwtExpiry time.Duration,
	internalToken string,
) *Module {
	handler := NewHandler(companyRepo, userRepo, membershipRepo, tenantUseCase, jwtSecret, jwtExpiry)

	group := router.Group("/provisioning", requireInternalToken(internalToken))
	group.Post("/companies", handler.CreateCompany)
	group.Post("/companies/:companyId/activate", handler.ActivateCompany)

	slog.Info("provisioning module initialized")

	return &Module{Handler: handler}
}
