// Package main provides the boarding CLI tool for interacting with BoardingPass services.
//
// The boarding CLI enables developers and CI systems to authenticate with BoardingPass
// services, query device information, upload configurations, execute commands, and
// complete provisioning workflows.
package main

import (
	"fmt"
	"os"

	"github.com/fzdarsky/boardingpass/internal/cli/clicontext"
	"github.com/fzdarsky/boardingpass/internal/cli/commands"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse global flags and extract command
	args, command := parseGlobalFlags(os.Args[1:])

	// Handle special commands
	switch command {
	case "--help", "-h", "help":
		printUsage()
		os.Exit(0)
	case "--version", "-v", "version":
		fmt.Printf("boarding version %s\n", version)
		os.Exit(0)
	}

	// Route to command implementations
	switch command {
	case "pass":
		commands.NewPassCommand().Execute(args)
	case "info":
		commands.NewInfoCommand().Execute(args)
	case "connections":
		commands.NewConnectionsCommand().Execute(args)
	case "load":
		commands.NewLoadCommand().Execute(args)
	case "command":
		commands.NewCommandCommand().Execute(args)
	case "complete":
		commands.NewCompleteCommand().Execute(args)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

// parseGlobalFlags processes global flags and returns remaining args and the command.
// Global flags like --assumeyes can appear anywhere in the argument list.
// Examples:
//
//	boarding -y pass --host localhost        (before command)
//	boarding pass -y --host localhost        (after command)
//	boarding pass --host localhost -y        (at the end)
func parseGlobalFlags(args []string) ([]string, string) {
	remainingArgs := make([]string, 0, len(args))
	var command string

	for i := range len(args) {
		arg := args[i]

		// Check for global flags
		if arg == "--assumeyes" || arg == "-y" {
			clicontext.SetAssumeYes(true)
			continue
		}

		// First non-flag argument is the command
		if command == "" && !isFlag(arg) {
			command = arg
			continue
		}

		// All other arguments are passed to the command
		remainingArgs = append(remainingArgs, arg)
	}

	return remainingArgs, command
}

// isFlag returns true if the argument looks like a flag (starts with -).
func isFlag(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `boarding - CLI tool for BoardingPass service provisioning

Usage:
  boarding <command> [flags]

Available Commands:
  pass         Authenticate with BoardingPass service
  info         Query system information (CPU, board, TPM, OS, FIPS)
  connections  Query network interface configuration
  load         Upload configuration directory to device
  command      Execute allow-listed command on device
  complete     Complete provisioning and terminate session

Global Flags:
  --help, -h        Show help information
  --version, -v     Show version information
  --assumeyes, -y   Automatically answer 'yes' to prompts (non-interactive mode)

Examples:
  # Authenticate with BoardingPass service
  boarding pass --host 192.168.1.100 --username admin

  # Authenticate in non-interactive mode (auto-accept certificate)
  boarding pass -y --host 192.168.1.100 --username admin --password secret

  # Query system information
  boarding info

  # Query network interfaces
  boarding connections

  # Upload configuration
  boarding load /path/to/config-directory

  # Execute command
  boarding command "systemctl restart networking"

  # Complete provisioning
  boarding complete

For detailed help on a specific command, run:
  boarding <command> --help

`)
}
