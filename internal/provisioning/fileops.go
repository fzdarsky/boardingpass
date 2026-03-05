package provisioning

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// fileOps abstracts file operations so the applier can use either
// direct OS calls (in tests with rootDir) or sudo-elevated commands
// (in production where files are owned by root).
type fileOps struct {
	useSudo bool
}

// mkdirAll creates the directory path and all parents.
func (f *fileOps) mkdirAll(ctx context.Context, path string, perm os.FileMode) error {
	if !f.useSudo {
		//nolint:gosec // G301: perm is caller-controlled (0755 for /etc directories)
		return os.MkdirAll(path, perm)
	}

	//nolint:gosec // G204: args are controlled paths validated against allow-list
	cmd := exec.CommandContext(ctx, "sudo", "mkdir", "-p", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sudo mkdir -p %s failed: %s: %w", path, out, err)
	}
	return nil
}

// installFile copies src to dst preserving the source file's permissions.
// Creates parent directories as needed.
// In non-sudo mode, uses atomicMove (rename or copy+delete).
// In sudo mode, uses install(1) which also sets root ownership.
func (f *fileOps) installFile(ctx context.Context, src, dst string) error {
	// Ensure parent directory exists (needed in both modes)
	if err := f.mkdirAll(ctx, filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	if !f.useSudo {
		return atomicMove(src, dst)
	}

	// Get source file permissions for the -m flag
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", src, err)
	}
	modeStr := fmt.Sprintf("%04o", info.Mode().Perm())

	// install(1) copies src to dst with given mode and root ownership
	//nolint:gosec // G204: args are controlled paths validated against allow-list
	cmd := exec.CommandContext(ctx, "sudo", "install", "-m", modeStr, src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sudo install %s → %s failed: %s: %w", src, dst, out, err)
	}

	// Remove source (staging file owned by boardingpass, no sudo needed)
	_ = os.Remove(src)
	return nil
}

// restoreFile copies src to dst preserving permissions (for rollback restore).
func (f *fileOps) restoreFile(ctx context.Context, src, dst string) error {
	if !f.useSudo {
		return copyFile(src, dst)
	}

	//nolint:gosec // G204: args are controlled backup/target paths
	cmd := exec.CommandContext(ctx, "sudo", "cp", "-f", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sudo cp %s → %s failed: %s: %w", src, dst, out, err)
	}
	return nil
}

// remove removes a file.
func (f *fileOps) remove(ctx context.Context, path string) error {
	if !f.useSudo {
		return os.Remove(path)
	}

	//nolint:gosec // G204: path is a controlled target validated against allow-list
	cmd := exec.CommandContext(ctx, "sudo", "rm", "-f", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sudo rm %s failed: %s: %w", path, out, err)
	}
	return nil
}

// backupCopy copies a file from src (possibly root-owned) to dst (user-owned backup dir).
// In sudo mode, uses "sudo install" to copy root-owned source files while setting
// ownership to the current user so backups remain readable for restore and cleanup.
func (f *fileOps) backupCopy(ctx context.Context, src, dst string, mode os.FileMode) error {
	if !f.useSudo {
		return copyFile(src, dst)
	}

	// Use sudo install to copy root-owned file with boardingpass ownership.
	// install(1) is already in the sudoers allow-list; cat is not.
	modeStr := fmt.Sprintf("%04o", mode.Perm())
	//nolint:gosec // G204: args are controlled paths validated against allow-list
	cmd := exec.CommandContext(ctx, "sudo", "install", "-m", modeStr,
		"-o", "boardingpass", "-g", "boardingpass", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to backup file %s: %s: %w", src, out, err)
	}
	return nil
}

// atomicMove attempts to move a file atomically from src to dst.
// It first tries os.Rename (atomic on same filesystem).
// If that fails with EXDEV (cross-device link), it falls back to copy+delete.
func atomicMove(src, dst string) error {
	// Try atomic rename first
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Check if error is cross-device link
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) && errors.Is(linkErr.Err, syscall.EXDEV) {
		// Fall back to copy+delete for cross-device moves
		//nolint:gosec // G304: src is a controlled staging file path, not user input
		srcFile, err := os.Open(src)
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer func() {
			_ = srcFile.Close()
		}()

		srcInfo, err := srcFile.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat source file: %w", err)
		}

		//nolint:gosec // G304: dst is a validated target path from allow-list
		dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			_ = dstFile.Close()
			_ = os.Remove(dst)
			return fmt.Errorf("failed to copy file contents: %w", err)
		}

		if err := dstFile.Sync(); err != nil {
			_ = dstFile.Close()
			_ = os.Remove(dst)
			return fmt.Errorf("failed to sync destination file: %w", err)
		}

		if err := dstFile.Close(); err != nil {
			return fmt.Errorf("failed to close destination file: %w", err)
		}

		if err := os.Remove(src); err != nil {
			return fmt.Errorf("failed to remove source file: %w", err)
		}

		return nil
	}

	return err
}
