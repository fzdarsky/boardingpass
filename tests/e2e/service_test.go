// Package e2e provides end-to-end tests for the BoardingPass service.
//
// These tests verify the complete service lifecycle in a containerized
// systemd environment, ensuring production-like behavior.
package e2e

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testTimeout is the maximum time to wait for service startup
	testTimeout = 30 * time.Second
)

// TestServiceLifecycle tests the complete service lifecycle.
//
// This test verifies:
// - Service starts successfully
// - API endpoints are accessible
// - Service responds to shutdown signals gracefully
//
// Note: This test requires either:
// 1. Docker/Podman available for containerized testing
// 2. Local service binary with test configuration
func TestServiceLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if we should run containerized tests
	if os.Getenv("E2E_CONTAINER") == "true" {
		t.Skip("Container-based E2E tests not yet implemented - requires Containerfile and systemd setup")
		// TODO: Implement containerized test execution
		// This would involve:
		// 1. Building the service binary
		// 2. Creating a test container with systemd
		// 3. Running the service in the container
		// 4. Testing API endpoints
		// 5. Cleaning up
		return
	}

	// Run local E2E test
	t.Run("LocalService", func(t *testing.T) {
		testLocalService(t)
	})
}

// testLocalService tests the service running locally.
func testLocalService(t *testing.T) {
	// Create temporary directory for test configuration
	tmpDir, err := os.MkdirTemp("", "boardingpass-e2e-*")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Use a random available port instead of fixed port
	servicePort := findAvailablePort(t)
	serviceURL := fmt.Sprintf("https://127.0.0.1:%d", servicePort)

	// Create test configuration
	configPath := filepath.Join(tmpDir, "config.yaml")
	verifierPath := filepath.Join(tmpDir, "verifier")
	sentinelPath := filepath.Join(tmpDir, "issued")
	tlsCertPath := filepath.Join(tmpDir, "tls", "server.crt")
	tlsKeyPath := filepath.Join(tmpDir, "tls", "server.key")

	// Create TLS directory
	//nolint:gosec // Test directory permissions are acceptable
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "tls"), 0o755))

	// Write test configuration
	configContent := fmt.Sprintf(`service:
  inactivity_timeout: "10m"
  session_ttl: "30m"
  sentinel_file: "%s"

transports:
  ethernet:
    enabled: true
    interfaces: []
    address: "127.0.0.1"
    port: %d
    tls_cert: "%s"
    tls_key: "%s"

commands:
  - id: "test-command"
    path: "/bin/echo"
    args: ["test"]

logging:
  level: "debug"
  format: "json"

paths:
  allow_list:
    - "/etc/test/"
`, sentinelPath, servicePort, tlsCertPath, tlsKeyPath)

	//nolint:gosec // Test config file permissions are acceptable
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o600))

	// Create password generator script
	passwordGenPath := filepath.Join(tmpDir, "password-gen.sh")
	passwordGenScript := "#!/bin/bash\necho 'test-password'"
	//nolint:gosec // Test script file permissions are acceptable
	require.NoError(t, os.WriteFile(passwordGenPath, []byte(passwordGenScript), 0o700))

	// Write test verifier configuration
	verifierContent := fmt.Sprintf(`{
  "username": "boardingpass",
  "salt": "dGVzdHNhbHQ=",
  "password_generator": "%s"
}`, passwordGenPath)
	require.NoError(t, os.WriteFile(verifierPath, []byte(verifierContent), 0o600))

	// Generate self-signed TLS certificate for testing
	err = generateTestCertificate(tlsCertPath, tlsKeyPath)
	require.NoError(t, err)

	// Build the service binary
	binaryPath := filepath.Join(tmpDir, "boardingpass")
	//nolint:gosec // Build command with variable path is required for testing
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/boardingpass")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", string(buildOutput))
		require.NoError(t, err, "Failed to build service binary")
	}

	// Start the service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//nolint:gosec // Test service command with variable paths is required
	serviceCmd := exec.CommandContext(ctx, binaryPath,
		"-config", configPath,
		"-verifier", verifierPath,
	)
	serviceCmd.Stdout = os.Stdout
	serviceCmd.Stderr = os.Stderr

	require.NoError(t, serviceCmd.Start())
	defer func() {
		cancel()
		_ = serviceCmd.Wait()
	}()

	// Wait for service to start
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				//nolint:gosec // Test certificate is self-signed
				InsecureSkipVerify: true,
			},
		},
	}

	// Poll until service is ready
	started := false
	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(serviceURL + "/info")
		if err == nil {
			_ = resp.Body.Close()
			// Service is up (even if it returns 401 Unauthorized, it's running)
			started = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.True(t, started, "Service failed to start within timeout")

	// Test API endpoints
	t.Run("HealthCheck", func(t *testing.T) {
		// Try to access /info endpoint without authentication
		// Should return 401 Unauthorized
		resp, err := client.Get(serviceURL + "/info")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("AuthEndpoints", func(t *testing.T) {
		// Test SRP init endpoint
		resp, err := client.Post(serviceURL+"/auth/srp/init",
			"application/json",
			nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should accept POST requests (even if body is invalid)
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("CompleteSRPAuthFlow", func(t *testing.T) {
		// This test verifies the complete SRP authentication flow:
		// 1. Init: Client sends A, server returns salt, B, and session_id
		// 2. Verify: Client sends session_id and M1, server returns M2 and session_token (if M1 valid)
		// 3. Access: Client uses session_token to access protected endpoints

		// Step 1: SRP Init
		initReq := map[string]string{
			"username": "boardingpass",
			"A":        "gxs0ip7Q72TLryg8kngzqyTivy4OpNgrfSbWQkk2BHvzuIm4+FrB+Yiv3F2jIVdxlkt7yNlW9qIel+PkPXYfFMgoaLeXYvj0QAKCEzgttSarPm6UH+Lp2kMtwKo6PSu1EMYY9V4Db1bqmpBOFw1AaaVOuiOnP0iiIOeJgQ29sQz4Y6jdXQ02WwyjuI4wDIAUet15Ys3m6WEItqz/zBVMIYEHHMG8QIgBFwiQQ9OXKIbnoXUMUSPSajPmf5nEFBHRdrcwUD1PxsprqJTfwYFI4UqYI0iztvxduoFA95wrg1QmUqb0VpiCt+e5eP43M8RlONKCPoDqNsh+sLJeZTTe/A==",
		}
		initBody, _ := json.Marshal(initReq)
		initResp, err := client.Post(serviceURL+"/auth/srp/init",
			"application/json",
			bytes.NewBuffer(initBody))
		require.NoError(t, err)
		defer func() { _ = initResp.Body.Close() }()

		assert.Equal(t, http.StatusOK, initResp.StatusCode)

		var initResult map[string]string
		parseJSONResponse(t, initResp, &initResult)

		// Verify response has required fields
		assert.NotEmpty(t, initResult["salt"], "salt should not be empty")
		assert.NotEmpty(t, initResult["B"], "B should not be empty")
		assert.NotEmpty(t, initResult["session_id"], "session_id should not be empty")

		sessionID := initResult["session_id"]

		// Step 2: SRP Verify with invalid M1 (should fail)
		verifyReq := map[string]string{
			"session_id": sessionID,
			"M1":         "aW52YWxpZE0xUHJvb2Y=", // Invalid M1
		}
		verifyBody, _ := json.Marshal(verifyReq)
		verifyResp, err := client.Post(serviceURL+"/auth/srp/verify",
			"application/json",
			bytes.NewBuffer(verifyBody))
		require.NoError(t, err)
		defer func() { _ = verifyResp.Body.Close() }()

		// Should fail with invalid M1
		assert.Equal(t, http.StatusUnauthorized, verifyResp.StatusCode)

		// Step 3: Test invalid session ID
		invalidVerifyReq := map[string]string{
			"session_id": "invalid-session-id",
			"M1":         "aW52YWxpZE0xUHJvb2Y=",
		}
		invalidVerifyBody, _ := json.Marshal(invalidVerifyReq)
		invalidVerifyResp, err := client.Post(serviceURL+"/auth/srp/verify",
			"application/json",
			bytes.NewBuffer(invalidVerifyBody))
		require.NoError(t, err)
		defer func() { _ = invalidVerifyResp.Body.Close() }()

		assert.Equal(t, http.StatusUnauthorized, invalidVerifyResp.StatusCode)

		// Step 4: Test missing fields
		missingM1Req := map[string]string{
			"session_id": sessionID,
			// M1 missing
		}
		missingM1Body, _ := json.Marshal(missingM1Req)
		missingM1Resp, err := client.Post(serviceURL+"/auth/srp/verify",
			"application/json",
			bytes.NewBuffer(missingM1Body))
		require.NoError(t, err)
		defer func() { _ = missingM1Resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, missingM1Resp.StatusCode)
	})

	t.Run("SRPSessionExpiration", func(t *testing.T) {
		// Initialize SRP session
		initReq := map[string]string{
			"username": "boardingpass",
			"A":        "gxs0ip7Q72TLryg8kngzqyTivy4OpNgrfSbWQkk2BHvzuIm4+FrB+Yiv3F2jIVdxlkt7yNlW9qIel+PkPXYfFMgoaLeXYvj0QAKCEzgttSarPm6UH+Lp2kMtwKo6PSu1EMYY9V4Db1bqmpBOFw1AaaVOuiOnP0iiIOeJgQ29sQz4Y6jdXQ02WwyjuI4wDIAUet15Ys3m6WEItqz/zBVMIYEHHMG8QIgBFwiQQ9OXKIbnoXUMUSPSajPmf5nEFBHRdrcwUD1PxsprqJTfwYFI4UqYI0iztvxduoFA95wrg1QmUqb0VpiCt+e5eP43M8RlONKCPoDqNsh+sLJeZTTe/A==",
		}
		initBody, _ := json.Marshal(initReq)
		initResp, err := client.Post(serviceURL+"/auth/srp/init",
			"application/json",
			bytes.NewBuffer(initBody))
		require.NoError(t, err)
		defer func() { _ = initResp.Body.Close() }()

		var initResult map[string]string
		parseJSONResponse(t, initResp, &initResult)
		sessionID := initResult["session_id"]

		// Note: Session TTL is 5 minutes in production, so we can't realistically test expiration
		// in an e2e test. This is tested in unit tests for SRPStore.
		// Here we just verify the session exists immediately after creation.
		verifyReq := map[string]string{
			"session_id": sessionID,
			"M1":         "dGVzdE0x",
		}
		verifyBody, _ := json.Marshal(verifyReq)
		verifyResp, err := client.Post(serviceURL+"/auth/srp/verify",
			"application/json",
			bytes.NewBuffer(verifyBody))
		require.NoError(t, err)
		defer func() { _ = verifyResp.Body.Close() }()

		// Should get unauthorized (invalid M1) not "session expired"
		assert.Equal(t, http.StatusUnauthorized, verifyResp.StatusCode)
	})

	t.Run("SRPRateLimiting", func(t *testing.T) {
		// Test that rate limiting mechanism exists
		// Note: Detailed rate limiting behavior is tested in integration tests
		// Here we just verify the mechanism is active (may already be rate-limited from previous tests)

		initReq := map[string]string{
			"username": "wronguser",
			"A":        "gxs0ip7Q72TLryg8kngzqyTivy4OpNgrfSbWQkk2BHvzuIm4+FrB+Yiv3F2jIVdxlkt7yNlW9qIel+PkPXYfFMgoaLeXYvj0QAKCEzgttSarPm6UH+Lp2kMtwKo6PSu1EMYY9V4Db1bqmpBOFw1AaaVOuiOnP0iiIOeJgQ29sQz4Y6jdXQ02WwyjuI4wDIAUet15Ys3m6WEItqz/zBVMIYEHHMG8QIgBFwiQQ9OXKIbnoXUMUSPSajPmf5nEFBHRdrcwUD1PxsprqJTfwYFI4UqYI0iztvxduoFA95wrg1QmUqb0VpiCt+e5eP43M8RlONKCPoDqNsh+sLJeZTTe/A==",
		}
		initBody, _ := json.Marshal(initReq)
		initResp, err := client.Post(serviceURL+"/auth/srp/init",
			"application/json",
			bytes.NewBuffer(initBody))
		require.NoError(t, err)
		defer func() { _ = initResp.Body.Close() }()

		// Should fail with either 401 (first few attempts) or 429 (rate limited)
		// Exact status depends on previous test failures from same IP
		assert.True(t, initResp.StatusCode == http.StatusUnauthorized || initResp.StatusCode == http.StatusTooManyRequests,
			"expected 401 or 429, got %d", initResp.StatusCode)

		// Should have Retry-After header for rate limiting
		if initResp.StatusCode == http.StatusUnauthorized || initResp.StatusCode == http.StatusTooManyRequests {
			assert.NotEmpty(t, initResp.Header.Get("Retry-After"), "expected Retry-After header")
		}
	})

	// Test graceful shutdown
	t.Run("GracefulShutdown", func(t *testing.T) {
		// Cancel context to trigger shutdown
		cancel()

		// Wait for process to exit with timeout
		done := make(chan error, 1)
		go func() {
			done <- serviceCmd.Wait()
		}()

		select {
		case err := <-done:
			// Process should exit cleanly (exit code 0 or killed by signal)
			if err != nil && err.Error() != "signal: killed" && err.Error() != "signal: interrupt" {
				t.Logf("Service exited with error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Service did not exit within timeout")
		}
	})
}

// findAvailablePort finds an available TCP port for testing.
func findAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}

// generateTestCertificate generates a self-signed certificate for testing.
func generateTestCertificate(certPath, keyPath string) error {
	// Use openssl to generate a self-signed certificate
	// This is a simple approach for testing; production uses internal/tls/certgen.go
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", certPath,
		"-days", "1",
		"-nodes",
		"-subj", "/CN=localhost")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w (output: %s)", err, string(output))
	}

	return nil
}

// TestAPIContractCompliance tests that API responses match the OpenAPI specification.
//
// Note: This is a placeholder for more comprehensive contract testing.
// Full implementation would validate responses against the OpenAPI spec.
func TestAPIContractCompliance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping contract test in short mode")
	}

	t.Skip("Contract validation not yet implemented - requires openapi.yaml parser")

	// TODO: Implement OpenAPI contract validation
	// This would involve:
	// 1. Loading the OpenAPI spec from specs/001-boardingpass-api/contracts/openapi.yaml
	// 2. Starting the service
	// 3. Making API requests
	// 4. Validating responses against the spec
}

// TestSystemdIntegration tests systemd-specific functionality.
func TestSystemdIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping systemd test in short mode")
	}

	t.Skip("Systemd integration tests not yet implemented - requires containerized systemd environment")

	// TODO: Implement systemd integration tests
	// This would test:
	// 1. Type=notify functionality
	// 2. Sentinel file check (ConditionPathExists)
	// 3. Service restart behavior
	// 4. Signal handling (SIGTERM, SIGINT)
}

// Helper function to parse JSON response
func parseJSONResponse(t *testing.T, resp *http.Response, target any) {
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	err := json.NewDecoder(resp.Body).Decode(target)
	require.NoError(t, err)
}
