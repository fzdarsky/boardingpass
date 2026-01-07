package main

import (
	"fmt"
	"os"

	"github.com/fzdarsky/boardingpass/internal/auth"
	tlspkg "github.com/fzdarsky/boardingpass/internal/tls"
)

const (
	// DefaultTLSCertPath is the default path for the TLS certificate
	DefaultTLSCertPath = "/var/lib/boardingpass/tls/server.crt"
	// DefaultTLSKeyPath is the default path for the TLS private key
	DefaultTLSKeyPath = "/var/lib/boardingpass/tls/server.key"
	// DefaultVerifierPath is the default path for the SRP verifier configuration
	DefaultVerifierPath = "/etc/boardingpass/verifier"
	// DefaultPasswordGenPath is the default path for the password generator script
	DefaultPasswordGenPath = "/usr/lib/boardingpass/generators/primary_mac"
	// DefaultUsername is the default SRP username
	DefaultUsername = "boardingpass"
	// DefaultCertValidDays is the default number of days the TLS certificate is valid
	DefaultCertValidDays = 365
)

// runInit performs initialization tasks: generates TLS certificates and verifier file.
// This command is idempotent - it only creates files if they don't already exist.
// Fails fast on any error (exit non-zero).
func runInit() error {
	// Task 1: Generate TLS certificate if it doesn't exist
	if err := ensureTLSCertificate(); err != nil {
		return fmt.Errorf("TLS certificate generation failed: %w", err)
	}

	// Task 2: Generate verifier file if it doesn't exist
	if err := ensureVerifierFile(); err != nil {
		return fmt.Errorf("verifier file generation failed: %w", err)
	}

	fmt.Println("Initialization completed successfully")
	return nil
}

// ensureTLSCertificate generates a TLS certificate if it doesn't already exist.
func ensureTLSCertificate() error {
	// Check if certificate already exists
	if tlspkg.CertificateExists(DefaultTLSCertPath, DefaultTLSKeyPath) {
		fmt.Printf("TLS certificate already exists at %s\n", DefaultTLSCertPath)
		return nil
	}

	fmt.Printf("Generating TLS certificate at %s...\n", DefaultTLSCertPath)

	// Ensure parent directory exists
	tlsDir := "/var/lib/boardingpass/tls"
	//nolint:gosec // G301: 0755 is acceptable for TLS directory (cert is public, key has 0600)
	if err := os.MkdirAll(tlsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create TLS directory: %w", err)
	}

	// Generate self-signed certificate with dynamic SANs
	if err := tlspkg.GenerateSelfSignedCert(DefaultTLSCertPath, DefaultTLSKeyPath, DefaultCertValidDays); err != nil {
		return fmt.Errorf("failed to generate TLS certificate: %w", err)
	}

	fmt.Printf("TLS certificate generated successfully\n")
	return nil
}

// ensureVerifierFile generates a verifier configuration file if it doesn't already exist.
func ensureVerifierFile() error {
	// Check if verifier file already exists
	if auth.VerifierExists(DefaultVerifierPath) {
		fmt.Printf("Verifier file already exists at %s\n", DefaultVerifierPath)
		return nil
	}

	fmt.Printf("Generating verifier file at %s...\n", DefaultVerifierPath)

	// Verify password generator script exists
	if _, err := os.Stat(DefaultPasswordGenPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("password generator script not found at %s", DefaultPasswordGenPath)
		}
		return fmt.Errorf("failed to check password generator script: %w", err)
	}

	// Generate verifier file with random salt
	if err := auth.GenerateVerifierFile(DefaultVerifierPath, DefaultUsername, DefaultPasswordGenPath); err != nil {
		return fmt.Errorf("failed to generate verifier file: %w", err)
	}

	fmt.Printf("Verifier file generated successfully\n")
	return nil
}
