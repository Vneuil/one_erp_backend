package http

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// rateLimit is a small fixed-window limiter keyed by client IP. It is per
// process, which is enough to blunt casual abuse of a public form; put a
// gateway limiter in front for anything stronger.
func rateLimit(max int, window time.Duration) fiber.Handler {
	type bucket struct {
		start time.Time
		n     int
	}
	var (
		mu      sync.Mutex
		buckets = map[string]*bucket{}
		swept   = time.Now()
	)
	return func(c *fiber.Ctx) error {
		now := time.Now()
		ip := c.IP()
		mu.Lock()
		if now.Sub(swept) > window {
			for k, b := range buckets {
				if now.Sub(b.start) > window {
					delete(buckets, k)
				}
			}
			swept = now
		}
		b := buckets[ip]
		if b == nil || now.Sub(b.start) > window {
			b = &bucket{start: now}
			buckets[ip] = b
		}
		b.n++
		blocked := b.n > max
		mu.Unlock()
		if blocked {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   fiber.Map{"code": "RATE_LIMITED", "message": "Too many requests, please try again shortly"},
			})
		}
		return c.Next()
	}
}
