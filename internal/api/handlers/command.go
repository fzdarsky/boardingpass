package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fzdarsky/boardingpass/internal/command"
	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// CommandHandler handles POST /command requests for executing allow-listed commands.
type CommandHandler struct {
	allowList *command.AllowList
	executor  command.CommandExecutor
	logger    *logging.Logger
}

// NewCommandHandler creates a new command handler.
func NewCommandHandler(cfg *config.Config, logger *logging.Logger) (*CommandHandler, error) {
	// Create executor
	executor, err := command.NewExecutor()
	if err != nil {
		return nil, fmt.Errorf("failed to create command executor: %w", err)
	}

	return NewCommandHandlerWithExecutor(cfg, executor, logger)
}

// NewCommandHandlerWithExecutor creates a new command handler with a custom executor.
// This is useful for testing with mock executors.
func NewCommandHandlerWithExecutor(cfg *config.Config, executor command.CommandExecutor, logger *logging.Logger) (*CommandHandler, error) {
	// T095: Create allow-list from configuration
	allowList, err := command.NewAllowList(cfg.Commands)
	if err != nil {
		return nil, fmt.Errorf("failed to create command allow-list: %w", err)
	}

	return &CommandHandler{
		allowList: allowList,
		executor:  executor,
		logger:    logger,
	}, nil
}

// ServeHTTP handles the POST /command endpoint.
//
// This endpoint:
// 1. Validates command ID against the allow-list (T095)
// 2. Executes the command via sudo
// 3. Captures stdout, stderr, and exit code (T096)
// 4. Logs execution with exit codes (T098)
//
// Authentication: Required (via middleware) (T097)
func (h *CommandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req protocol.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to decode command request", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Log request
	h.logger.InfoContext(r.Context(), "Command execution requested", map[string]any{
		"command_id": req.ID,
		"client_ip":  r.RemoteAddr,
	})

	// T095: Validate command ID against allow-list
	cmdDef, found := h.allowList.Get(req.ID)
	if !found {
		h.logger.WarnContext(r.Context(), "Command not in allow-list", map[string]any{
			"command_id": req.ID,
			"client_ip":  r.RemoteAddr,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error":   "command_not_allowed",
			"message": fmt.Sprintf("Command %q is not in the allow-list", req.ID),
		}); err != nil {
			h.logger.ErrorContext(r.Context(), "Failed to encode error response", map[string]any{
				"error":     err.Error(),
				"client_ip": r.RemoteAddr,
			})
		}
		return
	}

	// T096: Execute command and capture stdout/stderr (with sudo)
	response, err := h.executor.Execute(r.Context(), cmdDef, true)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Command execution failed", map[string]any{
			"command_id": req.ID,
			"error":      err.Error(),
			"client_ip":  r.RemoteAddr,
		})
		http.Error(w, fmt.Sprintf("Command execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// T098: Log execution with exit code
	h.logger.InfoContext(r.Context(), "Command executed", map[string]any{
		"command_id":  req.ID,
		"exit_code":   response.ExitCode,
		"stdout_size": len(response.Stdout),
		"stderr_size": len(response.Stderr),
		"client_ip":   r.RemoteAddr,
	})

	// T096: Return response with stdout/stderr and exit code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to encode response", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
	}
}
