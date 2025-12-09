//nolint:gosec,gofumpt // G301,G306: Test files use standard permissions; formatting is acceptable
package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()
	sentinelDir := filepath.Join(tmpDir, "etc", "boardingpass")
	require.NoError(t, os.MkdirAll(sentinelDir, 0755))

	tlsDir := filepath.Join(tmpDir, "var", "lib", "boardingpass", "tls")
	require.NoError(t, os.MkdirAll(tlsDir, 0755))

	// Create dummy command files
	rebootPath := filepath.Join(tmpDir, "usr", "sbin", "reboot")
	require.NoError(t, os.MkdirAll(filepath.Dir(rebootPath), 0755))
	require.NoError(t, os.WriteFile(rebootPath, []byte("#!/bin/sh\n"), 0755))

	systemctlPath := filepath.Join(tmpDir, "usr", "bin", "systemctl")
	require.NoError(t, os.MkdirAll(filepath.Dir(systemctlPath), 0755))
	require.NoError(t, os.WriteFile(systemctlPath, []byte("#!/bin/sh\n"), 0755))

	configYAML := `
service:
  inactivity_timeout: "10m"
  session_ttl: "30m"
  sentinel_file: "` + filepath.Join(sentinelDir, "issued") + `"

transports:
  ethernet:
    enabled: true
    interfaces: []
    address: ""
    port: 8443
    tls_cert: "` + filepath.Join(tlsDir, "server.crt") + `"
    tls_key: "` + filepath.Join(tlsDir, "server.key") + `"

commands:
  - id: "reboot"
    path: "` + rebootPath + `"
    args: []
  - id: "reload-networkd"
    path: "` + systemctlPath + `"
    args: ["reload", "systemd-networkd"]

logging:
  level: "info"
  format: "json"

paths:
  allow_list:
    - "/etc/systemd/"
    - "/etc/NetworkManager/"
`

	configFile := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(configYAML), 0644))

	cfg, err := config.Load(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "10m", cfg.Service.InactivityTimeout)
	assert.Equal(t, "30m", cfg.Service.SessionTTL)
	assert.Equal(t, filepath.Join(sentinelDir, "issued"), cfg.Service.SentinelFile)
	assert.True(t, cfg.Transports.Ethernet.Enabled)
	assert.Equal(t, 8443, cfg.Transports.Ethernet.Port)
	assert.Len(t, cfg.Commands, 2)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Len(t, cfg.Paths.AllowList, 2)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte("invalid: [yaml"), 0644))

	cfg, err := config.Load(configFile)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoad_FileNotFound(t *testing.T) {
	cfg, err := config.Load("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestGetInactivityTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     string
		expectError bool
		expected    time.Duration
	}{
		{
			name:        "valid 10 minutes",
			timeout:     "10m",
			expectError: false,
			expected:    10 * time.Minute,
		},
		{
			name:        "valid 1 hour",
			timeout:     "1h",
			expectError: false,
			expected:    1 * time.Hour,
		},
		{
			name:        "minimum 1 minute",
			timeout:     "1m",
			expectError: false,
			expected:    1 * time.Minute,
		},
		{
			name:        "below minimum",
			timeout:     "30s",
			expectError: true,
		},
		{
			name:        "invalid format",
			timeout:     "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Service: config.ServiceSettings{
					InactivityTimeout: tt.timeout,
				},
			}

			duration, err := cfg.GetInactivityTimeout()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, duration)
			}
		})
	}
}

func TestGetSessionTTL(t *testing.T) {
	tests := []struct {
		name        string
		ttl         string
		expectError bool
		expected    time.Duration
	}{
		{
			name:        "valid 30 minutes",
			ttl:         "30m",
			expectError: false,
			expected:    30 * time.Minute,
		},
		{
			name:        "valid 1 hour",
			ttl:         "1h",
			expectError: false,
			expected:    1 * time.Hour,
		},
		{
			name:        "minimum 5 minutes",
			ttl:         "5m",
			expectError: false,
			expected:    5 * time.Minute,
		},
		{
			name:        "below minimum",
			ttl:         "2m",
			expectError: true,
		},
		{
			name:        "invalid format",
			ttl:         "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Service: config.ServiceSettings{
					SessionTTL: tt.ttl,
				},
			}

			duration, err := cfg.GetSessionTTL()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, duration)
			}
		})
	}
}

