package provisioning

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathValidator validates file paths against an allow-list of permitted directories.
// It prevents writes to critical system files by explicitly allowing only specific subdirectories.
type PathValidator struct {
	allowedPaths []string
}

// NewPathValidator creates a new PathValidator with the given allow-list of paths.
// Each path should be an absolute directory path (e.g., "/etc/systemd/", "/etc/myapp/").
func NewPathValidator(allowedPaths []string) *PathValidator {
	return &PathValidator{allowedPaths: allowedPaths}
}

// ValidatePath checks if the given path is allowed according to the allow-list.
// The path is validated against the following rules:
// 1. Must not contain ".." (path traversal prevention)
// 2. Must be within one of the allowed directory prefixes
//
// Returns an error if the path is not allowed, nil if it is valid.
func (pv *PathValidator) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path %s contains path traversal sequence (..)", path)
	}

	// Normalize the path to absolute form
	// Paths in the bundle are relative to /etc, so prepend /etc
	absPath := filepath.Join("/etc", path)

	// Clean the path to remove any redundant separators or . elements
	absPath = filepath.Clean(absPath)

	// Ensure the path is still under /etc after cleaning
	if !strings.HasPrefix(absPath, "/etc/") {
		return fmt.Errorf("path %s is not within /etc", path)
	}

	// Check if path matches any allowed prefix
	for _, allowed := range pv.allowedPaths {
		// Ensure allowed path has trailing slash for prefix matching
		allowedPrefix := allowed
		if !strings.HasSuffix(allowedPrefix, "/") {
			allowedPrefix += "/"
		}

		// Check if absPath starts with the allowed prefix
		// or if absPath equals the allowed directory itself
		if strings.HasPrefix(absPath+"/", allowedPrefix) || absPath == strings.TrimSuffix(allowed, "/") {
			return nil // Path is allowed
		}
	}

	return fmt.Errorf("path %s is not in allow-list", absPath)
}

// ValidateAll validates all paths in a slice.
// Returns the first error encountered, or nil if all paths are valid.
func (pv *PathValidator) ValidateAll(paths []string) error {
	for _, path := range paths {
		if err := pv.ValidatePath(path); err != nil {
			return err
		}
	}
	return nil
}
