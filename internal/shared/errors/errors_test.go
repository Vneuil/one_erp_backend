package errors_test

import (
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

func TestAppErrorConstructors(t *testing.T) {
	t.Run("BadRequest", func(t *testing.T) {
		err := apperrors.NewBadRequest("invalid input", map[string]string{"field": "required"})
		if err.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.StatusCode)
		}
		if err.ErrorCode != "BAD_REQUEST" {
			t.Errorf("expected code BAD_REQUEST, got %s", err.ErrorCode)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		err := apperrors.NewNotFound("item not found")
		if err.StatusCode != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, err.StatusCode)
		}
		if err.ErrorCode != "NOT_FOUND" {
			t.Errorf("expected code NOT_FOUND, got %s", err.ErrorCode)
		}
	})

	t.Run("Conflict", func(t *testing.T) {
		err := apperrors.NewConflict("resource already exists")
		if err.StatusCode != http.StatusConflict {
			t.Errorf("expected status %d, got %d", http.StatusConflict, err.StatusCode)
		}
	})

	t.Run("Internal", func(t *testing.T) {
		raw := errors.New("db disk failure")
		err := apperrors.NewInternal(raw, "system error")
		if err.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, err.StatusCode)
		}
		if !errors.Is(err, raw) {
			t.Errorf("expected errors.Is to match raw error")
		}
	})
}
