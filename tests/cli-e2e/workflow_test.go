// Package clie2e provides end-to-end tests for the boarding CLI tool.
//
// These tests verify the complete CLI workflow in a containerized environment,
// testing against a real BoardingPass service deployed via make deploy.
package clie2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	// testTimeout is the maximum time to wait for container startup
	testTimeout = 60 * time.Second
	// servicePort is the port exposed by the BoardingPass service
	servicePort = 8443
	// containerRuntime is the container runtime to use (podman or docker)
	containerRuntime = "podman"
)

// TestCLIWorkflow tests the complete CLI provisioning workflow.
//
// This test verifies the full user journey:
// 1. Authenticate with 'boarding pass'
// 2. Query system info with 'boarding info'
// 3. Query network interfaces with 'boarding connections'
// 4. Upload configuration with 'boarding load'
// 5. Execute command with 'boarding command'
// 6. Complete session with 'boarding complete'
func TestCLIWorkflow(t *testing.T) {
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

	// Create test environment
	env := newTestEnvironment(ctx, t)
	defer env.cleanup()

	// Build CLI binary
	env.buildCLI(t)

	// Deploy service using make deploy
	env.deployService(t)

	// Wait for service to be ready
	env.waitForService(t)

	// Extract TLS certificate
	env.extractTLSCertificate(t)

	// Get the password from the container
	env.getPassword(t)

	// Run the full CLI workflow
	t.Run("AuthenticateWithPass", func(t *testing.T) {
		env.testPassCommand(t)
	})

	t.Run("QuerySystemInfo", func(t *testing.T) {
		env.testInfoCommand(t)
	})

	t.Run("QueryNetworkInterfaces", func(t *testing.T) {
		env.testConnectionsCommand(t)
	})

	t.Run("UploadConfiguration", func(t *testing.T) {
		env.testLoadCommand(t)
	})

	t.Run("ExecuteCommand", func(t *testing.T) {
		env.testCommandCommand(t)
	})

	t.Run("CompleteSession", func(t *testing.T) {
		env.testCompleteCommand(t)
	})
}

// testEnvironment holds all the state for an E2E test run.
type testEnvironment struct {
	ctx           context.Context
	t             *testing.T
	tmpDir        string
	containerName string
	serviceHost   string
	cliPath       string
	configDir     string
	cacheDir      string
	tlsCertPath   string // Path to TLS certificate for CLI to trust
	password      string // Password retrieved from the container
}

