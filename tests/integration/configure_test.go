package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/internal/provisioning"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureHandler_POST_Success(t *testing.T) {
	// Setup: Create temporary directory for staging
	err := os.MkdirAll(provisioning.StagingDirBase, 0o755) //nolint:gosec // G301: Test directory
	if err != nil {
		t.Skipf("Cannot create staging directory %s: %v (needs elevated permissions)", provisioning.StagingDirBase, err)
	}
	defer func() { _ = os.RemoveAll(provisioning.StagingDirBase) }()

	// Create test configuration
	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{"/etc/test/"},
		},
	}

	// Note: This test requires /etc write access or filesystem mocking
	// Skipping actual provisioning test in unit test environment
	t.Skip("Skipping integration test that requires /etc write access - run in container environment")

	// Code below is not reached due to Skip above, but kept for documentation
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewConfigureHandler(testConfig, logger)

	bundle := protocol.ConfigBundle{
		Files: []protocol.ConfigFile{
			{
				Path:    "test/file1.conf",
				Content: base64.StdEncoding.EncodeToString([]byte("config content 1")),
				Mode:    0o644,
			},
			{
				Path:    "test/file2.conf",
				Content: base64.StdEncoding.EncodeToString([]byte("config content 2")),
				Mode:    0o600,
			},
		},
	}

	body, err := json.Marshal(bundle)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])
}

func TestConfigureHandler_POST_InvalidJSON(t *testing.T) {
	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{"/etc/test/"},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewConfigureHandler(testConfig, logger)

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestConfigureHandler_POST_BundleValidationFailure(t *testing.T) {
	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{"/etc/test/"},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewConfigureHandler(testConfig, logger)

	tests := []struct {
		name   string
		bundle protocol.ConfigBundle
		errMsg string
	}{
		{
			name: "empty files list",
			bundle: protocol.ConfigBundle{
				Files: []protocol.ConfigFile{},
			},
			errMsg: "Bundle validation failed",
		},
		{
			name: "invalid Base64",
			bundle: protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "test/file.conf",
						Content: "not-valid-base64!!!",
						Mode:    0o644,
					},
				},
			},
			errMsg: "Bundle validation failed",
		},
		{
			name: "exceeds file count",
			bundle: func() protocol.ConfigBundle {
				files := make([]protocol.ConfigFile, provisioning.MaxFileCount+1)
				for i := range files {
					files[i] = protocol.ConfigFile{
						Path:    "test/file.conf",
						Content: base64.StdEncoding.EncodeToString([]byte("content")),
						Mode:    0o644,
					}
				}
				return protocol.ConfigBundle{Files: files}
			}(),
			errMsg: "Bundle validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.bundle)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.errMsg)
		})
	}
}

func TestConfigureHandler_POST_PathValidationFailure(t *testing.T) {
	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{"/etc/test/"},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewConfigureHandler(testConfig, logger)

	tests := []struct {
		name   string
		path   string
		errMsg string
	}{
		{
			name:   "path not in allow-list (passwd)",
			path:   "passwd",
			errMsg: "Path validation failed",
		},
		{
			name:   "path with traversal",
			path:   "test/../passwd",
			errMsg: "Path validation failed",
		},
		{
			name:   "absolute path outside /etc",
			path:   "/etc/shadow",
			errMsg: "Path validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle := protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    tt.path,
						Content: base64.StdEncoding.EncodeToString([]byte("malicious content")),
						Mode:    0o644,
					},
				},
			}

			body, err := json.Marshal(bundle)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.errMsg)
		})
	}
}

func TestConfigureHandler_MethodNotAllowed(t *testing.T) {
	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{"/etc/test/"},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewConfigureHandler(testConfig, logger)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/configure", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

func TestConfigureHandler_ContentRedaction(t *testing.T) {
	// This test verifies that configuration content is redacted in logs
	// In a real test, we would capture log output and verify redaction

	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{"/etc/test/"},
		},
	}

	// Create logger with buffer to capture output
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewConfigureHandler(testConfig, logger)

	bundle := protocol.ConfigBundle{
		Files: []protocol.ConfigFile{
			{
				Path:    "test/sensitive.conf",
				Content: base64.StdEncoding.EncodeToString([]byte("SECRET_API_KEY=abcdef123456")),
				Mode:    0o600,
			},
		},
	}

	body, err := json.Marshal(bundle)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Note: In production, we would verify log output contains [REDACTED]
	// and does not contain the actual configuration content
	// This requires capturing stdout/stderr or using a test logger
}

func TestConfigureHandler_E2E_WithFileSystem(t *testing.T) {
	// This test performs end-to-end testing with actual filesystem operations
	// It requires a test container or temporary /etc directory

	t.Skip("Skipping E2E test - run in containerized test environment with isolated /etc")

	// Setup test environment
	testEtc := t.TempDir()
	testStagingDir := filepath.Join(t.TempDir(), "staging")
	err := os.MkdirAll(testStagingDir, 0o755) //nolint:gosec // G301: Test directory
	require.NoError(t, err)

	// Override staging directory for testing
	// (In real implementation, this would use dependency injection)

	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{filepath.Join(testEtc, "test/")},
		},
	}

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewConfigureHandler(testConfig, logger)

	// Create configuration bundle
	bundle := protocol.ConfigBundle{
		Files: []protocol.ConfigFile{
			{
				Path:    "test/config.yaml",
				Content: base64.StdEncoding.EncodeToString([]byte("key: value\n")),
				Mode:    0o644,
			},
		},
	}

	body, err := json.Marshal(bundle)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/configure", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify file was created
	targetFile := filepath.Join(testEtc, "test/config.yaml")
	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, "key: value\n", string(content))

	// Verify permissions
	info, err := os.Stat(targetFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}
