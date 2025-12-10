package provisioning

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const (
	// StagingDirBase is the base directory for staging temporary files
	StagingDirBase = "/var/lib/boardingpass/staging"
)

// Applier handles atomic application of configuration bundles.
// It uses a temp-write → validate → atomic rename pattern for fail-safe operations.
type Applier struct {
	validator *PathValidator
	tempDir   string
	rollback  *Rollback
}

// NewApplier creates a new Applier with the given path validator.
// It creates a temporary staging directory for atomic operations.
func NewApplier(validator *PathValidator) (*Applier, error) {
	if validator == nil {
		return nil, fmt.Errorf("validator cannot be nil")
	}

	// Create unique temporary directory for this operation
	tempDir, err := os.MkdirTemp(StagingDirBase, "apply-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Initialize rollback tracker
	rollback, err := NewRollback(tempDir)
	if err != nil {
		_ = os.RemoveAll(tempDir) // Best effort cleanup
		return nil, fmt.Errorf("failed to initialize rollback: %w", err)
	}

	return &Applier{
		validator: validator,
		tempDir:   tempDir,
		rollback:  rollback,
	}, nil
}

// Apply applies a configuration bundle atomically.
// Steps:
// 1. Validate bundle (size, file count, Base64 encoding)
// 2. Validate all paths against allow-list
// 3. Decode and write all files to temp directory
// 4. Backup existing target files
// 5. Atomically rename all files to target paths
// 6. Clean up temp directory
//
// On any failure, rollback is performed to restore original state.
func (a *Applier) Apply(bundle *protocol.ConfigBundle) error {
	// Step 1: Validate bundle
	if err := ValidateBundle(bundle); err != nil {
		return fmt.Errorf("bundle validation failed: %w", err)
	}

	// Step 2: Validate all paths
	paths := make([]string, len(bundle.Files))
	for i, file := range bundle.Files {
		paths[i] = file.Path
	}
	if err := a.validator.ValidateAll(paths); err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	// Step 3: Decode and write files to temp directory
	stagedFiles := make(map[string]string) // maps relative path to temp path
	for _, file := range bundle.Files {
		decoded, err := DecodeFileContent(file.Content)
		if err != nil {
			return fmt.Errorf("failed to decode file %s: %w", file.Path, err)
		}

		// Write to temp directory
		tempPath := filepath.Join(a.tempDir, filepath.Base(file.Path))
		// #nosec G115 - file mode values are guaranteed to be within uint32 range
		if err := os.WriteFile(tempPath, decoded, os.FileMode(file.Mode)); err != nil {
			return fmt.Errorf("failed to write temp file %s: %w", file.Path, err)
		}

		stagedFiles[file.Path] = tempPath
	}

	// Step 4: Backup existing files and Step 5: Atomic rename
	for relPath, tempPath := range stagedFiles {
		targetPath := filepath.Join("/etc", relPath)

		// Ensure target directory exists
		targetDir := filepath.Dir(targetPath)
		//nolint:gosec // G301: 0755 is standard for /etc directories
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			// Rollback on failure
			if rollbackErr := a.rollback.Restore(); rollbackErr != nil {
				return fmt.Errorf("failed to create target directory %s (rollback also failed: %v): %w", targetDir, rollbackErr, err)
			}
			return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
		}

		// Backup existing file (if exists)
		if err := a.rollback.BackupFile(targetPath); err != nil {
			// Rollback on failure
			if rollbackErr := a.rollback.Restore(); rollbackErr != nil {
				return fmt.Errorf("failed to backup file %s (rollback also failed: %v): %w", targetPath, rollbackErr, err)
			}
			return fmt.Errorf("failed to backup file %s: %w", targetPath, err)
		}

		// Atomic rename (os.Rename is atomic on same filesystem)
		if err := os.Rename(tempPath, targetPath); err != nil {
			// Rollback on failure
			if rollbackErr := a.rollback.Restore(); rollbackErr != nil {
				return fmt.Errorf("failed to move file %s to %s (rollback also failed: %v): %w", tempPath, targetPath, rollbackErr, err)
			}
			return fmt.Errorf("failed to move file %s to %s: %w", tempPath, targetPath, err)
		}
	}

	// Step 6: Success - cleanup temp directory and backups
	if err := a.Cleanup(); err != nil {
		// Non-fatal: log but don't fail the operation
		// In production, this would be logged
		return nil
	}

	return nil
}

// Cleanup removes the temporary staging directory and all backups.
// Call this after successful provisioning.
func (a *Applier) Cleanup() error {
	// Clean up rollback backups
	if err := a.rollback.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup rollback: %w", err)
	}

	// Remove temp directory
	if err := os.RemoveAll(a.tempDir); err != nil {
		return fmt.Errorf("failed to cleanup temp directory: %w", err)
	}

	return nil
}

// Rollback restores all files to their pre-provisioning state.
// This is called automatically on failure during Apply.
func (a *Applier) Rollback() error {
	return a.rollback.Restore()
}
