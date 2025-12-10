package tls

import (
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"gopkg.in/yaml.v3"
)

const knownCertsFileName = "known_certs.yaml"

// CertificateEntry represents a known certificate fingerprint.
type CertificateEntry struct {
	Host        string    `yaml:"host"`
	Fingerprint string    `yaml:"fingerprint"`
	AcceptedAt  time.Time `yaml:"accepted_at"`
}

// CertificateStore manages known certificate fingerprints.
type CertificateStore struct {
	filePath string
	certs    map[string]CertificateEntry // Key: host, Value: certificate entry
}

// knownCertsFile represents the YAML structure of the known_certs.yaml file.
type knownCertsFile struct {
	Certificates []CertificateEntry `yaml:"certificates"`
}

// NewCertificateStore creates a new certificate store.
func NewCertificateStore() (*CertificateStore, error) {
	configDir, err := config.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// Ensure config directory exists
	if err := config.EnsureDir(configDir); err != nil {
		return nil, err
	}

	filePath := filepath.Join(configDir, knownCertsFileName)

	store := &CertificateStore{
		filePath: filePath,
		certs:    make(map[string]CertificateEntry),
	}

	// Load existing certificates if file exists
	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

// load loads known certificates from the YAML file.
func (s *CertificateStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, not an error
		}
		return fmt.Errorf("failed to read known certificates file: %w", err)
	}

	var file knownCertsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse known certificates file: %w", err)
	}

	// Build map from array
	for _, entry := range file.Certificates {
		s.certs[entry.Host] = entry
	}

	return nil
}

// save saves the known certificates to the YAML file.
func (s *CertificateStore) save() error {
	// Convert map to array
	certs := make([]CertificateEntry, 0, len(s.certs))
	for _, entry := range s.certs {
		certs = append(certs, entry)
	}

	file := knownCertsFile{
		Certificates: certs,
	}

	data, err := yaml.Marshal(&file)
	if err != nil {
		return fmt.Errorf("failed to marshal known certificates: %w", err)
	}

	// #nosec G306 - Certificate fingerprints are public information, 0644 is appropriate
	if err := os.WriteFile(s.filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write known certificates file: %w", err)
	}

	return nil
}

// IsKnown checks if a certificate fingerprint is known for the given host.
// Returns true if the fingerprint matches the stored value.
func (s *CertificateStore) IsKnown(host string, cert *x509.Certificate) bool {
	entry, exists := s.certs[host]
	if !exists {
		return false
	}

	actualFingerprint := ComputeFingerprint(cert)
	return entry.Fingerprint == actualFingerprint
}

// Add adds a new certificate fingerprint to the store.
func (s *CertificateStore) Add(host string, cert *x509.Certificate) error {
	fingerprint := ComputeFingerprint(cert)

	entry := CertificateEntry{
		Host:        host,
		Fingerprint: fingerprint,
		AcceptedAt:  time.Now(),
	}

	s.certs[host] = entry

	return s.save()
}

// Get retrieves the stored certificate entry for a host.
// Returns nil if not found.
func (s *CertificateStore) Get(host string) *CertificateEntry {
	if entry, exists := s.certs[host]; exists {
		return &entry
	}
	return nil
}

// Remove removes a certificate fingerprint from the store.
func (s *CertificateStore) Remove(host string) error {
	delete(s.certs, host)
	return s.save()
}

// VerifyFingerprint checks if the certificate matches the known fingerprint.
// Returns nil if the fingerprint matches or if no fingerprint is known.
// Returns an error if the fingerprint doesn't match (possible MITM attack).
func (s *CertificateStore) VerifyFingerprint(host string, cert *x509.Certificate) error {
	entry, exists := s.certs[host]
	if !exists {
		return nil // No known fingerprint, not an error (user will be prompted)
	}

	actualFingerprint := ComputeFingerprint(cert)
	if entry.Fingerprint != actualFingerprint {
		return fmt.Errorf("certificate fingerprint mismatch for %s\n"+
			"Expected: %s\n"+
			"Got:      %s\n"+
			"This could indicate a man-in-the-middle attack or certificate rotation.\n"+
			"If you trust this certificate, remove the old entry from: %s",
			host, entry.Fingerprint, actualFingerprint, s.filePath)
	}

	return nil
}
