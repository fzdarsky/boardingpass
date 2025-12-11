// Package config provides configuration loading and validation for the BoardingPass service.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the BoardingPass service configuration.
type Config struct {
	Service    ServiceSettings     `yaml:"service"`
	Transports TransportSettings   `yaml:"transports"`
	Commands   []CommandDefinition `yaml:"commands"`
	Logging    LoggingSettings     `yaml:"logging"`
	Paths      PathSettings        `yaml:"paths"`
}

// ServiceSettings contains service-level configuration.
type ServiceSettings struct {
	InactivityTimeout string `yaml:"inactivity_timeout"`
	SessionTTL        string `yaml:"session_ttl"`
	SentinelFile      string `yaml:"sentinel_file"`
}

// TransportSettings contains transport-specific configuration.
type TransportSettings struct {
	Ethernet EthernetTransport `yaml:"ethernet"`
}

// EthernetTransport contains Ethernet transport configuration.
type EthernetTransport struct {
	Enabled    bool     `yaml:"enabled"`
	Interfaces []string `yaml:"interfaces"`
	Address    string   `yaml:"address"`
	Port       int      `yaml:"port"`
	TLSCert    string   `yaml:"tls_cert"`
	TLSKey     string   `yaml:"tls_key"`
}

// CommandDefinition defines an allow-listed command.
type CommandDefinition struct {
	ID   string   `yaml:"id"`
	Path string   `yaml:"path"`
	Args []string `yaml:"args"`
}

// LoggingSettings contains logging configuration.
type LoggingSettings struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// PathSettings contains path allow-list configuration and optional root directory.
type PathSettings struct {
	AllowList     []string `yaml:"allow_list"`
	RootDirectory string   `yaml:"root_directory,omitempty"` // Optional chroot-like root for testing
}

// Load reads and parses the configuration file.
//
//nolint:gosec // G304: Config path is from command-line argument
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Allow environment variable override for root directory (useful for tests)
	if rootDir := os.Getenv("BOARDINGPASS_ROOT_DIR"); rootDir != "" {
		cfg.Paths.RootDirectory = rootDir
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// validate performs basic validation on the configuration.
// Detailed validation is in validate.go.
func (c *Config) validate() error {
	if c.Service.InactivityTimeout == "" {
		return fmt.Errorf("service.inactivity_timeout is required")
	}

	if c.Service.SessionTTL == "" {
		return fmt.Errorf("service.session_ttl is required")
	}

	if c.Service.SentinelFile == "" {
		return fmt.Errorf("service.sentinel_file is required")
	}

	if c.Transports.Ethernet.Enabled {
		if c.Transports.Ethernet.Port <= 0 || c.Transports.Ethernet.Port > 65535 {
			return fmt.Errorf("transports.ethernet.port must be between 1 and 65535")
		}

		if c.Transports.Ethernet.TLSCert == "" {
			return fmt.Errorf("transports.ethernet.tls_cert is required when ethernet is enabled")
		}

		if c.Transports.Ethernet.TLSKey == "" {
			return fmt.Errorf("transports.ethernet.tls_key is required when ethernet is enabled")
		}
	}

	// Validate root directory (if specified)
	if c.Paths.RootDirectory != "" {
		// Ensure it's an absolute path
		if !filepath.IsAbs(c.Paths.RootDirectory) {
			return fmt.Errorf("paths.root_directory must be an absolute path")
		}

		// Create if it doesn't exist (useful for tests)
		//nolint:gosec // G301: 0755 is standard for directory permissions
		if err := os.MkdirAll(c.Paths.RootDirectory, 0o755); err != nil {
			return fmt.Errorf("failed to create root directory: %w", err)
		}
	}

	return nil
}

// GetInactivityTimeout parses and returns the inactivity timeout duration.
func (c *Config) GetInactivityTimeout() (time.Duration, error) {
	duration, err := time.ParseDuration(c.Service.InactivityTimeout)
	if err != nil {
		return 0, fmt.Errorf("invalid inactivity_timeout: %w", err)
	}

	if duration < time.Minute {
		return 0, fmt.Errorf("inactivity_timeout must be at least 1 minute")
	}

	return duration, nil
}

// GetSessionTTL parses and returns the session TTL duration.
func (c *Config) GetSessionTTL() (time.Duration, error) {
	duration, err := time.ParseDuration(c.Service.SessionTTL)
	if err != nil {
		return 0, fmt.Errorf("invalid session_ttl: %w", err)
	}

	if duration < 5*time.Minute {
		return 0, fmt.Errorf("session_ttl must be at least 5 minutes")
	}

	return duration, nil
}

// GetCommandByID returns the command definition for the given ID.
func (c *Config) GetCommandByID(id string) (*CommandDefinition, bool) {
	for i := range c.Commands {
		if c.Commands[i].ID == id {
			return &c.Commands[i], true
		}
	}
	return nil, false
}

// IsPathAllowed checks if a path is in the allow-list.
func (c *Config) IsPathAllowed(path string) bool {
	if len(c.Paths.AllowList) == 0 {
		return false
	}

	for _, allowed := range c.Paths.AllowList {
		if len(path) >= len(allowed) && path[:len(allowed)] == allowed {
			return true
		}
	}

	return false
}
