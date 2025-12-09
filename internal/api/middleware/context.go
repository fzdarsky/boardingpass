package middleware

import (
	"context"

	"github.com/fzdarsky/boardingpass/internal/auth"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	sessionContextKey contextKey = "session"
)

// withSession stores a session in the request context.
func withSession(ctx context.Context, session *auth.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// GetSession retrieves the session from the request context.
// Returns nil if no session is present.
func GetSession(ctx context.Context) *auth.Session {
	session, ok := ctx.Value(sessionContextKey).(*auth.Session)
	if !ok {
		return nil
	}
	return session
}
