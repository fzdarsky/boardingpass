package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/command"
	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestCommandEndpoint_Contract validates the POST /command endpoint against OpenAPI spec.
//
// Contract Requirements (from OpenAPI 3.1 spec):
// - Endpoint: POST /command
// - Content-Type: application/json
// - Request Body: CommandRequest schema (id: string matching ^[a-z0-9-]+$)
// - Success Response: 200 OK with CommandResponse (exit_code, stdout, stderr)
// - Error Responses: 400 Bad Request, 401 Unauthorized, 403 Forbidden
// - Authentication: Required (Bearer token)
func TestCommandEndpoint_Contract(t *testing.T) {
	t.Run("POST /command - Success Response Schema", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockExecutor := command.NewMockCommandExecutor(ctrl)

		// Mock successful execution
		mockExecutor.EXPECT().
			Execute(gomock.Any(), gomock.Any(), true).
			Return(&protocol.CommandResponse{
				ExitCode: 0,
				Stdout:   "hello\n",
				Stderr:   "",
			}, nil).
			Times(1)

		testConfig := &config.Config{
			Commands: []config.CommandDefinition{
				{
					ID:   "echo-test",
					Path: "/bin/echo",
					Args: []string{"hello"},
				},
			},
		}

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
		require.NoError(t, err)

		// Valid request per OpenAPI spec
		reqBody := protocol.CommandRequest{
			ID: "echo-test",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Validate response structure matches OpenAPI spec
		// Success response should be:
		// {
		//   "exit_code": <int>,
		//   "stdout": "<string>",
		//   "stderr": "<string>"
		// }
		assert.Equal(t, http.StatusOK, w.Code)

		// Get body bytes before decoding
		bodyBytes := w.Body.Bytes()

		var response protocol.CommandResponse
		err = json.Unmarshal(bodyBytes, &response)
		require.NoError(t, err, "Response should be valid JSON matching CommandResponse schema")

		// Validate required fields per OpenAPI spec
		// Note: In Go, these fields will always exist (zero values if not set)
		// But we check they're present in the JSON
		assert.Contains(t, string(bodyBytes), "exit_code", "Response must contain 'exit_code' field")
		assert.Contains(t, string(bodyBytes), "stdout", "Response must contain 'stdout' field")
		assert.Contains(t, string(bodyBytes), "stderr", "Response must contain 'stderr' field")

		// Validate field types
		assert.IsType(t, 0, response.ExitCode, "exit_code should be integer")
		assert.IsType(t, "", response.Stdout, "stdout should be string")
		assert.IsType(t, "", response.Stderr, "stderr should be string")
	})

	t.Run("POST /command - Forbidden Response Schema", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockExecutor := command.NewMockCommandExecutor(ctrl)
		// No expectation - command not in allow-list, so executor won't be called

		testConfig := &config.Config{
			Commands: []config.CommandDefinition{
				{
					ID:   "echo-test",
					Path: "/bin/echo",
					Args: []string{"hello"},
				},
			},
		}

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
		require.NoError(t, err)

		// Request for command not in allow-list should return 403
		reqBody := protocol.CommandRequest{
			ID: "unknown-command",
		}

		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Validate error response format per OpenAPI spec
		assert.Equal(t, http.StatusForbidden, w.Code, "Command not in allow-list should return 403")

		var response map[string]string
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err, "Error response should be valid JSON")

		// OpenAPI spec error response format:
		// {
		//   "error": "command_not_allowed",
		//   "message": "descriptive message"
		// }
		assert.Contains(t, response, "error", "Error response must contain 'error' field")
		assert.Contains(t, response, "message", "Error response must contain 'message' field")
		assert.Equal(t, "command_not_allowed", response["error"])
	})

	t.Run("POST /command - Bad Request Response Schema", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockExecutor := command.NewMockCommandExecutor(ctrl)
		// No expectation - invalid JSON will fail before executor is called

		testConfig := &config.Config{
			Commands: []config.CommandDefinition{
				{
					ID:   "echo-test",
					Path: "/bin/echo",
					Args: []string{"hello"},
				},
			},
		}

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
		require.NoError(t, err)

		// Invalid request to trigger 400 error response
		req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Validate error response format
		assert.Equal(t, http.StatusBadRequest, w.Code, "Invalid JSON should return 400")
		assert.NotEmpty(t, w.Body.String(), "Error response should have body")
		assert.Contains(t, w.Body.String(), "Invalid request body", "Error message should be descriptive")
	})

	t.Run("POST /command - Request Body Validation", func(t *testing.T) {
		// Test various request bodies per OpenAPI spec

		tests := []struct {
			name           string
			requestBody    any
			expectedStatus int
			errorContains  string
		}{
			{
				name:           "valid command ID - lowercase",
				requestBody:    protocol.CommandRequest{ID: "echo-test"},
				expectedStatus: http.StatusOK,
			},
			{
				name:           "valid command ID - with numbers",
				requestBody:    protocol.CommandRequest{ID: "restart-network123"},
				expectedStatus: http.StatusForbidden, // Not in allow-list, but valid format
			},
			{
				name:           "valid command ID - with hyphens",
				requestBody:    protocol.CommandRequest{ID: "restart-network-manager"},
				expectedStatus: http.StatusForbidden, // Not in allow-list, but valid format
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := command.NewMockCommandExecutor(ctrl)

				// Only expect Execute call for valid command in allow-list
				if tt.expectedStatus == http.StatusOK {
					mockExecutor.EXPECT().
						Execute(gomock.Any(), gomock.Any(), true).
						Return(&protocol.CommandResponse{
							ExitCode: 0,
							Stdout:   "test\n",
							Stderr:   "",
						}, nil).
						Times(1)
				}

				testConfig := &config.Config{
					Commands: []config.CommandDefinition{
						{
							ID:   "echo-test",
							Path: "/bin/echo",
							Args: []string{"test"},
						},
					},
				}

				logger := logging.New(logging.LevelInfo, logging.FormatJSON)
				handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
				require.NoError(t, err)

				body, err := json.Marshal(tt.requestBody)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				assert.Equal(t, tt.expectedStatus, w.Code)
			})
		}
	})

	t.Run("POST /command - Method Validation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockExecutor := command.NewMockCommandExecutor(ctrl)
		// No expectation - wrong methods won't reach executor

		testConfig := &config.Config{
			Commands: []config.CommandDefinition{
				{
					ID:   "echo-test",
					Path: "/bin/echo",
					Args: []string{"hello"},
				},
			},
		}

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
		require.NoError(t, err)

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
				req := httptest.NewRequest(method, "/command", nil)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusMethodNotAllowed, w.Code,
					"Method %s should not be allowed", method)
			})
		}
	})

	t.Run("POST /command - CommandResponse Fields", func(t *testing.T) {
		// Validate CommandResponse schema fields per OpenAPI spec

		tests := []struct {
			name         string
			commandID    string
			wantExitCode int
		}{
			{
				name:         "exit code 0 - successful command",
				commandID:    "true-test",
				wantExitCode: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := command.NewMockCommandExecutor(ctrl)

				// Mock successful execution
				mockExecutor.EXPECT().
					Execute(gomock.Any(), gomock.Any(), true).
					Return(&protocol.CommandResponse{
						ExitCode: tt.wantExitCode,
						Stdout:   "",
						Stderr:   "",
					}, nil).
					Times(1)

				testConfig := &config.Config{
					Commands: []config.CommandDefinition{
						{
							ID:   "true-test",
							Path: "/bin/true",
							Args: []string{},
						},
					},
				}

				logger := logging.New(logging.LevelInfo, logging.FormatJSON)
				handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
				require.NoError(t, err)

				reqBody := protocol.CommandRequest{ID: tt.commandID}
				body, err := json.Marshal(reqBody)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)

				var response protocol.CommandResponse
				err = json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				assert.Equal(t, tt.wantExitCode, response.ExitCode)
				assert.NotNil(t, response.Stdout, "stdout should not be nil")
				assert.NotNil(t, response.Stderr, "stderr should not be nil")
			})
		}
	})

	t.Run("POST /command - Content-Type Validation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockExecutor := command.NewMockCommandExecutor(ctrl)

		// Mock successful execution
		mockExecutor.EXPECT().
			Execute(gomock.Any(), gomock.Any(), true).
			Return(&protocol.CommandResponse{
				ExitCode: 0,
				Stdout:   "hello\n",
				Stderr:   "",
			}, nil).
			Times(1)

		testConfig := &config.Config{
			Commands: []config.CommandDefinition{
				{
					ID:   "echo-test",
					Path: "/bin/echo",
					Args: []string{"hello"},
				},
			},
		}

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
		require.NoError(t, err)

		reqBody := protocol.CommandRequest{ID: "echo-test"}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Request without Content-Type header
		req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Should still accept request (Content-Type is not strictly enforced by Go's json.Decoder)
		// but API documentation specifies application/json
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /command - Non-Zero Exit Code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockExecutor := command.NewMockCommandExecutor(ctrl)

		// Mock failed command execution (exit code 1)
		mockExecutor.EXPECT().
			Execute(gomock.Any(), gomock.Any(), true).
			Return(&protocol.CommandResponse{
				ExitCode: 1,
				Stdout:   "",
				Stderr:   "",
			}, nil).
			Times(1)

		// Add a command that will fail
		testConfig := &config.Config{
			Commands: []config.CommandDefinition{
				{
					ID:   "false-test",
					Path: "/bin/false",
					Args: []string{},
				},
			},
		}

		logger := logging.New(logging.LevelInfo, logging.FormatJSON)
		handler, err := handlers.NewCommandHandlerWithExecutor(testConfig, mockExecutor, logger)
		require.NoError(t, err)

		reqBody := protocol.CommandRequest{ID: "false-test"}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Per OpenAPI spec, command execution returns 200 even if command fails
		assert.Equal(t, http.StatusOK, w.Code, "Command execution should return 200 even on non-zero exit")

		var response protocol.CommandResponse
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, 1, response.ExitCode, "Exit code should be 1 for failed command")
	})
}

// TestCommandEndpoint_OpenAPICompliance validates that the implementation
// matches the OpenAPI specification in specs/001-boardingpass-api/contracts/openapi.yaml
//
// This test would ideally use an OpenAPI validator library to check:
// - Request/response schema compliance
// - HTTP status codes
// - Content-Type headers
// - Authentication requirements
//
// For now, we perform manual validation of key contract requirements.
func TestCommandEndpoint_OpenAPICompliance(t *testing.T) {
	t.Skip("TODO: Implement full OpenAPI spec validation using validator library (e.g., kin-openapi)")

	// Future implementation would:
	// 1. Load OpenAPI spec from specs/001-boardingpass-api/contracts/openapi.yaml
	// 2. Validate all requests/responses against spec using validator
	// 3. Verify all documented status codes are implemented
	// 4. Verify authentication requirements are enforced
	// 5. Verify all required/optional fields are present
	// 6. Verify command ID pattern validation (^[a-z0-9-]+$)
}