func TestGetCommandByID(t *testing.T) {
	cfg := &config.Config{
		Commands: []config.CommandDefinition{
			{ID: "reboot", Path: "/usr/sbin/reboot", Args: []string{}},
			{ID: "reload-networkd", Path: "/usr/bin/systemctl", Args: []string{"reload", "systemd-networkd"}},
		},
	}

	t.Run("existing command", func(t *testing.T) {
		cmd, found := cfg.GetCommandByID("reboot")
		assert.True(t, found)
		assert.Equal(t, "reboot", cmd.ID)
		assert.Equal(t, "/usr/sbin/reboot", cmd.Path)
	})

	t.Run("non-existing command", func(t *testing.T) {
		cmd, found := cfg.GetCommandByID("nonexistent")
		assert.False(t, found)
		assert.Nil(t, cmd)
	})
}

func TestIsPathAllowed(t *testing.T) {
	cfg := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{
				"/etc/systemd/",
				"/etc/NetworkManager/",
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "allowed systemd path",
			path:     "/etc/systemd/network/10-eth0.network",
			expected: true,
		},
		{
			name:     "allowed NetworkManager path",
			path:     "/etc/NetworkManager/system-connections/wifi.nmconnection",
			expected: true,
		},
		{
			name:     "not allowed passwd",
			path:     "/etc/passwd",
			expected: false,
		},
		{
			name:     "not allowed shadow",
			path:     "/etc/shadow",
			expected: false,
		},
		{
			name:     "not allowed arbitrary path",
			path:     "/home/user/config",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, cfg.IsPathAllowed(tt.path))
		})
	}
}

func TestIsPathAllowed_EmptyAllowList(t *testing.T) {
	cfg := &config.Config{
		Paths: config.PathSettings{
			AllowList: []string{},
		},
	}

	assert.False(t, cfg.IsPathAllowed("/etc/systemd/network/10-eth0.network"))
}

func TestConfig_Validate_MissingFields(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectedErr string
	}{
		{
			name: "missing inactivity_timeout",
			yamlContent: `
service:
  session_ttl: "30m"
  sentinel_file: "/etc/boardingpass/issued"
`,
			expectedErr: "inactivity_timeout is required",
		},
		{
			name: "missing session_ttl",
			yamlContent: `
service:
  inactivity_timeout: "10m"
  sentinel_file: "/etc/boardingpass/issued"
`,
			expectedErr: "session_ttl is required",
		},
		{
			name: "missing sentinel_file",
			yamlContent: `
service:
  inactivity_timeout: "10m"
  session_ttl: "30m"
`,
			expectedErr: "sentinel_file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")
			require.NoError(t, os.WriteFile(configFile, []byte(tt.yamlContent), 0644))

			cfg, err := config.Load(configFile)
			assert.Error(t, err)
			assert.Nil(t, cfg)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestConfig_Validate_InvalidPort(t *testing.T) {
	tmpDir := t.TempDir()
	sentinelDir := filepath.Join(tmpDir, "etc", "boardingpass")
	require.NoError(t, os.MkdirAll(sentinelDir, 0755))

	configYAML := `
service:
  inactivity_timeout: "10m"
  session_ttl: "30m"
  sentinel_file: "` + filepath.Join(sentinelDir, "issued") + `"

transports:
  ethernet:
    enabled: true
    port: 99999
    tls_cert: "/var/lib/boardingpass/tls/server.crt"
    tls_key: "/var/lib/boardingpass/tls/server.key"
`

	configFile := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(configYAML), 0644))

	cfg, err := config.Load(configFile)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
}
