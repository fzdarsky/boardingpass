package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteEndpoint_Contract validates the POST /complete endpoint against OpenAPI spec.
//
// Contract Requirements (from OpenAPI 3.1 spec):
// - Endpoint: POST /complete
// - Content-Type: application/json
// - Success Response: 200 OK with CompleteResponse (status, sentinel_file)
// - Error Responses: 401 Unauthorized, 500 Internal Server Error
// - Authentication: Required (Bearer token)
func TestCompleteEndpoint_Contract(t *testing.T) {
	t.Run("POST /complete - Success Response Schema", func(t *testing.T) {
		tempDir := t.TempDir()
		sentinelPath := filepath.Join(tempDir, "issued")

		shutdownCalled := false

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler := handlers.NewCompleteHandler(sentinelPath, func(reason string) {
			shutdownCalled = true
		}, logger)

		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Validate response structure matches OpenAPI spec
		// Success response should be:
		// {
		//   "status": "shutting_down",
		//   "sentinel_file": "<path>"
		// }
		assert.Equal(t, http.StatusOK, w.Code)

		// Get body bytes before decoding
		bodyBytes := w.Body.Bytes()

		var response protocol.CompleteResponse
		err := json.Unmarshal(bodyBytes, &response)
		require.NoError(t, err, "Response should be valid JSON matching CompleteResponse schema")

		// Validate required fields per OpenAPI spec
		assert.Contains(t, string(bodyBytes), "status", "Response must contain 'status' field")
		assert.Contains(t, string(bodyBytes), "sentinel_file", "Response must contain 'sentinel_file' field")

		// Validate field types and values
		assert.IsType(t, "", response.Status, "status should be string")
		assert.Equal(t, "shutting_down", response.Status, "status should be 'shutting_down'")
		assert.IsType(t, "", response.SentinelFile, "sentinel_file should be string")
		assert.Equal(t, sentinelPath, response.SentinelFile)

		// Validate shutdown was triggered
		assert.True(t, shutdownCalled, "shutdown function should have been called")
	})

	t.Run("POST /complete - Method Validation", func(t *testing.T) {
		tempDir := t.TempDir()
		sentinelPath := filepath.Join(tempDir, "issued")

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler := handlers.NewCompleteHandler(sentinelPath, nil, logger)

		// OpenAPI spec only allows POST method
		disallowedMethods := []string{
			http.MethodGet,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodHead,
			http.MethodOptions,
		}

		for _, method := range disallowedMethods {
			t.Run(method, func(t *testing.T) {
				req := httptest.NewRequest(method, "/complete", nil)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusMethodNotAllowed, w.Code,
					"Method %s should not be allowed", method)
			})
		}
	})

	t.Run("POST /complete - Idempotent Behavior", func(t *testing.T) {
		tempDir := t.TempDir()
		sentinelPath := filepath.Join(tempDir, "issued")

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler := handlers.NewCompleteHandler(sentinelPath, nil, logger)

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/complete", nil)
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)

		var response1 protocol.CompleteResponse
		err := json.NewDecoder(w1.Body).Decode(&response1)
		require.NoError(t, err)

		// Second request (sentinel file already exists)
		req2 := httptest.NewRequest(http.MethodPost, "/complete", nil)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code, "Endpoint should be idempotent")

		var response2 protocol.CompleteResponse
		err = json.NewDecoder(w2.Body).Decode(&response2)
		require.NoError(t, err)

		// Responses should be identical
		assert.Equal(t, response1.Status, response2.Status)
		assert.Equal(t, response1.SentinelFile, response2.SentinelFile)
	})

	t.Run("POST /complete - Content-Type Validation", func(t *testing.T) {
		tempDir := t.TempDir()
		sentinelPath := filepath.Join(tempDir, "issued")

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler := handlers.NewCompleteHandler(sentinelPath, nil, logger)

		// Request without Content-Type header should still work
		// (handler doesn't parse request body)
		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("POST /complete - Response Headers", func(t *testing.T) {
		tempDir := t.TempDir()
		sentinelPath := filepath.Join(tempDir, "issued")

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler := handlers.NewCompleteHandler(sentinelPath, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Per OpenAPI spec, response should have application/json Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("POST /complete - Shutdown Callback Integration", func(t *testing.T) {
		tempDir := t.TempDir()
		sentinelPath := filepath.Join(tempDir, "issued")

		shutdownCallCount := 0
		var shutdownReason string

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler := handlers.NewCompleteHandler(sentinelPath, func(reason string) {
			shutdownCallCount++
			shutdownReason = reason
		}, logger)

		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 1, shutdownCallCount, "Shutdown callback should be called exactly once")
		assert.Equal(t, "provisioning completed", shutdownReason)
	})

	t.Run("POST /complete - Nil Shutdown Callback", func(t *testing.T) {
		tempDir := t.TempDir()
		sentinelPath := filepath.Join(tempDir, "issued")

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		// Handler with nil shutdown callback should not panic
		handler := handlers.NewCompleteHandler(sentinelPath, nil, logger)

		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		w := httptest.NewRecorder()

		// Should not panic
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestCompleteEndpoint_OpenAPICompliance validates that the implementation
// matches the OpenAPI specification in specs/001-boardingpass-api/contracts/openapi.yaml
//
// This test would ideally use an OpenAPI validator library to check:
// - Request/response schema compliance
// - HTTP status codes
// - Content-Type headers
// - Authentication requirements
//
// For now, we perform manual validation of key contract requirements.
func TestCompleteEndpoint_OpenAPICompliance(t *testing.T) {
	t.Skip("TODO: Implement full OpenAPI spec validation using validator library (e.g., kin-openapi)")

	// Future implementation would:
	// 1. Load OpenAPI spec from specs/001-boardingpass-api/contracts/openapi.yaml
	// 2. Validate all requests/responses against spec using validator
	// 3. Verify all documented status codes are implemented
	// 4. Verify authentication requirements are enforced
	// 5. Verify all required/optional fields are present
}
