package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/fzdarsky/boardingpass/internal/inventory"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// InfoHandler handles GET /info requests.
type InfoHandler struct {
	cacheMu     sync.RWMutex
	cachedInfo  *protocol.SystemInfo
	cacheExpiry time.Time
	cacheTTL    time.Duration
}

// NewInfoHandler creates a new InfoHandler with 1-second caching.
func NewInfoHandler() *InfoHandler {
	return &InfoHandler{
		cacheTTL: 1 * time.Second,
	}
}

// ServeHTTP handles GET /info requests and returns system information.
func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check cache first
	h.cacheMu.RLock()
	if h.cachedInfo != nil && time.Now().Before(h.cacheExpiry) {
		cachedInfo := h.cachedInfo
		h.cacheMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(cachedInfo); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
		return
	}
	h.cacheMu.RUnlock()

	// Cache miss or expired - gather fresh data
	info, err := gatherSystemInfo()
	if err != nil {
		http.Error(w, "Failed to gather system information", http.StatusInternalServerError)
		return
	}

	// Update cache
	h.cacheMu.Lock()
	h.cachedInfo = &info
	h.cacheExpiry = time.Now().Add(h.cacheTTL)
	h.cacheMu.Unlock()

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// gatherSystemInfo collects system information from all inventory components.
func gatherSystemInfo() (protocol.SystemInfo, error) {
	tpmInfo, err := inventory.GetTPMInfo()
	if err != nil {
		// Non-fatal: continue with empty TPM info
		tpmInfo = protocol.TPMInfo{Present: false}
	}

	boardInfo, err := inventory.GetBoardInfo()
	if err != nil {
		// Non-fatal: continue with unknown board info
		boardInfo = protocol.BoardInfo{
			Manufacturer: "Unknown",
			Model:        "Unknown",
			Serial:       "Unknown",
		}
	}

	cpuInfo := inventory.GetCPUInfo()

	osInfo, err := inventory.GetOSInfo()
	if err != nil {
		// Non-fatal: continue with unknown OS info
		osInfo = protocol.OSInfo{
			Distribution: "Unknown",
			Version:      "Unknown",
			FIPSEnabled:  false,
		}
	}

	return protocol.SystemInfo{
		TPM:   tpmInfo,
		Board: boardInfo,
		CPU:   cpuInfo,
		OS:    osInfo,
	}, nil
}
