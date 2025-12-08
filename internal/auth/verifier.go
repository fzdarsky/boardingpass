package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
)

// SRPVerifierConfig represents the SRP verifier configuration stored on disk.
// The verifier value (v) is computed dynamically at runtime by executing
// the password generator script.
type SRPVerifierConfig struct {
	Username          string `json:"username"`
	Salt              string `json:"salt"` // Base64-encoded
	PasswordGenerator string `json:"password_generator"`
}

// LoadVerifierConfig loads the SRP verifier configuration from the specified file path.
// Returns the parsed configuration or an error if loading or parsing fails.
func LoadVerifierConfig(path string) (*SRPVerifierConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read verifier config: %w", err)
	}

	var config SRPVerifierConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse verifier config: %w", err)
	}

	// Validate required fields
	if config.Username == "" {
		return nil, fmt.Errorf("username is required in verifier config")
	}
	if config.Salt == "" {
		return nil, fmt.Errorf("salt is required in verifier config")
	}
	if config.PasswordGenerator == "" {
		return nil, fmt.Errorf("password_generator is required in verifier config")
	}

	// Validate salt is valid base64
	if _, err := base64.StdEncoding.DecodeString(config.Salt); err != nil {
		return nil, fmt.Errorf("salt must be valid base64: %w", err)
	}

	return &config, nil
}

// GeneratePassword executes the password generator script and returns the device-unique password.
// The password is read from stdout and trimmed of whitespace.
func GeneratePassword(generatorPath string) (string, error) {
	cmd := exec.Command(generatorPath)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("password generator exited with error: %s (stderr: %s)", exitErr, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute password generator: %w", err)
	}

	password := strings.TrimSpace(string(output))
	if password == "" {
		return "", fmt.Errorf("password generator returned empty password")
	}

	return password, nil
}

// ComputeVerifier computes the SRP-6a verifier value: v = g^x % N
// where x = H(salt | H(username | ":" | password))
//
// Parameters:
//   - username: SRP username
//   - salt: Base64-encoded salt
//   - password: Device-unique password from generator script
//   - N: SRP group modulus (2048-bit safe prime)
//   - g: SRP group generator
//
// Returns the verifier as a big integer.
func ComputeVerifier(username, salt, password string, N, g *big.Int) (*big.Int, error) {
	// Decode salt from base64
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	// Compute x = H(salt | H(username | ":" | password))
	// Step 1: H(username | ":" | password)
	identityHash := sha256.New()
	identityHash.Write([]byte(username))
	identityHash.Write([]byte(":"))
	identityHash.Write([]byte(password))
	identityDigest := identityHash.Sum(nil)

	// Step 2: H(salt | identityDigest)
	xHash := sha256.New()
	xHash.Write(saltBytes)
	xHash.Write(identityDigest)
	xDigest := xHash.Sum(nil)

	// Convert hash to big integer
	x := new(big.Int).SetBytes(xDigest)

	// Compute v = g^x % N
	v := new(big.Int).Exp(g, x, N)

	return v, nil
}

// ComputeVerifierFromConfig is a convenience function that combines loading config,
// generating password, and computing the verifier.
func ComputeVerifierFromConfig(config *SRPVerifierConfig, N, g *big.Int) (*big.Int, error) {
	// Generate device-unique password
	password, err := GeneratePassword(config.PasswordGenerator)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	// Compute verifier
	verifier, err := ComputeVerifier(config.Username, config.Salt, password, N, g)
	if err != nil {
		return nil, fmt.Errorf("failed to compute verifier: %w", err)
	}

	// Clear password from memory immediately
	password = ""

	return verifier, nil
}
