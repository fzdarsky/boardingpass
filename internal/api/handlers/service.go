package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fzdarsky/boardingpass/internal/auth"
	"github.com/fzdarsky/boardingpass/pkg/version"
)

// serviceResponse is the JSON response for GET /.
type serviceResponse struct {
	Service string `json:"service"`
	Version string `json:"version,omitempty"`
}

// ServiceHandler handles GET / requests, returning service identity.
// Unauthenticated requests get {"service": "boardingpass"}.
// Authenticated requests also get the version field.
type ServiceHandler struct {
	sessionManager *auth.SessionManager
}

// NewServiceHandler creates a new ServiceHandler.
func NewServiceHandler(sm *auth.SessionManager) *ServiceHandler {
	return &ServiceHandler{sessionManager: sm}
}

// ServeHTTP handles GET / requests.
func (h *ServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := serviceResponse{Service: "boardingpass"}

	// Optionally include version for authenticated requests
	if token := extractBearerToken(r); token != "" {
		if _, err := h.sessionManager.ValidateSession(token); err == nil {
			resp.Version = version.Get().String()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// extractBearerToken extracts a Bearer token from the Authorization header.
// Returns empty string if no valid Bearer token is present.
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
