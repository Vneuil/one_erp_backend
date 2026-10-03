// Package systemlogs exposes a read-only "what just broke" view over
// foundation/logstore's in-memory ring buffer of recent server errors, so
// an admin can debug a 500 from inside the app instead of needing shell
// access to the server. See logstore's doc comment for how entries are
// captured and scoped per company.
package systemlogs

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"

	"github.com/divinecoid/one-backend/internal/foundation/logstore"
	"github.com/gofiber/fiber/v2"
)

// List handles GET /system-logs?limit=100 - admin-only, returns the
// caller's own company's most recent captured server errors, newest first.
func List(c *fiber.Ctx) error {
	claims := middleware.CurrentUser(c)
	if claims == nil {
		return apperrors.NewUnauthorized("Authentication required")
	}
	if claims.CompanyID == nil {
		return apperrors.NewBadRequest("No active company")
	}

	limit := c.QueryInt("limit", 100)
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	entries := logstore.List(*claims.CompanyID, limit)
	return response.OK(c, "Recent server errors", fiber.Map{
		"entries": entries,
		"count":   len(entries),
	})
}

func NewModule(router fiber.Router, jwtSecret string) {
	protected := middleware.Protected(jwtSecret)
	adminOnly := middleware.RequireRoles("admin")
	router.Get("/system-logs", protected, adminOnly, List)

	slog.Info("system logs module initialized")
}
