// Package handlers provides HTTP request handlers for the BoardingPass API.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/fzdarsky/boardingpass/internal/auth"
)

// AuthHandler handles SRP-6a authentication endpoints.
type AuthHandler struct {
	verifierConfig *auth.SRPVerifierConfig
	sessionManager *auth.SessionManager
	rateLimiter    *auth.RateLimiter
	logger         *log.Logger
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(
	verifierConfig *auth.SRPVerifierConfig,
	sessionManager *auth.SessionManager,
	rateLimiter *auth.RateLimiter,
	logger *log.Logger,
) *AuthHandler {
	return &AuthHandler{
		verifierConfig: verifierConfig,
		sessionManager: sessionManager,
		rateLimiter:    rateLimiter,
		logger:         logger,
	}
}

// SRPInitRequest represents the POST /auth/srp/init request body.
type SRPInitRequest struct {
	Username string `json:"username"`
	A        string `json:"A"` // Client ephemeral public value (Base64)
}

// SRPInitResponse represents the POST /auth/srp/init response body.
type SRPInitResponse struct {
	Salt string `json:"salt"` // Base64-encoded salt
	B    string `json:"B"`    // Server ephemeral public value (Base64)
}

// SRPVerifyRequest represents the POST /auth/srp/verify request body.
type SRPVerifyRequest struct {
	M1 string `json:"M1"` // Client proof (Base64)
}

// SRPVerifyResponse represents the POST /auth/srp/verify response body.
type SRPVerifyResponse struct {
	M2           string `json:"M2"`            // Server proof (Base64)
	SessionToken string `json:"session_token"` // Session token
}

// HandleSRPInit handles POST /auth/srp/init - initialize SRP handshake.
func (ah *AuthHandler) HandleSRPInit(w http.ResponseWriter, r *http.Request) {
	// Extract client IP for rate limiting
	clientIP := getClientIP(r)

	// Check rate limit
	locked, retryAfter, _ := ah.rateLimiter.CheckLimit(clientIP)
	if locked {
		// Client is locked out
		ah.logAuthEvent("srp_init_rate_limited", clientIP, "", "client locked out")
		w.Header().Set("Retry-After", fmt.Sprintf("%d", auth.FormatRetryAfter(retryAfter)))
		writeJSONError(w, http.StatusTooManyRequests, "too_many_requests",
			"Too many failed authentication attempts. Please try again later.")
		return
	}

	// Parse request body
	var req SRPInitRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		ah.logAuthEvent("srp_init_invalid_request", clientIP, "", fmt.Sprintf("parse error: %v", err))
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate required fields
	if req.Username == "" {
		ah.logAuthEvent("srp_init_missing_username", clientIP, "", "username missing")
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Missing required field: username")
		return
	}
	if req.A == "" {
		ah.logAuthEvent("srp_init_missing_A", clientIP, req.Username, "ephemeral public value A missing")
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Missing required field: A")
		return
	}

	// Verify username matches configured username
	if req.Username != ah.verifierConfig.Username {
		ah.logAuthEvent("srp_init_invalid_username", clientIP, req.Username, "username mismatch")
		// Don't reveal whether username is valid - treat as auth failure
		delay := ah.rateLimiter.RecordFailure(clientIP)
		w.Header().Set("Retry-After", fmt.Sprintf("%d", auth.FormatRetryAfter(delay)))
		writeJSONError(w, http.StatusUnauthorized, "authentication_failed", "Authentication failed")
		return
	}

	// Compute verifier from password generator
	N, g, _ := auth.GetGroupParameters()
	verifier, err := auth.ComputeVerifierFromConfig(ah.verifierConfig, N, g)
	if err != nil {
		ah.logAuthEvent("srp_init_verifier_error", clientIP, req.Username, fmt.Sprintf("verifier computation failed: %v", err))
		writeJSONError(w, http.StatusInternalServerError, "internal_server_error", "Internal server error")
		return
	}

	// Create SRP server instance
	server, err := auth.NewSRPServer(req.Username, ah.verifierConfig.Salt, verifier)
	if err != nil {
		ah.logAuthEvent("srp_init_server_error", clientIP, req.Username, fmt.Sprintf("SRP server creation failed: %v", err))
		writeJSONError(w, http.StatusInternalServerError, "internal_server_error", "Internal server error")
		return
	}

	// Initialize SRP session
	salt, B, err := server.Init(req.A)
	if err != nil {
		ah.logAuthEvent("srp_init_failed", clientIP, req.Username, fmt.Sprintf("SRP init failed: %v", err))
		// Invalid A value - treat as auth failure
		delay := ah.rateLimiter.RecordFailure(clientIP)
		w.Header().Set("Retry-After", fmt.Sprintf("%d", auth.FormatRetryAfter(delay)))
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid ephemeral public value A")
		return
	}

	// Store server instance in session for verify step
	// TODO: Implement server instance storage (in-memory map with TTL)
	// For now, we'll need to recompute in verify (stateless approach)

	// Return salt and B
	resp := SRPInitResponse{
		Salt: salt,
		B:    B,
	}

	ah.logAuthEvent("srp_init_success", clientIP, req.Username, "SRP init successful")
	writeJSONResponse(w, http.StatusOK, resp)
}

// HandleSRPVerify handles POST /auth/srp/verify - verify client proof and issue session token.
func (ah *AuthHandler) HandleSRPVerify(w http.ResponseWriter, r *http.Request) {
	// Extract client IP for rate limiting
	clientIP := getClientIP(r)

	// Check rate limit
	locked, retryAfter, _ := ah.rateLimiter.CheckLimit(clientIP)
	if locked {
		// Client is locked out
		ah.logAuthEvent("srp_verify_rate_limited", clientIP, "", "client locked out")
		w.Header().Set("Retry-After", fmt.Sprintf("%d", auth.FormatRetryAfter(retryAfter)))
		writeJSONError(w, http.StatusTooManyRequests, "too_many_requests",
			"Too many failed authentication attempts. Please try again later.")
		return
	}

	// Parse request body
	var req SRPVerifyRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		ah.logAuthEvent("srp_verify_invalid_request", clientIP, "", fmt.Sprintf("parse error: %v", err))
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate required fields
	if req.M1 == "" {
		ah.logAuthEvent("srp_verify_missing_M1", clientIP, "", "proof M1 missing")
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Missing required field: M1")
		return
	}

	// TODO: Retrieve server instance from storage
	// For now, this is a placeholder showing the flow
	// In a real implementation, we would need to:
	// 1. Store the SRP server instance after Init (with TTL)
	// 2. Retrieve it here using a session ID or similar
	// 3. Verify M1 using the stored server instance

	// Placeholder: This would fail in real use without server instance storage
	ah.logAuthEvent("srp_verify_not_implemented", clientIP, "", "server instance storage not implemented")
	writeJSONError(w, http.StatusNotImplemented, "not_implemented",
		"SRP verify requires server instance storage (not yet implemented)")

	// The following code shows the intended flow once storage is implemented:
	/*
		// Verify client proof M1
		M2, err := server.Verify(req.M1)
		if err != nil {
			ah.logAuthEvent("srp_verify_failed", clientIP, username, fmt.Sprintf("verification failed: %v", err))
			// Authentication failed
			delay := ah.rateLimiter.RecordFailure(clientIP)
			w.Header().Set("Retry-After", fmt.Sprintf("%d", auth.FormatRetryAfter(delay)))
			writeJSONError(w, http.StatusUnauthorized, "authentication_failed", "Authentication failed")
			return
		}

		// Authentication successful - clear rate limit
		ah.rateLimiter.RecordSuccess(clientIP)

		// Create session token
		sessionToken, err := ah.sessionManager.CreateSession(username)
		if err != nil {
			ah.logAuthEvent("srp_verify_session_error", clientIP, username, fmt.Sprintf("session creation failed: %v", err))
			writeJSONError(w, http.StatusInternalServerError, "internal_server_error", "Internal server error")
			return
		}

		// Clear server secrets from memory
		server.ClearSecrets()

		// Return M2 and session token
		resp := SRPVerifyResponse{
			M2:           M2,
			SessionToken: sessionToken,
		}

		ah.logAuthEvent("srp_verify_success", clientIP, username, "authentication successful")
		writeJSONResponse(w, http.StatusOK, resp)
	*/
}

// logAuthEvent logs an authentication event with secret redaction.
func (ah *AuthHandler) logAuthEvent(event, clientIP, username, details string) {
	// Redact sensitive fields
	redactedDetails := details
	// In a real implementation, we would redact: A, B, M1, M2, session tokens, etc.
	// For now, we just log the event with basic info

	ah.logger.Printf("[AUTH] event=%s client_ip=%s username=%s details=%s",
		event, clientIP, username, redactedDetails)
}

// getClientIP extracts the client IP address from the request.
// Checks X-Forwarded-For header first (for proxies), then RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take first IP in comma-separated list
		ips := splitAndTrim(xff, ",")
		if len(ips) > 0 && ips[0] != "" {
			return ips[0]
		}
	}

	// Fall back to RemoteAddr
	// RemoteAddr format: "IP:port" or "[IPv6]:port"
	remoteAddr := r.RemoteAddr
	// Strip port
	if idx := lastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}

// writeJSONResponse writes a JSON success response.
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Error already written to response, log it
		return
	}
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

// Helper functions

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i:i+len(sep)] == sep {
			parts = append(parts, trim(s[start:i]))
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, trim(s[start:]))
	return parts
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for start < end && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
