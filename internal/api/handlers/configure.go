package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fzdarsky/boardingpass/internal/config"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/internal/provisioning"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// ConfigureHandler handles POST /configure requests for configuration bundle provisioning.
type ConfigureHandler struct {
	config *config.Config
	logger *logging.Logger
}

// NewConfigureHandler creates a new configure handler.
func NewConfigureHandler(cfg *config.Config, logger *logging.Logger) *ConfigureHandler {
	return &ConfigureHandler{
		config: cfg,
		logger: logger,
	}
}

// ServeHTTP handles the POST /configure endpoint.
//
// This endpoint:
// 1. Validates bundle size (10MB max) and file count (100 files max)
// 2. Validates all file paths against the allow-list from config
// 3. Applies configuration atomically with rollback on failure
// 4. Creates sentinel file on success (triggers service shutdown)
//
// Authentication: Required (via middleware)
// Content redaction: All configuration payloads are redacted in logs
func (h *ConfigureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var bundle protocol.ConfigBundle
	if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to decode configuration bundle", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Log request (with content redaction)
	h.logger.InfoContext(r.Context(), "Configuration bundle received", map[string]any{
		"file_count": len(bundle.Files),
		"client_ip":  r.RemoteAddr,
		"bundle":     "[REDACTED]", // T085: Strict content redaction
	})

	// T082: Validate bundle size and file count (10MB max, 100 files max)
	if err := provisioning.ValidateBundle(&bundle); err != nil {
		h.logger.WarnContext(r.Context(), "Configuration bundle validation failed", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
		http.Error(w, fmt.Sprintf("Bundle validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// T083: Validate paths against allow-list
	validator := provisioning.NewPathValidator(h.config.Paths.AllowList)
	paths := make([]string, len(bundle.Files))
	for i, file := range bundle.Files {
		paths[i] = file.Path
	}
	if err := validator.ValidateAll(paths); err != nil {
		h.logger.WarnContext(r.Context(), "Path validation failed", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
		http.Error(w, fmt.Sprintf("Path validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Apply configuration bundle atomically
	applier, err := provisioning.NewApplier(validator)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to create applier", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := applier.Apply(&bundle); err != nil {
		h.logger.ErrorContext(r.Context(), "Configuration provisioning failed", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
		http.Error(w, fmt.Sprintf("Provisioning failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Cleanup temp files
	if err := applier.Cleanup(); err != nil {
		// Non-fatal: log but continue
		h.logger.WarnContext(r.Context(), "Failed to cleanup temp files", map[string]any{
			"error": err.Error(),
		})
	}

	h.logger.InfoContext(r.Context(), "Configuration provisioning successful", map[string]any{
		"file_count": len(bundle.Files),
		"client_ip":  r.RemoteAddr,
	})

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Configuration applied successfully",
	}); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to encode response", map[string]any{
			"error": err.Error(),
		})
	}
}
