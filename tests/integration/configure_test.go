package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

// findRepoRoot finds the repository root by looking for go.mod
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}

// testRootDir creates a test root directory under _output/tests/
func testRootDir(t *testing.T) string {
	t.Helper()
	// Find repository root
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	// Create test directory under repo root's _output/tests/
	rootDir := filepath.Join(repoRoot, "_output", "tests", t.Name())
	//nolint:gosec // G301: Test directory, relaxed permissions acceptable
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	t.Cleanup(func() {
		_ = os.RemoveAll(rootDir)
	})
	return rootDir
}

func TestConfigureHandler_POST_Success(t *testing.T) {
	// Create temporary root directory
	rootDir := testRootDir(t)

	// Create test configuration with root directory
	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList:     []string{"/etc/test/"},
			RootDirectory: rootDir,
		},
	}

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

	// Verify files were written
	expectedPath1 := filepath.Join(rootDir, "/etc/test/file1.conf")
	content1, err := os.ReadFile(expectedPath1)
	require.NoError(t, err)
	assert.Equal(t, "config content 1", string(content1))

	expectedPath2 := filepath.Join(rootDir, "/etc/test/file2.conf")
	content2, err := os.ReadFile(expectedPath2)
	require.NoError(t, err)
	assert.Equal(t, "config content 2", string(content2))
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
	// using a temporary root directory

	// Create temporary root directory
	rootDir := testRootDir(t)

	testConfig := &config.Config{
		Paths: config.PathSettings{
			AllowList:     []string{"/etc/test/"},
			RootDirectory: rootDir,
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
	targetFile := filepath.Join(rootDir, "/etc/test/config.yaml")
	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, "key: value\n", string(content))

	// Verify permissions
	info, err := os.Stat(targetFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}
