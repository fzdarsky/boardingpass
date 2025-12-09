package provisioning

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRollback(t *testing.T) {
	tempDir := t.TempDir()

	rollback, err := NewRollback(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, rollback)
	assert.NotEmpty(t, rollback.tempDir)
	assert.NotNil(t, rollback.backups)

	// Verify backup directory was created
	backupDir := filepath.Join(tempDir, "backup")
	info, err := os.Stat(backupDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewRollback_EmptyTempDir(t *testing.T) {
	rollback, err := NewRollback("")
	assert.Nil(t, rollback)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tempDir cannot be empty")
}

func TestRollback_BackupFile_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	rollback, err := NewRollback(tempDir)
	require.NoError(t, err)

	// Backup a file that doesn't exist (should succeed with no backup created)
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	err = rollback.BackupFile(nonExistentFile)
	assert.NoError(t, err)

	// No backup should be recorded
	assert.Empty(t, rollback.backups)
}

func TestRollback_BackupFile_ExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	rollback, err := NewRollback(tempDir)
	require.NoError(t, err)

	// Create a test file
	targetFile := filepath.Join(tempDir, "test.txt")
	originalContent := []byte("original content")
	err = os.WriteFile(targetFile, originalContent, 0o644) //nolint:gosec // G306: Test file
	require.NoError(t, err)

	// Backup the file
	err = rollback.BackupFile(targetFile)
	require.NoError(t, err)

	// Verify backup was recorded
	assert.Len(t, rollback.backups, 1)
	assert.Contains(t, rollback.backups, targetFile)

	// Verify backup file exists and has correct content
	backupPath := rollback.backups[targetFile]
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, backupContent)
}

func TestRollback_BackupFile_Directory(t *testing.T) {
	tempDir := t.TempDir()
	rollback, err := NewRollback(tempDir)
	require.NoError(t, err)

	// Create a directory
	targetDir := filepath.Join(tempDir, "testdir")
	err = os.Mkdir(targetDir, 0o755) //nolint:gosec // G301: Test directory
	require.NoError(t, err)

	// Attempt to backup directory (should fail)
	err = rollback.BackupFile(targetDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a regular file")
}

func TestRollback_Restore_NewFile(t *testing.T) {
	tempDir := t.TempDir()
	rollback, err := NewRollback(tempDir)
	require.NoError(t, err)

	// Create a new file that didn't exist before
	newFile := filepath.Join(tempDir, "newfile.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0o644) //nolint:gosec // G306: Test file
	require.NoError(t, err)

	// Record as if we tried to backup this file (but it didn't exist)
	// Simulate by adding to backups map with non-existent backup
	rollback.backups[newFile] = filepath.Join(rollback.tempDir, "nonexistent.backup")

	// Restore should remove the new file
	err = rollback.Restore()
	require.NoError(t, err)

	// File should be removed
	_, err = os.Stat(newFile)
	assert.True(t, os.IsNotExist(err))
}

func TestRollback_Restore_ModifiedFile(t *testing.T) {
	tempDir := t.TempDir()
	rollback, err := NewRollback(tempDir)
	require.NoError(t, err)

	// Create original file
	targetFile := filepath.Join(tempDir, "test.txt")
	originalContent := []byte("original content")
	err = os.WriteFile(targetFile, originalContent, 0o644) //nolint:gosec // G306: Test file
	require.NoError(t, err)

	// Backup the file
	err = rollback.BackupFile(targetFile)
	require.NoError(t, err)

	// Modify the file
	modifiedContent := []byte("modified content")
	err = os.WriteFile(targetFile, modifiedContent, 0o644) //nolint:gosec // G306: Test file
	require.NoError(t, err)

	// Verify file was modified
	currentContent, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, modifiedContent, currentContent)

	// Restore should revert to original
	err = rollback.Restore()
	require.NoError(t, err)

	// Verify file was restored
	restoredContent, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, restoredContent)
}

