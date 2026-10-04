package rbac

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/modules/rbac/domain"
	userDomain "github.com/divinecoid/one-backend/internal/modules/user/domain"
	"github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// approvalRule marks one endpoint as an approval-type action - approving,
// rejecting, paying, posting or finalizing a document. Those actions need the
// stronger CanApprove grant (or a manager/admin account), on top of the
// module-level CanManage check that Enforce already applies to every write.
type approvalRule struct {
	method string
	path   *regexp.Regexp
	module string
}

func rule(method, pattern, module string) approvalRule {
	return approvalRule{method: method, path: regexp.MustCompile("^" + pattern + "$"), module: module}
}

const idSeg = `[^/]+`

// approvalRules is the single, auditable list of endpoints that need approval
// rights. Keeping it here (rather than sprinkled across each module's routes)
// means a new approval endpoint is one line, and the whole list can be reviewed
// in one place. The generic Approval Center (/approval/requests/...) is not
// listed: it already matches the actor against each workflow step's role.
var approvalRules = []approvalRule{
	rule(http.MethodPost, `/api/v1/leaves/`+idSeg+`/(approve|reject)`, "hrm"),
	rule(http.MethodPost, `/api/v1/reimbursements/`+idSeg+`/(approve|reject|mark-paid)`, "hrm"),
	rule(http.MethodPost, `/api/v1/hrm/(corrections|shift-changes|overtime|cash-advances)/`+idSeg+`/(approve|reject)`, "hrm"),
	rule(http.MethodPost, `/api/v1/hr-letters/`+idSeg+`/(issue|cancel|apply)`, "hrm"),
	rule(http.MethodPost, `/api/v1/payroll/entries/calculate`, "hrm"),
	rule(http.MethodPut, `/api/v1/payroll/entries/`+idSeg+`/status`, "hrm"),
	rule(http.MethodPut, `/api/v1/kpi/`+idSeg+`/status`, "hrm"),
	rule(http.MethodPost, `/api/v1/commission/records/`+idSeg+`/(approve|mark-paid)`, "commission"),
	rule(http.MethodPost, `/api/v1/sales/orders/`+idSeg+`/(approve|reject)`, "sales"),
	rule(http.MethodPost, `/api/v1/procurement/(requests|orders)/`+idSeg+`/(approve|reject)`, "procurement"),
	rule(http.MethodPost, `/api/v1/inventory/batches/`+idSeg+`/write-off`, "inventory"),
	rule(http.MethodPost, `/api/v1/inventory/opname/`+idSeg+`/finalize`, "inventory"),
	rule(http.MethodPost, `/api/v1/finance/journal-entries/`+idSeg+`/(post|reverse)`, "finance"),
	rule(http.MethodPost, `/api/v1/finance/petty-cash/`+idSeg+`/transactions/`+idSeg+`/(approve|reject)`, "finance"),
	rule(http.MethodPost, `/api/v1/finance/capital/drawings`, "finance"),
	rule(http.MethodPost, `/api/v1/project-work-orders/`+idSeg+`/issue`, "project"),
	rule(http.MethodPost, `/api/v1/pos/transactions/`+idSeg+`/(void|refund)`, "pos"),
	rule(http.MethodPost, `/api/v1/manufacturing/production-orders/`+idSeg+`/release`, "manufacturing"),
	rule(http.MethodPost, `/api/v1/recruitment/candidates/`+idSeg+`/hire`, "recruitment"),
}

// approvalModuleFor returns the permission module an approval-type request
// belongs to, and whether the request is one at all.
func approvalModuleFor(method, path string) (string, bool) {
	path = strings.TrimRight(path, "/")
	for _, r := range approvalRules {
		if r.method == method && r.path.MatchString(path) {
			return r.module, true
		}
	}
	return "", false
}

// mayApprove is the authorization decision, kept free of HTTP and database
// concerns so it can be tested exhaustively.
//
//   - a JWT admin can always approve (the primary account is never locked out);
//   - a user with a custom Role can approve only where that role's permission for
//     the module has CanApprove;
//   - a user without a custom Role falls back to the legacy account role:
//     admin and manager may approve, staff may not.
func mayApprove(jwtRole string, user *userDomain.User, perm *domain.Permission) bool {
	if strings.EqualFold(jwtRole, string(userDomain.RoleAdmin)) {
		return true
	}
	if user == nil {
		return false
	}
	if user.RoleID != nil {
		return perm != nil && perm.CanApprove
	}
	return user.Role == userDomain.RoleAdmin || user.Role == userDomain.RoleManager
}

// ApprovalGate rejects approval-type requests from users who lack approval
// rights. Unlike Enforce it fails CLOSED: if the user cannot be loaded, the
// approval is refused rather than allowed.
func ApprovalGate(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		module, ok := approvalModuleFor(c.Method(), c.Path())
		if !ok {
			return c.Next()
		}
		claims := middleware.CurrentUser(c)
		if claims == nil {
			return c.Next() // not authenticated yet; the route's own auth rejects it
		}
		if strings.EqualFold(claims.Role, string(userDomain.RoleAdmin)) {
			return c.Next()
		}

		var user userDomain.User
		if err := db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewForbidden("Your account could not be found; you cannot approve documents")
			}
			return errors.NewInternal(err, "Unable to verify your approval permission")
		}

		var perm *domain.Permission
		if user.RoleID != nil {
			var p domain.Permission
			switch err := db.Where("role_id = ? AND module = ?", *user.RoleID, module).First(&p).Error; err {
			case nil:
				perm = &p
			case gorm.ErrRecordNotFound:
			default:
				return errors.NewInternal(err, "Unable to verify your approval permission")
			}
		}

		if !mayApprove(claims.Role, &user, perm) {
			return errors.NewForbidden("You do not have permission to approve, reject or pay documents in this module")
		}
		return c.Next()
	}
}
