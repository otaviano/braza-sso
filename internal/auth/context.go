package auth

import "context"

// ContextKey is the type used for request context keys set by auth middleware.
type ContextKey string

const (
	// ContextKeyUserID is the context key for the authenticated user's UUID string.
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyEmail is the context key for the authenticated user's email.
	ContextKeyEmail ContextKey = "email"
)

// UserIDFromContext extracts the authenticated user ID from a request context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextKeyUserID).(string)
	return v, ok
}
