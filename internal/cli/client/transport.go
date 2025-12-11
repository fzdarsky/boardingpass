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
		// For TOFU, we need to handle certificate verification ourselves
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyPeerCertificate = func(_ [][]byte, _ [][]*x509.Certificate) error {
			// This callback is called even with InsecureSkipVerify=true
			// We use it to implement TOFU verification
			return nil // Actual verification happens in RoundTrip
		}
	}

	baseTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &TOFUTransport{
		base:      baseTransport,
		certStore: certStore,
		host:      host,
	}, nil
}

// RoundTrip implements the http.RoundTripper interface with TOFU certificate verification.
func (t *TOFUTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// If we're using a custom CA (RootCAs is set), skip TOFU and use normal verification
	if t.base.TLSClientConfig.RootCAs != nil {
		return t.base.RoundTrip(req)
	}

	// For TOFU: first try the request to get the certificate
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		// Check if this is a certificate error (it shouldn't be since we have InsecureSkipVerify)
		return nil, err
	}

	// Get the peer certificate from the connection state
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("no TLS certificate received from server")
	}

	cert := resp.TLS.PeerCertificates[0]

	// Verify certificate against known fingerprints
	if err := t.certStore.VerifyFingerprint(t.host, cert); err != nil {
		_ = resp.Body.Close()
		return nil, err
	}

	// If certificate is not known, prompt user
	if !t.certStore.IsKnown(t.host, cert) {
		_ = resp.Body.Close()

		// Prompt user to accept certificate
		if !cliTLS.PromptAcceptCertificate(t.host, cert) {
			return nil, fmt.Errorf("certificate rejected by user")
		}

		// Add to known certificates
		if err := t.certStore.Add(t.host, cert); err != nil {
			return nil, fmt.Errorf("failed to save certificate: %w", err)
		}

		// Retry the request now that certificate is accepted
		return t.base.RoundTrip(req)
	}

	// Certificate is known and matches - return response
	return resp, nil
}
