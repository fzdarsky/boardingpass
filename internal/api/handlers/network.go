package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fzdarsky/boardingpass/internal/network"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// NetworkHandler handles GET /network requests.
type NetworkHandler struct{}

// NewNetworkHandler creates a new NetworkHandler.
func NewNetworkHandler() *NetworkHandler {
	return &NetworkHandler{}
}

// ServeHTTP handles GET /network requests and returns network configuration.
func (h *NetworkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Gather network information
	interfaces, err := network.GetInterfaces()
	if err != nil {
		http.Error(w, "Failed to gather network information", http.StatusInternalServerError)
		return
	}

	// Build response
	config := protocol.NetworkConfig{
		Interfaces: interfaces,
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
