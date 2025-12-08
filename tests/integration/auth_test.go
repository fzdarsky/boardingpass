package integration_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/api/middleware"
	"github.com/fzdarsky/boardingpass/internal/auth"
)

// TestSRPHandshakeFlow tests the complete SRP-6a authentication flow.
func TestSRPHandshakeFlow(t *testing.T) {
	// Skip if we don't have a test password generator
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup test environment
	setup := setupTestAuth(t)
	defer setup.cleanup()

	// Test flow:
	// 1. POST /auth/srp/init → get salt and B
	// 2. POST /auth/srp/verify → get M2 and session token
	// 3. Use session token to access protected endpoint

	// Step 1: SRP Init
	initReq := map[string]string{
		"username": setup.username,
		"A":        generateTestEphemeralA(t),
	}

	initBody, _ := json.Marshal(initReq)
	initResp := httptest.NewRecorder()
	initHTTPReq := httptest.NewRequest("POST", "/auth/srp/init", bytes.NewReader(initBody))
	initHTTPReq.Header.Set("Content-Type", "application/json")

	setup.authHandler.HandleSRPInit(initResp, initHTTPReq)

	if initResp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", initResp.Code, initResp.Body.String())
	}

	var initRespData struct {
		Salt string `json:"salt"`
		B    string `json:"B"`
	}
	if err := json.Unmarshal(initResp.Body.Bytes(), &initRespData); err != nil {
		t.Fatalf("failed to parse init response: %v", err)
	}

	if initRespData.Salt == "" {
		t.Error("expected non-empty salt")
	}
	if initRespData.B == "" {
		t.Error("expected non-empty B")
	}

	// Note: Step 2 (SRP Verify) would require completing the SRP client-side
	// computation and proper server instance storage, which is marked as TODO
	// in the handler implementation.
}

// TestSRPInit_InvalidUsername tests authentication with wrong username.
func TestSRPInit_InvalidUsername(t *testing.T) {
	setup := setupTestAuth(t)
	defer setup.cleanup()

	req := map[string]string{
		"username": "wronguser",
		"A":        generateTestEphemeralA(t),
	}

	body, _ := json.Marshal(req)
	resp := httptest.NewRecorder()
	httpReq := httptest.NewRequest("POST", "/auth/srp/init", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	setup.authHandler.HandleSRPInit(resp, httpReq)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.Code)
	}
}

// TestSRPInit_MissingFields tests validation of required fields.
func TestSRPInit_MissingFields(t *testing.T) {
	setup := setupTestAuth(t)
	defer setup.cleanup()

	tests := []struct {
		name string
		req  map[string]string
	}{
		{
			name: "missing username",
			req: map[string]string{
				"A": generateTestEphemeralA(t),
			},
		},
		{
			name: "missing A",
			req: map[string]string{
				"username": setup.username,
			},
		},
		{
			name: "empty body",
			req:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			resp := httptest.NewRecorder()
			httpReq := httptest.NewRequest("POST", "/auth/srp/init", bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")

			setup.authHandler.HandleSRPInit(resp, httpReq)

			if resp.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", resp.Code)
			}
		})
	}
}

// TestRateLimiting tests progressive delay rate limiting.
func TestRateLimiting(t *testing.T) {
	setup := setupTestAuth(t)
	defer setup.cleanup()

	// Make multiple failed auth attempts
	for i := 0; i < 4; i++ {
		req := map[string]string{
			"username": "wronguser",
			"A":        generateTestEphemeralA(t),
		}

		body, _ := json.Marshal(req)
		resp := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/auth/srp/init", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.RemoteAddr = "192.168.1.100:12345"

		setup.authHandler.HandleSRPInit(resp, httpReq)

		if i < 3 {
			// First 3 failures should return 401
			if resp.Code != http.StatusUnauthorized {
				t.Errorf("attempt %d: expected status 401, got %d", i+1, resp.Code)
			}
			// Should have Retry-After header
			if resp.Header().Get("Retry-After") == "" {
				t.Errorf("attempt %d: expected Retry-After header", i+1)
			}
		} else {
			// 4th failure should return 429 (too many requests)
			if resp.Code != http.StatusTooManyRequests {
				t.Errorf("attempt %d: expected status 429, got %d", i+1, resp.Code)
			}
		}
	}
}

