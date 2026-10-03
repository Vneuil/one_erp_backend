// Package inbox lets a module drop an item in a user's notification center
// without depending on the module that stores notifications.
package inbox

import "context"

// Sender delivers one notification to the user with the given email. It is
// best effort: callers must not fail the action that triggered it.
type Sender func(ctx context.Context, recipientEmail, title, body, link string)
