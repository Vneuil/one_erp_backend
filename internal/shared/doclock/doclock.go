// Package doclock enforces document immutability and SuperAdmin override rules.
package doclock

import (
	"strings"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

// IsDocumentFinal reports whether the document is in a finalized state.
func IsDocumentFinal(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return s == "final" || s == "approved" || s == "completed" || s == "posted"
}

// CheckModification verifies if a document at a given status can be modified.
// For final/approved documents, regular users (including supervisors) are blocked (403),
// while SuperAdmin (Owner) is granted override privileges.
func CheckModification(status string, isSuperAdmin bool) error {
	if IsDocumentFinal(status) {
		if !isSuperAdmin {
			return apperrors.NewForbidden("Dokumen telah final dan dikunci")
		}
	}
	return nil
}

