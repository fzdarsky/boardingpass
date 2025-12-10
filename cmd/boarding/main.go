// Package main provides the boarding CLI tool for interacting with BoardingPass services.
//
// The boarding CLI enables developers and CI systems to authenticate with BoardingPass
// services, query device information, upload configurations, execute commands, and
// complete provisioning workflows.
package main

import (
	"fmt"
	"os"

	"github.com/fzdarsky/boardingpass/internal/cli/commands"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

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
		commands.NewPassCommand().Execute(os.Args[2:])
	case "info":
		commands.NewInfoCommand().Execute(os.Args[2:])
	case "connections":
		commands.NewConnectionsCommand().Execute(os.Args[2:])
	case "load":
		commands.NewLoadCommand().Execute(os.Args[2:])
	case "command":
		commands.NewCommandCommand().Execute(os.Args[2:])
	case "complete":
		commands.NewCompleteCommand().Execute(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n\n", command)
		printUsage()
		os.Exit(1)
	}
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
  --help, -h     Show help information
  --version, -v  Show version information

Examples:
  # Authenticate with BoardingPass service
  boarding pass --host 192.168.1.100 --username admin

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
