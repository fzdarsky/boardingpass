package tls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/fzdarsky/boardingpass/internal/logging"
)

// CertManager manages the TLS certificate with on-demand regeneration.
// When a TLS handshake arrives on an IP not in the certificate's SANs,
// the cert is regenerated (reusing the existing private key) to include
// all currently active network interfaces.
type CertManager struct {
	certPath  string
	keyPath   string
	validDays int
	logger    *logging.Logger

	mu      sync.RWMutex
	current *tls.Certificate
	sanIPs  map[string]bool
}

// NewCertManager creates a CertManager that loads the initial certificate
// from disk and serves it via GetCertificate, regenerating when needed.
func NewCertManager(certPath, keyPath string, validDays int, logger *logging.Logger) (*CertManager, error) {
	cm := &CertManager{
		certPath:  certPath,
		keyPath:   keyPath,
		validDays: validDays,
		logger:    logger,
	}

	if err := cm.loadCert(); err != nil {
		return nil, fmt.Errorf("failed to load initial certificate: %w", err)
	}

	return cm, nil
}

// GetCertificate is called by the TLS stack on each handshake. It checks
// whether the listener's local IP is covered by the current cert's SANs.
// If not, the cert is regenerated to include all current network interfaces.
func (cm *CertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	localIP := localAddrIP(hello.Conn)
	if localIP == "" {
		cm.mu.RLock()
		defer cm.mu.RUnlock()
		return cm.current, nil
	}

	cm.mu.RLock()
	if cm.sanIPs[localIP] {
		defer cm.mu.RUnlock()
		return cm.current, nil
	}
	cm.mu.RUnlock()

	// IP not in SANs — regenerate cert under write lock
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Double-check after acquiring write lock
	if cm.sanIPs[localIP] {
		return cm.current, nil
	}

	cm.logger.Info("regenerating TLS certificate for new interface IP", map[string]any{
		"ip": localIP,
	})

	if err := RegenerateCert(cm.certPath, cm.keyPath, cm.validDays); err != nil {
		cm.logger.Warn("failed to regenerate certificate, serving existing cert", map[string]any{
			"error": err.Error(),
			"ip":    localIP,
		})
		return cm.current, nil
	}

	if err := cm.loadCertLocked(); err != nil {
		cm.logger.Warn("failed to reload regenerated certificate", map[string]any{
			"error": err.Error(),
		})
	}

	return cm.current, nil
}

// ServerTLSConfig returns a tls.Config using this CertManager's GetCertificate callback.
func (cm *CertManager) ServerTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: nil, // Use defaults for TLS 1.3

		GetCertificate: cm.GetCertificate,

		NextProtos:               []string{"http/1.1"},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   true,
		ClientAuth:               tls.NoClientCert,
	}
}

// loadCert loads the certificate from disk and updates the cached state.
func (cm *CertManager) loadCert() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.loadCertLocked()
}

// loadCertLocked loads the certificate from disk. Caller must hold cm.mu write lock.
//
//nolint:gosec // G304: File paths are from config
func (cm *CertManager) loadCertLocked() error {
	cert, err := tls.LoadX509KeyPair(cm.certPath, cm.keyPath)
	if err != nil {
		return fmt.Errorf("failed to load TLS key pair: %w", err)
	}

	// Parse the leaf certificate to extract SANs
	certPEM, err := os.ReadFile(cm.certPath)
	if err != nil {
		return fmt.Errorf("failed to read cert file: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	sanIPs := make(map[string]bool, len(x509Cert.IPAddresses))
	for _, ip := range x509Cert.IPAddresses {
		sanIPs[ip.String()] = true
	}

	cm.current = &cert
	cm.sanIPs = sanIPs

	return nil
}

// localAddrIP extracts the IP string from a net.Conn's local address.
// Returns empty string if the address cannot be parsed.
func localAddrIP(conn net.Conn) string {
	if conn == nil {
		return ""
	}
	addr := conn.LocalAddr()
	if addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return ""
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	return ip.String()
}

// NewServerConfig creates a TLS configuration for the HTTPS server.
//
// Deprecated: Use NewCertManager + ServerTLSConfig for dynamic SAN support.
func NewServerConfig(certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		CipherSuites: nil,
		NextProtos:   []string{"http/1.1"},

		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   true,
		ClientAuth:               tls.NoClientCert,
	}, nil
}

// GetFIPSCipherSuites returns the list of FIPS 140-3 compliant cipher suites.
// For TLS 1.3, these are the default suites provided by Go's crypto/tls.
func GetFIPSCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	}
}
