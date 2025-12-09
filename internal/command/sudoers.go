package command

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultSudoersPath is the default location for the boardingpass sudoers file.
	DefaultSudoersPath = "/etc/sudoers.d/boardingpass"

	// RequiredUser is the user that must have sudo permissions.
	RequiredUser = "boardingpass"
)

// SudoersValidator validates the sudoers configuration file.
type SudoersValidator struct {
	path string
}

// NewSudoersValidator creates a new sudoers validator.
func NewSudoersValidator(path string) *SudoersValidator {
	if path == "" {
		path = DefaultSudoersPath
	}
	return &SudoersValidator{
		path: path,
	}
}

// Validate checks if the sudoers file exists and has basic required configuration.
// It verifies:
// - File exists and is readable
// - Contains !requiretty for boardingpass user
// - Contains at least one NOPASSWD entry for boardingpass user
func (v *SudoersValidator) Validate() error {
	// Check if file exists
	file, err := os.Open(v.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("sudoers file not found at %s", v.path)
		}
		return fmt.Errorf("failed to open sudoers file: %w", err)
	}
	defer file.Close() //nolint:errcheck // Read-only operation, close error is not actionable

	var hasRequireTTY bool
	var hasNOPASSWD bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for !requiretty directive
		if strings.Contains(line, "Defaults:"+RequiredUser) && strings.Contains(line, "!requiretty") {
			hasRequireTTY = true
		}

		// Check for NOPASSWD entries
		if strings.HasPrefix(line, RequiredUser+" ") && strings.Contains(line, "NOPASSWD:") {
			hasNOPASSWD = true
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read sudoers file: %w", err)
	}

	// Validate required configuration
	if !hasRequireTTY {
		return fmt.Errorf("sudoers file missing required 'Defaults:%s !requiretty' directive", RequiredUser)
	}

	if !hasNOPASSWD {
		return fmt.Errorf("sudoers file missing required NOPASSWD entries for user %s", RequiredUser)
	}

	return nil
}

// ValidateCommand checks if a specific command path is permitted in the sudoers file.
// This is a basic check that looks for the command path in NOPASSWD entries.
func (v *SudoersValidator) ValidateCommand(cmdPath string) error {
	file, err := os.Open(v.path)
	if err != nil {
		return fmt.Errorf("failed to open sudoers file: %w", err)
	}
	defer file.Close() //nolint:errcheck // Read-only operation, close error is not actionable

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if this line permits the command
		if strings.HasPrefix(line, RequiredUser+" ") &&
			strings.Contains(line, "NOPASSWD:") &&
			strings.Contains(line, cmdPath) {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read sudoers file: %w", err)
	}

	return fmt.Errorf("command %s not found in sudoers file", cmdPath)
}
