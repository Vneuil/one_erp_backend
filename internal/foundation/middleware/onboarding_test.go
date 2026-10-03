package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fakeResolver struct{ calls atomic.Int32 }

func (f *fakeResolver) GetDB(context.Context, uuid.UUID) (*gorm.DB, error) {
	f.calls.Add(1)
	return &gorm.DB{}, nil
}

func newOnboardingApp(t *testing.T, enabled bool, claims **utils.JWTClaims, run func(context.Context, *gorm.DB, tenant.OnboardingUser) error, res TenantDBResolver) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if *claims != nil {
			c.Locals(ContextKeyUser, *claims)
		}
		return c.Next()
	})
	app.Use(EnsureUserOnboarded(enabled, res, func(uuid.UUID) string { return "Ani" }, run))
	app.Get("/api/v1/hrm/employees", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/api/v1/auth/me", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func hit(t *testing.T, app *fiber.App, path string) int {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

func TestOnboardsOncePerMemberAndDoesNotBlockRequests(t *testing.T) {
	company := uuid.New()
	claims := &utils.JWTClaims{UserID: uuid.New(), CompanyID: &company, Email: "ani@x.com", Role: "admin"}
	var runs atomic.Int32
	var seen tenant.OnboardingUser
	run := func(_ context.Context, _ *gorm.DB, u tenant.OnboardingUser) error { runs.Add(1); seen = u; return nil }
	app := newOnboardingApp(t, true, &claims, run, &fakeResolver{})

	for i := 0; i < 3; i++ {
		if code := hit(t, app, "/api/v1/hrm/employees"); code != 200 {
			t.Fatalf("request %d returned %d", i, code)
		}
	}
	if runs.Load() != 1 {
		t.Fatalf("onboarding should run once per member, ran %d times", runs.Load())
	}
	if seen.Email != "ani@x.com" || seen.Name != "Ani" || seen.Role != "admin" || seen.CompanyID != company {
		t.Fatalf("hook got the wrong member: %+v", seen)
	}

	// A different member of the same company is onboarded separately.
	claims = &utils.JWTClaims{UserID: uuid.New(), CompanyID: &company, Email: "budi@x.com", Role: "staff"}
	hit(t, app, "/api/v1/hrm/employees")
	if runs.Load() != 2 {
		t.Fatalf("second member should trigger its own onboarding, runs=%d", runs.Load())
	}
}

func TestOnboardingFailureNeverBreaksTheRequestAndBacksOff(t *testing.T) {
	company := uuid.New()
	claims := &utils.JWTClaims{UserID: uuid.New(), CompanyID: &company, Email: "ani@x.com"}
	var runs atomic.Int32
	run := func(context.Context, *gorm.DB, tenant.OnboardingUser) error {
		runs.Add(1)
		return errors.New("schema not migrated")
	}
	app := newOnboardingApp(t, true, &claims, run, &fakeResolver{})

	for i := 0; i < 3; i++ {
		if code := hit(t, app, "/api/v1/hrm/employees"); code != 200 {
			t.Fatalf("a failing hook must not fail the request, got %d", code)
		}
	}
	if runs.Load() != 1 {
		t.Fatalf("a failed member must back off instead of retrying every request, ran %d times", runs.Load())
	}
}

func TestOnboardingSkipsWhenDisabledUnauthenticatedOrExempt(t *testing.T) {
	company := uuid.New()
	var runs atomic.Int32
	run := func(context.Context, *gorm.DB, tenant.OnboardingUser) error { runs.Add(1); return nil }

	claims := &utils.JWTClaims{UserID: uuid.New(), CompanyID: &company, Email: "ani@x.com"}
	hit(t, newOnboardingApp(t, false, &claims, run, &fakeResolver{}), "/api/v1/hrm/employees")

	var none *utils.JWTClaims
	hit(t, newOnboardingApp(t, true, &none, run, &fakeResolver{}), "/api/v1/hrm/employees")

	noCompany := &utils.JWTClaims{UserID: uuid.New(), Email: "ani@x.com"}
	hit(t, newOnboardingApp(t, true, &noCompany, run, &fakeResolver{}), "/api/v1/hrm/employees")

	noEmail := &utils.JWTClaims{UserID: uuid.New(), CompanyID: &company}
	hit(t, newOnboardingApp(t, true, &noEmail, run, &fakeResolver{}), "/api/v1/hrm/employees")

	res := &fakeResolver{}
	hit(t, newOnboardingApp(t, true, &claims, run, res), "/api/v1/auth/me") // exempt path
	if runs.Load() != 0 || res.calls.Load() != 0 {
		t.Fatalf("onboarding must not run for disabled/unauthenticated/company-less/exempt requests (runs=%d, tenant lookups=%d)", runs.Load(), res.calls.Load())
	}
}