// newTestEnvironment creates a new test environment.
func newTestEnvironment(ctx context.Context, t *testing.T) *testEnvironment {
	tmpDir, err := os.MkdirTemp("", "cli-e2e-*")
	require.NoError(t, err)

	// Use timestamp for unique container name
	containerName := fmt.Sprintf("boardingpass-cli-e2e-%d", time.Now().Unix())

	// Create config and cache directories
	configDir := filepath.Join(tmpDir, ".config", "boardingpass")
	cacheDir := filepath.Join(tmpDir, ".cache", "boardingpass")
	require.NoError(t, os.MkdirAll(configDir, 0o750))
	require.NoError(t, os.MkdirAll(cacheDir, 0o750))

	return &testEnvironment{
		ctx:           ctx,
		t:             t,
		tmpDir:        tmpDir,
		containerName: containerName,
		serviceHost:   "127.0.0.1",
		configDir:     configDir,
		cacheDir:      cacheDir,
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

// buildCLI builds the boarding CLI binary.
func (e *testEnvironment) buildCLI(t *testing.T) {
	t.Helper()

	binDir := filepath.Join(e.tmpDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o750))

	// Build CLI binary
	e.cliPath = filepath.Join(binDir, "boarding")
	t.Logf("Building CLI binary: %s", e.cliPath)
	//nolint:gosec // Build command with variable path is required for testing
	buildCLICmd := exec.Command("go", "build",
		"-o", e.cliPath,
		"../../cmd/boarding")
	// buildCLICmd.Dir is not set, so it runs from the test directory (tests/cli-e2e)
	// From there, ../../cmd/boarding correctly points to the cmd/boarding package
	buildCLIOutput, err := buildCLICmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", string(buildCLIOutput))
		require.NoError(t, err, "Failed to build CLI binary")
	}

	t.Logf("CLI binary built successfully")
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
		if strings.Contains(logs, "panic") || strings.Contains(logs, "fatal") {
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
}

// getPassword retrieves the password from the container by running the password generator.
func (e *testEnvironment) getPassword(t *testing.T) {
	t.Helper()
	t.Logf("Retrieving password from container...")

	// Run the password generator script in the container
	output := e.execInContainer(t, "/usr/lib/boardingpass/generators/primary_mac")
	e.password = strings.TrimSpace(output)

	require.NotEmpty(t, e.password, "Password should not be empty")
	t.Logf("Retrieved password from container (length: %d)", len(e.password))
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

// runCLI runs the boarding CLI with the given arguments.
func (e *testEnvironment) runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	// Set environment variables for CLI
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("BOARDING_HOST=%s", e.serviceHost),
		fmt.Sprintf("BOARDING_PORT=%d", servicePort),
		fmt.Sprintf("BOARDING_CA_CERT=%s", e.tlsCertPath), // Trust our test certificate
		fmt.Sprintf("HOME=%s", e.tmpDir),                  // Use tmp dir for config/cache
	)

	var stdoutBuf, stderrBuf bytes.Buffer

	//nolint:gosec // CLI path is controlled by test
	cmd := exec.Command(e.cliPath, args...)
	cmd.Env = env
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Logf("CLI execution error: %v", err)
		}
	}

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	t.Logf("CLI command: %s %v", e.cliPath, args)
	t.Logf("Exit code: %d", exitCode)
	if stdout != "" {
		t.Logf("Stdout:\n%s", stdout)
	}
	if stderr != "" {
		t.Logf("Stderr:\n%s", stderr)
	}

	return stdout, stderr, exitCode
}

// testPassCommand tests the 'boarding pass' authentication command.
func (e *testEnvironment) testPassCommand(t *testing.T) {
	t.Helper()

	// Run authentication command with the password from the container
	_, stderr, exitCode := e.runCLI(t,
		"pass",
		"--username", "boardingpass",
		"--password", e.password,
	)

	// Check that authentication succeeded
	assert.Equal(t, 0, exitCode, "Authentication should succeed")
	assert.Contains(t, stderr, "Authentication successful", "Should show success message")
	assert.Contains(t, stderr, "Session token saved", "Should confirm token saved")

	// Verify session token was created
	files, err := os.ReadDir(e.cacheDir)
	require.NoError(t, err)

	found := false
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "session-") && strings.HasSuffix(f.Name(), ".token") {
			found = true
			// Verify permissions
			info, err := f.Info()
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
				"Session token should have 0600 permissions")
			break
		}
	}
	assert.True(t, found, "Session token file should be created")
}

// testInfoCommand tests the 'boarding info' command.
func (e *testEnvironment) testInfoCommand(t *testing.T) {
	t.Helper()

	// Test YAML output (default)
	stdout, stderr, exitCode := e.runCLI(t, "info")
	assert.Equal(t, 0, exitCode, "Info command should succeed")
	assert.Empty(t, stderr, "Should have no errors")
	assert.Contains(t, stdout, "cpu:", "Should contain CPU info in YAML format")
	assert.Contains(t, stdout, "os:", "Should contain OS info in YAML format")

	// Test JSON output
	stdout, stderr, exitCode = e.runCLI(t, "info", "--output", "json")
	assert.Equal(t, 0, exitCode, "Info command with JSON should succeed")
	assert.Empty(t, stderr, "Should have no errors")

	// Parse JSON to verify it's valid
	var infoData map[string]any
	err := json.Unmarshal([]byte(stdout), &infoData)
	assert.NoError(t, err, "Output should be valid JSON")
	assert.NotNil(t, infoData["cpu"], "Should contain CPU info")
	assert.NotNil(t, infoData["os"], "Should contain OS info")
}

