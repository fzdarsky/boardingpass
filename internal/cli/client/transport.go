package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	cliTLS "github.com/fzdarsky/boardingpass/internal/cli/tls"
)

// TOFUTransport is a custom HTTP transport that implements Trust-On-First-Use
// for TLS certificates.
type TOFUTransport struct {
	base      *http.Transport
	certStore *cliTLS.CertificateStore
	host      string
}

// NewTOFUTransport creates a new TOFU transport for the specified host.
// If caCertPath is provided, it will be used for certificate validation instead of TOFU.
func NewTOFUTransport(host string, caCertPath string) (*TOFUTransport, error) {
	// Create certificate store
	certStore, err := cliTLS.NewCertificateStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate store: %w", err)
	}

	// Create base TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	transport := &TOFUTransport{
		certStore: certStore,
		host:      host,
	}

	// If custom CA is provided, use it for validation
	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath) // #nosec G304 - caCertPath is user-provided config
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = certPool
	} else {
		// For TOFU, we verify certificates in the VerifyPeerCertificate callback
		// This runs during TLS handshake, before the request body is sent
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyPeerCertificate = transport.verifyTOFU
	}

	transport.base = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return transport, nil
}

// verifyTOFU implements Trust-On-First-Use certificate verification.
// This is called during the TLS handshake, before any request body is sent.
func (t *TOFUTransport) verifyTOFU(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	// With InsecureSkipVerify=true, verifiedChains is empty
	// We need to parse the raw certificates ourselves
	if len(rawCerts) == 0 {
		return fmt.Errorf("no TLS certificate received from server")
	}

	// Parse the first (peer) certificate
	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse server certificate: %w", err)
	}

	// Verify certificate against known fingerprints
	if err := t.certStore.VerifyFingerprint(t.host, cert); err != nil {
		return err
	}

	// If certificate is not known, prompt user
	if !t.certStore.IsKnown(t.host, cert) {
		// Prompt user to accept certificate
		if !cliTLS.PromptAcceptCertificate(t.host, cert) {
			return fmt.Errorf("certificate rejected by user")
		}

		// Add to known certificates
		if err := t.certStore.Add(t.host, cert); err != nil {
			return fmt.Errorf("failed to save certificate: %w", err)
		}
	}

	return nil
}

// RoundTrip implements the http.RoundTripper interface.
// Certificate verification happens in the TLS callback, so this just delegates to the base transport.
func (t *TOFUTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req)
}
