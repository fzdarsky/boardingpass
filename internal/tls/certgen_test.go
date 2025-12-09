//nolint:errcheck,gosec,gofumpt // Test file - unchecked errors and relaxed file permissions acceptable
package tls_test

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	tlspkg "github.com/fzdarsky/boardingpass/internal/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	// Verify certificate file exists
	assert.FileExists(t, certPath)

	// Verify key file exists
	assert.FileExists(t, keyPath)

	// Check certificate permissions (0644)
	certInfo, err := os.Stat(certPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), certInfo.Mode().Perm())

	// Check key permissions (0600)
	keyInfo, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), keyInfo.Mode().Perm())
}

func TestGenerateSelfSignedCert_ValidCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	// Read and parse certificate
	certPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	assert.Equal(t, "CERTIFICATE", block.Type)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	// Verify certificate properties
	assert.Contains(t, cert.Subject.Organization, "BoardingPass")
	assert.Equal(t, "BoardingPass Bootstrap Service", cert.Subject.CommonName)

	// Verify validity period
	now := time.Now()
	assert.True(t, cert.NotBefore.Before(now))
	assert.True(t, cert.NotAfter.After(now))

	// Verify it's valid for approximately 365 days
	validDuration := cert.NotAfter.Sub(cert.NotBefore)
	expectedDuration := 365 * 24 * time.Hour
	assert.InDelta(t, expectedDuration, validDuration, float64(time.Hour))

	// Verify key usage
	assert.True(t, cert.KeyUsage&x509.KeyUsageDigitalSignature != 0)
	assert.True(t, cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0)

	// Verify extended key usage
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)

	// Verify SANs
	assert.Contains(t, cert.DNSNames, "localhost")
	assert.Contains(t, cert.DNSNames, "boardingpass.local")
}

func TestGenerateSelfSignedCert_ValidPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	// Read and parse private key
	keyPEM, err := os.ReadFile(keyPath)
	require.NoError(t, err)

	block, _ := pem.Decode(keyPEM)
	require.NotNil(t, block)
	assert.Equal(t, "EC PRIVATE KEY", block.Type)

	_, err = x509.ParseECPrivateKey(block.Bytes)
	require.NoError(t, err)
}

func TestGenerateSelfSignedCert_InvalidPath(t *testing.T) {
	err := tlspkg.GenerateSelfSignedCert("/nonexistent/dir/cert.pem", "/nonexistent/dir/key.pem", 365)
	assert.Error(t, err)
}

func TestCertificateExists(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	// Should not exist initially
	assert.False(t, tlspkg.CertificateExists(certPath, keyPath))

	// Generate certificate
	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	// Should exist now
	assert.True(t, tlspkg.CertificateExists(certPath, keyPath))

	// Should not exist if cert is missing
	os.Remove(certPath)
	assert.False(t, tlspkg.CertificateExists(certPath, keyPath))

	// Regenerate
	err = tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	// Should not exist if key is missing
	os.Remove(keyPath)
	assert.False(t, tlspkg.CertificateExists(certPath, keyPath))
}

func TestValidateCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	// Generate valid certificate
	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 365)
	require.NoError(t, err)

	// Should be valid
	err = tlspkg.ValidateCertificate(certPath)
	assert.NoError(t, err)
}

func TestValidateCertificate_FileNotFound(t *testing.T) {
	err := tlspkg.ValidateCertificate("/nonexistent/cert.pem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read certificate")
}

func TestValidateCertificate_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "invalid-cert.pem")

	// Write invalid PEM data
	err := os.WriteFile(certPath, []byte("invalid pem data"), 0644)
	require.NoError(t, err)

	err = tlspkg.ValidateCertificate(certPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode PEM block")
}

func TestGenerateSelfSignedCert_CustomValidDays(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "server.crt")
	keyPath := filepath.Join(tmpDir, "server.key")

	// Generate certificate valid for 30 days
	err := tlspkg.GenerateSelfSignedCert(certPath, keyPath, 30)
	require.NoError(t, err)

	// Read and parse certificate
	certPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	// Verify it's valid for approximately 30 days
	validDuration := cert.NotAfter.Sub(cert.NotBefore)
	expectedDuration := 30 * 24 * time.Hour
	assert.InDelta(t, expectedDuration, validDuration, float64(time.Hour))
}
