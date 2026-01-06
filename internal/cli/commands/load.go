package commands

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fzdarsky/boardingpass/internal/cli/config"
	"github.com/fzdarsky/boardingpass/internal/cli/session"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const (
	maxFiles     = 100
	maxTotalSize = 10 * 1024 * 1024 // 10 MB
)

// LoadCommand implements the 'load' command for uploading configuration files.
type LoadCommand struct{}

// NewLoadCommand creates a new load command instance.
func NewLoadCommand() *LoadCommand {
	return &LoadCommand{}
}

// Execute runs the load command with the provided arguments.
func (c *LoadCommand) Execute(args []string) {
	fs := flag.NewFlagSet("load", flag.ExitOnError)

	// Define flags
	host := fs.String("host", "", "BoardingPass service hostname or IP")
	port := fs.Int("port", 0, "BoardingPass service port")
	caCert := fs.String("ca-cert", "", "Path to custom CA certificate bundle")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: boarding load [flags] <directory>

Upload configuration files from a directory to the device for provisioning.
Files are uploaded atomically - either all succeed or all fail.

Requires prior authentication via 'boarding pass'.

Limits:
  - Maximum 100 files
  - Maximum 10 MB total size

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Upload configuration directory
  boarding load --host 192.168.1.100 /path/to/config

  # Using environment variables for host/port
  export BOARDING_HOST=192.168.1.100
  export BOARDING_PORT=8443
  boarding load /etc/my-config
`)
	}

	if err := fs.Parse(args); err != nil {
		exitWithError("failed to parse flags: %v", err)
	}

	// Get directory from positional argument
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: directory path is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	directory := fs.Arg(0)

	// Load base configuration
	cfg, err := config.Load()
	if err != nil {
		exitWithError("failed to load configuration: %v", err)
	}

	// Apply command-line flags (highest priority)
	cfg.ApplyFlags(*host, *port, *caCert)

	// Execute load
	if err := c.loadConfig(cfg, directory); err != nil {
		exitWithError("%v", err)
	}
}

// loadConfig scans a directory, validates files, and uploads them to the device.
func (c *LoadCommand) loadConfig(cfg *config.Config, directory string) error {
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

	// Scan directory for files
	fmt.Fprintf(os.Stderr, "Scanning directory: %s\n", directory)
	files, err := c.scanDirectory(directory)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found in directory")
	}

	fmt.Fprintf(os.Stderr, "Found %d file(s)\n", len(files))

	// Create config bundle
	bundle := &protocol.ConfigBundle{
		Files: files,
	}

	// Upload configuration
	fmt.Fprintf(os.Stderr, "Uploading configuration...\n")
	if err := apiClient.PostConfigure(bundle); err != nil {
		return fmt.Errorf("failed to upload configuration: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Configuration uploaded successfully (%d file(s))\n", len(files))
	return nil
}

// scanDirectory walks a directory tree and collects all files for upload.
// It validates file count and total size limits.
func (c *LoadCommand) scanDirectory(directory string) ([]protocol.ConfigFile, error) {
	// Verify directory exists
	info, err := os.Stat(directory)
	if err != nil {
		return nil, fmt.Errorf("directory not found: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", directory)
	}

	var files []protocol.ConfigFile
	var totalSize int64

	// Walk directory tree
	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access %s: %w", path, err)
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check file count limit
		if len(files) >= maxFiles {
			return fmt.Errorf("file count limit exceeded (maximum %d files)", maxFiles)
		}

		// Check total size limit
		totalSize += info.Size()
		if totalSize > maxTotalSize {
			return fmt.Errorf("total size limit exceeded (maximum 10 MB)")
		}

		// Read file content
		content, err := os.ReadFile(path) // #nosec G304 - path is validated via filepath.Walk within user-provided directory
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Compute relative path from base directory
		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}

		// Add to bundle
		files = append(files, protocol.ConfigFile{
			Path:    relPath,
			Content: base64.StdEncoding.EncodeToString(content),
			Mode:    int(info.Mode().Perm()),
		})

		// Progress feedback
		fmt.Fprintf(os.Stderr, "  [%d/%d] %s (%d bytes)\n", len(files), maxFiles, relPath, info.Size())

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
