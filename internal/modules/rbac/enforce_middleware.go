package rbac

import (
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/modules/rbac/domain"
	userDomain "github.com/divinecoid/one-backend/internal/modules/user/domain"
	"github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// pathsExemptFromEnforcement never require a permission check - either
// because they must stay reachable to reach a permission-granting screen
// at all (auth, rbac's own endpoints), or they're not really "modules"
// (activity-logs is read-only self-service, tenant/company are handled by
// their own middleware/role checks already).
var pathsExemptFromEnforcement = []string{
	"/api/v1/auth",
	"/api/v1/rbac",
	"/api/v1/platform",
	"/api/v1/tenant",
	"/api/v1/companies",
	// Self-service HR (own corrections, overtime, notifications...): every
	// handler only touches the caller's own records.
	"/api/v1/hr-self",
}

// Enforce is a generic, opt-in permission gate: mounted globally (see
// cmd/server/main.go), it runs AFTER modules/activitylog.Logger in the
// middleware chain conceptually but must itself run BEFORE the route
// handler, so unlike Logger it does its work before calling c.Next()
// rather than after.
//
// A user with no custom Role assigned (User.RoleID nil) is let through
// unconditionally - this is what makes RBAC opt-in rather than a
// flag-day lockout: every existing account keeps working exactly as it
// did before this feature existed, until a company admin explicitly
// assigns someone a custom role. Once assigned, access to a module with
// no permission row for that role is denied.
func Enforce(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		for _, exempt := range pathsExemptFromEnforcement {
			if strings.HasPrefix(path, exempt) {
				return c.Next()
			}
		}

		claims := middleware.CurrentUser(c)
		if claims == nil {
			return c.Next()
		}

		// The legacy admin role string is always a full bypass - never
		// lock out the primary account a company was created with.
		if strings.EqualFold(claims.Role, "admin") {
			return c.Next()
		}

		var user userDomain.User
		if err := db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
			// Can't resolve the user record - fail open rather than break
			// every request on a transient DB hiccup.
			return c.Next()
		}
		if user.RoleID == nil {
			// RBAC is opt-in, so an account with no custom role is otherwise unrestricted.
			// Salary data is the exception: a plain staff account never sees it.
			if staffBlockedFromPath(path, user.Role) {
				return errors.NewForbidden("Payroll data is only available to administrators, managers and roles granted the HR module")
			}
			return c.Next()
		}

		module := moduleFromPath(path)
		if module == "" {
			return c.Next()
		}

		var perm domain.Permission
		err := db.Where("role_id = ? AND module = ?", *user.RoleID, module).First(&perm).Error
		if err != nil {
			return errors.NewForbidden("Your role does not have access to this module")
		}

		isMutating := c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead && c.Method() != fiber.MethodOptions
		if isMutating && !perm.CanManage {
			return errors.NewForbidden("Your role does not have permission to modify this module")
		}
		if !isMutating && !perm.CanView {
			return errors.NewForbidden("Your role does not have permission to view this module")
		}

		return c.Next()
	}
}

// restrictedForPlainStaff are areas an account with no custom role and the
// plain staff role cannot reach, even though RBAC is otherwise opt-in.
// Other people's warnings, terminations and mutations are as sensitive as payroll.
var restrictedForPlainStaff = []string{"/api/v1/payroll", "/api/v1/hr-letters"}

// staffBlockedFromPath reports whether a legacy account (no custom role) with
// the given account role is kept out of path.
func staffBlockedFromPath(path string, role userDomain.UserRole) bool {
	if role == userDomain.RoleAdmin || role == userDomain.RoleManager {
		return false
	}
	for _, p := range restrictedForPlainStaff {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

// moduleAliases maps the first URL segment of routes that don't share their
// name with a permission module (see allModules) onto the module they belong
// to. Without this, a user with a custom role could never be granted access to
// e.g. /leaves or /products: no permission row can exist for those segments,
// so every request would be denied.
var moduleAliases = map[string]string{
	"leaves":                "hrm",
	"reimbursements":        "hrm",
	"payroll":               "hrm",
	"kpi":                   "hrm",
	"cooperative":           "hrm",
	"products":              "product",
	"reports":               "report",
	"projects":              "project",
	"project-tasks":         "project",
	"project-work-orders":   "project",
	"project-profitability": "project",
	"project-tickets":       "project",
	"devices":               "device",
	"tax":                   "finance",
	"hr-letters":            "hrm",
	"currencies":            "finance",
	"exchange-rates":        "finance",
	"commission":            "commission",
	"warehouse":             "warehouse",
}

func moduleFromPath(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	seg := ""
	for i, s := range segments {
		if s == "v1" && i+1 < len(segments) {
			seg = segments[i+1]
			break
		}
	}
	if seg == "" && len(segments) > 0 {
		seg = segments[0]
	}
	if alias, ok := moduleAliases[seg]; ok {
		return alias
	}
	return seg
}
