package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	defaultPort    = 8443
	configFileName = "config.yaml"
	envHost        = "BOARDING_HOST"
	envPort        = "BOARDING_PORT"
	envCACert      = "BOARDING_CA_CERT"
	minPort        = 1
	maxPort        = 65535
)

// Config holds the configuration for the boarding CLI tool.
type Config struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	CACert string `yaml:"ca_cert"`
}

// Load loads configuration from file, environment variables, and applies defaults.
// Precedence order (highest to lowest):
// 1. Environment variables
// 2. Config file
// 3. Defaults
//
// Note: Command-line flags are applied by individual commands after calling Load().
func Load() (*Config, error) {
	cfg := &Config{
		Port: defaultPort,
	}

	// Layer 1: Load from config file (lowest priority)
	if err := cfg.loadFromFile(); err != nil {
		// Config file is optional, so we only return error if file exists but is invalid
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Layer 2: Load from environment variables (medium priority)
	cfg.loadFromEnv()

	// Validation
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFromFile loads configuration from the YAML config file.
func (c *Config) loadFromFile() error {
	configDir, err := UserConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, configFileName)

	data, err := os.ReadFile(configPath) // #nosec G304 - configPath is user config directory
	if err != nil {
		return err
	}

	var fileConfig Config
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Merge file config (only non-zero values)
	if fileConfig.Host != "" {
		c.Host = fileConfig.Host
	}
	if fileConfig.Port != 0 {
		c.Port = fileConfig.Port
	}
	if fileConfig.CACert != "" {
		c.CACert = fileConfig.CACert
	}

	return nil
}

// loadFromEnv loads configuration from environment variables.
func (c *Config) loadFromEnv() {
	if host := os.Getenv(envHost); host != "" {
		c.Host = host
	}

	if portStr := os.Getenv(envPort); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			c.Port = port
		}
	}

	if caCert := os.Getenv(envCACert); caCert != "" {
		c.CACert = caCert
	}
}

// ApplyFlags applies command-line flag values to the configuration.
// This should be called after Load() to apply the highest priority values.
func (c *Config) ApplyFlags(host string, port int, caCert string) {
	if host != "" {
		c.Host = host
	}
	if port != 0 {
		c.Port = port
	}
	if caCert != "" {
		c.CACert = caCert
	}
}

// Validate validates the configuration values.
func (c *Config) Validate() error {
	// Host validation is lenient - we allow empty host here because
	// some commands may not need it. Individual commands should check
	// if host is required for their operation.

	// Port validation
	if c.Port < minPort || c.Port > maxPort {
		return fmt.Errorf("invalid port %d: must be between %d and %d", c.Port, minPort, maxPort)
	}

	// CA cert validation - if specified, file must exist
	if c.CACert != "" {
		if _, err := os.Stat(c.CACert); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("CA certificate file not found: %s", c.CACert)
			}
			return fmt.Errorf("failed to access CA certificate file %s: %w", c.CACert, err)
		}
	}

	return nil
}

// RequireHost checks if host is set and returns an error with helpful message if not.
func (c *Config) RequireHost() error {
	if c.Host == "" {
		return fmt.Errorf("BoardingPass service host not specified\n"+
			"Use --host flag, %s environment variable, or add 'host:' to config file:\n"+
			"  Config file location: <UserConfigDir>/boardingpass/config.yaml\n"+
			"  Example: host: boardingpass.local", envHost)
	}
	return nil
}

// Address returns the host:port address for connecting to BoardingPass service.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
