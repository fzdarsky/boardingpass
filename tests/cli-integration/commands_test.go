package integration_test

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/cli/client"
	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/fzdarsky/boardingpass/internal/cli/session"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseServerAddr parses a server address into host and port.
func parseServerAddr(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return host, port
}

// TestInfoCommand tests the info command integration.
func TestInfoCommand(t *testing.T) {
	t.Skip("TODO: Implement test with proper TLS certificate handling")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/info", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-token-123", r.Header.Get("Authorization"))

		// Return mock system info
		info := protocol.SystemInfo{
			TPM: protocol.TPMInfo{
				Present: true,
			},
			Board: protocol.BoardInfo{
				Manufacturer: "Test Manufacturer",
				Model:        "Test Board",
				Serial:       "TEST123",
			},
			CPU: protocol.CPUInfo{
				Architecture: "x86_64",
			},
			OS: protocol.OSInfo{
				Distribution: "Ubuntu",
				Version:      "22.04",
				FIPSEnabled:  false,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	}))
	defer server.Close()

	// This would be the actual test implementation - skipped due to TLS complexity
	// In real tests, we would configure the client to trust the test server certificate
	var info *protocol.SystemInfo
	var err error
	require.NoError(t, err)

	// Verify response
	assert.True(t, info.TPM.Present)
	assert.Equal(t, "Test Manufacturer", info.Board.Manufacturer)
	assert.Equal(t, "Test Board", info.Board.Model)
	assert.Equal(t, "x86_64", info.CPU.Architecture)
	assert.Equal(t, "Ubuntu", info.OS.Distribution)
	assert.False(t, info.OS.FIPSEnabled)
}

// TestConnectionsCommand tests the connections command integration.
func TestConnectionsCommand(t *testing.T) {
	t.Skip("TODO: Implement test with proper TLS certificate handling")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/network", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-token-456", r.Header.Get("Authorization"))

		// Return mock network config
		network := protocol.NetworkConfig{
			Interfaces: []protocol.NetworkInterface{
				{
					Name:       "eth0",
					MACAddress: "00:11:22:33:44:55",
					LinkState:  "up",
					IPAddresses: []protocol.IPAddress{
						{
							IP:     "192.168.1.100",
							Prefix: 24,
							Family: "inet",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(network)
	}))
	defer server.Close()

	// Setup test environment
	setupTestEnv(t)

	// Parse server URL
	host, port := parseServerAddr(t, server.Listener.Addr().String())

	// Create mock config
	cfg := &config.Config{
		Host: host,
		Port: port,
	}

	// Create client
	apiClient, err := client.NewClient(cfg)
	require.NoError(t, err)

	// Set session token
	apiClient.SetSessionToken("test-token-456")

	// Get network config
	network, err := apiClient.GetNetwork()
	require.NoError(t, err)

	// Verify response
	require.Len(t, network.Interfaces, 1)
	assert.Equal(t, "eth0", network.Interfaces[0].Name)
	assert.Equal(t, "00:11:22:33:44:55", network.Interfaces[0].MACAddress)
	assert.Equal(t, "up", network.Interfaces[0].LinkState)
	require.Len(t, network.Interfaces[0].IPAddresses, 1)
	assert.Equal(t, "192.168.1.100", network.Interfaces[0].IPAddresses[0].IP)
}

// TestLoadCommand tests the load command integration.
func TestLoadCommand(t *testing.T) {
	t.Skip("TODO: Implement test with proper TLS certificate handling")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/configure", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer test-token-789", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var bundle protocol.ConfigBundle
		err := json.NewDecoder(r.Body).Decode(&bundle)
		require.NoError(t, err)

		// Verify bundle
		require.Len(t, bundle.Files, 2)
		assert.Equal(t, "file1.txt", bundle.Files[0].Path)
		assert.Equal(t, "file2.txt", bundle.Files[1].Path)

		// Decode and verify content
		content1, err := base64.StdEncoding.DecodeString(bundle.Files[0].Content)
		require.NoError(t, err)
		assert.Equal(t, "content1", string(content1))

		content2, err := base64.StdEncoding.DecodeString(bundle.Files[1].Content)
		require.NoError(t, err)
		assert.Equal(t, "content2", string(content2))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Setup test environment
	setupTestEnv(t)

	// Parse server URL
	host, port := parseServerAddr(t, server.Listener.Addr().String())

	// Create mock config
	cfg := &config.Config{
		Host: host,
		Port: port,
	}

	// Create client
	apiClient, err := client.NewClient(cfg)
	require.NoError(t, err)

	// Set session token
	apiClient.SetSessionToken("test-token-789")

	// Create config bundle
	bundle := &protocol.ConfigBundle{
		Files: []protocol.ConfigFile{
			{
				Path:    "file1.txt",
				Content: base64.StdEncoding.EncodeToString([]byte("content1")),
				Mode:    0o644,
			},
			{
				Path:    "file2.txt",
				Content: base64.StdEncoding.EncodeToString([]byte("content2")),
				Mode:    0o644,
			},
		},
	}

	// Upload configuration
	err = apiClient.PostConfigure(bundle)
	require.NoError(t, err)
}

// TestLoadCommand_DirectoryScanning tests directory scanning functionality.
func TestLoadCommand_DirectoryScanning(t *testing.T) {
	t.Skip("TODO: Implement test with proper TLS certificate handling")

	// Create temporary directory with test files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config.yaml")
	file2 := filepath.Join(tmpDir, "secret.txt")

	require.NoError(t, os.WriteFile(file1, []byte("test config"), 0o644)) // #nosec G306 - test file, relaxed permissions acceptable
	require.NoError(t, os.WriteFile(file2, []byte("test secret"), 0o600))

	// Create mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bundle protocol.ConfigBundle
		err := json.NewDecoder(r.Body).Decode(&bundle)
		require.NoError(t, err)

		// Verify 2 files were scanned
		assert.Len(t, bundle.Files, 2)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Setup test environment
	setupTestEnv(t)

	// Parse server URL
	host, port := parseServerAddr(t, server.Listener.Addr().String())

	// Create mock config
	cfg := &config.Config{
		Host: host,
		Port: port,
	}

	// Create client
	apiClient, err := client.NewClient(cfg)
	require.NoError(t, err)
	apiClient.SetSessionToken("test-token")

	// Scan and upload
	var files []protocol.ConfigFile
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return err
		}

		files = append(files, protocol.ConfigFile{
			Path:    relPath,
			Content: base64.StdEncoding.EncodeToString(content),
			Mode:    int(info.Mode().Perm()),
		})

		return nil
	})
	require.NoError(t, err)

	bundle := &protocol.ConfigBundle{Files: files}
	err = apiClient.PostConfigure(bundle)
	require.NoError(t, err)
}

