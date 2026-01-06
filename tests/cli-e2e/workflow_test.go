// Package clie2e provides end-to-end tests for the boarding CLI tool.
//
// These tests verify the complete CLI workflow in a containerized environment,
// testing against a real BoardingPass service running in a UBI9 init container.
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
	// containerImage is the UBI9 init container image
	containerImage = "registry.redhat.io/ubi9/ubi-init:latest"
	// servicePort is the port exposed by the BoardingPass service
	servicePort = 8443
)

// containerRuntime holds the detected container runtime (podman or docker)
var containerRuntime string

func init() {
	// Detect container runtime
	if _, err := exec.LookPath("podman"); err == nil {
		containerRuntime = "podman"
	} else if _, err := exec.LookPath("docker"); err == nil {
		containerRuntime = "docker"
	}
}

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

	if containerRuntime == "" {
		t.Skip("No container runtime (podman/docker) available")
	}

	// Check if running in CI without container support
	if os.Getenv("CI") == "true" && os.Getenv("E2E_SKIP_CONTAINER") == "true" {
		t.Skip("Skipping container-based E2E test in CI")
	}

	ctx := context.Background()

	// Create test environment
	env := newTestEnvironment(ctx, t)
	defer env.cleanup()

	// Build binaries
	env.buildBinaries(t)

	// Start containerized BoardingPass service
	env.startService(t)

	// Wait for service to be ready
	env.waitForService(t)

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
	containerID   string
	containerName string
	serviceHost   string
	cliPath       string
	servicePath   string
	configDir     string
	cacheDir      string
	tlsCertPath   string // Path to TLS certificate for CLI to trust
}

// newTestEnvironment creates a new test environment.
func newTestEnvironment(ctx context.Context, t *testing.T) *testEnvironment {
	tmpDir, err := os.MkdirTemp("", "cli-e2e-*")
	require.NoError(t, err)

	containerName := fmt.Sprintf("boardingpass-e2e-%d", time.Now().Unix())

	// Create config and cache directories
	configDir := filepath.Join(tmpDir, "config")
	cacheDir := filepath.Join(tmpDir, "cache")
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
	// Stop and remove container
	if e.containerID != "" {
		//nolint:gosec // Test command with variable container ID is required
		stopCmd := exec.Command(containerRuntime, "stop", e.containerID)
		_ = stopCmd.Run()

		//nolint:gosec // Test command with variable container ID is required
		rmCmd := exec.Command(containerRuntime, "rm", "-f", e.containerID)
		_ = rmCmd.Run()
	}

	// Remove temporary directory
	_ = os.RemoveAll(e.tmpDir)
}

// buildBinaries builds the BoardingPass service and CLI binaries.
func (e *testEnvironment) buildBinaries(t *testing.T) {
	t.Helper()

	binDir := filepath.Join(e.tmpDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o750))

	// Build service binary
	e.servicePath = filepath.Join(binDir, "boardingpass")
	t.Logf("Building service binary: %s", e.servicePath)
	//nolint:gosec // Build command with variable path is required for testing
	buildCmd := exec.Command("go", "build",
		"-o", e.servicePath,
		"../../cmd/boardingpass")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", string(buildOutput))
		require.NoError(t, err, "Failed to build service binary")
	}

	// Build CLI binary
	e.cliPath = filepath.Join(binDir, "boarding")
	t.Logf("Building CLI binary: %s", e.cliPath)
	//nolint:gosec // Build command with variable path is required for testing
	buildCLICmd := exec.Command("go", "build",
		"-o", e.cliPath,
		"../../cmd/boarding")
	buildCLIOutput, err := buildCLICmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", string(buildCLIOutput))
		require.NoError(t, err, "Failed to build CLI binary")
	}

	t.Logf("Binaries built successfully")
}

