package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fzdarsky/boardingpass/internal/lifecycle"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// CompleteHandler handles POST /complete requests for marking provisioning complete.
type CompleteHandler struct {
	sentinel     *lifecycle.Sentinel
	shutdownFunc func(string)
	logger       *logging.Logger
}

// NewCompleteHandler creates a new complete handler.
func NewCompleteHandler(sentinelPath string, shutdownFunc func(string), logger *logging.Logger) *CompleteHandler {
	return &CompleteHandler{
		sentinel:     lifecycle.NewSentinel(sentinelPath),
		shutdownFunc: shutdownFunc,
		logger:       logger,
	}
}

// ServeHTTP handles the POST /complete endpoint.
//
// This endpoint:
// 1. Creates the sentinel file to prevent service from starting again
// 2. Initiates graceful service shutdown
// 3. Returns response confirming shutdown is in progress
//
// Authentication: Required (via middleware)
func (h *CompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.InfoContext(r.Context(), "Provisioning completion requested", map[string]any{
		"client_ip": r.RemoteAddr,
	})

	// Create sentinel file
	if err := h.sentinel.Create(); err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to create sentinel file", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
		http.Error(w, fmt.Sprintf("Failed to create sentinel file: %v", err), http.StatusInternalServerError)
		return
	}

	h.logger.InfoContext(r.Context(), "Sentinel file created", map[string]any{
		"path":      h.sentinel.Path(),
		"client_ip": r.RemoteAddr,
	})

	// Initiate graceful shutdown if shutdown function is provided
	if h.shutdownFunc != nil {
		h.shutdownFunc("provisioning completed")
		h.logger.InfoContext(r.Context(), "Graceful shutdown initiated", map[string]any{
			"client_ip": r.RemoteAddr,
		})
	}

	// Return success response
	response := protocol.CompleteResponse{
		Status:       "shutting_down",
		SentinelFile: h.sentinel.Path(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.ErrorContext(r.Context(), "Failed to encode response", map[string]any{
			"error":     err.Error(),
			"client_ip": r.RemoteAddr,
		})
	}
}
