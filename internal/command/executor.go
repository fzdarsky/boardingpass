package command

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// CommandExecutor defines the interface for executing commands.
// This interface is defined at the consumer for testing purposes.
//
//nolint:revive // Name is intentionally CommandExecutor for clarity in handler context
type CommandExecutor interface {
	Execute(ctx context.Context, cmd *config.CommandDefinition, runUsingSudo bool) (*protocol.CommandResponse, error)
}

// Executor executes commands via sudo with output capture.
type Executor struct {
	sudoPath string
}

// NewExecutor creates a new command executor.
// It verifies that sudo is available in the system.
func NewExecutor() (*Executor, error) {
	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		return nil, fmt.Errorf("sudo not found in PATH: %w", err)
	}

	return &Executor{
		sudoPath: sudoPath,
	}, nil
}

// Execute runs a command from the allow-list, optionally with sudo privileges.
// It captures stdout, stderr, and the exit code.
//
// If runUsingSudo is true, the command is executed as: sudo <path> <args...>
// If runUsingSudo is false, the command is executed as: <path> <args...>
// Context cancellation will terminate the command process.
func (e *Executor) Execute(ctx context.Context, cmd *config.CommandDefinition, runUsingSudo bool) (*protocol.CommandResponse, error) {
	if cmd == nil {
		return nil, fmt.Errorf("command definition cannot be nil")
	}

	var command *exec.Cmd

	if runUsingSudo {
		// Build command arguments: sudo <path> <args...>
		args := make([]string, 0, len(cmd.Args)+1)
		args = append(args, cmd.Path)
		args = append(args, cmd.Args...)
		//nolint:gosec // G204: Command execution with allow-listed commands is by design
		command = exec.CommandContext(ctx, e.sudoPath, args...)
	} else {
		// Run command directly without sudo
		//nolint:gosec // G204: Command execution with allow-listed commands is by design
		command = exec.CommandContext(ctx, cmd.Path, cmd.Args...)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	// Execute command
	err := command.Run()

	// Check if context was cancelled
	if ctx.Err() != nil {
		return nil, fmt.Errorf("command execution cancelled: %w", ctx.Err())
	}

	// Build response
	response := &protocol.CommandResponse{
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	// Extract exit code from error
	if err != nil {
		// Check if it's an exit error with a status code
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				response.ExitCode = status.ExitStatus()
			} else {
				// Fallback if we can't get the exit code
				response.ExitCode = 1
			}
		} else {
			// Non-exit error (e.g., command not found)
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
	}

	return response, nil
}
