// Package commands provides CLI command implementations for the boarding tool.
package commands

import (
	"fmt"
	"os"

	"github.com/fzdarsky/boardingpass/internal/cli/client"
	"github.com/fzdarsky/boardingpass/internal/cli/config"
)

// createClient creates a new API client from the configuration.
// It handles config loading, flag merging, and client initialization.
func createClient(cfg *config.Config) (*client.Client, error) {
	// Validate that host is set
	if err := cfg.RequireHost(); err != nil {
		return nil, err
	}

	// Create HTTP client
	apiClient, err := client.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return apiClient, nil
}

// exitWithError prints an error message to stderr and exits with status 1.
func exitWithError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
