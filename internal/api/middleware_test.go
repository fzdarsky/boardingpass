//nolint:errcheck,revive // Test file - unchecked errors and unused params are acceptable
package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/middleware"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddleware(t *testing.T) {
	var logOutput bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&logOutput, &logOutput)

	handler := middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "test response", rr.Body.String())

	// Verify logging output
	logLine := logOutput.String()
	assert.NotEmpty(t, logLine)

	var logEntry map[string]any
	require.NoError(t, json.Unmarshal([]byte(logLine), &logEntry))

	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "http request", logEntry["message"])

	fields, ok := logEntry["fields"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "GET", fields["method"])
	assert.Equal(t, "/test", fields["path"])
	assert.Equal(t, "test-agent", fields["user_agent"])
	assert.Equal(t, float64(200), fields["status"])
}

func TestLoggingMiddleware_StatusCode(t *testing.T) {
	var logOutput bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&logOutput, &logOutput)

	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logOutput.Reset()

			handler := middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			var logEntry map[string]any
			require.NoError(t, json.Unmarshal(logOutput.Bytes(), &logEntry))

			fields := logEntry["fields"].(map[string]any)
			assert.Equal(t, float64(tt.statusCode), fields["status"])
		})
	}
}

func TestErrorHandlerMiddleware_Panic(t *testing.T) {
	var logOutput bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&logOutput, &logOutput)

	handler := middleware.ErrorHandler(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should return 500 with error response
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var errResp protocol.ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	assert.Equal(t, protocol.ErrCodeSystemError, errResp.Code)

	// Verify panic was logged
	assert.Contains(t, logOutput.String(), "panic recovered")
}

func TestWriteJSON(t *testing.T) {
	data := map[string]any{
		"key": "value",
		"num": 42,
	}

	rr := httptest.NewRecorder()
	middleware.WriteJSON(rr, data, http.StatusOK)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var result map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &result))
	assert.Equal(t, "value", result["key"])
	assert.Equal(t, float64(42), result["num"])
}

func TestWriteJSONError(t *testing.T) {
	err := protocol.NewUnauthorizedError()
	rr := httptest.NewRecorder()

	middleware.WriteJSONError(rr, err, http.StatusUnauthorized)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var errResp protocol.ErrorResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	assert.Equal(t, protocol.ErrCodeUnauthorized, errResp.Code)
	assert.Equal(t, "Authentication required", errResp.Message)
}

func TestHTTPStatusForErrorCode(t *testing.T) {
	tests := []struct {
		code           protocol.ErrorCode
		expectedStatus int
	}{
		// 400 Bad Request
		{protocol.ErrCodeInvalidRequest, http.StatusBadRequest},
		{protocol.ErrCodeInvalidPath, http.StatusBadRequest},
		{protocol.ErrCodeBundleTooLarge, http.StatusBadRequest},

		// 401 Unauthorized
		{protocol.ErrCodeUnauthorized, http.StatusUnauthorized},
		{protocol.ErrCodeAuthenticationFailed, http.StatusUnauthorized},
		{protocol.ErrCodeSessionExpired, http.StatusUnauthorized},

		// 403 Forbidden
		{protocol.ErrCodePathNotAllowed, http.StatusForbidden},
		{protocol.ErrCodeCommandNotAllowed, http.StatusForbidden},
		{protocol.ErrCodeAlreadyProvisioned, http.StatusForbidden},

		// 404 Not Found
		{protocol.ErrCodeVerifierNotFound, http.StatusNotFound},

		// 429 Too Many Requests
		{protocol.ErrCodeRateLimitExceeded, http.StatusTooManyRequests},

		// 500 Internal Server Error
		{protocol.ErrCodeSystemError, http.StatusInternalServerError},
		{protocol.ErrCodeFileSystemError, http.StatusInternalServerError},
		{protocol.ErrCodeTLSError, http.StatusInternalServerError},

		// 502 Bad Gateway
		{protocol.ErrCodeCommandFailed, http.StatusBadGateway},

		// 503 Service Unavailable
		{protocol.ErrCodeShuttingDown, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			status := middleware.HTTPStatusForErrorCode(tt.code)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestLoggingMiddleware_BytesWritten(t *testing.T) {
	var logOutput bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&logOutput, &logOutput)

	responseBody := "test response body"
	handler := middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(responseBody))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var logEntry map[string]any
	require.NoError(t, json.Unmarshal(logOutput.Bytes(), &logEntry))

	fields := logEntry["fields"].(map[string]any)
	assert.Equal(t, float64(len(responseBody)), fields["bytes"])
}

func TestErrorHandlerMiddleware_NoError(t *testing.T) {
	var logOutput bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&logOutput, &logOutput)

	handler := middleware.ErrorHandler(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())

	// No panic log should be present
	assert.NotContains(t, logOutput.String(), "panic recovered")
}
