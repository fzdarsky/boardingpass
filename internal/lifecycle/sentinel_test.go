package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSentinel(t *testing.T) {
	t.Run("with custom path", func(t *testing.T) {
		s := NewSentinel("/custom/path")
		if s.Path() != "/custom/path" {
			t.Errorf("expected path /custom/path, got %s", s.Path())
		}
	})

	t.Run("with default path", func(t *testing.T) {
		s := NewSentinel("")
		if s.Path() != DefaultSentinelPath {
			t.Errorf("expected path %s, got %s", DefaultSentinelPath, s.Path())
		}
	})
}

func TestSentinel_Exists(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	sentinelPath := filepath.Join(tempDir, "issued")

	s := NewSentinel(sentinelPath)

	t.Run("file does not exist", func(t *testing.T) {
		exists, err := s.Exists()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("expected file to not exist")
		}
	})

	t.Run("file exists", func(t *testing.T) {
		// Create the file
		if err := os.WriteFile(sentinelPath, []byte("test"), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		exists, err := s.Exists()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected file to exist")
		}
	})
}

func TestSentinel_Create(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	sentinelPath := filepath.Join(tempDir, "boardingpass", "issued")

	s := NewSentinel(sentinelPath)

	t.Run("creates file successfully", func(t *testing.T) {
		err := s.Create()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file exists
		exists, err := s.Exists()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected file to be created")
		}

		// Verify file contents
		content, err := os.ReadFile(sentinelPath)
		if err != nil {
			t.Fatalf("failed to read sentinel file: %v", err)
		}
		if len(content) == 0 {
			t.Error("expected file to have content")
		}
	})

	t.Run("creates parent directory if needed", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "nested", "path", "issued")
		s := NewSentinel(nestedPath)

		err := s.Create()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file exists
		exists, err := s.Exists()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected file to be created")
		}
	})

	t.Run("idempotent - creating existing file succeeds", func(t *testing.T) {
		existingPath := filepath.Join(tempDir, "existing")
		s := NewSentinel(existingPath)

		// Create first time
		if err := s.Create(); err != nil {
			t.Fatalf("first create failed: %v", err)
		}

		// Create second time should succeed
		if err := s.Create(); err != nil {
			t.Fatalf("second create failed: %v", err)
		}
	})
}

func TestSentinel_Path(t *testing.T) {
	testPath := "/test/path/sentinel"
	s := NewSentinel(testPath)

	if s.Path() != testPath {
		t.Errorf("expected path %s, got %s", testPath, s.Path())
	}
}
