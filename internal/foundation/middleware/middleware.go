package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/logstore"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// recordServerError captures a 500-class failure into logstore so it shows
// up on the in-app "Backend Logs" page, scoped to whichever company's
// request triggered it (see logstore.List's tenant-scoping doc comment).
// Best-effort: c may not have claims set yet (e.g. a panic before Protected
// ran), in which case the entry is simply not company-attributable and
// logstore.List will never surface it to anyone - it still went to slog.
func recordServerError(c *fiber.Ctx, status int, message string) {
	claims := CurrentUser(c)
	entry := logstore.Entry{
		Status:    status,
		Method:    c.Method(),
		Path:      c.Path(),
		Message:   message,
		RequestID: c.GetRespHeader("X-Request-ID"),
	}
	if claims != nil {
		entry.CompanyID = claims.CompanyID
		entry.UserEmail = claims.Email
	}
	logstore.Record(entry)
}

// RequestID returns the request ID middleware
func RequestID() fiber.Handler {
	return requestid.New(requestid.Config{
		Header: "X-Request-ID",
	})
}

// CORS returns CORS middleware restricted to the deployed frontend's own
// origin plus the local dev servers, rather than "*" - the API is called
// with a Bearer token (not cookies), so an open CORS policy doesn't enable
// CSRF, but it does let any website script authenticated requests using a
// token obtained elsewhere (XSS, a leaked token, a malicious browser
// extension). frontendBaseURL is cfg.Frontend.BaseURL - the same origin the
// app itself is served from in each environment.
func CORS(frontendBaseURL string) fiber.Handler {
	origins := []string{"http://localhost:3000", "http://localhost:3001"}
	if frontendBaseURL != "" {
		origins = append(origins, frontendBaseURL)
	}
	return cors.New(cors.Config{
		AllowOrigins: strings.Join(origins, ","),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods: "GET, POST, HEAD, PUT, PATCH, DELETE, OPTIONS",
	})
}

// Logger returns structured HTTP request logger middleware using slog
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()
		if err != nil {
			if catchErr := c.App().ErrorHandler(c, err); catchErr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
			err = nil
		}

		latency := time.Since(start)
		status := c.Response().StatusCode()
		reqID := c.GetRespHeader("X-Request-ID")

		attrs := []any{
			"status", status,
			"method", c.Method(),
			"path", c.Path(),
			"ip", c.IP(),
			"latency_ms", float64(latency.Microseconds()) / 1000.0,
			"request_id", reqID,
		}

		if status >= 500 {
			slog.Error("http request error", attrs...)
		} else if status >= 400 {
			slog.Warn("http request client error", attrs...)
		} else {
			slog.Info("http request completed", attrs...)
		}

		return err
	}
}

// Recover returns a panic recovery middleware
func Recover() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				slog.Error("panic recovered",
					"error", fmt.Sprintf("%v", r),
					"stack", stack,
					"path", c.Path(),
					"method", c.Method(),
				)
				recordServerError(c, http.StatusInternalServerError, fmt.Sprintf("panic: %v", r))

				_ = response.Error(
					c,
					http.StatusInternalServerError,
					"INTERNAL_ERROR",
					"An unexpected error occurred",
					nil,
				)
			}
		}()
		return c.Next()
	}
}

// ErrorHandler is the global Fiber error handler
func ErrorHandler(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		if appErr.StatusCode >= 500 {
			msg := appErr.Message
			if appErr.RawError != nil {
				slog.Error("app internal error", "error", appErr.RawError, "path", c.Path())
				msg = fmt.Sprintf("%s: %v", appErr.Message, appErr.RawError)
			}
			recordServerError(c, appErr.StatusCode, msg)
		}
		return response.Error(c, appErr.StatusCode, appErr.ErrorCode, appErr.Message, appErr.Details)
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		if fiberErr.Code >= 500 {
			recordServerError(c, fiberErr.Code, fiberErr.Message)
		}
		return response.Error(c, fiberErr.Code, "HTTP_ERROR", fiberErr.Message, nil)
	}

	slog.Error("unhandled error", "error", err, "path", c.Path())
	recordServerError(c, http.StatusInternalServerError, err.Error())
	return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
}
