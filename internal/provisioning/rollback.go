package provisioning

import (
	"fmt"
	"os"
	"path/filepath"
)

// Rollback tracks file operations for potential rollback on failure.
// It maintains a list of files that have been modified and their backup locations.
type Rollback struct {
	backups map[string]string // maps target path to backup path
	tempDir string            // temporary directory for backups
}

// NewRollback creates a new Rollback tracker with a temporary backup directory.
func NewRollback(tempDir string) (*Rollback, error) {
	if tempDir == "" {
		return nil, fmt.Errorf("tempDir cannot be empty")
	}

	// Create backup directory within tempDir
	backupDir := filepath.Join(tempDir, "backup")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &Rollback{
		backups: make(map[string]string),
		tempDir: backupDir,
	}, nil
}

// BackupFile creates a backup of the target file before modification.
// If the target file doesn't exist, no backup is created (new file).
// Returns nil on success, error on failure.
func (r *Rollback) BackupFile(targetPath string) error {
	// Check if file exists
	info, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		// File doesn't exist, no backup needed
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat target file %s: %w", targetPath, err)
	}

	// Only backup regular files
	if !info.Mode().IsRegular() {
		return fmt.Errorf("target %s is not a regular file", targetPath)
	}

	// Generate backup path
	backupPath := filepath.Join(r.tempDir, filepath.Base(targetPath)+".backup")

	// Copy file to backup location
	if err := copyFile(targetPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup file %s: %w", targetPath, err)
	}

	// Record backup
	r.backups[targetPath] = backupPath

	return nil
}

// Restore restores all backed-up files to their original locations.
// This is called on provisioning failure to undo partial changes.
// Returns an error if any restore operation fails.
func (r *Rollback) Restore() error {
	var restoreErrors []error

	for targetPath, backupPath := range r.backups {
		// Check if backup exists
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			// Backup doesn't exist, file was newly created, remove it
			if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
				restoreErrors = append(restoreErrors, fmt.Errorf("failed to remove %s: %w", targetPath, err))
			}
			continue
		}

		// Restore from backup
		if err := copyFile(backupPath, targetPath); err != nil {
			restoreErrors = append(restoreErrors, fmt.Errorf("failed to restore %s: %w", targetPath, err))
		}
	}

	if len(restoreErrors) > 0 {
		return fmt.Errorf("rollback failed with %d errors: %v", len(restoreErrors), restoreErrors)
	}

	return nil
}

// Cleanup removes all backup files. Call after successful provisioning.
func (r *Rollback) Cleanup() error {
	if r.tempDir == "" {
		return nil
	}

	// Remove entire backup directory
	if err := os.RemoveAll(r.tempDir); err != nil {
		return fmt.Errorf("failed to cleanup backups: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	// Read source file
	//nolint:gosec // G304: src is a controlled backup file path, not user input
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Get source file permissions
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Write to destination with same permissions
	if err := os.WriteFile(dst, data, info.Mode()); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}
