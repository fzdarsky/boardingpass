//nolint:gofumpt // Test file - formatting is acceptable
package protocol_test

import (
	"encoding/json"
	"testing"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorResponse_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *protocol.ErrorResponse
		expected string
	}{
		{
			name: "without details",
			err: &protocol.ErrorResponse{
				Code:    protocol.ErrCodeUnauthorized,
				Message: "Authentication required",
			},
			expected: "UNAUTHORIZED: Authentication required",
		},
		{
			name: "with details",
			err: &protocol.ErrorResponse{
				Code:    protocol.ErrCodeInvalidPath,
				Message: "Invalid file path",
				Details: "/etc/passwd",
			},
			expected: "INVALID_PATH: Invalid file path (/etc/passwd)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestErrorResponse_JSON(t *testing.T) {
	tests := []struct {
		name     string
		err      *protocol.ErrorResponse
		expected string
	}{
		{
			name: "without details",
			err: &protocol.ErrorResponse{
				Code:    protocol.ErrCodeSessionExpired,
				Message: "Session token has expired",
			},
			expected: `{"code":"SESSION_EXPIRED","message":"Session token has expired"}`,
		},
		{
			name: "with details",
			err: &protocol.ErrorResponse{
				Code:    protocol.ErrCodeBundleTooLarge,
				Message: "Configuration bundle too large",
				Details: "15000000 bytes exceeds maximum of 10485760 bytes",
			},
			expected: `{"code":"BUNDLE_TOO_LARGE","message":"Configuration bundle too large","details":"15000000 bytes exceeds maximum of 10485760 bytes"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.err)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			var decoded protocol.ErrorResponse
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.err.Code, decoded.Code)
			assert.Equal(t, tt.err.Message, decoded.Message)
			assert.Equal(t, tt.err.Details, decoded.Details)
		})
	}
}

func TestNewError(t *testing.T) {
	err := protocol.NewError(protocol.ErrCodeUnauthorized, "Authentication required")
	assert.Equal(t, protocol.ErrCodeUnauthorized, err.Code)
	assert.Equal(t, "Authentication required", err.Message)
	assert.Empty(t, err.Details)
}

func TestNewErrorWithDetails(t *testing.T) {
	err := protocol.NewErrorWithDetails(
		protocol.ErrCodeInvalidPath,
		"Invalid file path",
		"/etc/passwd",
	)
	assert.Equal(t, protocol.ErrCodeInvalidPath, err.Code)
	assert.Equal(t, "Invalid file path", err.Message)
	assert.Equal(t, "/etc/passwd", err.Details)
}

func TestAuthenticationErrors(t *testing.T) {
	tests := []struct {
		name       string
		fn         func() *protocol.ErrorResponse
		code       protocol.ErrorCode
		message    string
		hasDetails bool
	}{
		{
			name: "NewAuthenticationFailedError",
			fn: func() *protocol.ErrorResponse {
				return protocol.NewAuthenticationFailedError("SRP verification failed")
			},
			code:       protocol.ErrCodeAuthenticationFailed,
			message:    "Authentication failed",
			hasDetails: true,
		},
		{
			name:       "NewInvalidCredentialsError",
			fn:         protocol.NewInvalidCredentialsError,
			code:       protocol.ErrCodeInvalidCredentials,
			message:    "Invalid username or password",
			hasDetails: false,
		},
		{
			name:       "NewSessionExpiredError",
			fn:         protocol.NewSessionExpiredError,
			code:       protocol.ErrCodeSessionExpired,
			message:    "Session token has expired",
			hasDetails: false,
		},
		{
			name:       "NewSessionInvalidError",
			fn:         protocol.NewSessionInvalidError,
			code:       protocol.ErrCodeSessionInvalid,
			message:    "Session token is invalid",
			hasDetails: false,
		},
		{
			name:       "NewUnauthorizedError",
			fn:         protocol.NewUnauthorizedError,
			code:       protocol.ErrCodeUnauthorized,
			message:    "Authentication required",
			hasDetails: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			if tt.hasDetails {
				assert.NotEmpty(t, err.Details)
			}
		})
	}
}

func TestNewRateLimitExceededError(t *testing.T) {
	err := protocol.NewRateLimitExceededError(60)
	assert.Equal(t, protocol.ErrCodeRateLimitExceeded, err.Code)
	assert.Equal(t, "Rate limit exceeded", err.Message)
	assert.Contains(t, err.Details, "60 seconds")
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		fn      func() *protocol.ErrorResponse
		code    protocol.ErrorCode
		message string
	}{
		{
			name:    "NewInvalidRequestError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewInvalidRequestError("missing field: username") },
			code:    protocol.ErrCodeInvalidRequest,
			message: "Invalid request",
		},
		{
			name:    "NewInvalidPathError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewInvalidPathError("/etc/passwd") },
			code:    protocol.ErrCodeInvalidPath,
			message: "Invalid file path",
		},
		{
			name:    "NewInvalidCommandError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewInvalidCommandError("unknown") },
			code:    protocol.ErrCodeInvalidCommand,
			message: "Invalid command ID",
		},
		{
			name:    "NewPathNotAllowedError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewPathNotAllowedError("/etc/shadow") },
			code:    protocol.ErrCodePathNotAllowed,
			message: "Path not in allow-list",
		},
		{
			name:    "NewCommandNotAllowedError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewCommandNotAllowedError("rm-rf") },
			code:    protocol.ErrCodeCommandNotAllowed,
			message: "Command not in allow-list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.NotEmpty(t, err.Details)
		})
	}
}

func TestNewBundleTooLargeError(t *testing.T) {
	err := protocol.NewBundleTooLargeError(15000000, 10485760)
	assert.Equal(t, protocol.ErrCodeBundleTooLarge, err.Code)
	assert.Equal(t, "Configuration bundle too large", err.Message)
	assert.Contains(t, err.Details, "15000000 bytes")
	assert.Contains(t, err.Details, "10485760 bytes")
}

func TestNewTooManyFilesError(t *testing.T) {
	err := protocol.NewTooManyFilesError(150, 100)
	assert.Equal(t, protocol.ErrCodeTooManyFiles, err.Code)
	assert.Equal(t, "Too many files in bundle", err.Message)
	assert.Contains(t, err.Details, "150 files")
	assert.Contains(t, err.Details, "100 files")
}

func TestNewInvalidFileModeError(t *testing.T) {
	err := protocol.NewInvalidFileModeError(0777)
	assert.Equal(t, protocol.ErrCodeInvalidFileMode, err.Code)
	assert.Equal(t, "Invalid file mode", err.Message)
	assert.Contains(t, err.Details, "777")
}

func TestSystemErrors(t *testing.T) {
	tests := []struct {
		name    string
		fn      func() *protocol.ErrorResponse
		code    protocol.ErrorCode
		message string
	}{
		{
			name:    "NewSystemError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewSystemError("unexpected error") },
			code:    protocol.ErrCodeSystemError,
			message: "System error",
		},
		{
			name:    "NewFileSystemError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewFileSystemError("disk full") },
			code:    protocol.ErrCodeFileSystemError,
			message: "Filesystem error",
		},
		{
			name:    "NewTPMError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewTPMError("TPM not responding") },
			code:    protocol.ErrCodeTPMError,
			message: "TPM error",
		},
		{
			name:    "NewNetworkError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewNetworkError("NetworkManager unavailable") },
			code:    protocol.ErrCodeNetworkError,
			message: "Network error",
		},
		{
			name:    "NewTLSError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewTLSError("certificate invalid") },
			code:    protocol.ErrCodeTLSError,
			message: "TLS error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.NotEmpty(t, err.Details)
		})
	}
}

func TestNewCommandFailedError(t *testing.T) {
	err := protocol.NewCommandFailedError(127)
	assert.Equal(t, protocol.ErrCodeCommandFailed, err.Code)
	assert.Equal(t, "Command execution failed", err.Message)
	assert.Contains(t, err.Details, "127")
}

func TestLifecycleErrors(t *testing.T) {
	tests := []struct {
		name       string
		fn         func() *protocol.ErrorResponse
		code       protocol.ErrorCode
		message    string
		hasDetails bool
	}{
		{
			name:       "NewAlreadyProvisionedError",
			fn:         protocol.NewAlreadyProvisionedError,
			code:       protocol.ErrCodeAlreadyProvisioned,
			message:    "Device has already been provisioned",
			hasDetails: false,
		},
		{
			name:       "NewSentinelFileExistsError",
			fn:         func() *protocol.ErrorResponse { return protocol.NewSentinelFileExistsError("/etc/boardingpass/issued") },
			code:       protocol.ErrCodeSentinelFileExists,
			message:    "Sentinel file already exists",
			hasDetails: true,
		},
		{
			name:       "NewShuttingDownError",
			fn:         protocol.NewShuttingDownError,
			code:       protocol.ErrCodeShuttingDown,
			message:    "Service is shutting down",
			hasDetails: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			if tt.hasDetails {
				assert.NotEmpty(t, err.Details)
			}
		})
	}
}

func TestConfigurationErrors(t *testing.T) {
	tests := []struct {
		name    string
		fn      func() *protocol.ErrorResponse
		code    protocol.ErrorCode
		message string
	}{
		{
			name:    "NewConfigurationError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewConfigurationError("invalid YAML") },
			code:    protocol.ErrCodeConfigurationError,
			message: "Configuration error",
		},
		{
			name:    "NewVerifierNotFoundError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewVerifierNotFoundError("/etc/boardingpass/verifier") },
			code:    protocol.ErrCodeVerifierNotFound,
			message: "SRP verifier file not found",
		},
		{
			name:    "NewInvalidConfigurationError",
			fn:      func() *protocol.ErrorResponse { return protocol.NewInvalidConfigurationError("missing field: port") },
			code:    protocol.ErrCodeInvalidConfiguration,
			message: "Invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.NotEmpty(t, err.Details)
		})
	}
}
