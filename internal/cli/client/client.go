package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const (
	defaultTimeout  = 30 * time.Second
	contentTypeJSON = "application/json"
	maxRetries      = 3
	initialBackoff  = 500 * time.Millisecond
	maxBackoff      = 5 * time.Second
)

// Client is an HTTP client for the BoardingPass API.
type Client struct {
	baseURL      string
	httpClient   *http.Client
	sessionToken string
}

// NewClient creates a new BoardingPass API client.
func NewClient(cfg *config.Config) (*Client, error) {
	// Create TOFU transport
	transport, err := NewTOFUTransport(cfg.Address(), cfg.CACert)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   defaultTimeout,
	}

	baseURL := fmt.Sprintf("https://%s", cfg.Address())

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}, nil
}

// SetSessionToken sets the session token for authenticated requests.
func (c *Client) SetSessionToken(token string) {
	c.sessionToken = token
}

// SRPInit initiates SRP-6a authentication (Phase 1).
//
//nolint:gocritic // A is capitalized per RFC 5054 SRP-6a specification
func (c *Client) SRPInit(username, A string) (*protocol.SRPInitResponse, error) {
	req := protocol.SRPInitRequest{
		Username: username,
		A:        A,
	}

	var resp protocol.SRPInitResponse
	if err := c.post("/auth/srp/init", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SRPVerify completes SRP-6a authentication (Phase 2).
//
//nolint:gocritic // M1 is capitalized per RFC 5054 SRP-6a specification
func (c *Client) SRPVerify(sessionID, M1 string) (*protocol.SRPVerifyResponse, error) {
	req := protocol.SRPVerifyRequest{
		SessionID: sessionID,
		M1:        M1,
	}

	var resp protocol.SRPVerifyResponse
	if err := c.post("/auth/srp/verify", req, &resp); err != nil {
		return nil, err
	}

	// Store session token from response
	c.sessionToken = resp.SessionToken

	return &resp, nil
}

// GetInfo retrieves system information from the device.
func (c *Client) GetInfo() (*protocol.SystemInfo, error) {
	var info protocol.SystemInfo
	if err := c.get("/info", &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetNetwork retrieves network interface configuration from the device.
func (c *Client) GetNetwork() (*protocol.NetworkConfig, error) {
	var network protocol.NetworkConfig
	if err := c.get("/network", &network); err != nil {
		return nil, err
	}
	return &network, nil
}

// ExecuteCommand executes an allow-listed command on the device.
func (c *Client) ExecuteCommand(commandID string) (*protocol.CommandResponse, error) {
	req := protocol.CommandRequest{
		ID: commandID,
	}

	var resp protocol.CommandResponse
	if err := c.post("/command", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PostConfigure uploads a configuration bundle to the device.
func (c *Client) PostConfigure(bundle *protocol.ConfigBundle) error {
	return c.post("/configure", bundle, nil)
}

// Complete signals provisioning completion to the device.
func (c *Client) Complete() (*protocol.CompleteResponse, error) {
	var resp protocol.CompleteResponse
	if err := c.post("/complete", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// get performs a GET request to the specified path.
func (c *Client) get(path string, response any) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	return c.doRequest(req, response)
}

// post performs a POST request to the specified path.
func (c *Client) post(path string, body any, response any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", contentTypeJSON)
	}

	return c.doRequest(req, response)
}

// doRequest executes an HTTP request with authentication, retry logic, and error handling.
func (c *Client) doRequest(req *http.Request, response any) error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone request body for retries (if present)
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			_ = req.Body.Close()
		}

		// Add session token if available
		if c.sessionToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.sessionToken)
		}

		// Restore body for this attempt
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Execute request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Check if error is retryable
			if isRetryable(err) && attempt < maxRetries {
				lastErr = fmt.Errorf("request failed (attempt %d/%d): %w", attempt+1, maxRetries+1, err)
				time.Sleep(backoff)
				backoff = min(backoff*2, maxBackoff)
				continue
			}
			return fmt.Errorf("request failed: %w", err)
		}

		// Read response body
		respBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Handle error responses
		if resp.StatusCode >= 400 {
			// Check if status code is retryable (5xx server errors)
			if resp.StatusCode >= 500 && attempt < maxRetries {
				lastErr = fmt.Errorf("server error (HTTP %d, attempt %d/%d)", resp.StatusCode, attempt+1, maxRetries+1)
				time.Sleep(backoff)
				backoff = min(backoff*2, maxBackoff)
				continue
			}
			return c.handleErrorResponse(resp.StatusCode, respBytes)
		}

		// Parse successful response
		if response != nil {
			if err := json.Unmarshal(respBytes, response); err != nil {
				// Provide helpful error for malformed JSON
				if len(respBytes) > 100 {
					return fmt.Errorf("failed to parse response (invalid JSON): %w", err)
				}
				return fmt.Errorf("failed to parse response (invalid JSON, body: %s): %w", string(respBytes), err)
			}
		}

		return nil
	}

	return lastErr
}

// isRetryable checks if an error is transient and should be retried.
func isRetryable(err error) bool {
	// Network timeout errors
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// DNS temporary errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsTemporary {
		return true
	}

	// Connection refused (server may be starting up)
	if errors.Is(err, errors.New("connection refused")) {
		return true
	}

	return false
}

// handleErrorResponse converts HTTP error responses to user-friendly errors.
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	// Try to parse as API error
	var apiError struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Message != "" {
		return fmt.Errorf("%s (HTTP %d)", apiError.Message, statusCode)
	}

	// Fallback to generic error messages
	switch statusCode {
	case http.StatusUnauthorized:
		return &AuthError{Message: "Session expired or invalid. Run 'boarding pass' to re-authenticate"}
	case http.StatusForbidden:
		return fmt.Errorf("permission denied (HTTP 403)")
	case http.StatusNotFound:
		return fmt.Errorf("endpoint not found - possible version mismatch (HTTP 404)")
	case http.StatusInternalServerError:
		return fmt.Errorf("server error - contact administrator (HTTP 500)")
	case http.StatusServiceUnavailable:
		return fmt.Errorf("service unavailable - try again later (HTTP 503)")
	default:
		return fmt.Errorf("request failed with status %d: %s", statusCode, string(body))
	}
}

// AuthError represents an authentication/authorization error.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// IsAuthError checks if an error is an authentication error.
func IsAuthError(err error) bool {
	_, ok := err.(*AuthError)
	return ok
}
