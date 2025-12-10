package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/fzdarsky/boardingpass/internal/cli/session"
)

// CompleteCommand implements the 'complete' command for completing provisioning.
type CompleteCommand struct{}

// NewCompleteCommand creates a new complete command instance.
func NewCompleteCommand() *CompleteCommand {
	return &CompleteCommand{}
}

// Execute runs the complete command with the provided arguments.
func (c *CompleteCommand) Execute(args []string) {
	fs := flag.NewFlagSet("complete", flag.ExitOnError)

	// Define flags
	host := fs.String("host", "", "BoardingPass service hostname or IP")
	port := fs.Int("port", 0, "BoardingPass service port")
	caCert := fs.String("ca-cert", "", "Path to custom CA certificate bundle")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: boarding complete [flags]

Signal provisioning completion to the BoardingPass service.
This triggers the service to finalize provisioning and terminate.
The session token is deleted after completion.

Requires prior authentication via 'boarding pass'.

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Complete provisioning
  boarding complete --host 192.168.1.100

  # Using environment variables for host/port
  export BOARDING_HOST=192.168.1.100
  export BOARDING_PORT=8443
  boarding complete
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

	// Execute complete
	if err := c.completeProvisioning(cfg); err != nil {
		exitWithError("%v", err)
	}
}

// completeProvisioning signals completion to the service and deletes the session token.
func (c *CompleteCommand) completeProvisioning(cfg *config.Config) error {
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

	// Signal completion
	fmt.Fprintf(os.Stderr, "Completing provisioning...\n")

	resp, err := apiClient.Complete()
	if err != nil {
		return fmt.Errorf("failed to complete provisioning: %w", err)
	}

	// Delete session token
	if err := store.Delete(cfg.Host, cfg.Port); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to delete session token: %v\n", err)
	}

	// Display completion message
	fmt.Fprintf(os.Stderr, "Provisioning completed successfully.\n")
	fmt.Fprintf(os.Stderr, "Status: %s\n", resp.Status)
	if resp.Message != nil && *resp.Message != "" {
		fmt.Fprintf(os.Stderr, "Message: %s\n", *resp.Message)
	}
	fmt.Fprintf(os.Stderr, "Sentinel file: %s\n", resp.SentinelFile)

	return nil
}
