package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/fzdarsky/boardingpass/internal/cli/session"
)

// CommandCommand implements the 'command' command for executing allow-listed commands.
type CommandCommand struct{}

// NewCommandCommand creates a new command execution command instance.
func NewCommandCommand() *CommandCommand {
	return &CommandCommand{}
}

// Execute runs the command execution command with the provided arguments.
func (c *CommandCommand) Execute(args []string) {
	fs := flag.NewFlagSet("command", flag.ExitOnError)

	// Define flags
	host := fs.String("host", "", "BoardingPass service hostname or IP")
	port := fs.Int("port", 0, "BoardingPass service port")
	caCert := fs.String("ca-cert", "", "Path to custom CA certificate bundle")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: boarding command [flags] <command-id>

Execute an allow-listed command on the device.
Requires prior authentication via 'boarding pass'.

The command-id must match one of the allow-listed commands configured
on the BoardingPass service. The command output (stdout/stderr) and
exit code are displayed.

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Execute an allow-listed command
  boarding command --host 192.168.1.100 restart-service

  # Using environment variables for host/port
  export BOARDING_HOST=192.168.1.100
  export BOARDING_PORT=8443
  boarding command show-status
`)
	}

	if err := fs.Parse(args); err != nil {
		exitWithError("failed to parse flags: %v", err)
	}

	// Get command ID from positional argument
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: command-id is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	commandID := fs.Arg(0)

	// Load base configuration
	cfg, err := config.Load()
	if err != nil {
		exitWithError("failed to load configuration: %v", err)
	}

	// Apply command-line flags (highest priority)
	cfg.ApplyFlags(*host, *port, *caCert)

	// Execute command
	if err := c.executeCommand(cfg, commandID); err != nil {
		exitWithError("%v", err)
	}
}

// executeCommand executes an allow-listed command on the device and displays the output.
func (c *CommandCommand) executeCommand(cfg *config.Config, commandID string) error {
	// Create API client
	apiClient, err := createClient(cfg)
	if err != nil {
		return err
	}

	// Load session token
	store, err := session.NewStore()
	if err != nil {
		return fmt.Errorf("failed to access session store: %w", err)
	}

	token, err := store.Load(cfg.Host, cfg.Port)
	if err != nil {
		return fmt.Errorf("failed to load session token: %w", err)
	}

	if token == "" {
		return fmt.Errorf("no active session. Run 'boarding pass' to authenticate")
	}

	apiClient.SetSessionToken(token)

	// Execute command
	fmt.Fprintf(os.Stderr, "Executing command '%s'...\n", commandID)

	resp, err := apiClient.ExecuteCommand(commandID)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Display stdout
	if resp.Stdout != "" {
		_, _ = fmt.Fprint(os.Stdout, resp.Stdout)
	}

	// Display stderr
	if resp.Stderr != "" {
		_, _ = fmt.Fprint(os.Stderr, resp.Stderr)
	}

	// Exit with the command's exit code
	if resp.ExitCode != 0 {
		fmt.Fprintf(os.Stderr, "\nCommand exited with code %d\n", resp.ExitCode)
		os.Exit(resp.ExitCode)
	}

	return nil
}
