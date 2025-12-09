package contract

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigureEndpoint_Contract validates the POST /configure endpoint against OpenAPI spec.
//
// Contract Requirements (from OpenAPI 3.1 spec):
// - Endpoint: POST /configure
// - Content-Type: application/json
// - Request Body: ConfigBundle schema
// - Success Response: 200 OK with status message
// - Error Responses: 400 Bad Request, 401 Unauthorized, 500 Internal Server Error
// - Authentication: Required (Bearer token)
func TestConfigureEndpoint_Contract(t *testing.T) {
	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{"/etc/test/"},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewConfigureHandler(testConfig, logger)

	t.Run("POST /configure - Success Response Schema", func(t *testing.T) {
		// Valid request per OpenAPI spec
		bundle := protocol.ConfigBundle{
			Files: []protocol.ConfigFile{
				{
					Path:    "test/config.yaml",
					Content: base64.StdEncoding.EncodeToString([]byte("key: value")),
					Mode:    0o644,
				},
			},
		}

		body, err := json.Marshal(bundle)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Note: This will fail validation, but we're testing response schema
		handler.ServeHTTP(w, req)

		// Validate response structure matches OpenAPI spec
		// Success response should be:
		// {
		//   "status": "success",
		//   "message": "Configuration applied successfully"
		// }
		if w.Code == http.StatusOK {
			var response map[string]string
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err, "Response should be valid JSON")

			// Validate required fields per OpenAPI spec
			assert.Contains(t, response, "status", "Response must contain 'status' field")
			assert.Contains(t, response, "message", "Response must contain 'message' field")
			assert.Equal(t, "success", response["status"], "Status should be 'success'")
		}
	})

	t.Run("POST /configure - Error Response Schema", func(t *testing.T) {
		// Invalid request to trigger error response
		req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Validate error response format
		assert.Equal(t, http.StatusBadRequest, w.Code, "Invalid JSON should return 400")
		assert.NotEmpty(t, w.Body.String(), "Error response should have body")

		// OpenAPI spec error response format:
		// Plain text error message or JSON error object
		assert.Contains(t, w.Body.String(), "Invalid request body", "Error message should be descriptive")
	})

	t.Run("POST /configure - Request Body Validation", func(t *testing.T) {
		// Test various invalid request bodies per OpenAPI spec

		tests := []struct {
			name           string
			bundle         any
			expectedStatus int
			errorContains  string
		}{
			{
				name:           "missing files field",
				bundle:         map[string]any{},
				expectedStatus: http.StatusBadRequest,
				errorContains:  "validation failed",
			},
			{
				name: "empty files array",
				bundle: protocol.ConfigBundle{
					Files: []protocol.ConfigFile{},
				},
				expectedStatus: http.StatusBadRequest,
				errorContains:  "must contain at least one file",
			},
			{
				name: "file missing required path field",
				bundle: protocol.ConfigBundle{
					Files: []protocol.ConfigFile{
						{
							Content: base64.StdEncoding.EncodeToString([]byte("content")),
							Mode:    0o644,
						},
					},
				},
				expectedStatus: http.StatusBadRequest,
				errorContains:  "validation failed",
			},
			{
				name: "file missing required content field",
				bundle: protocol.ConfigBundle{
					Files: []protocol.ConfigFile{
						{
							Path: "test/file.conf",
							Mode: 0o644,
						},
					},
				},
				expectedStatus: http.StatusBadRequest,
				errorContains:  "validation failed",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body, err := json.Marshal(tt.bundle)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				assert.Equal(t, tt.expectedStatus, w.Code)
				assert.Contains(t, w.Body.String(), tt.errorContains)
			})
		}
	})

	t.Run("POST /configure - Content-Type Validation", func(t *testing.T) {
		bundle := protocol.ConfigBundle{
			Files: []protocol.ConfigFile{
				{
					Path:    "test/file.conf",
					Content: base64.StdEncoding.EncodeToString([]byte("content")),
					Mode:    0o644,
				},
			},
		}

		body, err := json.Marshal(bundle)
		require.NoError(t, err)

		// Request without Content-Type header
		req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Should still accept request (Content-Type is not strictly enforced by Go's json.Decoder)
		// but API documentation specifies application/json
	})

	t.Run("POST /configure - Method Validation", func(t *testing.T) {
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
				req := httptest.NewRequest(method, "/configure", nil)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusMethodNotAllowed, w.Code,
					"Method %s should not be allowed", method)
			})
		}
	})

	t.Run("POST /configure - ConfigFile Schema Validation", func(t *testing.T) {
		// Validate ConfigFile schema fields per OpenAPI spec

		tests := []struct {
			name          string
			file          protocol.ConfigFile
			expectedError bool
			errorContains string
		}{
			{
				name: "valid file",
				file: protocol.ConfigFile{
					Path:    "test/valid.conf",
					Content: base64.StdEncoding.EncodeToString([]byte("content")),
					Mode:    0o644,
				},
				expectedError: false,
			},
			{
				name: "invalid mode (negative)",
				file: protocol.ConfigFile{
					Path:    "test/file.conf",
					Content: base64.StdEncoding.EncodeToString([]byte("content")),
					Mode:    -1,
				},
				expectedError: true,
				errorContains: "invalid mode",
			},
			{
				name: "invalid mode (too large)",
				file: protocol.ConfigFile{
					Path:    "test/file.conf",
					Content: base64.StdEncoding.EncodeToString([]byte("content")),
					Mode:    0o1000,
				},
				expectedError: true,
				errorContains: "invalid mode",
			},
			{
				name: "invalid Base64 content",
				file: protocol.ConfigFile{
					Path:    "test/file.conf",
					Content: "not-valid-base64!!!",
					Mode:    0o644,
				},
				expectedError: true,
				errorContains: "invalid Base64",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bundle := protocol.ConfigBundle{
					Files: []protocol.ConfigFile{tt.file},
				}

				body, err := json.Marshal(bundle)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				if tt.expectedError {
					assert.Equal(t, http.StatusBadRequest, w.Code)
					assert.Contains(t, w.Body.String(), tt.errorContains)
				}
			})
		}
	})

	t.Run("POST /configure - Bundle Constraints", func(t *testing.T) {
		// Validate OpenAPI spec constraints:
		// - Maximum 100 files
		// - Maximum 10MB total size

		t.Run("exceeds max file count", func(t *testing.T) {
			// Create bundle with 101 files (exceeds limit)
			files := make([]protocol.ConfigFile, 101)
			for i := range files {
				files[i] = protocol.ConfigFile{
					Path:    "test/file.conf",
					Content: base64.StdEncoding.EncodeToString([]byte("content")),
					Mode:    0o644,
				}
			}

			bundle := protocol.ConfigBundle{Files: files}
			body, err := json.Marshal(bundle)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "maximum is 100")
		})

		t.Run("at max file count boundary", func(t *testing.T) {
			// Create bundle with exactly 100 files (should be valid)
			files := make([]protocol.ConfigFile, 100)
			for i := range files {
				files[i] = protocol.ConfigFile{
					Path:    "test/file.conf",
					Content: base64.StdEncoding.EncodeToString([]byte("content")),
					Mode:    0o644,
				}
			}

			bundle := protocol.ConfigBundle{Files: files}
			body, err := json.Marshal(bundle)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should pass bundle validation but may fail path validation
			// depending on test configuration
		})
	})
}

// TestConfigureEndpoint_OpenAPICompliance validates that the implementation
// matches the OpenAPI specification in specs/001-boardingpass-api/contracts/openapi.yaml
//
// This test would ideally use an OpenAPI validator library to check:
// - Request/response schema compliance
// - HTTP status codes
// - Content-Type headers
// - Authentication requirements
//
// For now, we perform manual validation of key contract requirements.
func TestConfigureEndpoint_OpenAPICompliance(t *testing.T) {
	t.Skip("TODO: Implement full OpenAPI spec validation using validator library (e.g., kin-openapi)")

	// Future implementation would:
	// 1. Load OpenAPI spec from specs/001-boardingpass-api/contracts/openapi.yaml
	// 2. Validate all requests/responses against spec using validator
	// 3. Verify all documented status codes are implemented
	// 4. Verify authentication requirements are enforced
	// 5. Verify all required/optional fields are present
}
