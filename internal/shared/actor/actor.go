// Package actor carries the authenticated caller's identity through a
// request context, so use cases can make ownership decisions (for example
// "you cannot approve your own request") without every method taking an
// extra parameter.
package actor

import "context"

type emailKey struct{}

// WithEmail returns ctx carrying the caller's email address.
func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailKey{}, email)
}

// EmailFrom returns the caller's email, or "" when the context carries none
// (background jobs, seeds, tests).
func EmailFrom(ctx context.Context) string {
	email, _ := ctx.Value(emailKey{}).(string)
	return email
}
