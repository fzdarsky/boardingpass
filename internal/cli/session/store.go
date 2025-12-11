// Package session provides session token storage for the boarding CLI tool.
package session

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
)

const (
	tokenFileMode = 0o600 // Owner read/write only
)

// Store manages session token persistence in the OS cache directory.
type Store struct {
	dir string
}

// NewStore creates a new session token store.
// The store uses the OS-specific cache directory.
func NewStore() (*Store, error) {
	cacheDir, err := config.UserCacheDir()
	if err != nil {
		return nil, err
	}

	// Ensure cache directory exists with secure permissions
	if err := config.EnsureDir(cacheDir); err != nil {
		return nil, err
	}

	return &Store{
		dir: cacheDir,
	}, nil
}

// Save saves a session token for the specified host and port.
// The token is stored in a file with 0600 permissions (owner read/write only).
func (s *Store) Save(host string, port int, token string) error {
	filename := s.tokenFilename(host, port)

	// Write token to file with secure permissions
	if err := os.WriteFile(filename, []byte(token), tokenFileMode); err != nil {
		return fmt.Errorf("failed to save session token: %w", err)
	}

	return nil
}

// Load loads the session token for the specified host and port.
// Returns empty string if no token exists.
func (s *Store) Load(host string, port int) (string, error) {
	filename := s.tokenFilename(host, port)

	data, err := os.ReadFile(filename) // #nosec G304 - filename is generated from hash of host:port
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No token exists, not an error
		}
		return "", fmt.Errorf("failed to read session token: %w", err)
	}

	// Trim any trailing whitespace/newlines
	token := strings.TrimSpace(string(data))
	return token, nil
}

// Delete deletes the session token for the specified host and port.
func (s *Store) Delete(host string, port int) error {
	filename := s.tokenFilename(host, port)

	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, not an error
		}
		return fmt.Errorf("failed to delete session token: %w", err)
	}

	return nil
}

// tokenFilename generates a filename for the session token based on host:port.
// The filename uses the first 16 hex characters of the SHA-256 hash of "host:port".
// Format: session-<hash>.token
// Example: session-a1b2c3d4e5f6g7h8.token
func (s *Store) tokenFilename(host string, port int) string {
	// Create unique identifier from host:port
	identifier := fmt.Sprintf("%s:%d", host, port)

	// Compute SHA-256 hash
	hash := sha256.Sum256([]byte(identifier))

	// Use first 16 hex characters (8 bytes) for filename
	hashStr := fmt.Sprintf("%x", hash[:8])

	// Create filename
	filename := fmt.Sprintf("session-%s.token", hashStr)

	return filepath.Join(s.dir, filename)
}
