package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/fzdarsky/boardingpass/internal/cli/output"
	"github.com/fzdarsky/boardingpass/internal/cli/session"
)

// ConnectionsCommand implements the 'connections' command for querying network interfaces.
type ConnectionsCommand struct{}

// NewConnectionsCommand creates a new connections command instance.
func NewConnectionsCommand() *ConnectionsCommand {
	return &ConnectionsCommand{}
}

// Execute runs the connections command with the provided arguments.
func (c *ConnectionsCommand) Execute(args []string) {
	fs := flag.NewFlagSet("connections", flag.ExitOnError)

	// Define flags
	outputFormat := fs.String("output", "yaml", "Output format (yaml or json)")
	host := fs.String("host", "", "BoardingPass service hostname or IP")
	port := fs.Int("port", 0, "BoardingPass service port")
	caCert := fs.String("ca-cert", "", "Path to custom CA certificate bundle")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: boarding connections [flags]

Query device network interface configuration including interface names,
MAC addresses, link states, and IP addresses.
Requires prior authentication via 'boarding pass'.

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Output Formats:
  yaml  YAML format (default)
  json  JSON format

Examples:
  # Query network interfaces (default YAML output)
  boarding connections --host 192.168.1.100

  # Query network interfaces with JSON output
  boarding connections --host 192.168.1.100 --output json

  # Using environment variables for host/port
  export BOARDING_HOST=192.168.1.100
  export BOARDING_PORT=8443
  boarding connections
`)
	}

	if err := fs.Parse(args); err != nil {
		exitWithError("failed to parse flags: %v", err)
	}

	// Load base configuration
	cfg, err := config.Load()
	if err != nil {
		exitWithError("failed to load configuration: %v", err)
	}

	// Apply command-line flags (highest priority)
	cfg.ApplyFlags(*host, *port, *caCert)

	// Parse output format
	format, err := output.ParseFormat(*outputFormat)
	if err != nil {
		exitWithError("%v", err)
	}

	// Execute connections query
	if err := c.getConnections(cfg, format); err != nil {
		exitWithError("%v", err)
	}
}

// getConnections queries network interface configuration from the device and displays it.
func (c *ConnectionsCommand) getConnections(cfg *config.Config, format output.Format) error {
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

	// Query network configuration
	network, err := apiClient.GetNetwork()
	if err != nil {
		return fmt.Errorf("failed to query network configuration: %w", err)
	}

	// Format and display output
	formatted, err := output.FormatData(network, format)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)
	return nil
}
