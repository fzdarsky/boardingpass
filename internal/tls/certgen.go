// Package tls provides TLS certificate generation and configuration for the BoardingPass service.
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

// getSystemHostname returns the system hostname, or empty string if unavailable.
func getSystemHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return hostname
}

// getNetworkIPs returns all non-loopback IP addresses from active network interfaces.
func getNetworkIPs() []net.IP {
	var ips []net.IP

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip loopback IPs
			if ip == nil || ip.IsLoopback() {
				continue
			}

			ips = append(ips, ip)
		}
	}

	return ips
}

// buildSANs constructs DNS names and IP addresses for certificate SANs.
// Includes static SANs (localhost, boardingpass.local) plus dynamic SANs
// (hostname, network IPs, mDNS-style names).
func buildSANs() (dnsNames []string, ipAddresses []net.IP) {
	// Static DNS SANs
	dnsNames = []string{
		"localhost",
		"boardingpass.local",
	}

	// Static IP SANs
	ipAddresses = []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("::1"),
	}

	// Add system hostname
	if hostname := getSystemHostname(); hostname != "" {
		dnsNames = append(dnsNames, hostname)

		// Add mDNS-style name: boardingpass-<hostname>.local
		mdnsName := "boardingpass-" + strings.ToLower(hostname) + ".local"
		dnsNames = append(dnsNames, mdnsName)
	}

	// Add all network interface IPs
	networkIPs := getNetworkIPs()
	ipAddresses = append(ipAddresses, networkIPs...)

	return dnsNames, ipAddresses
}

// GenerateSelfSignedCert generates a self-signed TLS certificate and key.
// Returns the paths to the generated cert and key files.
//
//nolint:gosec // G304: File paths are from config, G302: 0644 is appropriate for certs
func GenerateSelfSignedCert(certPath, keyPath string, validDays int) error {
	// Generate ECDSA private key (P-256 curve, FIPS 140-3 compliant)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate a random serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Build SANs (includes static + dynamic hostname/IPs)
	dnsNames, ipAddresses := buildSANs()

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"BoardingPass"},
			CommonName:   "BoardingPass Bootstrap Service",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Duration(validDays) * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		// Add SANs for maximum compatibility (static + dynamic)
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
	}

	// Create self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer func() {
		if cerr := certFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close cert file: %w", cerr)
		}
	}()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write cert: %w", err)
	}

	// Set certificate file permissions (read-only for all)
	//nolint:gofumpt // formatting is acceptable
	if err := os.Chmod(certPath, 0644); err != nil {
		return fmt.Errorf("failed to set cert permissions: %w", err)
	}

	// Write private key to file
	keyFile, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer func() {
		if cerr := keyFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close key file: %w", cerr)
		}
	}()

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	// Set key file permissions (read-only for owner)
	//nolint:gofumpt // formatting is acceptable
	if err := os.Chmod(keyPath, 0600); err != nil {
		return fmt.Errorf("failed to set key permissions: %w", err)
	}

	return nil
}

// CertificateExists checks if both certificate and key files exist.
func CertificateExists(certPath, keyPath string) bool {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// ValidateCertificate checks if a certificate file is valid and not expired.
//
//nolint:gosec // G304: Certificate path is from config
func ValidateCertificate(certPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check if certificate is expired
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid")
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}

	return nil
}
