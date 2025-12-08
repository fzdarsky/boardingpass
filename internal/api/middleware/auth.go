package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fzdarsky/boardingpass/internal/auth"
)

// AuthMiddleware provides session token authentication for HTTP handlers.
type AuthMiddleware struct {
	sessionManager *auth.SessionManager
}

// NewAuthMiddleware creates a new authentication middleware.
func NewAuthMiddleware(sm *auth.SessionManager) *AuthMiddleware {
	return &AuthMiddleware{
		sessionManager: sm,
	}
}

// Require is an HTTP middleware that enforces authentication.
// It validates the session token from the Authorization header and
// rejects requests with missing or invalid tokens.
func (am *AuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		// Format: "Authorization: Bearer <token>"
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Missing authorization header")
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid authorization header format")
			return
		}

		token := parts[1]
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Missing session token")
			return
		}

		// Validate session token
		session, err := am.sessionManager.ValidateSession(token)
		if err != nil {
			if err == auth.ErrSessionNotFound {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid session token")
				return
			}
			if err == auth.ErrSessionExpired {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Session token expired")
				return
			}
			// Other validation errors
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired session token")
			return
		}

		// Session is valid - store in request context for handlers to use
		ctx := r.Context()
		ctx = withSession(ctx, session)
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error":   errorCode,
		"message": message,
	}

	_ = json.NewEncoder(w).Encode(response)
}
