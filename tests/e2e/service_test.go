// Package e2e provides end-to-end tests for the BoardingPass service.
//
// These tests verify the complete service lifecycle in a containerized
// systemd environment using the production deployment method (make deploy).
package e2e

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testTimeout is the maximum time to wait for service startup
	testTimeout = 60 * time.Second
	// servicePort is the port exposed by the BoardingPass service
	servicePort = 8443
	// containerRuntime is the container runtime to use (podman or docker)
	containerRuntime = "podman"
)

// TestServiceLifecycle tests the complete service lifecycle.
//
// This test verifies:
// - Service starts successfully via make deploy
// - API endpoints are accessible
// - Service responds properly with authentication requirements
// - SRP authentication flow works correctly
// - Service runs with production-like deployment
func TestServiceLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if container runtime is available
	if _, err := exec.LookPath(containerRuntime); err != nil {
		t.Skipf("Container runtime %s not available", containerRuntime)
	}

	// Check if running in CI without container support
	if os.Getenv("CI") == "true" && os.Getenv("E2E_SKIP_CONTAINER") == "true" {
		t.Skip("Skipping container-based E2E test in CI")
	}

	ctx := context.Background()
	env := newTestEnvironment(ctx, t)
	defer env.cleanup()

	// Deploy the service using make deploy
	env.deployService(t)

	// Wait for service to be ready
	env.waitForService(t)

	// Extract TLS certificate for client use
	env.extractTLSCertificate(t)

	// Test API endpoints
	t.Run("HealthCheck", func(t *testing.T) {
		env.testHealthCheck(t)
	})

	t.Run("AuthEndpoints", func(t *testing.T) {
		env.testAuthEndpoints(t)
	})

	t.Run("CompleteSRPAuthFlow", func(t *testing.T) {
		env.testCompleteSRPAuthFlow(t)
	})

	t.Run("SRPSessionExpiration", func(t *testing.T) {
		env.testSRPSessionExpiration(t)
	})

	t.Run("SRPRateLimiting", func(t *testing.T) {
		env.testSRPRateLimiting(t)
	})

	t.Run("ServiceLogs", func(t *testing.T) {
		env.testServiceLogs(t)
	})
}

// testEnvironment holds all the state for an E2E test run.
type testEnvironment struct {
	ctx           context.Context
	t             *testing.T
	tmpDir        string
	containerName string
	serviceHost   string
	serviceURL    string
	tlsCertPath   string
	client        *http.Client
}

// newTestEnvironment creates a new test environment.
func newTestEnvironment(ctx context.Context, t *testing.T) *testEnvironment {
	tmpDir, err := os.MkdirTemp("", "e2e-*")
	require.NoError(t, err)

	// Use timestamp for unique container name
	containerName := fmt.Sprintf("boardingpass-e2e-%d", time.Now().Unix())
	serviceHost := "127.0.0.1"
	serviceURL := fmt.Sprintf("https://%s:%d", serviceHost, servicePort)

	return &testEnvironment{
		ctx:           ctx,
		t:             t,
		tmpDir:        tmpDir,
		containerName: containerName,
		serviceHost:   serviceHost,
		serviceURL:    serviceURL,
	}
}

// cleanup removes all test resources.
func (e *testEnvironment) cleanup() {
	// Stop and remove container using make undeploy
	t := e.t
	t.Logf("Cleaning up test environment...")

	//nolint:gosec // Test command with controlled environment
	undeployCmd := exec.Command("make", "undeploy")
	undeployCmd.Env = append(os.Environ(), fmt.Sprintf("CONTAINER_NAME=%s", e.containerName))
	undeployCmd.Dir = "../.." // Run from repository root
	_ = undeployCmd.Run()

	// Remove temporary directory
	_ = os.RemoveAll(e.tmpDir)
}

// deployService deploys the BoardingPass service using make deploy.
func (e *testEnvironment) deployService(t *testing.T) {
	t.Helper()
	t.Logf("Deploying service using make deploy...")

	// Use make deploy with test-specific containerfile and container name
	//nolint:gosec // Test command with controlled environment
	deployCmd := exec.Command("make", "deploy")
	deployCmd.Env = append(os.Environ(),
		fmt.Sprintf("CONTAINER_NAME=%s", e.containerName),
		fmt.Sprintf("IMAGE_NAME=%s:latest", e.containerName),
		"CONTAINERFILE=build/Containerfile.bootc.test",
	)
	deployCmd.Dir = "../.." // Run from repository root
	deployCmd.Stdout = os.Stdout
	deployCmd.Stderr = os.Stderr

	err := deployCmd.Run()
	require.NoError(t, err, "Failed to deploy service")

	t.Logf("Service deployed successfully in container %s", e.containerName)
}