// TestCommandExecution tests the command execution integration.
func TestCommandExecution(t *testing.T) {
	t.Skip("TODO: Implement test with proper TLS certificate handling")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/command", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer test-token-cmd", r.Header.Get("Authorization"))

		// Parse request
		var req protocol.CommandRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "show-status", req.ID)

		// Return command response
		resp := protocol.CommandResponse{
			ExitCode: 0,
			Stdout:   "Status: OK\n",
			Stderr:   "",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Setup test environment
	setupTestEnv(t)

	// Parse server URL
	host, port := parseServerAddr(t, server.Listener.Addr().String())

	// Create mock config
	cfg := &config.Config{
		Host: host,
		Port: port,
	}

	// Create client
	apiClient, err := client.NewClient(cfg)
	require.NoError(t, err)

	// Set session token
	apiClient.SetSessionToken("test-token-cmd")

	// Execute command
	resp, err := apiClient.ExecuteCommand("show-status")
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 0, resp.ExitCode)
	assert.Equal(t, "Status: OK\n", resp.Stdout)
	assert.Equal(t, "", resp.Stderr)
}

// TestCommandExecution_NonZeroExitCode tests command execution with non-zero exit code.
func TestCommandExecution_NonZeroExitCode(t *testing.T) {
	t.Skip("TODO: Implement test with proper TLS certificate handling")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return command response with error
		resp := protocol.CommandResponse{
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "Error: command failed\n",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Setup test environment
	setupTestEnv(t)

	// Parse server URL
	host, port := parseServerAddr(t, server.Listener.Addr().String())

	// Create mock config
	cfg := &config.Config{
		Host: host,
		Port: port,
	}

	// Create client
	apiClient, err := client.NewClient(cfg)
	require.NoError(t, err)
	apiClient.SetSessionToken("test-token")

	// Execute command
	resp, err := apiClient.ExecuteCommand("failing-command")
	require.NoError(t, err)

	// Verify non-zero exit code
	assert.Equal(t, 1, resp.ExitCode)
	assert.Equal(t, "Error: command failed\n", resp.Stderr)
}

// TestCompleteCommand tests the complete command integration.
func TestCompleteCommand(t *testing.T) {
	t.Skip("TODO: Implement test with proper TLS certificate handling")

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/complete", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer test-token-complete", r.Header.Get("Authorization"))

		// Return complete response
		message := "Provisioning completed successfully"
		resp := protocol.CompleteResponse{
			Status:       "completed",
			SentinelFile: "/var/run/boardingpass.issued",
			Message:      &message,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Setup test environment
	setupTestEnv(t)

	// Parse server URL
	host, port := parseServerAddr(t, server.Listener.Addr().String())

	// Create mock config
	cfg := &config.Config{
		Host: host,
		Port: port,
	}

	// Create client
	apiClient, err := client.NewClient(cfg)
	require.NoError(t, err)

	// Set session token
	apiClient.SetSessionToken("test-token-complete")

	// Complete provisioning
	resp, err := apiClient.Complete()
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, "completed", resp.Status)
	assert.Equal(t, "/var/run/boardingpass.issued", resp.SentinelFile)
	require.NotNil(t, resp.Message)
	assert.Equal(t, "Provisioning completed successfully", *resp.Message)
}

// TestCompleteCommand_TokenDeletion tests that session token is deleted after completion.
func TestCompleteCommand_TokenDeletion(t *testing.T) {
	// Setup test environment
	setupTestEnv(t)

	// Create session store
	store, err := session.NewStore()
	require.NoError(t, err)

	host := "test.local"
	port := 8443
	token := "test-token-to-delete"

	// Save token
	err = store.Save(host, port, token)
	require.NoError(t, err)

	// Verify token exists
	loaded, err := store.Load(host, port)
	require.NoError(t, err)
	assert.Equal(t, token, loaded)

	// Delete token (simulating completion)
	err = store.Delete(host, port)
	require.NoError(t, err)

	// Verify token is deleted
	loaded, err = store.Load(host, port)
	require.NoError(t, err)
	assert.Equal(t, "", loaded)
}
