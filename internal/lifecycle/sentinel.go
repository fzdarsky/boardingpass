// Package lifecycle manages service lifecycle including sentinel files,
// inactivity tracking, and graceful shutdown coordination.
package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultSentinelPath is the default location for the sentinel file.
	// When this file exists, the service will refuse to start.
	DefaultSentinelPath = "/etc/boardingpass/issued"
)

// Sentinel manages the sentinel file that prevents the service from starting
// after provisioning is complete.
type Sentinel struct {
	path string
}

// NewSentinel creates a new sentinel file manager.
// If path is empty, uses DefaultSentinelPath.
func NewSentinel(path string) *Sentinel {
	if path == "" {
		path = DefaultSentinelPath
	}
	return &Sentinel{
		path: path,
	}
}

// Exists checks if the sentinel file exists.
// Returns true if the file exists, false otherwise.
func (s *Sentinel) Exists() (bool, error) {
	_, err := os.Stat(s.path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check sentinel file: %w", err)
}

// Create creates the sentinel file to indicate provisioning is complete.
// This will prevent the service from starting on subsequent boots.
func (s *Sentinel) Create() error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create the sentinel file with read-only permissions
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o400)
	if err != nil {
		if os.IsExist(err) {
			// File already exists, this is OK
			return nil
		}
		return fmt.Errorf("failed to create sentinel file: %w", err)
	}
	defer f.Close() //nolint:errcheck // Write-only operation, close error is not critical

	// Write a timestamp to the file for debugging
	if _, err := f.WriteString("BoardingPass provisioning completed\n"); err != nil {
		return fmt.Errorf("failed to write sentinel file: %w", err)
	}

	return nil
}

// Path returns the configured sentinel file path.
func (s *Sentinel) Path() string {
	return s.path
}
