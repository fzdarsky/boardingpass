package commands

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fzdarsky/boardingpass/internal/cli/client"
	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/fzdarsky/boardingpass/internal/cli/session"
	"golang.org/x/term"
)

// PassCommand implements the 'pass' command for authentication.
type PassCommand struct{}

// NewPassCommand creates a new pass command instance.
func NewPassCommand() *PassCommand {
	return &PassCommand{}
}

// Execute runs the pass command with the provided arguments.
func (c *PassCommand) Execute(args []string) {
	fs := flag.NewFlagSet("pass", flag.ExitOnError)

	// Define flags
	username := fs.String("username", "", "Username for authentication")
	password := fs.String("password", "", "Password for authentication (prompts if not provided)")
	host := fs.String("host", "", "BoardingPass service hostname or IP")
	port := fs.Int("port", 0, "BoardingPass service port")
	caCert := fs.String("ca-cert", "", "Path to custom CA certificate bundle")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: boarding pass [flags]

Authenticate with BoardingPass service using SRP-6a protocol.
The session token is stored for subsequent commands.

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Interactive (prompts for username and password)
  boarding pass --host 192.168.1.100

  # With username (prompts for password)
  boarding pass --host 192.168.1.100 --username admin

  # Non-interactive (for CI/CD)
  boarding pass --host 192.168.1.100 --username admin --password secret123

  # With custom CA certificate
  boarding pass --host internal.corp --ca-cert /etc/ssl/ca.pem --username admin
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

	// Validate configuration
	if err := cfg.RequireHost(); err != nil {
		exitWithError("%v", err)
	}

	// Get username
	user := *username
	if user == "" {
		user = promptUsername()
	}

	// Get password
	pass := *password
	if pass == "" {
		pass = promptPassword()
	}

	// Perform SRP authentication
	if err := c.authenticate(cfg, user, pass); err != nil {
		exitWithError("authentication failed: %v", err)
	}
}

// authenticate performs SRP-6a authentication and stores the session token.
func (c *PassCommand) authenticate(cfg *config.Config, username, password string) error {
	// Create API client
	apiClient, err := createClient(cfg)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Authenticating with %s...\n", cfg.Address())

	// Initialize SRP client
	srpClient := client.NewSRPClient(username, password)
	defer srpClient.ClearSecrets()

	// Phase 1: Generate ephemeral keypair and send Init request
	A, err := srpClient.GenerateEphemeralKeypair()
	if err != nil {
		return fmt.Errorf("failed to generate ephemeral keypair: %w", err)
	}

	initResp, err := apiClient.SRPInit(username, A)
	if err != nil {
		return fmt.Errorf("SRP init failed: %w", err)
	}

	// Set server response (salt and B)
	if err := srpClient.SetServerResponse(initResp.Salt, initResp.B); err != nil {
		return fmt.Errorf("invalid server response: %w", err)
	}

	// Compute shared secret and session key
	if err := srpClient.ComputeSharedSecret(); err != nil {
		return fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// Phase 2: Compute client proof and send Verify request
	M1, err := srpClient.ComputeClientProof()
	if err != nil {
		return fmt.Errorf("failed to compute client proof: %w", err)
	}

	verifyResp, err := apiClient.SRPVerify(initResp.SessionID, M1)
	if err != nil {
		return fmt.Errorf("SRP verify failed: %w", err)
	}

	// Verify server proof
	if err := srpClient.VerifyServerProof(verifyResp.M2); err != nil {
		return fmt.Errorf("server authentication failed: %w", err)
	}

	// Save session token
	store, err := session.NewStore()
	if err != nil {
		return fmt.Errorf("failed to access session store: %w", err)
	}

	if err := store.Save(cfg.Host, cfg.Port, verifyResp.SessionToken); err != nil {
		return fmt.Errorf("failed to save session token: %w", err)
	}

	// Save connection config for future commands (so --host isn't required next time)
	if err := cfg.Save(); err != nil {
		// Log warning but don't fail - authentication already succeeded
		fmt.Fprintf(os.Stderr, "Warning: failed to save connection config: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Authentication successful. Session token saved.\n")
	return nil
}

// promptUsername prompts the user to enter their username.
func promptUsername() string {
	fmt.Fprintf(os.Stderr, "Username: ")
	reader := bufio.NewReader(os.Stdin)
	username, _ := reader.ReadString('\n')
	return strings.TrimSpace(username)
}

// promptPassword prompts the user to enter their password (hidden input).
func promptPassword() string {
	fmt.Fprintf(os.Stderr, "Password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintf(os.Stderr, "\n")
	if err != nil {
		exitWithError("failed to read password: %v", err)
	}
	return string(password)
}