// startService starts the BoardingPass service in a UBI9 init container.
func (e *testEnvironment) startService(t *testing.T) {
	t.Helper()

	// Create container configuration
	serviceDir := filepath.Join(e.tmpDir, "service")
	require.NoError(t, os.MkdirAll(serviceDir, 0o750))

	// Generate TLS certificates for the service
	tlsDir := filepath.Join(serviceDir, "tls")
	require.NoError(t, os.MkdirAll(tlsDir, 0o750))
	tlsCertPath := filepath.Join(tlsDir, "server.crt")
	tlsKeyPath := filepath.Join(tlsDir, "server.key")

	t.Logf("Generating TLS certificates...")
	err := e.generateTestCertificate(tlsCertPath, tlsKeyPath)
	require.NoError(t, err, "Failed to generate TLS certificates")

	// Store cert path for CLI to use
	e.tlsCertPath = tlsCertPath

	// Create test configuration with TLS paths for container
	configContent := fmt.Sprintf(`service:
  inactivity_timeout: "10m"
  session_ttl: "30m"
  sentinel_file: "/tmp/boardingpass-issued"

transports:
  ethernet:
    enabled: true
    interfaces: []
    address: "0.0.0.0"
    port: %d
    tls_cert: "/opt/boardingpass/tls/server.crt"
    tls_key: "/opt/boardingpass/tls/server.key"

commands:
  - id: "echo-test"
    path: "/bin/echo"
    args: ["success"]
  - id: "systemctl-status"
    path: "/bin/systemctl"
    args: ["status", "sshd"]

logging:
  level: "info"
  format: "json"

paths:
  allow_list:
    - "/etc/boardingpass/"
`, servicePort)

	configPath := filepath.Join(serviceDir, "config.yaml")
	//nolint:gosec // Test config file permissions are acceptable
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o600))

	// Create password generator script
	passwordGenPath := filepath.Join(serviceDir, "password-gen.sh")
	passwordGenScript := "#!/bin/bash\necho 'test-password-123'"
	//nolint:gosec // Test script file permissions are acceptable
	require.NoError(t, os.WriteFile(passwordGenPath, []byte(passwordGenScript), 0o755))

	// Create verifier configuration
	verifierPath := filepath.Join(serviceDir, "verifier")
	verifierContent := `{
  "username": "admin",
  "salt": "dGVzdHNhbHQ=",
  "password_generator": "/opt/boardingpass/config/password-gen.sh"
}`
	require.NoError(t, os.WriteFile(verifierPath, []byte(verifierContent), 0o600))

	// Pull UBI9 init image if not already present
	t.Logf("Pulling container image: %s", containerImage)
	//nolint:gosec // Container image is a const
	pullCmd := exec.Command(containerRuntime, "pull", containerImage)
	pullOutput, err := pullCmd.CombinedOutput()
	if err != nil {
		t.Logf("Pull output: %s", string(pullOutput))
		// Don't fail if image already exists
		if !strings.Contains(string(pullOutput), "already present") {
			t.Logf("Warning: Failed to pull image (may already exist): %v", err)
		}
	}

	// Start container with systemd
	t.Logf("Starting container: %s", e.containerName)
	//nolint:gosec // Container name and paths are controlled by test
	runArgs := []string{
		"run",
		"-d",
		"--name", e.containerName,
		"-p", fmt.Sprintf("%d:%d", servicePort, servicePort),
		"--tmpfs", "/run",
		"--tmpfs", "/tmp",
		containerImage,
	}

	//nolint:gosec // Container runtime is validated at init time
	runCmd := exec.Command(containerRuntime, runArgs...)
	runOutput, err := runCmd.CombinedOutput()
	if err != nil {
		t.Logf("Run output: %s", string(runOutput))
		require.NoError(t, err, "Failed to start container")
	}

	e.containerID = strings.TrimSpace(string(runOutput))
	t.Logf("Container started: %s", e.containerID)

	// Wait a moment for container to initialize
	time.Sleep(2 * time.Second)

	// Install sudo (required by command executor)
	t.Logf("Installing sudo in container")
	e.execInContainer(t, "dnf", "install", "-y", "sudo")

	// Create directories in container
	t.Logf("Setting up container directories")
	e.execInContainer(t, "mkdir", "-p", "/opt/boardingpass/bin")
	e.execInContainer(t, "mkdir", "-p", "/opt/boardingpass/config")
	e.execInContainer(t, "mkdir", "-p", "/opt/boardingpass/tls")

	// Copy service binary to container
	t.Logf("Copying service binary to container")
	e.copyToContainer(t, e.servicePath, "/opt/boardingpass/bin/boardingpass")
	e.execInContainer(t, "chmod", "+x", "/opt/boardingpass/bin/boardingpass")

	// Copy configuration files to container
	t.Logf("Copying configuration files to container")
	e.copyToContainer(t, configPath, "/opt/boardingpass/config/config.yaml")
	e.copyToContainer(t, verifierPath, "/opt/boardingpass/config/verifier")
	e.copyToContainer(t, passwordGenPath, "/opt/boardingpass/config/password-gen.sh")
	e.execInContainer(t, "chmod", "+x", "/opt/boardingpass/config/password-gen.sh")

	// Copy TLS certificates to container
	t.Logf("Copying TLS certificates to container")
	e.copyToContainer(t, tlsCertPath, "/opt/boardingpass/tls/server.crt")
	e.copyToContainer(t, tlsKeyPath, "/opt/boardingpass/tls/server.key")

	// Start service in background
	t.Logf("Starting BoardingPass service in container")
	startCmd := "/opt/boardingpass/bin/boardingpass -config /opt/boardingpass/config/config.yaml -verifier /opt/boardingpass/config/verifier > /var/log/boardingpass.log 2>&1 &"
	e.execInContainer(t, "bash", "-c", startCmd)

	t.Logf("Service started in container")
}

