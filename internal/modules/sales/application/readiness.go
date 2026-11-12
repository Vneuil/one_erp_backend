package application

import (
	"strings"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

var (
	packagingStates  = []string{"pending", "available", "partial", "unavailable"}
	coaStates        = []string{"not_required", "pending", "ready"}
	productionStates = []string{"not_started", "in_production", "completed"}
)

// normalizeState lowercases v and checks it is one of allowed; empty falls back to def.
func normalizeState(field, v, def string, allowed []string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return def, nil
	}
	for _, a := range allowed {
		if v == a {
			return v, nil
		}
	}
	return "", apperrors.NewBadRequest(field + " must be one of: " + strings.Join(allowed, ", "))
}
