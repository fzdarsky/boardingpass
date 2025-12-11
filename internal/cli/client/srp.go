// Package client provides HTTP client functionality for the boarding CLI tool.
package client

import (
	"github.com/fzdarsky/boardingpass/pkg/srp"
)

// SRPClient wraps the shared SRP client implementation for use in the CLI.
// This wrapper exists for backward compatibility and to maintain the CLI's API.
type SRPClient struct {
	*srp.Client
}

// NewSRPClient creates a new SRP client for authentication.
func NewSRPClient(username, password string) *SRPClient {
	return &SRPClient{
		Client: srp.NewClient(username, password),
	}
}
