package tls_test

import (
	"crypto/tls"
	"path/filepath"
	"testing"

	tlspkg "github.com/fzdarsky/boardingpass/internal/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	// Generate test certificate
	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	// Create TLS config
	cfg, err := tlspkg.NewServerConfig(certPath, keyPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify TLS version requirements
	assert.Equal(t, uint16(tls.VersionTLS13), cfg.MinVersion)
	assert.Equal(t, uint16(tls.VersionTLS13), cfg.MaxVersion)

	// Verify session tickets disabled
	assert.True(t, cfg.SessionTicketsDisabled)

	// Verify no client cert required
	assert.Equal(t, tls.NoClientCert, cfg.ClientAuth)

	// Verify certificate loaded
	assert.Len(t, cfg.Certificates, 1)
}

func TestNewServerConfig_InvalidCertPath(t *testing.T) {
	cfg, err := tlspkg.NewServerConfig("/nonexistent/cert.pem", "/nonexistent/key.pem")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to load TLS certificate")
}

func TestNewServerConfig_MismatchedKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	cert1Path := filepath.Join(tmpDir, "cert1.pem")
	key1Path := filepath.Join(tmpDir, "key1.pem")
	cert2Path := filepath.Join(tmpDir, "cert2.pem")
	key2Path := filepath.Join(tmpDir, "key2.pem")

	// Generate two different certificates
	err := tlspkg.GenerateSelfSignedCert(cert1Path, key1Path, 365)
	require.NoError(t, err)

	err = tlspkg.GenerateSelfSignedCert(cert2Path, key2Path, 365)
	require.NoError(t, err)

	// Try to load cert1 with key2 (mismatched)
	cfg, err := tlspkg.NewServerConfig(cert1Path, key2Path)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetFIPSCipherSuites(t *testing.T) {
	suites := tlspkg.GetFIPSCipherSuites()

	// Verify we have TLS 1.3 cipher suites
	assert.NotEmpty(t, suites)

	// Verify specific FIPS-compliant suites are included
	expectedSuites := map[uint16]bool{
		tls.TLS_AES_128_GCM_SHA256:       true,
		tls.TLS_AES_256_GCM_SHA384:       true,
		tls.TLS_CHACHA20_POLY1305_SHA256: true,
	}

	for _, suite := range suites {
		assert.True(t, expectedSuites[suite], "Unexpected cipher suite: %x", suite)
	}
}

func TestNewServerConfig_TLS13Only(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	cfg, err := tlspkg.NewServerConfig(certPath, keyPath)
	require.NoError(t, err)

	// Verify only TLS 1.3 is supported (no TLS 1.2 or earlier)
	assert.Equal(t, uint16(tls.VersionTLS13), cfg.MinVersion)
	assert.Equal(t, uint16(tls.VersionTLS13), cfg.MaxVersion)

	// Ensure TLS 1.2 is not acceptable
	assert.Greater(t, cfg.MinVersion, uint16(tls.VersionTLS12))
}
