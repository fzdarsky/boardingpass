// Package command provides command execution functionality for the BoardingPass service.
package command

import (
	"fmt"
	"regexp"

	"github.com/fzdarsky/boardingpass/internal/config"
)

// AllowList validates and retrieves commands from the allow-list.
type AllowList struct {
	commands map[string]*config.CommandDefinition
}

var commandIDPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// NewAllowList creates a new AllowList from the provided command definitions.
func NewAllowList(commands []config.CommandDefinition) (*AllowList, error) {
	if len(commands) == 0 {
		return nil, fmt.Errorf("command allow-list cannot be empty")
	}

	commandMap := make(map[string]*config.CommandDefinition, len(commands))
	for i := range commands {
		cmd := &commands[i]

		// Validate command ID format
		if !commandIDPattern.MatchString(cmd.ID) {
			return nil, fmt.Errorf("invalid command ID %q: must match pattern ^[a-z0-9-]+$", cmd.ID)
		}

		// Validate command path is not empty
		if cmd.Path == "" {
			return nil, fmt.Errorf("command %q has empty path", cmd.ID)
		}

		// Check for duplicate IDs
		if _, exists := commandMap[cmd.ID]; exists {
			return nil, fmt.Errorf("duplicate command ID: %q", cmd.ID)
		}

		commandMap[cmd.ID] = cmd
	}

	return &AllowList{
		commands: commandMap,
	}, nil
}

// Get retrieves a command definition by ID.
// Returns the command definition and true if found, or nil and false if not found.
func (a *AllowList) Get(id string) (*config.CommandDefinition, bool) {
	cmd, ok := a.commands[id]
	return cmd, ok
}

// IsAllowed checks if a command ID is in the allow-list.
func (a *AllowList) IsAllowed(id string) bool {
	_, ok := a.commands[id]
	return ok
}

// Count returns the number of commands in the allow-list.
func (a *AllowList) Count() int {
	return len(a.commands)
}