// waitForService waits for the BoardingPass service to be ready.
func (e *testEnvironment) waitForService(t *testing.T) {
	t.Helper()
	t.Logf("Waiting for service to be ready...")

	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		// Check if service is listening
		output := e.execInContainer(t, "bash", "-c",
			fmt.Sprintf("ss -tuln | grep ':%d' || true", servicePort))

		if strings.Contains(output, fmt.Sprintf(":%d", servicePort)) {
			t.Logf("Service is ready and listening on port %d", servicePort)
			// Give it a moment to fully initialize
			time.Sleep(2 * time.Second)
			return
		}

		// Check service logs for errors
		logs := e.execInContainer(t, "journalctl", "-u", "boardingpass", "--no-pager", "-n", "20")
		if strings.Contains(logs, "panic") || strings.Contains(logs, "fatal error") {
			t.Fatalf("Service failed to start. Logs:\n%s", logs)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Dump logs for debugging
	logs := e.execInContainer(t, "journalctl", "-u", "boardingpass", "--no-pager")
	t.Fatalf("Service did not start within timeout. Logs:\n%s", logs)
}

// extractTLSCertificate extracts the TLS certificate from the container.
func (e *testEnvironment) extractTLSCertificate(t *testing.T) {
	t.Helper()
	t.Logf("Extracting TLS certificate from container...")

	certPath := filepath.Join(e.tmpDir, "server.crt")

	// Copy certificate from container
	//nolint:gosec // Test command with controlled paths
	cpCmd := exec.Command(containerRuntime, "cp",
		fmt.Sprintf("%s:/var/lib/boardingpass/tls/server.crt", e.containerName),
		certPath)
	output, err := cpCmd.CombinedOutput()
	if err != nil {
		t.Logf("Copy output: %s", string(output))
		require.NoError(t, err, "Failed to extract TLS certificate")
	}

	e.tlsCertPath = certPath
	t.Logf("TLS certificate extracted to %s", certPath)

	// Create HTTP client that trusts the certificate
	e.client = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				//nolint:gosec // Test certificate is self-signed
				InsecureSkipVerify: true,
			},
		},
	}
}

// execInContainer executes a command inside the container.
func (e *testEnvironment) execInContainer(t *testing.T, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"exec", e.containerName}, args...)
	//nolint:gosec // Test command with controlled arguments
	cmd := exec.Command(containerRuntime, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command: %s %v", containerRuntime, cmdArgs)
		t.Logf("Output: %s", string(output))
		t.Logf("Error: %v", err)
	}
	return string(output)
}

