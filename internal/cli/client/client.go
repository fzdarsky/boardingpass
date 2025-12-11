package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const (
	defaultTimeout  = 30 * time.Second
	contentTypeJSON = "application/json"
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
func (c *Client) SRPVerify(M1 string) (*protocol.SRPVerifyResponse, error) {
	req := protocol.SRPVerifyRequest{
		M1: M1,
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

// doRequest executes an HTTP request with authentication and error handling.
func (c *Client) doRequest(req *http.Request, response any) error {
	// Add session token if available
	if c.sessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.sessionToken)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		return c.handleErrorResponse(resp.StatusCode, bodyBytes)
	}

	// Parse successful response
	if response != nil {
		if err := json.Unmarshal(bodyBytes, response); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
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
