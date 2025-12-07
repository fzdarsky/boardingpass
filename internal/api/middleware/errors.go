package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// ErrorHandler returns middleware that recovers from panics and handles errors.
func ErrorHandler(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered", map[string]any{
						"error": err,
						"path":  r.URL.Path,
					})

					WriteJSONError(w, protocol.NewSystemError("internal server error"), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If encoding fails, there's not much we can do
		// The status code has already been written
		return
	}
}

// WriteJSONError writes a JSON error response.
func WriteJSONError(w http.ResponseWriter, err *protocol.ErrorResponse, statusCode int) {
	WriteJSON(w, err, statusCode)
}

// HTTPStatusForErrorCode maps protocol error codes to HTTP status codes.
func HTTPStatusForErrorCode(code protocol.ErrorCode) int {
	switch code {
	// 400 Bad Request
	case protocol.ErrCodeInvalidRequest,
		protocol.ErrCodeInvalidPath,
		protocol.ErrCodeInvalidCommand,
		protocol.ErrCodeBundleTooLarge,
		protocol.ErrCodeTooManyFiles,
		protocol.ErrCodeInvalidFileMode,
		protocol.ErrCodeInvalidConfiguration:
		return http.StatusBadRequest

	// 401 Unauthorized
	case protocol.ErrCodeUnauthorized,
		protocol.ErrCodeAuthenticationFailed,
		protocol.ErrCodeInvalidCredentials,
		protocol.ErrCodeSessionExpired,
		protocol.ErrCodeSessionInvalid:
		return http.StatusUnauthorized

	// 403 Forbidden
	case protocol.ErrCodePathNotAllowed,
		protocol.ErrCodeCommandNotAllowed,
		protocol.ErrCodeAlreadyProvisioned,
		protocol.ErrCodeSentinelFileExists:
		return http.StatusForbidden

	// 404 Not Found
	case protocol.ErrCodeVerifierNotFound:
		return http.StatusNotFound

	// 429 Too Many Requests
	case protocol.ErrCodeRateLimitExceeded:
		return http.StatusTooManyRequests

	// 500 Internal Server Error
	case protocol.ErrCodeSystemError,
		protocol.ErrCodeFileSystemError,
		protocol.ErrCodeTPMError,
		protocol.ErrCodeNetworkError,
		protocol.ErrCodeTLSError,
		protocol.ErrCodeConfigurationError:
		return http.StatusInternalServerError

	// 503 Service Unavailable
	case protocol.ErrCodeShuttingDown:
		return http.StatusServiceUnavailable

	// 502 Bad Gateway (command execution failure)
	case protocol.ErrCodeCommandFailed:
		return http.StatusBadGateway

	default:
		return http.StatusInternalServerError
	}
}
