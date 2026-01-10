package main

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/cli/clicontext"
)

func TestParseGlobalFlags(t *testing.T) {
	tests := []struct {
		name              string
		input             []string
		expectedCommand   string
		expectedArgs      []string
		expectedAssumeYes bool
	}{
		{
			name:              "global flag before command",
			input:             []string{"-y", "pass", "--host", "localhost"},
			expectedCommand:   "pass",
			expectedArgs:      []string{"--host", "localhost"},
			expectedAssumeYes: true,
		},
		{
			name:              "global flag after command",
			input:             []string{"pass", "-y", "--host", "localhost"},
			expectedCommand:   "pass",
			expectedArgs:      []string{"--host", "localhost"},
			expectedAssumeYes: true,
		},
		{
			name:              "global flag at end",
			input:             []string{"pass", "--host", "localhost", "-y"},
			expectedCommand:   "pass",
			expectedArgs:      []string{"--host", "localhost"},
			expectedAssumeYes: true,
		},
		{
			name:              "long form global flag",
			input:             []string{"pass", "--assumeyes", "--host", "localhost"},
			expectedCommand:   "pass",
			expectedArgs:      []string{"--host", "localhost"},
			expectedAssumeYes: true,
		},
		{
			name:              "no global flag",
			input:             []string{"pass", "--host", "localhost"},
			expectedCommand:   "pass",
			expectedArgs:      []string{"--host", "localhost"},
			expectedAssumeYes: false,
		},
		{
			name:              "multiple global flags",
			input:             []string{"-y", "pass", "--host", "localhost", "-y"},
			expectedCommand:   "pass",
			expectedArgs:      []string{"--host", "localhost"},
			expectedAssumeYes: true,
		},
		{
			name:              "command only",
			input:             []string{"info"},
			expectedCommand:   "info",
			expectedArgs:      []string{},
			expectedAssumeYes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global context before each test
			clicontext.SetAssumeYes(false)

			// Parse flags
			args, command := parseGlobalFlags(tt.input)

			// Verify command
			if command != tt.expectedCommand {
				t.Errorf("parseGlobalFlags() command = %v, want %v", command, tt.expectedCommand)
			}

			// Verify remaining args
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("parseGlobalFlags() args length = %v, want %v", len(args), len(tt.expectedArgs))
			} else {
				for i, arg := range args {
					if arg != tt.expectedArgs[i] {
						t.Errorf("parseGlobalFlags() args[%d] = %v, want %v", i, arg, tt.expectedArgs[i])
					}
				}
			}

			// Verify AssumeYes flag
			if clicontext.AssumeYes() != tt.expectedAssumeYes {
				t.Errorf("parseGlobalFlags() AssumeYes = %v, want %v", clicontext.AssumeYes(), tt.expectedAssumeYes)
			}
		})
	}
}

func TestIsFlag(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		expected bool
	}{
		{"short flag", "-y", true},
		{"long flag", "--assumeyes", true},
		{"command", "pass", false},
		{"value", "localhost", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFlag(tt.arg); got != tt.expected {
				t.Errorf("isFlag(%q) = %v, want %v", tt.arg, got, tt.expected)
			}
		})
	}
}
