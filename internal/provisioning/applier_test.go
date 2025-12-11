package provisioning

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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

func TestNewApplier(t *testing.T) {
	// Create temporary root directory for testing
	rootDir := testRootDir(t)

	validator := NewPathValidator([]string{"/etc/test/"})
	applier, err := NewApplier(validator, rootDir)
	require.NoError(t, err)
	require.NotNil(t, applier)
	assert.NotEmpty(t, applier.tempDir)

	// Verify staging directory was created under root
	expectedStaging := filepath.Join(rootDir, "var/lib/boardingpass/staging")
	_, err = os.Stat(expectedStaging)
	require.NoError(t, err, "staging directory should be created")

	// Cleanup
	_ = applier.Cleanup() // nolint:errcheck
}

func TestNewApplier_NilValidator(t *testing.T) {
	applier, err := NewApplier(nil, "")
	assert.Nil(t, applier)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validator cannot be nil")
}

func TestApplier_Apply_ValidBundle(t *testing.T) {
	// Create temporary root directory for testing
	rootDir := testRootDir(t)

	validator := NewPathValidator([]string{"/etc/test/"})
	applier, err := NewApplier(validator, rootDir)
	require.NoError(t, err)

	bundle := &protocol.ConfigBundle{
		Files: []protocol.ConfigFile{
			{
				Path:    "test/config.yaml", // Will be written to <rootDir>/etc/test/config.yaml
				Content: base64.StdEncoding.EncodeToString([]byte("test: config")),
				Mode:    0o644,
			},
		},
	}

	err = applier.Apply(bundle)
	require.NoError(t, err)

	// Verify file was written to correct location
	expectedPath := filepath.Join(rootDir, "/etc/test/config.yaml")
	content, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Equal(t, "test: config", string(content))
}

func TestApplier_Apply_InvalidBundle(t *testing.T) {
	// Create temporary root directory for testing
	rootDir := testRootDir(t)

	validator := NewPathValidator([]string{"/etc/test/"})
	applier, err := NewApplier(validator, rootDir)
	require.NoError(t, err)

	defer applier.Cleanup() //nolint:errcheck

	tests := []struct {
		name   string
		bundle *protocol.ConfigBundle
		errMsg string
	}{
		{
			name:   "nil bundle",
			bundle: nil,
			errMsg: "bundle validation failed",
		},
		{
			name: "empty files",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{},
			},
			errMsg: "bundle validation failed",
		},
		{
			name: "invalid Base64",
			bundle: &protocol.ConfigBundle{
				Files: []protocol.ConfigFile{
					{
						Path:    "test/config.yaml",
						Content: "invalid-base64!!!",
						Mode:    0o644,
					},
				},
			},
			errMsg: "bundle validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applier.Apply(tt.bundle)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestApplier_Apply_PathValidationFailure(t *testing.T) {
	// Create temporary root directory for testing
	rootDir := testRootDir(t)

	// Validator only allows /etc/test/
	validator := NewPathValidator([]string{"/etc/test/"})
	applier, err := NewApplier(validator, rootDir)
	require.NoError(t, err)
	defer applier.Cleanup() //nolint:errcheck

	// Bundle with path outside allow-list
	bundle := &protocol.ConfigBundle{
		Files: []protocol.ConfigFile{
			{
				Path:    "passwd", // Not in /etc/test/
				Content: base64.StdEncoding.EncodeToString([]byte("malicious")),
				Mode:    0o644,
			},
		},
	}

	err = applier.Apply(bundle)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path validation failed")
}

func TestApplier_Cleanup(t *testing.T) {
	// Create temporary root directory for testing
	rootDir := testRootDir(t)

	validator := NewPathValidator([]string{"/etc/test/"})
	applier, err := NewApplier(validator, rootDir)
	require.NoError(t, err)

	// Temp directory should exist
	_, err = os.Stat(applier.tempDir)
	assert.NoError(t, err)

	// Cleanup
	_ = applier.Cleanup() //nolint:errcheck
	assert.NoError(t, err)

	// Temp directory should be removed
	_, err = os.Stat(applier.tempDir)
	assert.True(t, os.IsNotExist(err))
}

func TestApplier_MultipleFiles(t *testing.T) {
	// This test validates the logic of handling multiple files
	// Create temporary root directory for testing
	rootDir := testRootDir(t)

	validator := NewPathValidator([]string{"/etc/test/"})
	applier, err := NewApplier(validator, rootDir)
	require.NoError(t, err)
	defer applier.Cleanup() //nolint:errcheck

	bundle := &protocol.ConfigBundle{
		Files: []protocol.ConfigFile{
			{
				Path:    "test/file1.conf",
				Content: base64.StdEncoding.EncodeToString([]byte("content1")),
				Mode:    0o644,
			},
			{
				Path:    "test/file2.conf",
				Content: base64.StdEncoding.EncodeToString([]byte("content2")),
				Mode:    0o600,
			},
		},
	}

	// Validate bundle structure
	err = ValidateBundle(bundle)
	assert.NoError(t, err)

	// Validate paths
	paths := []string{bundle.Files[0].Path, bundle.Files[1].Path}
	err = validator.ValidateAll(paths)
	assert.NoError(t, err)

	// Note: Actual Apply() would need /etc write access
	// This is covered by integration tests
}

func TestApplier_DecodeAndStageFiles(t *testing.T) {
	// Test the file staging logic
	// Create temporary root directory for testing
	rootDir := testRootDir(t)

	validator := NewPathValidator([]string{"/etc/test/"})
	applier, err := NewApplier(validator, rootDir)
	require.NoError(t, err)
	defer applier.Cleanup() //nolint:errcheck

	testContent := "test configuration content"
	encoded := base64.StdEncoding.EncodeToString([]byte(testContent))

	// Decode content
	decoded, err := DecodeFileContent(encoded)
	require.NoError(t, err)
	assert.Equal(t, []byte(testContent), decoded)

	// Write to temp directory (simulating staging)
	tempPath := filepath.Join(applier.tempDir, "test.conf")
	err = os.WriteFile(tempPath, decoded, 0o644) //nolint:gosec // G306: Test file
	require.NoError(t, err)

	// Verify file was written
	readBack, err := os.ReadFile(tempPath)
	require.NoError(t, err)
	assert.Equal(t, []byte(testContent), readBack)

	// Verify permissions
	info, err := os.Stat(tempPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}