// TestAuthMiddleware_ValidToken tests authentication middleware with valid token.
func TestAuthMiddleware_ValidToken(t *testing.T) {
	setup := setupTestAuth(t)
	defer setup.cleanup()

	// Create a valid session
	token, err := setup.sessionManager.CreateSession("testuser")
	if err != nil {
		t.Fatal(err)
	}

	// Create a test handler that requires authentication
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := middleware.GetSession(r.Context())
		if session == nil {
			t.Error("expected session in context")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with auth middleware
	handler := setup.authMiddleware.Require(protectedHandler)

	// Make request with valid token
	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

// TestAuthMiddleware_MissingToken tests authentication middleware without token.
func TestAuthMiddleware_MissingToken(t *testing.T) {
	setup := setupTestAuth(t)
	defer setup.cleanup()

	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without valid token")
	})

	handler := setup.authMiddleware.Require(protectedHandler)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	// No Authorization header

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.Code)
	}
}

// TestAuthMiddleware_InvalidToken tests authentication middleware with invalid token.
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	setup := setupTestAuth(t)
	defer setup.cleanup()

	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
	})

	handler := setup.authMiddleware.Require(protectedHandler)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.Code)
	}
}

// Test setup helpers

type testAuthSetup struct {
	username       string
	verifierConfig *auth.SRPVerifierConfig
	sessionManager *auth.SessionManager
	rateLimiter    *auth.RateLimiter
	authHandler    *handlers.AuthHandler
	authMiddleware *middleware.AuthMiddleware
	cleanup        func()
}

func setupTestAuth(t *testing.T) *testAuthSetup {
	// Create test password generator script
	script := "#!/bin/bash\necho 'test-password-12345'"
	tmpfile, err := os.CreateTemp("", "generator-*.sh")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tmpfile.Name(), 0700); err != nil {
		t.Fatal(err)
	}

	// Create verifier config
	verifierConfig := &auth.SRPVerifierConfig{
		Username:          "boardingpass",
		Salt:              base64.StdEncoding.EncodeToString([]byte("testsalt")),
		PasswordGenerator: tmpfile.Name(),
	}

	// Create session manager
	secret, _ := auth.GenerateSessionSecret()
	sessionManager := auth.NewSessionManager(secret, 30*time.Minute)

	// Create rate limiter
	rateLimiter := auth.NewRateLimiter()

	// Create logger
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	// Create auth handler
	authHandler := handlers.NewAuthHandler(verifierConfig, sessionManager, rateLimiter, logger)

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(sessionManager)

	cleanup := func() {
		os.Remove(tmpfile.Name())
		sessionManager.Stop()
		rateLimiter.Stop()
	}

	return &testAuthSetup{
		username:       "boardingpass",
		verifierConfig: verifierConfig,
		sessionManager: sessionManager,
		rateLimiter:    rateLimiter,
		authHandler:    authHandler,
		authMiddleware: authMiddleware,
		cleanup:        cleanup,
	}
}

func generateTestEphemeralA(t *testing.T) string {
	// Generate a valid-looking ephemeral public value for testing
	// In a real SRP client, this would be computed as g^a % N
	N, gParam, _ := auth.GetGroupParameters()

	if N == nil {
		t.Fatal("N is nil from GetGroupParameters")
	}
	if gParam == nil {
		t.Fatal("g is nil from GetGroupParameters")
	}

	// Generate random a
	a := make([]byte, 32)
	_, err := rand.Read(a)
	if err != nil {
		// Fallback to a simple value for testing
		for i := range a {
			a[i] = byte(i)
		}
	}

	// Compute A = g^a % N
	aBig := new(big.Int).SetBytes(a)
	A := new(big.Int).Exp(gParam, aBig, N)

	return base64.StdEncoding.EncodeToString(A.Bytes())
}
