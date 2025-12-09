package command_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/command"
)

func TestNewSudoersValidator(t *testing.T) {
	validator := command.NewSudoersValidator("")
	if validator == nil {
		t.Error("expected non-nil validator")
	}

	customPath := "/custom/sudoers"
	validator = command.NewSudoersValidator(customPath)
	if validator == nil {
		t.Error("expected non-nil validator with custom path")
	}
}

func TestSudoersValidator_Validate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sudoers file",
			content: `# Sudoers configuration for BoardingPass
Defaults:boardingpass !requiretty
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl reboot
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart NetworkManager
`,
			wantErr: false,
		},
		{
			name: "missing !requiretty directive",
			content: `# Sudoers configuration
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl reboot
`,
			wantErr: true,
			errMsg:  "missing required 'Defaults:boardingpass !requiretty' directive",
		},
		{
			name: "missing NOPASSWD entries",
			content: `# Sudoers configuration
Defaults:boardingpass !requiretty
`,
			wantErr: true,
			errMsg:  "missing required NOPASSWD entries",
		},
		{
			name: "valid with comments and empty lines",
			content: `# Comment line

# Another comment
Defaults:boardingpass !requiretty

# Command definitions
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl reboot

`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary sudoers file
			tmpDir := t.TempDir()
			sudoersPath := filepath.Join(tmpDir, "sudoers")

			if err := os.WriteFile(sudoersPath, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("failed to create temp sudoers file: %v", err)
			}

			validator := command.NewSudoersValidator(sudoersPath)
			err := validator.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSudoersValidator_Validate_FileNotFound(t *testing.T) {
	validator := command.NewSudoersValidator("/nonexistent/sudoers")
	err := validator.Validate()

	if err == nil {
		t.Error("expected error for non-existent file, got nil")
		return
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("expected error containing 'not found', got %q", err.Error())
	}
}

func TestSudoersValidator_ValidateCommand(t *testing.T) {
	content := `# Sudoers configuration
Defaults:boardingpass !requiretty
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl reboot --force
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart NetworkManager
`

	tmpDir := t.TempDir()
	sudoersPath := filepath.Join(tmpDir, "sudoers")

	if err := os.WriteFile(sudoersPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to create temp sudoers file: %v", err)
	}

	validator := command.NewSudoersValidator(sudoersPath)

	tests := []struct {
		name    string
		cmdPath string
		wantErr bool
	}{
		{
			name:    "existing command - systemctl",
			cmdPath: "/usr/bin/systemctl",
			wantErr: false,
		},
		{
			name:    "non-existing command",
			cmdPath: "/usr/bin/unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCommand(tt.cmdPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSudoersValidator_ValidateCommand_FileNotFound(t *testing.T) {
	validator := command.NewSudoersValidator("/nonexistent/sudoers")
	err := validator.ValidateCommand("/usr/bin/systemctl")

	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
