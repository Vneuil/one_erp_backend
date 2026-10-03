// Package sod enforces segregation of duties: the person who raised a document
// must not be the one who approves (or pays) it.
package sod

import (
	"context"
	"os"
	"strings"

	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

// AllowSelfApprovalEnv, when set to "true", switches the check off. It exists
// for very small companies where one person legitimately raises and approves
// everything; it defaults to off (self-approval is blocked).
const AllowSelfApprovalEnv = "ALLOW_SELF_APPROVAL"

// SamePerson reports whether two email addresses identify the same person
// (case- and whitespace-insensitive). Empty values never match, so documents
// with no recorded owner are not blocked.
func SamePerson(a, b string) bool {
	a, b = strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	return a != "" && a == b
}

// ForbidSelfApproval returns a 403 when the caller in ctx is the document's
// owner. If the context has no caller (background work) or the owner is
// unknown, it allows the action.
func ForbidSelfApproval(ctx context.Context, ownerEmail string) error {
	if os.Getenv(AllowSelfApprovalEnv) == "true" {
		return nil
	}
	if SamePerson(actor.EmailFrom(ctx), ownerEmail) {
		return apperrors.NewForbidden("You cannot approve or pay your own request; ask another approver")
	}
	return nil
}
