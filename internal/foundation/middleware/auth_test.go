package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/gofiber/fiber/v2"
)

func TestRequireRolesChecksContextRoles(t *testing.T) {
	var claims *utils.JWTClaims
	var ctxRoles []string

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if ae, ok := err.(*apperrors.AppError); ok {
				return c.SendStatus(ae.StatusCode)
			}
			return c.SendStatus(fiber.StatusInternalServerError)
		},
	})
	app.Use(func(c *fiber.Ctx) error {
		if claims != nil {
			c.Locals(ContextKeyUser, claims)
		}
		if len(ctxRoles) > 0 {
			c.SetUserContext(actor.WithRoles(c.UserContext(), ctxRoles))
		}
		return c.Next()
	})
	app.Patch("/vendors/status", RequireRoles("admin", "manager"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	call := func() int {
		resp, err := app.Test(httptest.NewRequest("PATCH", "/vendors/status", nil))
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode
	}

	// Plain staff without Manager RBAC role is forbidden.
	claims = &utils.JWTClaims{Role: "staff"}
	ctxRoles = []string{"staff"}
	if got := call(); got != fiber.StatusForbidden {
		t.Fatalf("plain staff status = %d, want 403", got)
	}

	// Account with legacy role "staff" but assigned RBAC role "Manager" is allowed.
	claims = &utils.JWTClaims{Role: "staff"}
	ctxRoles = []string{"staff", "Manager"}
	if got := call(); got != fiber.StatusOK {
		t.Fatalf("staff with Manager RBAC role status = %d, want 200", got)
	}

	// Legacy manager without context roles is still allowed.
	claims = &utils.JWTClaims{Role: "manager"}
	ctxRoles = nil
	if got := call(); got != fiber.StatusOK {
		t.Fatalf("legacy manager status = %d, want 200", got)
	}
}
