// Package command provides command execution functionality for the BoardingPass service.
//
//go:generate go tool mockgen -destination=mock_executor.go -package=command github.com/fzdarsky/boardingpass/internal/command CommandExecutor
package command
