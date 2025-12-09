package command_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/command"
	"github.com/fzdarsky/boardingpass/internal/config"
)

func TestNewAllowList(t *testing.T) {
	tests := []struct {
		name     string
		commands []config.CommandDefinition
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid commands",
			commands: []config.CommandDefinition{
				{ID: "reboot", Path: "/usr/bin/systemctl", Args: []string{"reboot", "--force"}},
				{ID: "restart-network", Path: "/usr/bin/systemctl", Args: []string{"restart", "NetworkManager"}},
			},
			wantErr: false,
		},
		{
			name:     "empty command list",
			commands: []config.CommandDefinition{},
			wantErr:  true,
			errMsg:   "command allow-list cannot be empty",
		},
		{
			name: "invalid command ID - uppercase",
			commands: []config.CommandDefinition{
				{ID: "REBOOT", Path: "/usr/bin/systemctl", Args: []string{"reboot"}},
			},
			wantErr: true,
			errMsg:  "invalid command ID",
		},
		{
			name: "invalid command ID - special chars",
			commands: []config.CommandDefinition{
				{ID: "restart_network", Path: "/usr/bin/systemctl", Args: []string{"restart"}},
			},
			wantErr: true,
			errMsg:  "invalid command ID",
		},
		{
			name: "empty command path",
			commands: []config.CommandDefinition{
				{ID: "reboot", Path: "", Args: []string{"reboot"}},
			},
			wantErr: true,
			errMsg:  "has empty path",
		},
		{
			name: "duplicate command ID",
			commands: []config.CommandDefinition{
				{ID: "reboot", Path: "/usr/bin/systemctl", Args: []string{"reboot"}},
				{ID: "reboot", Path: "/sbin/reboot", Args: []string{}},
			},
			wantErr: true,
			errMsg:  "duplicate command ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowList, err := command.NewAllowList(tt.commands)

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
				return
			}

			if allowList == nil {
				t.Error("expected non-nil allow list")
			}
		})
	}
}

func TestAllowList_Get(t *testing.T) {
	commands := []config.CommandDefinition{
		{ID: "reboot", Path: "/usr/bin/systemctl", Args: []string{"reboot", "--force"}},
		{ID: "restart-network", Path: "/usr/bin/systemctl", Args: []string{"restart", "NetworkManager"}},
	}

	allowList, err := command.NewAllowList(commands)
	if err != nil {
		t.Fatalf("failed to create allow list: %v", err)
	}

	tests := []struct {
		name      string
		id        string
		wantFound bool
		wantPath  string
	}{
		{
			name:      "existing command - reboot",
			id:        "reboot",
			wantFound: true,
			wantPath:  "/usr/bin/systemctl",
		},
		{
			name:      "existing command - restart-network",
			id:        "restart-network",
			wantFound: true,
			wantPath:  "/usr/bin/systemctl",
		},
		{
			name:      "non-existing command",
			id:        "unknown",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, found := allowList.Get(tt.id)

			if found != tt.wantFound {
				t.Errorf("Get(%q) found = %v, want %v", tt.id, found, tt.wantFound)
			}

			if tt.wantFound {
				if cmd == nil {
					t.Errorf("expected non-nil command for %q", tt.id)
					return
				}
				if cmd.Path != tt.wantPath {
					t.Errorf("Get(%q) path = %q, want %q", tt.id, cmd.Path, tt.wantPath)
				}
			} else if cmd != nil {
				t.Errorf("expected nil command for %q, got %+v", tt.id, cmd)
			}
		})
	}
}

func TestAllowList_IsAllowed(t *testing.T) {
	commands := []config.CommandDefinition{
		{ID: "reboot", Path: "/usr/bin/systemctl", Args: []string{"reboot"}},
	}

	allowList, err := command.NewAllowList(commands)
	if err != nil {
		t.Fatalf("failed to create allow list: %v", err)
	}

	if !allowList.IsAllowed("reboot") {
		t.Error("expected 'reboot' to be allowed")
	}

	if allowList.IsAllowed("unknown") {
		t.Error("expected 'unknown' to not be allowed")
	}
}

func TestAllowList_Count(t *testing.T) {
	commands := []config.CommandDefinition{
		{ID: "reboot", Path: "/usr/bin/systemctl", Args: []string{"reboot"}},
		{ID: "restart-network", Path: "/usr/bin/systemctl", Args: []string{"restart", "NetworkManager"}},
	}

	allowList, err := command.NewAllowList(commands)
	if err != nil {
		t.Fatalf("failed to create allow list: %v", err)
	}

	count := allowList.Count()
	if count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