// testConnectionsCommand tests the 'boarding connections' command.
func (e *testEnvironment) testConnectionsCommand(t *testing.T) {
	t.Helper()

	// Test YAML output (default)
	stdout, stderr, exitCode := e.runCLI(t, "connections")
	assert.Equal(t, 0, exitCode, "Connections command should succeed")
	assert.Empty(t, stderr, "Should have no errors")
	assert.Contains(t, stdout, "interfaces:", "Should contain interfaces in YAML format")

	// Test JSON output
	stdout, stderr, exitCode = e.runCLI(t, "connections", "--output", "json")
	assert.Equal(t, 0, exitCode, "Connections command with JSON should succeed")
	assert.Empty(t, stderr, "Should have no errors")

	// Parse JSON to verify it's valid
	var networkData map[string]any
	err := json.Unmarshal([]byte(stdout), &networkData)
	assert.NoError(t, err, "Output should be valid JSON")
	assert.NotNil(t, networkData["interfaces"], "Should contain interfaces")
}

// testLoadCommand tests the 'boarding load' configuration upload command.
func (e *testEnvironment) testLoadCommand(t *testing.T) {
	t.Helper()

	// Create test configuration directory with boardingpass subdirectory
	// Files will be uploaded to /etc/<relative-path>, so we need boardingpass/ prefix
	// to match the allow-list: /etc/boardingpass/
	configDir := filepath.Join(e.tmpDir, "test-config", "boardingpass")
	require.NoError(t, os.MkdirAll(configDir, 0o750))

	// Create some test files
	testFiles := map[string]string{
		"config.yaml": "key: value\n",
		"script.sh":   "#!/bin/bash\necho 'test'\n",
		"data.json":   `{"test": true}`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(configDir, filename)
		//nolint:gosec // Test file permissions are acceptable
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
	}

	// Run load command (pass parent dir so relative paths include "boardingpass/")
	parentDir := filepath.Dir(configDir)
	_, stderr, exitCode := e.runCLI(t, "load", parentDir)
	assert.Equal(t, 0, exitCode, "Load command should succeed")
	assert.Contains(t, stderr, "Configuration uploaded successfully",
		"Should show success message")
}

// testCommandCommand tests the 'boarding command' execution command.
func (e *testEnvironment) testCommandCommand(t *testing.T) {
	t.Helper()

	// Execute allowed command
	stdout, _, exitCode := e.runCLI(t, "command", "echo-test")
	assert.Equal(t, 0, exitCode, "Command execution should succeed")
	assert.Contains(t, stdout, "success", "Should contain command output")

	// Try to execute non-allowed command (should fail)
	_, stderr, exitCode := e.runCLI(t, "command", "rm -rf /")
	assert.NotEqual(t, 0, exitCode, "Non-allowed command should fail")
	assert.NotEmpty(t, stderr, "Should have error message")
}

// testCompleteCommand tests the 'boarding complete' session termination command.
func (e *testEnvironment) testCompleteCommand(t *testing.T) {
	t.Helper()

	// Run complete command
	_, stderr, exitCode := e.runCLI(t, "complete")
	assert.Equal(t, 0, exitCode, "Complete command should succeed")
	assert.Contains(t, stderr, "Provisioning completed successfully", "Should show success message")

	// Verify session token was deleted
	files, err := os.ReadDir(e.cacheDir)
	require.NoError(t, err)

	for _, f := range files {
		assert.False(t,
			strings.HasPrefix(f.Name(), "session-") && strings.HasSuffix(f.Name(), ".token"),
			"Session token should be deleted")
	}

	// Verify subsequent commands fail without authentication
	_, stderr, exitCode = e.runCLI(t, "info")
	assert.NotEqual(t, 0, exitCode, "Commands should fail after session completion")
	assert.Contains(t, stderr, "no active session", "Should require re-authentication")
}
