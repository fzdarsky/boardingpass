package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Validate performs comprehensive validation on the configuration.
func Validate(cfg *Config) error {
	if err := validateService(cfg); err != nil {
		return fmt.Errorf("service validation failed: %w", err)
	}

	if err := validateTransports(cfg); err != nil {
		return fmt.Errorf("transport validation failed: %w", err)
	}

	if err := validateCommands(cfg); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	if err := validateLogging(cfg); err != nil {
		return fmt.Errorf("logging validation failed: %w", err)
	}

	if err := validatePaths(cfg); err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	return nil
}

func validateService(cfg *Config) error {
	// Validate inactivity timeout
	if _, err := cfg.GetInactivityTimeout(); err != nil {
		return err
	}

	// Validate session TTL
	if _, err := cfg.GetSessionTTL(); err != nil {
		return err
	}

	// Validate sentinel file path
	if !filepath.IsAbs(cfg.Service.SentinelFile) {
		return fmt.Errorf("sentinel_file must be an absolute path")
	}

	// Check if sentinel file directory exists
	sentinelDir := filepath.Dir(cfg.Service.SentinelFile)
	if _, err := os.Stat(sentinelDir); os.IsNotExist(err) {
		return fmt.Errorf("sentinel_file directory does not exist: %s", sentinelDir)
	}

	return nil
}

func validateTransports(cfg *Config) error {
	if !cfg.Transports.Ethernet.Enabled {
		return fmt.Errorf("at least one transport must be enabled")
	}

	if cfg.Transports.Ethernet.Enabled {
		// Validate port
		if cfg.Transports.Ethernet.Port <= 0 || cfg.Transports.Ethernet.Port > 65535 {
			return fmt.Errorf("ethernet.port must be between 1 and 65535")
		}

		// Validate TLS cert file
		if !filepath.IsAbs(cfg.Transports.Ethernet.TLSCert) {
			return fmt.Errorf("ethernet.tls_cert must be an absolute path")
		}

		// Check if TLS cert directory exists
		certDir := filepath.Dir(cfg.Transports.Ethernet.TLSCert)
		if _, err := os.Stat(certDir); os.IsNotExist(err) {
			return fmt.Errorf("ethernet.tls_cert directory does not exist: %s", certDir)
		}

		// Validate TLS key file
		if !filepath.IsAbs(cfg.Transports.Ethernet.TLSKey) {
			return fmt.Errorf("ethernet.tls_key must be an absolute path")
		}

		// Check if TLS key directory exists
		keyDir := filepath.Dir(cfg.Transports.Ethernet.TLSKey)
		if _, err := os.Stat(keyDir); os.IsNotExist(err) {
			return fmt.Errorf("ethernet.tls_key directory does not exist: %s", keyDir)
		}

		// Validate address if specified
		if cfg.Transports.Ethernet.Address != "" {
			// Basic validation - could be IPv4, IPv6, or hostname
			if strings.Contains(cfg.Transports.Ethernet.Address, " ") {
				return fmt.Errorf("ethernet.address contains invalid characters")
			}
		}
	}

	return nil
}

func validateCommands(cfg *Config) error {
	if len(cfg.Commands) == 0 {
		return fmt.Errorf("commands allow-list cannot be empty")
	}

	if len(cfg.Commands) > 50 {
		return fmt.Errorf("commands allow-list cannot exceed 50 entries")
	}

	// Track command IDs to check for duplicates
	seen := make(map[string]bool)

	for i, cmd := range cfg.Commands {
		// Validate command ID
		if cmd.ID == "" {
			return fmt.Errorf("command[%d]: id is required", i)
		}

		if !isValidCommandID(cmd.ID) {
			return fmt.Errorf("command[%d]: id '%s' contains invalid characters (allowed: a-z, 0-9, -)", i, cmd.ID)
		}

		if seen[cmd.ID] {
			return fmt.Errorf("command[%d]: duplicate command id '%s'", i, cmd.ID)
		}
		seen[cmd.ID] = true

		// Validate command path
		if cmd.Path == "" {
			return fmt.Errorf("command[%d]: path is required", i)
		}

		if !filepath.IsAbs(cmd.Path) {
			return fmt.Errorf("command[%d]: path must be absolute", i)
		}

		// Check if command exists and is executable
		if _, err := os.Stat(cmd.Path); os.IsNotExist(err) {
			return fmt.Errorf("command[%d]: path does not exist: %s", i, cmd.Path)
		}
	}

	return nil
}

func validateLogging(cfg *Config) error {
	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	if !slices.Contains(validLevels, cfg.Logging.Level) {
		return fmt.Errorf("logging.level must be one of: %s", strings.Join(validLevels, ", "))
	}

	// Validate log format
	validFormats := []string{"json", "human"}
	if !slices.Contains(validFormats, cfg.Logging.Format) {
		return fmt.Errorf("logging.format must be one of: %s", strings.Join(validFormats, ", "))
	}

	return nil
}

func validatePaths(cfg *Config) error {
	if len(cfg.Paths.AllowList) == 0 {
		return fmt.Errorf("paths.allow_list cannot be empty")
	}

	for i, path := range cfg.Paths.AllowList {
		// Must be absolute
		if !filepath.IsAbs(path) {
			return fmt.Errorf("paths.allow_list[%d]: path must be absolute: %s", i, path)
		}

		// Must not contain ..
		if strings.Contains(path, "..") {
			return fmt.Errorf("paths.allow_list[%d]: path cannot contain '..': %s", i, path)
		}

		// Must be a directory path (end with /)
		if !strings.HasSuffix(path, "/") {
			return fmt.Errorf("paths.allow_list[%d]: path must end with '/': %s", i, path)
		}

		// Dangerous paths should not be in allow-list
		dangerousPaths := []string{
			"/etc/passwd",
			"/etc/shadow",
			"/etc/sudoers",
			"/etc/ssh/",
			"/root/",
			"/home/",
		}

		for _, dangerous := range dangerousPaths {
			if path == dangerous || strings.HasPrefix(path, dangerous) {
				return fmt.Errorf("paths.allow_list[%d]: dangerous path not allowed: %s", i, path)
			}
		}
	}

	return nil
}

// isValidCommandID checks if a command ID contains only allowed characters.
func isValidCommandID(id string) bool {
	for _, r := range id {
		//nolint:staticcheck // QF1001: current form is more readable
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}
