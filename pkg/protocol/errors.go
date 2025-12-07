// Package protocol defines shared data structures and error codes for the BoardingPass API.
package protocol

import "fmt"

// ErrorCode represents a standardized error code for the BoardingPass API.
type ErrorCode string

// API error codes.
const (
	// ErrCodeAuthenticationFailed indicates authentication failed.
	ErrCodeAuthenticationFailed ErrorCode = "AUTHENTICATION_FAILED"
	// ErrCodeInvalidCredentials indicates invalid credentials were provided.
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	// ErrCodeSessionExpired indicates the session has expired.
	ErrCodeSessionExpired ErrorCode = "SESSION_EXPIRED"
	// ErrCodeSessionInvalid indicates the session token is invalid.
	ErrCodeSessionInvalid ErrorCode = "SESSION_INVALID"
	// ErrCodeRateLimitExceeded indicates too many requests were made.
	ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	// ErrCodeUnauthorized indicates the request is not authorized.
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"

	// ErrCodeInvalidRequest indicates the request payload is invalid.
	ErrCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	// ErrCodeInvalidPath indicates an invalid file path.
	ErrCodeInvalidPath ErrorCode = "INVALID_PATH"
	// ErrCodeInvalidCommand indicates an invalid command ID.
	ErrCodeInvalidCommand ErrorCode = "INVALID_COMMAND"
	// ErrCodeBundleTooLarge indicates the configuration bundle is too large.
	ErrCodeBundleTooLarge ErrorCode = "BUNDLE_TOO_LARGE"
	// ErrCodeTooManyFiles indicates too many files in the configuration bundle.
	ErrCodeTooManyFiles ErrorCode = "TOO_MANY_FILES"
	// ErrCodeInvalidFileMode indicates invalid file permissions.
	ErrCodeInvalidFileMode ErrorCode = "INVALID_FILE_MODE"
	// ErrCodePathNotAllowed indicates the path is not in the allow-list.
	ErrCodePathNotAllowed ErrorCode = "PATH_NOT_ALLOWED"
	// ErrCodeCommandNotAllowed indicates the command is not in the allow-list.
	ErrCodeCommandNotAllowed ErrorCode = "COMMAND_NOT_ALLOWED"

	// ErrCodeSystemError indicates a system-level error occurred.
	ErrCodeSystemError ErrorCode = "SYSTEM_ERROR"
	// ErrCodeFileSystemError indicates a filesystem error occurred.
	ErrCodeFileSystemError ErrorCode = "FILESYSTEM_ERROR"
	// ErrCodeCommandFailed indicates a command execution failed.
	ErrCodeCommandFailed ErrorCode = "COMMAND_FAILED"
	// ErrCodeTPMError indicates a TPM-related error occurred.
	ErrCodeTPMError ErrorCode = "TPM_ERROR"
	// ErrCodeNetworkError indicates a network-related error occurred.
	ErrCodeNetworkError ErrorCode = "NETWORK_ERROR"
	// ErrCodeTLSError indicates a TLS-related error occurred.
	ErrCodeTLSError ErrorCode = "TLS_ERROR"

	// ErrCodeAlreadyProvisioned indicates the device is already provisioned.
	ErrCodeAlreadyProvisioned ErrorCode = "ALREADY_PROVISIONED"
	// ErrCodeSentinelFileExists indicates the sentinel file already exists.
	ErrCodeSentinelFileExists ErrorCode = "SENTINEL_FILE_EXISTS"
	// ErrCodeShuttingDown indicates the service is shutting down.
	ErrCodeShuttingDown ErrorCode = "SHUTTING_DOWN"

	// ErrCodeConfigurationError indicates a configuration error.
	ErrCodeConfigurationError ErrorCode = "CONFIGURATION_ERROR"
	// ErrCodeVerifierNotFound indicates the SRP verifier was not found.
	ErrCodeVerifierNotFound ErrorCode = "VERIFIER_NOT_FOUND"
	// ErrCodeInvalidConfiguration indicates invalid configuration.
	ErrCodeInvalidConfiguration ErrorCode = "INVALID_CONFIGURATION"
)

// ErrorResponse represents a standardized API error response.
type ErrorResponse struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *ErrorResponse) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewError creates a new ErrorResponse.
func NewError(code ErrorCode, message string) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails creates a new ErrorResponse with details.
func NewErrorWithDetails(code ErrorCode, message, details string) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Common error constructors for convenience

