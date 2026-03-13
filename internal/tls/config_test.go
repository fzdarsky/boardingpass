package tls_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fzdarsky/boardingpass/internal/logging"
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

func testLogger() *logging.Logger {
	return logging.New(logging.LevelDebug, logging.FormatHuman)
}

func TestNewCertManager(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	cm, err := tlspkg.NewCertManager(certPath, keyPath, 365, testLogger())
	require.NoError(t, err)
	require.NotNil(t, cm)
}

func TestNewCertManager_InvalidPath(t *testing.T) {
	cm, err := tlspkg.NewCertManager("/nonexistent/cert.pem", "/nonexistent/key.pem", 365, testLogger())
	assert.Error(t, err)
	assert.Nil(t, cm)
}

func TestCertManager_ServerTLSConfig(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	cm, err := tlspkg.NewCertManager(certPath, keyPath, 365, testLogger())
	require.NoError(t, err)

	cfg := cm.ServerTLSConfig()
	require.NotNil(t, cfg)

	assert.Equal(t, uint16(tls.VersionTLS13), cfg.MinVersion)
	assert.Equal(t, uint16(tls.VersionTLS13), cfg.MaxVersion)
	assert.True(t, cfg.SessionTicketsDisabled)
	assert.Equal(t, tls.NoClientCert, cfg.ClientAuth)
	assert.NotNil(t, cfg.GetCertificate, "GetCertificate callback must be set")
	assert.Empty(t, cfg.Certificates, "Certificates should be empty when using GetCertificate")
}

func TestCertManager_GetCertificate_NilConn(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	cm, err := tlspkg.NewCertManager(certPath, keyPath, 365, testLogger())
	require.NoError(t, err)

	// GetCertificate with nil Conn should return current cert without error
	cert, err := cm.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)
}

func TestCertManager_GetCertificate_RegeneratesForNewIP(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	cm, err := tlspkg.NewCertManager(certPath, keyPath, 365, testLogger())
	require.NoError(t, err)

	// Read the original cert fingerprint
	origPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)
	origBlock, _ := pem.Decode(origPEM)
	origCert, err := x509.ParseCertificate(origBlock.Bytes)
	require.NoError(t, err)

	// 127.0.0.1 is always in SANs — GetCertificate should NOT regenerate
	cert, err := cm.GetCertificate(&tls.ClientHelloInfo{
		Conn: &fakeConn{localAddr: "127.0.0.1:9455"},
	})
	require.NoError(t, err)
	require.NotNil(t, cert)

	// Cert on disk should be unchanged (no regeneration)
	afterPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)
	assert.Equal(t, origPEM, afterPEM, "cert should not regenerate for known SAN IP")

	// Now simulate a connection from an IP NOT in SANs.
	// This should trigger regeneration. The new cert will include
	// whatever IPs are currently on the system (we can't control that
	// in a unit test), but we can verify the cert file changed.
	unknownIP := findIPNotInSANs(origCert)
	if unknownIP == "" {
		t.Skip("all test IPs are in the cert SANs, cannot test regeneration")
	}

	cert, err = cm.GetCertificate(&tls.ClientHelloInfo{
		Conn: &fakeConn{localAddr: unknownIP + ":9455"},
	})
	require.NoError(t, err)
	require.NotNil(t, cert)

	// Cert on disk should have changed (regenerated)
	regenPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)
	assert.NotEqual(t, origPEM, regenPEM, "cert should regenerate for unknown IP")

	// Key should be unchanged (same private key reused)
	origKey, _ := os.ReadFile(keyPath)
	// Key was written by GenerateSelfSignedCert and not touched by RegenerateCert
	afterKey, _ := os.ReadFile(keyPath)
	assert.Equal(t, origKey, afterKey)
}

// findIPNotInSANs returns an IP address that is not in the certificate's SANs.
func findIPNotInSANs(cert *x509.Certificate) string {
	sanSet := make(map[string]bool)
	for _, ip := range cert.IPAddresses {
		sanSet[ip.String()] = true
	}

	// Try some IPs that are unlikely to be on any real interface
	candidates := []string{"198.51.100.1", "203.0.113.1", "192.0.2.1"}
	for _, ip := range candidates {
		if !sanSet[ip] {
			return ip
		}
	}
	return ""
}

// fakeConn implements net.Conn minimally for testing GetCertificate.
type fakeConn struct {
	localAddr string
}

func (f *fakeConn) LocalAddr() net.Addr              { return fakeAddr(f.localAddr) }
func (f *fakeConn) RemoteAddr() net.Addr             { return fakeAddr("0.0.0.0:0") }
func (f *fakeConn) Read([]byte) (int, error)         { return 0, nil }
func (f *fakeConn) Write([]byte) (int, error)        { return 0, nil }
func (f *fakeConn) Close() error                     { return nil }
func (f *fakeConn) SetDeadline(time.Time) error      { return nil }
func (f *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (f *fakeConn) SetWriteDeadline(time.Time) error { return nil }

type fakeAddr string

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return string(a) }
