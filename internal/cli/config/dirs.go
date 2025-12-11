// Package config provides configuration management for the boarding CLI tool.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const appName = "boardingpass"

// UserConfigDir returns the OS-specific user configuration directory for boardingpass.
// On Linux: ~/.config/boardingpass
// On macOS: ~/Library/Application Support/boardingpass
// On Windows: %APPDATA%\boardingpass
func UserConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	appConfigDir := filepath.Join(configDir, appName)
	return appConfigDir, nil
}

// UserCacheDir returns the OS-specific user cache directory for boardingpass.
// On Linux: ~/.cache/boardingpass
// On macOS: ~/Library/Caches/boardingpass
// On Windows: %LocalAppData%\boardingpass\cache
func UserCacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache directory: %w", err)
	}

	appCacheDir := filepath.Join(cacheDir, appName)
	return appCacheDir, nil
}

// EnsureDir creates a directory and all parent directories if they don't exist.
// It sets the directory permissions to 0700 (owner read/write/execute only).
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}
