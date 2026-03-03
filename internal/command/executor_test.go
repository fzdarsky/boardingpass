package command_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/fzdarsky/boardingpass/internal/command"
	"github.com/fzdarsky/boardingpass/internal/config"
)

// checkSudoAvailable checks if passwordless sudo is available for testing.
// Returns true if sudo can be executed without password.
func checkSudoAvailable() bool {
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	return err == nil
}

func TestNewExecutor(t *testing.T) {
	executor, err := command.NewExecutor()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if executor == nil {
		t.Error("expected non-nil executor")
	}
}

// lookPathOrSkip resolves a command path, skipping the test if the command is not found.
func lookPathOrSkip(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("skipping: %q not found in PATH", name)
	}
	return path
}

func TestExecutor_Execute(t *testing.T) {
	executor, err := command.NewExecutor()
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	echoPath := lookPathOrSkip(t, "echo")
	truePath := lookPathOrSkip(t, "true")
	falsePath := lookPathOrSkip(t, "false")

	tests := []struct {
		name         string
		cmd          *config.CommandDefinition
		wantExitCode int
		wantStdout   string
		wantStderr   string
		wantErr      bool
	}{
		{
			name: "successful command - echo",
			cmd: &config.CommandDefinition{
				ID:   "echo-test",
				Path: echoPath,
				Args: []string{"hello", "world"},
			},
			wantExitCode: 0,
			wantStdout:   "hello world\n",
			wantStderr:   "",
			wantErr:      false,
		},
		{
			name: "successful command - true",
			cmd: &config.CommandDefinition{
				ID:   "true-test",
				Path: truePath,
				Args: []string{},
			},
			wantExitCode: 0,
			wantStdout:   "",
			wantStderr:   "",
			wantErr:      false,
		},
		{
			name: "failed command - false",
			cmd: &config.CommandDefinition{
				ID:   "false-test",
				Path: falsePath,
				Args: []string{},
			},
			wantExitCode: 1,
			wantStdout:   "",
			wantStderr:   "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := executor.Execute(ctx, tt.cmd, false, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if response == nil {
				t.Error("expected non-nil response")
				return
			}

			if response.ExitCode != tt.wantExitCode {
				t.Errorf("ExitCode = %d, want %d", response.ExitCode, tt.wantExitCode)
			}

			if response.Stdout != tt.wantStdout {
				t.Errorf("Stdout = %q, want %q", response.Stdout, tt.wantStdout)
			}

			if response.Stderr != tt.wantStderr {
				t.Errorf("Stderr = %q, want %q", response.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestExecutor_Execute_NilCommand(t *testing.T) {
	executor, err := command.NewExecutor()
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	_, err = executor.Execute(ctx, nil, false, nil)

	if err == nil {
		t.Error("expected error for nil command, got nil")
	}
}

func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	executor, err := command.NewExecutor()
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Try to run a command that sleeps longer than the timeout
	sleepPath := lookPathOrSkip(t, "sleep")
	cmd := &config.CommandDefinition{
		ID:   "sleep-test",
		Path: sleepPath,
		Args: []string{"5"},
	}

	_, err = executor.Execute(ctx, cmd, false, nil)

	// Should get an error due to context cancellation
	if err == nil {
		t.Error("expected error due to context cancellation, got nil")
	}
}

func TestExecutor_Execute_StderrCapture(t *testing.T) {
	executor, err := command.NewExecutor()
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	// Use a command that writes to stderr
	shPath := lookPathOrSkip(t, "sh")
	cmd := &config.CommandDefinition{
		ID:   "stderr-test",
		Path: shPath,
		Args: []string{"-c", "echo error >&2"},
	}

	ctx := context.Background()
	response, err := executor.Execute(ctx, cmd, false, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if response == nil {
		t.Error("expected non-nil response")
		return
	}

	if response.Stderr != "error\n" {
		t.Errorf("Stderr = %q, want %q", response.Stderr, "error\n")
	}
}
