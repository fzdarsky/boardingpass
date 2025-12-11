// Package tls provides TLS certificate management for the boarding CLI tool.
package tls

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

// ComputeFingerprint computes the SHA-256 fingerprint of a TLS certificate.
// The fingerprint is returned in the format "SHA256:<base64-encoded-hash>".
//
// This is used for Trust-on-First-Use (TOFU) certificate verification.
func ComputeFingerprint(cert *x509.Certificate) string {
	// Compute SHA-256 hash of the DER-encoded certificate
	hash := sha256.Sum256(cert.Raw)

	// Encode as base64
	encoded := base64.StdEncoding.EncodeToString(hash[:])

	// Format as "SHA256:<hash>"
	return fmt.Sprintf("SHA256:%s", encoded)
}

// FingerprintMatches checks if a certificate's fingerprint matches the expected value.
func FingerprintMatches(cert *x509.Certificate, expected string) bool {
	actual := ComputeFingerprint(cert)
	return actual == expected
}
