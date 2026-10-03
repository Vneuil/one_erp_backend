package activitylog

import (
	"context"
	"log/slog"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/activitylog/domain"
	"github.com/divinecoid/one-backend/internal/modules/activitylog/infrastructure"
	"github.com/gofiber/fiber/v2"
)

// pathsExcludedFromLogging are endpoints that are either not tenant-scoped
// (auth, activity-logs itself) or too noisy/irrelevant to audit.
var pathsExcludedFromLogging = []string{
	"/api/v1/auth",
	"/api/v1/activity-logs",
}

// Logger is a generic audit-trail middleware: after any mutating request
// (POST/PUT/PATCH/DELETE) that reaches a tenant-scoped handler and
// succeeds, it writes one ActivityLog row. Mounted globally, once, before
// any module registers its routes (see cmd/server/main.go) - Fiber calls
// middleware registered with app.Use() before any route-specific
// middleware a module later attaches to its own group, and because this
// calls c.Next() first, by the time execution resumes here the inner
// Protected/TenantContext middleware (and the actual handler) have already
// run and populated c.Locals, so CurrentUser(c) and TenantDB(c) are
// available without this middleware needing to parse the JWT itself.
//
// Logging is best-effort: a failure to write the log never fails the
// request, and is only logged at debug level to avoid noise.
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()

		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return err
		}

		path := c.Path()
		for _, excluded := range pathsExcludedFromLogging {
			if strings.HasPrefix(path, excluded) {
				return err
			}
		}

		status := c.Response().StatusCode()
		if status >= 400 {
			return err
		}

		tenantDB := middleware.TenantDB(c)
		claims := middleware.CurrentUser(c)
		if tenantDB == nil || claims == nil {
			return err
		}

		entry := &domain.ActivityLog{
			UserID:     claims.UserID,
			UserEmail:  claims.Email,
			Method:     method,
			Path:       path,
			Module:     moduleFromPath(path),
			StatusCode: status,
			IPAddress:  c.IP(),
		}

		ctx := tenantctx.WithTenantID(context.Background(), claims.TenantID)
		repo := infrastructure.NewActivityLogRepository(tenantDB)
		if logErr := repo.Create(ctx, entry); logErr != nil {
			slog.Debug("failed to write activity log", "error", logErr, "path", path)
		}

		return err
	}
}

// moduleFromPath extracts the module segment from a versioned API path,
// e.g. "/api/v1/sales/orders" -> "sales".
func moduleFromPath(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segments {
		if seg == "v1" && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	if len(segments) > 0 {
		return segments[0]
	}
	return ""
}
