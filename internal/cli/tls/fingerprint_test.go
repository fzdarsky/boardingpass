package tls_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	cliTLS "github.com/fzdarsky/boardingpass/internal/cli/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeFingerprint(t *testing.T) {
	// Create test certificate
	cert := createTestCertificate(t, "test.local")

	// Compute fingerprint
	fingerprint := cliTLS.ComputeFingerprint(cert)

	// Verify format
	assert.NotEmpty(t, fingerprint)
	assert.Contains(t, fingerprint, "SHA256:", "fingerprint should start with SHA256:")
	assert.Greater(t, len(fingerprint), len("SHA256:"), "fingerprint should contain hash data")
}

func TestComputeFingerprint_Consistency(t *testing.T) {
	// Create test certificate
	cert := createTestCertificate(t, "test.local")

	// Compute fingerprint multiple times
	fp1 := cliTLS.ComputeFingerprint(cert)
	fp2 := cliTLS.ComputeFingerprint(cert)
	fp3 := cliTLS.ComputeFingerprint(cert)

	// All fingerprints should be identical
	assert.Equal(t, fp1, fp2, "fingerprint should be consistent")
	assert.Equal(t, fp1, fp3, "fingerprint should be consistent")
}

func TestComputeFingerprint_UniqueCerts(t *testing.T) {
	// Create two different certificates
	cert1 := createTestCertificate(t, "host1.local")
	cert2 := createTestCertificate(t, "host2.local")

	// Compute fingerprints
	fp1 := cliTLS.ComputeFingerprint(cert1)
	fp2 := cliTLS.ComputeFingerprint(cert2)

	// Fingerprints should be different
	assert.NotEqual(t, fp1, fp2, "different certificates should have different fingerprints")
}

func TestComputeFingerprint_SameCN_DifferentKeys(t *testing.T) {
	// Create two certificates with same CN but different keys
	cert1 := createTestCertificate(t, "same.local")
	cert2 := createTestCertificate(t, "same.local")

	// Compute fingerprints
	fp1 := cliTLS.ComputeFingerprint(cert1)
	fp2 := cliTLS.ComputeFingerprint(cert2)

	// Fingerprints should be different (different private keys)
	assert.NotEqual(t, fp1, fp2, "certificates with same CN but different keys should have different fingerprints")
}

func TestFingerprintMatches_Match(t *testing.T) {
	// Create test certificate
	cert := createTestCertificate(t, "test.local")

	// Compute expected fingerprint
	expected := cliTLS.ComputeFingerprint(cert)

	// Verify match
	matches := cliTLS.FingerprintMatches(cert, expected)
	assert.True(t, matches, "fingerprint should match expected value")
}

func TestFingerprintMatches_NoMatch(t *testing.T) {
	// Create test certificate
	cert := createTestCertificate(t, "test.local")

	// Use invalid/different fingerprint
	invalidFingerprint := "SHA256:invalid-fingerprint-value"

	// Verify no match
	matches := cliTLS.FingerprintMatches(cert, invalidFingerprint)
	assert.False(t, matches, "fingerprint should not match invalid value")
}

func TestFingerprintMatches_DifferentCerts(t *testing.T) {
	// Create two different certificates
	cert1 := createTestCertificate(t, "host1.local")
	cert2 := createTestCertificate(t, "host2.local")

	// Get fingerprint of cert1
	fp1 := cliTLS.ComputeFingerprint(cert1)

	// Verify cert2 doesn't match cert1's fingerprint
	matches := cliTLS.FingerprintMatches(cert2, fp1)
	assert.False(t, matches, "different certificate should not match")
}

func TestFingerprintMatches_EmptyFingerprint(t *testing.T) {
	cert := createTestCertificate(t, "test.local")

	matches := cliTLS.FingerprintMatches(cert, "")
	assert.False(t, matches, "empty fingerprint should not match")
}

func TestFingerprintMatches_MalformedFingerprint(t *testing.T) {
	cert := createTestCertificate(t, "test.local")

	tests := []struct {
		name        string
		fingerprint string
	}{
		{
			name:        "missing SHA256 prefix",
			fingerprint: "abcdef123456",
		},
		{
			name:        "wrong prefix",
			fingerprint: "MD5:abcdef123456",
		},
		{
			name:        "invalid base64",
			fingerprint: "SHA256:!!!invalid!!!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := cliTLS.FingerprintMatches(cert, tt.fingerprint)
			assert.False(t, matches, "malformed fingerprint should not match")
		})
	}
}

// Helper function to create a test certificate

func createTestCertificate(t *testing.T, commonName string) *x509.Certificate {
	t.Helper()

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"BoardingPass Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{commonName},
	}

	// Create self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Parse certificate
	cert, err := x509.ParseCertificate(derBytes)
	require.NoError(t, err)

	return cert
}
