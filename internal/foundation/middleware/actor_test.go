package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/divinecoid/one-backend/internal/shared/utils"
	"github.com/gofiber/fiber/v2"
)

func TestActorContextExposesCallerEmailToHandlers(t *testing.T) {
	var claims *utils.JWTClaims
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if claims != nil {
			c.Locals(ContextKeyUser, claims)
		}
		return c.Next()
	})
	app.Use(ActorContext())
	app.Get("/who", func(c *fiber.Ctx) error { return c.SendString(actor.EmailFrom(c.UserContext())) })

	get := func() string {
		resp, err := app.Test(httptest.NewRequest("GET", "/who", nil))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}

	claims = &utils.JWTClaims{Email: "ani@x.com"}
	if got := get(); got != "ani@x.com" {
		t.Fatalf("handler saw actor %q, want ani@x.com", got)
	}
	claims = nil
	if got := get(); got != "" {
		t.Fatalf("unauthenticated request should carry no actor, got %q", got)
	}
}
