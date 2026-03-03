package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fzdarsky/boardingpass/internal/lifecycle"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const rebootDelay = 2 * time.Second

// CompleteHandler handles POST /complete requests for marking provisioning complete.
type CompleteHandler struct {
	sentinel     *lifecycle.Sentinel
	shutdownFunc func(string)
	rebootFunc   func()
	logger       *logging.Logger
}

// NewCompleteHandler creates a new complete handler.
func NewCompleteHandler(sentinelPath string, shutdownFunc func(string), rebootFunc func(), logger *logging.Logger) *CompleteHandler {
	return &CompleteHandler{
		sentinel:     lifecycle.NewSentinel(sentinelPath),
		shutdownFunc: shutdownFunc,
		rebootFunc:   rebootFunc,
		logger:       logger,
	}
}

// ServeHTTP handles the POST /complete endpoint.
//
// This endpoint:
// 1. Parses optional request body for reboot flag
// 2. Creates the sentinel file to prevent service from starting again
// 3. Initiates graceful service shutdown or schedules reboot
// 4. Returns response confirming shutdown/reboot is in progress
//
// Authentication: Required (via middleware)
func (h *CompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse optional request body
	var req protocol.CompleteRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.ErrorContext(r.Context(), "Failed to decode complete request", map[string]any{
				"error":     err.Error(),
				"client_ip": r.RemoteAddr,
			})
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
	}

	h.logger.InfoContext(r.Context(), "Provisioning completion requested", map[string]any{
		"reboot":    req.Reboot,
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

	// Determine status based on reboot flag
	status := "shutting_down"
	if req.Reboot {
		status = "rebooting"
	}

	// Return success response before initiating shutdown/reboot
	response := protocol.CompleteResponse{
		Status:       status,
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

	// After sending response, initiate shutdown or reboot
	if req.Reboot && h.rebootFunc != nil {
		h.logger.InfoContext(r.Context(), "Scheduling reboot", map[string]any{
			"delay":     rebootDelay.String(),
			"client_ip": r.RemoteAddr,
		})
		go func() {
			time.Sleep(rebootDelay)
			h.rebootFunc()
		}()
	} else if h.shutdownFunc != nil {
		h.shutdownFunc("provisioning completed")
		h.logger.InfoContext(r.Context(), "Graceful shutdown initiated", map[string]any{
			"client_ip": r.RemoteAddr,
		})
	}
}