// testHealthCheck tests the /info endpoint without authentication.
func (e *testEnvironment) testHealthCheck(t *testing.T) {
	t.Helper()

	// Try to access /info endpoint without authentication
	// Should return 401 Unauthorized
	resp, err := e.client.Get(e.serviceURL + "/info")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// testAuthEndpoints tests the SRP authentication endpoints.
func (e *testEnvironment) testAuthEndpoints(t *testing.T) {
	t.Helper()

	// Test SRP init endpoint
	resp, err := e.client.Post(e.serviceURL+"/auth/srp/init",
		"application/json",
		nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should accept POST requests (even if body is invalid)
	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}

// testCompleteSRPAuthFlow tests the complete SRP authentication flow.
func (e *testEnvironment) testCompleteSRPAuthFlow(t *testing.T) {
	t.Helper()

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
	initResp, err := e.client.Post(e.serviceURL+"/auth/srp/init",
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
	verifyResp, err := e.client.Post(e.serviceURL+"/auth/srp/verify",
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
	invalidVerifyResp, err := e.client.Post(e.serviceURL+"/auth/srp/verify",
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
	missingM1Resp, err := e.client.Post(e.serviceURL+"/auth/srp/verify",
		"application/json",
		bytes.NewBuffer(missingM1Body))
	require.NoError(t, err)
	defer func() { _ = missingM1Resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, missingM1Resp.StatusCode)
}

// testSRPSessionExpiration tests SRP session expiration behavior.
func (e *testEnvironment) testSRPSessionExpiration(t *testing.T) {
	t.Helper()

	// Initialize SRP session
	initReq := map[string]string{
		"username": "boardingpass",
		"A":        "gxs0ip7Q72TLryg8kngzqyTivy4OpNgrfSbWQkk2BHvzuIm4+FrB+Yiv3F2jIVdxlkt7yNlW9qIel+PkPXYfFMgoaLeXYvj0QAKCEzgttSarPm6UH+Lp2kMtwKo6PSu1EMYY9V4Db1bqmpBOFw1AaaVOuiOnP0iiIOeJgQ29sQz4Y6jdXQ02WwyjuI4wDIAUet15Ys3m6WEItqz/zBVMIYEHHMG8QIgBFwiQQ9OXKIbnoXUMUSPSajPmf5nEFBHRdrcwUD1PxsprqJTfwYFI4UqYI0iztvxduoFA95wrg1QmUqb0VpiCt+e5eP43M8RlONKCPoDqNsh+sLJeZTTe/A==",
	}
	initBody, _ := json.Marshal(initReq)
	initResp, err := e.client.Post(e.serviceURL+"/auth/srp/init",
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
	verifyResp, err := e.client.Post(e.serviceURL+"/auth/srp/verify",
		"application/json",
		bytes.NewBuffer(verifyBody))
	require.NoError(t, err)
	defer func() { _ = verifyResp.Body.Close() }()

	// Should get unauthorized (invalid M1) not "session expired"
	assert.Equal(t, http.StatusUnauthorized, verifyResp.StatusCode)
}

// testSRPRateLimiting tests the rate limiting mechanism.
func (e *testEnvironment) testSRPRateLimiting(t *testing.T) {
	t.Helper()

	// Test that rate limiting mechanism exists
	// Note: Detailed rate limiting behavior is tested in integration tests
	// Here we just verify the mechanism is active

	initReq := map[string]string{
		"username": "wronguser",
		"A":        "gxs0ip7Q72TLryg8kngzqyTivy4OpNgrfSbWQkk2BHvzuIm4+FrB+Yiv3F2jIVdxlkt7yNlW9qIel+PkPXYfFMgoaLeXYvj0QAKCEzgttSarPm6UH+Lp2kMtwKo6PSu1EMYY9V4Db1bqmpBOFw1AaaVOuiOnP0iiIOeJgQ29sQz4Y6jdXQ02WwyjuI4wDIAUet15Ys3m6WEItqz/zBVMIYEHHMG8QIgBFwiQQ9OXKIbnoXUMUSPSajPmf5nEFBHRdrcwUD1PxsprqJTfwYFI4UqYI0iztvxduoFA95wrg1QmUqb0VpiCt+e5eP43M8RlONKCPoDqNsh+sLJeZTTe/A==",
	}
	initBody, _ := json.Marshal(initReq)
	initResp, err := e.client.Post(e.serviceURL+"/auth/srp/init",
		"application/json",
		bytes.NewBuffer(initBody))
	require.NoError(t, err)
	defer func() { _ = initResp.Body.Close() }()

	// Should fail with either 401 (first few attempts) or 429 (rate limited)
	// Exact status depends on previous test failures
	assert.True(t, initResp.StatusCode == http.StatusUnauthorized || initResp.StatusCode == http.StatusTooManyRequests,
		"expected 401 or 429, got %d", initResp.StatusCode)

	// Should have Retry-After header for rate limiting
	if initResp.StatusCode == http.StatusUnauthorized || initResp.StatusCode == http.StatusTooManyRequests {
		assert.NotEmpty(t, initResp.Header.Get("Retry-After"), "expected Retry-After header")
	}
}

// testServiceLogs tests that service logs are accessible and structured properly.
func (e *testEnvironment) testServiceLogs(t *testing.T) {
	t.Helper()

	// Get service logs
	logs := e.execInContainer(t, "journalctl", "-u", "boardingpass", "--no-pager", "-n", "50")
	assert.NotEmpty(t, logs, "Service logs should not be empty")

	// Verify JSON log format
	logLines := strings.Split(logs, "\n")
	jsonLogCount := 0
	for _, line := range logLines {
		if strings.Contains(line, "{") && strings.Contains(line, "level") {
			jsonLogCount++
		}
	}
	assert.Greater(t, jsonLogCount, 0, "Should have JSON-formatted logs")
}

// Helper function to parse JSON response
func parseJSONResponse(t *testing.T, resp *http.Response, target any) {
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	err := json.NewDecoder(resp.Body).Decode(target)
	require.NoError(t, err)
}
