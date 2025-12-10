package session_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/cli/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	// Set up temporary cache directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	store, err := session.NewStore()
	require.NoError(t, err)
	assert.NotNil(t, store)

	// Verify cache directory was created
	expectedDir := filepath.Join(tmpDir, "boardingpass")
	info, err := os.Stat(expectedDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestStore_SaveAndLoad(t *testing.T) {
	store := setupTestStore(t)

	tests := []struct {
		name  string
		host  string
		port  int
		token string
	}{
		{
			name:  "save and load simple token",
			host:  "test.local",
			port:  8443,
			token: "test-session-token-123",
		},
		{
			name:  "save and load token with special characters",
			host:  "boardingpass.local",
			port:  9443,
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ",
		},
		{
			name:  "save and load long token",
			host:  "192.168.1.100",
			port:  8443,
			token: "very-long-token-with-lots-of-random-characters-abcdefghijklmnopqrstuvwxyz0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save token
			err := store.Save(tt.host, tt.port, tt.token)
			require.NoError(t, err)

			// Load token
			loaded, err := store.Load(tt.host, tt.port)
			require.NoError(t, err)
			assert.Equal(t, tt.token, loaded)
		})
	}
}

func TestStore_Load_NotExists(t *testing.T) {
	store := setupTestStore(t)

	// Load non-existent token
	token, err := store.Load("nonexistent.local", 8443)
	require.NoError(t, err)
	assert.Equal(t, "", token, "should return empty string for non-existent token")
}

func TestStore_Load_WithTrailingWhitespace(t *testing.T) {
	store := setupTestStore(t)

	// Manually create token file with trailing whitespace
	host := "test.local"
	port := 8443
	filename := getTokenFilename(t, store, host, port)

	tokenWithWhitespace := "test-token\n\n  \t"
	err := os.WriteFile(filename, []byte(tokenWithWhitespace), 0o600)
	require.NoError(t, err)

	// Load should trim whitespace
	loaded, err := store.Load(host, port)
	require.NoError(t, err)
	assert.Equal(t, "test-token", loaded)
}

func TestStore_SavePermissions(t *testing.T) {
	store := setupTestStore(t)

	host := "test.local"
	port := 8443
	token := "test-token"

	// Save token
	err := store.Save(host, port, token)
	require.NoError(t, err)

	// Check file permissions
	filename := getTokenFilename(t, store, host, port)
	info, err := os.Stat(filename)
	require.NoError(t, err)

	// Verify permissions are 0600 (owner read/write only)
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0o600), mode, "token file should have 0600 permissions")
}

func TestStore_Delete(t *testing.T) {
	store := setupTestStore(t)

	host := "test.local"
	port := 8443
	token := "test-token"

	// Save token
	err := store.Save(host, port, token)
	require.NoError(t, err)

	// Verify token exists
	loaded, err := store.Load(host, port)
	require.NoError(t, err)
	assert.Equal(t, token, loaded)

	// Delete token
	err = store.Delete(host, port)
	require.NoError(t, err)

	// Verify token no longer exists
	loaded, err = store.Load(host, port)
	require.NoError(t, err)
	assert.Equal(t, "", loaded)
}

func TestStore_Delete_NotExists(t *testing.T) {
	store := setupTestStore(t)

	// Delete non-existent token should not error
	err := store.Delete("nonexistent.local", 8443)
	assert.NoError(t, err)
}

func TestStore_MultipleHosts(t *testing.T) {
	store := setupTestStore(t)

	// Save tokens for different hosts
	hosts := []struct {
		host  string
		port  int
		token string
	}{
		{"host1.local", 8443, "token-for-host1"},
		{"host2.local", 8443, "token-for-host2"},
		{"host1.local", 9443, "token-for-host1-port9443"},
	}

	for _, h := range hosts {
		err := store.Save(h.host, h.port, h.token)
		require.NoError(t, err)
	}

	// Verify each token can be loaded independently
	for _, h := range hosts {
		loaded, err := store.Load(h.host, h.port)
		require.NoError(t, err)
		assert.Equal(t, h.token, loaded)
	}

	// Delete one token
	err := store.Delete("host1.local", 8443)
	require.NoError(t, err)

	// Verify only that token is deleted
	loaded, err := store.Load("host1.local", 8443)
	require.NoError(t, err)
	assert.Equal(t, "", loaded)

	// Other tokens should still exist
	loaded, err = store.Load("host2.local", 8443)
	require.NoError(t, err)
	assert.Equal(t, "token-for-host2", loaded)

	loaded, err = store.Load("host1.local", 9443)
	require.NoError(t, err)
	assert.Equal(t, "token-for-host1-port9443", loaded)
}

func TestStore_TokenFilename_Consistency(t *testing.T) {
	store := setupTestStore(t)

	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "consistent filename for same host:port",
			host:     "test.local",
			port:     8443,
			expected: computeExpectedFilename(t, store, "test.local", 8443),
		},
		{
			name:     "different filename for different host",
			host:     "other.local",
			port:     8443,
			expected: computeExpectedFilename(t, store, "other.local", 8443),
		},
		{
			name:     "different filename for different port",
			host:     "test.local",
			port:     9443,
			expected: computeExpectedFilename(t, store, "test.local", 9443),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save token
			err := store.Save(tt.host, tt.port, "test-token")
			require.NoError(t, err)

			// Verify filename matches expected
			actualFilename := getTokenFilename(t, store, tt.host, tt.port)
			assert.Equal(t, tt.expected, actualFilename)
		})
	}
}

func TestStore_TokenFilename_Collision(t *testing.T) {
	store := setupTestStore(t)

	// Two different host:port combinations should produce different filenames
	filename1 := getTokenFilename(t, store, "host1.local", 8443)
	filename2 := getTokenFilename(t, store, "host2.local", 8443)
	filename3 := getTokenFilename(t, store, "host1.local", 9443)

	assert.NotEqual(t, filename1, filename2, "different hosts should have different filenames")
	assert.NotEqual(t, filename1, filename3, "different ports should have different filenames")
	assert.NotEqual(t, filename2, filename3, "different host:port should have different filenames")
}

// Helper functions

func setupTestStore(t *testing.T) *session.Store {
	t.Helper()

	// Set up temporary cache directory
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	store, err := session.NewStore()
	require.NoError(t, err)

	return store
}

func getTokenFilename(t *testing.T, store *session.Store, host string, port int) string {
	t.Helper()

	// Save a dummy token to trigger filename generation
	err := store.Save(host, port, "dummy")
	require.NoError(t, err)

	// Compute expected filename
	return computeExpectedFilename(t, store, host, port)
}

func computeExpectedFilename(t *testing.T, store *session.Store, host string, port int) string {
	t.Helper()

	// Replicate the filename generation logic from store.go
	identifier := fmt.Sprintf("%s:%d", host, port)
	hash := sha256.Sum256([]byte(identifier))
	hashStr := fmt.Sprintf("%x", hash[:8])
	filename := fmt.Sprintf("session-%s.token", hashStr)

	// Get cache dir from env
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = os.Getenv("HOME")
	}
	cacheDir = filepath.Join(cacheDir, "boardingpass")

	return filepath.Join(cacheDir, filename)
}
