package middleware

import (
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/gofiber/fiber/v2"
)

// ActorContext stores the authenticated caller's email in the request context
// (see shared/actor). It must be mounted after the middleware that parses the
// JWT; a request without valid claims simply carries no actor.
func ActorContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if claims := CurrentUser(c); claims != nil && claims.Email != "" {
			c.SetUserContext(actor.WithEmail(c.UserContext(), claims.Email))
		}
		return c.Next()
	}
}
