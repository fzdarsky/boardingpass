package auth_test

import (
	"encoding/base64"
	"math/big"
	"os"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/auth"
)

func TestLoadVerifierConfig(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
		errContains string
	}{
		{
			name: "valid config",
			configJSON: `{
				"username": "boardingpass",
				"salt": "dGVzdHNhbHQxMjM0NTY3ODkw",
				"password_generator": "/usr/bin/test-generator"
			}`,
			expectError: false,
		},
		{
			name: "missing username",
			configJSON: `{
				"salt": "dGVzdHNhbHQxMjM0NTY3ODkw",
				"password_generator": "/usr/bin/test-generator"
			}`,
			expectError: true,
			errContains: "username is required",
		},
		{
			name: "missing salt",
			configJSON: `{
				"username": "boardingpass",
				"password_generator": "/usr/bin/test-generator"
			}`,
			expectError: true,
			errContains: "salt is required",
		},
		{
			name: "missing password_generator",
			configJSON: `{
				"username": "boardingpass",
				"salt": "dGVzdHNhbHQxMjM0NTY3ODkw"
			}`,
			expectError: true,
			errContains: "password_generator is required",
		},
		{
			name: "invalid base64 salt",
			configJSON: `{
				"username": "boardingpass",
				"salt": "not-valid-base64!!!",
				"password_generator": "/usr/bin/test-generator"
			}`,
			expectError: true,
			errContains: "salt must be valid base64",
		},
		{
			name:        "invalid JSON",
			configJSON:  `{invalid json}`,
			expectError: true,
			errContains: "failed to parse verifier config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpfile, err := os.CreateTemp("", "verifier-*.json")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.configJSON)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Load config
			config, err := auth.LoadVerifierConfig(tmpfile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if config == nil {
					t.Error("expected non-nil config")
				}
			}
		})
	}
}

func TestLoadVerifierConfig_FileNotFound(t *testing.T) {
	_, err := auth.LoadVerifierConfig("/nonexistent/path/verifier.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !contains(err.Error(), "failed to read verifier config") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGeneratePassword(t *testing.T) {
	tests := []struct {
		name        string
		scriptBody  string
		expectError bool
		errContains string
	}{
		{
			name:        "valid password output",
			scriptBody:  "#!/bin/bash\necho 'test-password-12345'",
			expectError: false,
		},
		{
			name:        "empty output",
			scriptBody:  "#!/bin/bash\necho ''",
			expectError: true,
			errContains: "empty password",
		},
		{
			name:        "whitespace only",
			scriptBody:  "#!/bin/bash\necho '   '",
			expectError: true, // Trimmed to empty, caught by empty check
			errContains: "empty password",
		},
		{
			name:        "script exits with error",
			scriptBody:  "#!/bin/bash\nexit 1",
			expectError: true,
			errContains: "password generator exited with error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary script
			tmpfile, err := os.CreateTemp("", "generator-*.sh")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.scriptBody)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Make executable
			if err := os.Chmod(tmpfile.Name(), 0700); err != nil {
				t.Fatal(err)
			}

			// Generate password
			password, err := auth.GeneratePassword(tmpfile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if password == "" {
					t.Error("expected non-empty password")
				}
			}
		})
	}
}

func TestGeneratePassword_NonexistentScript(t *testing.T) {
	_, err := auth.GeneratePassword("/nonexistent/script.sh")
	if err == nil {
		t.Error("expected error for nonexistent script")
	}
}

func TestComputeVerifier(t *testing.T) {
	// Get SRP group parameters
	N, g, _ := auth.GetGroupParameters()

	tests := []struct {
		name     string
		username string
		salt     string
		password string
	}{
		{
			name:     "standard computation",
			username: "boardingpass",
			salt:     base64.StdEncoding.EncodeToString([]byte("testsalt12345678")),
			password: "test-password",
		},
		{
			name:     "different username",
			username: "testuser",
			salt:     base64.StdEncoding.EncodeToString([]byte("differentsalt123")),
			password: "different-password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier, err := auth.ComputeVerifier(tt.username, tt.salt, tt.password, N, g)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if verifier == nil {
				t.Error("expected non-nil verifier")
			}
			if verifier.Cmp(big.NewInt(0)) <= 0 {
				t.Error("expected positive verifier")
			}
			if verifier.Cmp(N) >= 0 {
				t.Error("expected verifier < N")
			}
		})
	}
}

func TestComputeVerifier_InvalidSalt(t *testing.T) {
	N, g, _ := auth.GetGroupParameters()

	_, err := auth.ComputeVerifier("test", "not-valid-base64!!!", "password", N, g)
	if err == nil {
		t.Error("expected error for invalid salt")
	}
	if !contains(err.Error(), "failed to decode salt") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestComputeVerifier_Deterministic(t *testing.T) {
	// Verifier should be deterministic for same inputs
	N, g, _ := auth.GetGroupParameters()

	username := "boardingpass"
	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	password := "test-password"

	v1, err := auth.ComputeVerifier(username, salt, password, N, g)
	if err != nil {
		t.Fatal(err)
	}

	v2, err := auth.ComputeVerifier(username, salt, password, N, g)
	if err != nil {
		t.Fatal(err)
	}

	if v1.Cmp(v2) != 0 {
		t.Error("verifier computation is not deterministic")
	}
}

func TestComputeVerifierFromConfig(t *testing.T) {
	// Create temporary password generator script
	script := "#!/bin/bash\necho 'device-unique-password'"
	tmpfile, err := os.CreateTemp("", "generator-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tmpfile.Name(), 0700); err != nil {
		t.Fatal(err)
	}

	// Create config
	config := &auth.SRPVerifierConfig{
		Username:          "boardingpass",
		Salt:              base64.StdEncoding.EncodeToString([]byte("testsalt")),
		PasswordGenerator: tmpfile.Name(),
	}

	N, g, _ := auth.GetGroupParameters()

	verifier, err := auth.ComputeVerifierFromConfig(config, N, g)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if verifier == nil {
		t.Error("expected non-nil verifier")
	}
	if verifier.Cmp(big.NewInt(0)) <= 0 {
		t.Error("expected positive verifier")
	}
}

func TestComputeVerifierFromConfig_GeneratorFails(t *testing.T) {
	// Create script that fails
	script := "#!/bin/bash\nexit 1"
	tmpfile, err := os.CreateTemp("", "generator-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tmpfile.Name(), 0700); err != nil {
		t.Fatal(err)
	}

	config := &auth.SRPVerifierConfig{
		Username:          "boardingpass",
		Salt:              base64.StdEncoding.EncodeToString([]byte("testsalt")),
		PasswordGenerator: tmpfile.Name(),
	}

	N, g, _ := auth.GetGroupParameters()

	_, err = auth.ComputeVerifierFromConfig(config, N, g)
	if err == nil {
		t.Error("expected error when password generator fails")
	}
	if !contains(err.Error(), "failed to generate password") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
