package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Load_Defaults(t *testing.T) {
	// Clear environment
	clearEnv(t)

	// Ensure no config file exists
	setupNoConfigFile(t)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "", cfg.Host, "default host should be empty")
	assert.Equal(t, 8443, cfg.Port, "default port should be 8443")
	assert.Equal(t, "", cfg.CACert, "default ca_cert should be empty")
}

func TestConfig_Load_FromFile(t *testing.T) {
	// Create temp CA cert file for tests
	tempCACert := createTempFile(t)

	tests := []struct {
		name       string
		fileConfig string
		wantHost   string
		wantPort   int
		wantCACert string
		wantError  bool
	}{
		{
			name: "valid config with all fields",
			fileConfig: `host: test.local
port: 9443
ca_cert: ` + tempCACert,
			wantHost:   "test.local",
			wantPort:   9443,
			wantCACert: tempCACert,
		},
		{
			name:       "valid config with only host",
			fileConfig: `host: myhost.local`,
			wantHost:   "myhost.local",
			wantPort:   8443, // default
			wantCACert: "",   // default
		},
		{
			name:       "valid config with only port",
			fileConfig: `port: 7443`,
			wantHost:   "", // default
			wantPort:   7443,
			wantCACert: "", // default
		},
		{
			name:       "empty config file",
			fileConfig: ``,
			wantHost:   "",   // default
			wantPort:   8443, // default
			wantCACert: "",   // default
		},
		{
			name:       "invalid yaml",
			fileConfig: `host: [invalid`,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv(t)

			// Setup config file
			setupConfigFile(t, tt.fileConfig)

			cfg, err := config.Load()

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, cfg.Host)
			assert.Equal(t, tt.wantPort, cfg.Port)
			assert.Equal(t, tt.wantCACert, cfg.CACert)
		})
	}
}

func TestConfig_Load_FromEnv(t *testing.T) {
	// Create temp CA cert file for tests
	tempCACert := createTempFile(t)

	tests := []struct {
		name       string
		envHost    string
		envPort    string
		envCACert  string
		wantHost   string
		wantPort   int
		wantCACert string
	}{
		{
			name:       "all env vars set",
			envHost:    "env.local",
			envPort:    "9999",
			envCACert:  tempCACert,
			wantHost:   "env.local",
			wantPort:   9999,
			wantCACert: tempCACert,
		},
		{
			name:     "only host env var",
			envHost:  "env-host.local",
			wantHost: "env-host.local",
			wantPort: 8443, // default
		},
		{
			name:     "only port env var",
			envPort:  "7777",
			wantPort: 7777,
		},
		{
			name:     "invalid port env var (ignored)",
			envPort:  "invalid",
			wantPort: 8443, // default (invalid port ignored)
			wantHost: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment and set test values
			clearEnv(t)
			if tt.envHost != "" {
				t.Setenv("BOARDING_HOST", tt.envHost)
			}
			if tt.envPort != "" {
				t.Setenv("BOARDING_PORT", tt.envPort)
			}
			if tt.envCACert != "" {
				t.Setenv("BOARDING_CA_CERT", tt.envCACert)
			}

			// Ensure no config file
			setupNoConfigFile(t)

			cfg, err := config.Load()
			require.NoError(t, err)

			assert.Equal(t, tt.wantHost, cfg.Host)
			assert.Equal(t, tt.wantPort, cfg.Port)
			assert.Equal(t, tt.wantCACert, cfg.CACert)
		})
	}
}

func TestConfig_Precedence_EnvOverFile(t *testing.T) {
	// Create temp CA cert files
	fileCACert := createTempFile(t)
	envCACert := createTempFile(t)

	// Setup config file
	setupConfigFile(t, `host: file.local
port: 8888
ca_cert: `+fileCACert)

	// Set environment variables (should override file)
	t.Setenv("BOARDING_HOST", "env.local")
	t.Setenv("BOARDING_PORT", "9999")
	t.Setenv("BOARDING_CA_CERT", envCACert)

	cfg, err := config.Load()
	require.NoError(t, err)

	// Environment variables should override file values
	assert.Equal(t, "env.local", cfg.Host, "env should override file for host")
	assert.Equal(t, 9999, cfg.Port, "env should override file for port")
	assert.Equal(t, envCACert, cfg.CACert, "env should override file for ca_cert")
}

func TestConfig_Precedence_FlagsOverEnv(t *testing.T) {
	// Setup config file
	setupConfigFile(t, `host: file.local
port: 8888`)

	// Set environment variables
	t.Setenv("BOARDING_HOST", "env.local")
	t.Setenv("BOARDING_PORT", "9999")

	cfg, err := config.Load()
	require.NoError(t, err)

	// Apply flags (highest priority)
	cfg.ApplyFlags("flag.local", 7777, "/flag/ca.pem")

	// Flags should override everything
	assert.Equal(t, "flag.local", cfg.Host, "flags should override env and file for host")
	assert.Equal(t, 7777, cfg.Port, "flags should override env and file for port")
	assert.Equal(t, "/flag/ca.pem", cfg.CACert, "flags should override env and file for ca_cert")
}

