// Package config provides configuration loading and validation for the BoardingPass service.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultTLSCertPath is the default path for the TLS certificate
	DefaultTLSCertPath = "/var/lib/boardingpass/tls/server.crt"
	// DefaultTLSKeyPath is the default path for the TLS private key
	DefaultTLSKeyPath = "/var/lib/boardingpass/tls/server.key"
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
	InactivityTimeout string       `yaml:"inactivity_timeout"`
	SessionTTL        string       `yaml:"session_ttl"`
	SentinelFile      string       `yaml:"sentinel_file"`
	Port              int          `yaml:"port"`
	TLSCert           string       `yaml:"tls_cert"`
	TLSKey            string       `yaml:"tls_key"`
	MDNS              MDNSSettings `yaml:"mdns"`
}

// MDNSSettings contains mDNS service announcement configuration.
type MDNSSettings struct {
	Enabled      *bool  `yaml:"enabled,omitempty"`       // default: true
	InstanceName string `yaml:"instance_name,omitempty"` // default: "BoardingPass-<hostname>"
}

// IsEnabled returns whether mDNS announcements are enabled.
// Defaults to true when not explicitly set.
func (m *MDNSSettings) IsEnabled() bool {
	return m.Enabled == nil || *m.Enabled
}

// TransportSettings contains transport-specific configuration.
type TransportSettings struct {
	Ethernet  EthernetTransport  `yaml:"ethernet"`
	WiFi      WiFiTransport      `yaml:"wifi"`
	Bluetooth BluetoothTransport `yaml:"bluetooth"`
	USB       USBTransport       `yaml:"usb"`
}

// EthernetTransport contains Ethernet transport configuration.
type EthernetTransport struct {
	Enabled    bool     `yaml:"enabled"`
	Interfaces []string `yaml:"interfaces"`
	Address    string   `yaml:"address"`
}

// WiFiTransport contains WiFi access point transport configuration.
type WiFiTransport struct {
	Enabled   bool   `yaml:"enabled"`
	Interface string `yaml:"interface"`
	SSID      string `yaml:"ssid"`
	Password  string `yaml:"password,omitempty"`
	Channel   int    `yaml:"channel"`
	Address   string `yaml:"address"`
}

// BluetoothTransport contains Bluetooth PAN transport configuration.
type BluetoothTransport struct {
	Enabled    bool   `yaml:"enabled"`
	Adapter    string `yaml:"adapter"`
	DeviceName string `yaml:"device_name"`
	Address    string `yaml:"address"`
}

// USBTransport contains USB tethering transport configuration.
type USBTransport struct {
	Enabled         bool   `yaml:"enabled"`
	InterfacePrefix string `yaml:"interface_prefix"`
	Address         string `yaml:"address,omitempty"`
}

// CommandDefinition defines an allow-listed command.
type CommandDefinition struct {
	ID        string   `yaml:"id"`
	Path      string   `yaml:"path"`
	Args      []string `yaml:"args"`
	MaxParams int      `yaml:"max_params"`     // 0 means no params accepted
	Sudo      *bool    `yaml:"sudo,omitempty"` // nil or true = use sudo (default), false = run directly
}

// NeedsSudo returns whether this command should be executed via sudo.
// Defaults to true when not explicitly set.
func (c *CommandDefinition) NeedsSudo() bool {
	return c.Sudo == nil || *c.Sudo
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

	// Apply defaults for service-level settings
	if cfg.Service.Port == 0 {
		cfg.Service.Port = 9455
	}
	if cfg.Service.TLSCert == "" {
		cfg.Service.TLSCert = DefaultTLSCertPath
	}
	if cfg.Service.TLSKey == "" {
		cfg.Service.TLSKey = DefaultTLSKeyPath
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

	if c.Service.Port <= 0 || c.Service.Port > 65535 {
		return fmt.Errorf("service.port must be between 1 and 65535")
	}

	if c.Service.TLSCert == "" {
		return fmt.Errorf("service.tls_cert is required")
	}

	if c.Service.TLSKey == "" {
		return fmt.Errorf("service.tls_key is required")
	}

	if err := c.validateWiFi(); err != nil {
		return err
	}

	if err := c.validateBluetooth(); err != nil {
		return err
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

func (c *Config) validateWiFi() error {
	if !c.Transports.WiFi.Enabled {
		return nil
	}

	// interface is optional — auto-detected at runtime if empty

	if c.Transports.WiFi.Password != "" && len(c.Transports.WiFi.Password) < 8 {
		return fmt.Errorf("transports.wifi.password must be at least 8 characters")
	}

	ch := c.Transports.WiFi.Channel
	if ch != 0 && (ch < 1 || ch > 165) {
		return fmt.Errorf("transports.wifi.channel must be between 1 and 165")
	}

	return nil
}

func (c *Config) validateBluetooth() error {
	if !c.Transports.Bluetooth.Enabled {
		return nil
	}
	// Adapter defaults to hci0 if empty — no mandatory fields beyond enabled
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