// NewAuthenticationFailedError creates an authentication failed error.
func NewAuthenticationFailedError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeAuthenticationFailed, "Authentication failed", details)
}

// NewInvalidCredentialsError creates an invalid credentials error.
func NewInvalidCredentialsError() *ErrorResponse {
	return NewError(ErrCodeInvalidCredentials, "Invalid username or password")
}

// NewSessionExpiredError creates a session expired error.
func NewSessionExpiredError() *ErrorResponse {
	return NewError(ErrCodeSessionExpired, "Session token has expired")
}

// NewSessionInvalidError creates a session invalid error.
func NewSessionInvalidError() *ErrorResponse {
	return NewError(ErrCodeSessionInvalid, "Session token is invalid")
}

// NewRateLimitExceededError creates a rate limit exceeded error.
func NewRateLimitExceededError(retryAfter int) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeRateLimitExceeded, "Rate limit exceeded", fmt.Sprintf("Retry after %d seconds", retryAfter))
}

// NewUnauthorizedError creates an unauthorized error.
func NewUnauthorizedError() *ErrorResponse {
	return NewError(ErrCodeUnauthorized, "Authentication required")
}

// NewInvalidRequestError creates an invalid request error.
func NewInvalidRequestError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeInvalidRequest, "Invalid request", details)
}

// NewInvalidPathError creates an invalid path error.
func NewInvalidPathError(path string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeInvalidPath, "Invalid file path", path)
}

// NewInvalidCommandError creates an invalid command error.
func NewInvalidCommandError(commandID string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeInvalidCommand, "Invalid command ID", commandID)
}

// NewBundleTooLargeError creates a bundle too large error.
func NewBundleTooLargeError(size, maxSize int64) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeBundleTooLarge, "Configuration bundle too large", fmt.Sprintf("%d bytes exceeds maximum of %d bytes", size, maxSize))
}

// NewTooManyFilesError creates a too many files error.
func NewTooManyFilesError(count, maxCount int) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeTooManyFiles, "Too many files in bundle", fmt.Sprintf("%d files exceeds maximum of %d files", count, maxCount))
}

// NewInvalidFileModeError creates an invalid file mode error.
func NewInvalidFileModeError(mode int) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeInvalidFileMode, "Invalid file mode", fmt.Sprintf("Mode %o is invalid", mode))
}

// NewPathNotAllowedError creates a path not allowed error.
func NewPathNotAllowedError(path string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodePathNotAllowed, "Path not in allow-list", path)
}

// NewCommandNotAllowedError creates a command not allowed error.
func NewCommandNotAllowedError(commandID string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeCommandNotAllowed, "Command not in allow-list", commandID)
}

// NewSystemError creates a system error.
func NewSystemError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeSystemError, "System error", details)
}

// NewFileSystemError creates a filesystem error.
func NewFileSystemError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeFileSystemError, "Filesystem error", details)
}

// NewCommandFailedError creates a command failed error.
func NewCommandFailedError(exitCode int) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeCommandFailed, "Command execution failed", fmt.Sprintf("Exit code: %d", exitCode))
}

// NewTPMError creates a TPM error.
func NewTPMError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeTPMError, "TPM error", details)
}

// NewNetworkError creates a network error.
func NewNetworkError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeNetworkError, "Network error", details)
}

// NewTLSError creates a TLS error.
func NewTLSError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeTLSError, "TLS error", details)
}

// NewAlreadyProvisionedError creates an already provisioned error.
func NewAlreadyProvisionedError() *ErrorResponse {
	return NewError(ErrCodeAlreadyProvisioned, "Device has already been provisioned")
}

// NewSentinelFileExistsError creates a sentinel file exists error.
func NewSentinelFileExistsError(path string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeSentinelFileExists, "Sentinel file already exists", path)
}

// NewShuttingDownError creates a shutting down error.
func NewShuttingDownError() *ErrorResponse {
	return NewError(ErrCodeShuttingDown, "Service is shutting down")
}

// NewConfigurationError creates a configuration error.
func NewConfigurationError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeConfigurationError, "Configuration error", details)
}

// NewVerifierNotFoundError creates a verifier not found error.
func NewVerifierNotFoundError(path string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeVerifierNotFound, "SRP verifier file not found", path)
}

// NewInvalidConfigurationError creates an invalid configuration error.
func NewInvalidConfigurationError(details string) *ErrorResponse {
	return NewErrorWithDetails(ErrCodeInvalidConfiguration, "Invalid configuration", details)
}