func TestConfig_Precedence_FullStack(t *testing.T) {
	// Create temp CA cert file
	fileCACert := createTempFile(t)

	// Setup all three layers
	setupConfigFile(t, `host: file.local
port: 8888
ca_cert: `+fileCACert)

	t.Setenv("BOARDING_HOST", "env.local")
	// Note: not setting env port, so file value should be used

	cfg, err := config.Load()
	require.NoError(t, err)

	// Before flags: env host, file port, file ca_cert
	assert.Equal(t, "env.local", cfg.Host)
	assert.Equal(t, 8888, cfg.Port)
	assert.Equal(t, fileCACert, cfg.CACert)

	// Apply partial flags (only host and port)
	cfg.ApplyFlags("flag.local", 7777, "")

	// After flags: flag host, flag port, file ca_cert
	assert.Equal(t, "flag.local", cfg.Host)
	assert.Equal(t, 7777, cfg.Port)
	assert.Equal(t, fileCACert, cfg.CACert, "ca_cert from file should remain when flag is empty")
}

func TestConfig_ApplyFlags_EmptyValues(t *testing.T) {
	cfg := &config.Config{
		Host:   "existing.local",
		Port:   8443,
		CACert: "/existing/ca.pem",
	}

	// Apply empty flags - should not change existing values
	cfg.ApplyFlags("", 0, "")

	assert.Equal(t, "existing.local", cfg.Host, "empty flag should not change host")
	assert.Equal(t, 8443, cfg.Port, "zero flag should not change port")
	assert.Equal(t, "/existing/ca.pem", cfg.CACert, "empty flag should not change ca_cert")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: config.Config{
				Host: "test.local",
				Port: 8443,
			},
			wantErr: false,
		},
		{
			name: "valid config with ca_cert (temp file)",
			cfg: config.Config{
				Host:   "test.local",
				Port:   8443,
				CACert: createTempFile(t),
			},
			wantErr: false,
		},
		{
			name: "port too low",
			cfg: config.Config{
				Host: "test.local",
				Port: 0,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "port too high",
			cfg: config.Config{
				Host: "test.local",
				Port: 70000,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "ca_cert file not found",
			cfg: config.Config{
				Host:   "test.local",
				Port:   8443,
				CACert: "/nonexistent/ca.pem",
			},
			wantErr: true,
			errMsg:  "CA certificate file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_RequireHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{
			name:    "host set",
			host:    "test.local",
			wantErr: false,
		},
		{
			name:    "host empty",
			host:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Host: tt.host,
				Port: 8443,
			}

			err := cfg.RequireHost()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "BoardingPass service host not specified")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Address(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		wantAddr string
	}{
		{
			name:     "standard port",
			host:     "boardingpass.local",
			port:     8443,
			wantAddr: "boardingpass.local:8443",
		},
		{
			name:     "custom port",
			host:     "192.168.1.100",
			port:     9999,
			wantAddr: "192.168.1.100:9999",
		},
		{
			name:     "empty host",
			host:     "",
			port:     8443,
			wantAddr: ":8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Host: tt.host,
				Port: tt.port,
			}

			addr := cfg.Address()
			assert.Equal(t, tt.wantAddr, addr)
		})
	}
}

// Helper functions

func clearEnv(t *testing.T) {
	t.Helper()
	// Clear relevant environment variables
	_ = os.Unsetenv("BOARDING_HOST")
	_ = os.Unsetenv("BOARDING_PORT")
	_ = os.Unsetenv("BOARDING_CA_CERT")
}

func setupConfigFile(t *testing.T, content string) {
	t.Helper()

	// Create temporary config directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "boardingpass")
	require.NoError(t, os.MkdirAll(configDir, 0o755)) // #nosec G301 - test directory, relaxed permissions acceptable

	// Write config file
	configPath := filepath.Join(configDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644)) // #nosec G306 - test file, relaxed permissions acceptable

	// Override config directory lookup
	// Note: This requires modifying config.UserConfigDir() to be testable
	// For now, we'll use a temporary directory approach
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
}

func setupNoConfigFile(t *testing.T) {
	t.Helper()

	// Create temporary directory with no config file
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
}

func createTempFile(t *testing.T) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "ca-cert-*.pem")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	// Write some dummy content
	_, err = tmpFile.WriteString("dummy cert content")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	return tmpFile.Name()
}
