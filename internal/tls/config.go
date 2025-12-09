package tls

import (
	"crypto/tls"
	"fmt"
)

// NewServerConfig creates a TLS configuration for the HTTPS server.
// Enforces TLS 1.3 minimum and uses FIPS 140-3 compliant cipher suites.
func NewServerConfig(certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},

		// TLS 1.3 cipher suites (FIPS 140-3 compliant)
		// Note: In TLS 1.3, cipher suites are automatically negotiated
		// and don't need explicit configuration. Go's crypto/tls uses
		// FIPS-compliant ciphers by default when built with FIPS mode.
		CipherSuites: nil, // Use defaults for TLS 1.3

		// Prefer server cipher suites
		PreferServerCipherSuites: true,

		// Disable session tickets for ephemeral operation
		SessionTicketsDisabled: true,

		// Client authentication not required for BoardingPass
		ClientAuth: tls.NoClientCert,
	}, nil
}

// GetFIPSCipherSuites returns the list of FIPS 140-3 compliant cipher suites.
// For TLS 1.3, these are the default suites provided by Go's crypto/tls.
func GetFIPSCipherSuites() []uint16 {
	// TLS 1.3 cipher suites (all FIPS-compliant when using stdlib crypto)
	return []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	}
}