func TestRollback_Restore_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	rollback, err := NewRollback(tempDir)
	require.NoError(t, err)

	// Create and backup multiple files
	file1 := filepath.Join(tempDir, "file1.txt")
	content1 := []byte("content 1")
	err = os.WriteFile(file1, content1, 0o644) //nolint:gosec // G306: Test file
	require.NoError(t, err)
	err = rollback.BackupFile(file1)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "file2.txt")
	content2 := []byte("content 2")
	err = os.WriteFile(file2, content2, 0o600)
	require.NoError(t, err)
	err = rollback.BackupFile(file2)
	require.NoError(t, err)

	// Modify both files
	err = os.WriteFile(file1, []byte("modified 1"), 0o644) //nolint:gosec // G306: Test file
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("modified 2"), 0o600)
	require.NoError(t, err)

	// Restore both
	err = rollback.Restore()
	require.NoError(t, err)

	// Verify both were restored
	restored1, err := os.ReadFile(file1)
	require.NoError(t, err)
	assert.Equal(t, content1, restored1)

	restored2, err := os.ReadFile(file2)
	require.NoError(t, err)
	assert.Equal(t, content2, restored2)
}

func TestRollback_Cleanup(t *testing.T) {
	tempDir := t.TempDir()
	rollback, err := NewRollback(tempDir)
	require.NoError(t, err)

	// Create and backup a file
	targetFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(targetFile, []byte("content"), 0o644) //nolint:gosec // G306: Test file
	require.NoError(t, err)
	err = rollback.BackupFile(targetFile)
	require.NoError(t, err)

	// Verify backup directory exists
	_, err = os.Stat(rollback.tempDir)
	assert.NoError(t, err)

	// Cleanup
	err = rollback.Cleanup()
	require.NoError(t, err)

	// Verify backup directory was removed
	_, err = os.Stat(rollback.tempDir)
	assert.True(t, os.IsNotExist(err))
}

func TestRollback_Cleanup_EmptyTempDir(t *testing.T) {
	rollback := &Rollback{
		backups: make(map[string]string),
		tempDir: "",
	}

	// Cleanup with empty tempDir should not fail
	err := rollback.Cleanup()
	assert.NoError(t, err)
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tempDir, "source.txt")
	srcContent := []byte("source content")
	err := os.WriteFile(srcFile, srcContent, 0o600)
	require.NoError(t, err)

	// Copy file
	dstFile := filepath.Join(tempDir, "dest.txt")
	err = copyFile(srcFile, dstFile)
	require.NoError(t, err)

	// Verify destination exists and has same content
	dstContent, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcContent, dstContent)

	// Verify permissions were preserved
	srcInfo, err := os.Stat(srcFile)
	require.NoError(t, err)
	dstInfo, err := os.Stat(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcInfo.Mode().Perm(), dstInfo.Mode().Perm())
}

func TestCopyFile_NonExistentSource(t *testing.T) {
	tempDir := t.TempDir()

	srcFile := filepath.Join(tempDir, "nonexistent.txt")
	dstFile := filepath.Join(tempDir, "dest.txt")

	err := copyFile(srcFile, dstFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source file")
}

func TestCopyFile_PermissionPreservation(t *testing.T) {
	tempDir := t.TempDir()

	// Test various permission modes
	modes := []os.FileMode{0o600, 0o644, 0o755, 0o400}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			srcFile := filepath.Join(tempDir, "src_"+mode.String()+".txt")
			err := os.WriteFile(srcFile, []byte("content"), mode)
			require.NoError(t, err)

			dstFile := filepath.Join(tempDir, "dst_"+mode.String()+".txt")
			err = copyFile(srcFile, dstFile)
			require.NoError(t, err)

			dstInfo, err := os.Stat(dstFile)
			require.NoError(t, err)
			assert.Equal(t, mode, dstInfo.Mode().Perm())
		})
	}
}