// execInContainer executes a command inside the container.
func (e *testEnvironment) execInContainer(t *testing.T, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"exec", e.containerID}, args...)
	//nolint:gosec // Test command with controlled arguments
	cmd := exec.Command(containerRuntime, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command output: %s", string(output))
		t.Logf("Command failed: %v", err)
	}
	return string(output)
}

// copyToContainer copies a file from the host to the container.
func (e *testEnvironment) copyToContainer(t *testing.T, srcPath, dstPath string) {
	t.Helper()

	//nolint:gosec // Test command with controlled paths
	cpCmd := exec.Command(containerRuntime, "cp", srcPath, e.containerID+":"+dstPath)
	output, err := cpCmd.CombinedOutput()
	if err != nil {
		t.Logf("Copy output: %s", string(output))
		require.NoError(t, err, "Failed to copy %s to container", srcPath)
	}
}

// generateTestCertificate generates a self-signed TLS certificate for testing.
func (e *testEnvironment) generateTestCertificate(certPath, keyPath string) error {
	// Create a temporary config file for openssl with SAN extension
	configContent := `[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
`

	configPath := filepath.Join(filepath.Dir(certPath), "openssl.cnf")
	//nolint:gosec // Test config file
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		return fmt.Errorf("failed to create openssl config: %w", err)
	}
	defer func() { _ = os.Remove(configPath) }()

	// Generate self-signed certificate with SAN
	//nolint:gosec // Test certificate generation with controlled paths
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", certPath,
		"-days", "1",
		"-nodes",
		"-config", configPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w (output: %s)", err, string(output))
	}

	return nil
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
		logs := e.execInContainer(t, "tail", "-20", "/var/log/boardingpass.log")
		if strings.Contains(logs, "panic") || strings.Contains(logs, "fatal") {
			t.Fatalf("Service failed to start. Logs:\n%s", logs)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Dump logs for debugging
	logs := e.execInContainer(t, "cat", "/var/log/boardingpass.log")
	t.Fatalf("Service did not start within timeout. Logs:\n%s", logs)
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

	// Also create config directories in the expected locations
	userConfigDir := filepath.Join(e.tmpDir, ".config", "boardingpass")
	userCacheDir := filepath.Join(e.tmpDir, ".cache", "boardingpass")
	require.NoError(t, os.MkdirAll(userConfigDir, 0o750))
	require.NoError(t, os.MkdirAll(userCacheDir, 0o750))

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

	// Run authentication command
	// Note: The password should match what the password generator script returns
	// Note: CA cert is passed via BOARDING_CA_CERT environment variable
	_, stderr, exitCode := e.runCLI(t,
		"pass",
		"--username", "admin",
		"--password", "test-password-123",
	)

	// Check that authentication succeeded
	assert.Equal(t, 0, exitCode, "Authentication should succeed")
	assert.Contains(t, stderr, "Authentication successful", "Should show success message")
	assert.Contains(t, stderr, "Session token saved", "Should confirm token saved")

	// Verify session token was created
	sessionTokenPath := filepath.Join(e.tmpDir, ".cache", "boardingpass")
	files, err := os.ReadDir(sessionTokenPath)
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

	// Get session token path before completing
	sessionTokenPath := filepath.Join(e.tmpDir, ".cache", "boardingpass")

	// Run complete command
	_, stderr, exitCode := e.runCLI(t, "complete")
	assert.Equal(t, 0, exitCode, "Complete command should succeed")
	assert.Contains(t, stderr, "Provisioning completed successfully", "Should show success message")

	// Verify session token was deleted
	files, err := os.ReadDir(sessionTokenPath)
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
