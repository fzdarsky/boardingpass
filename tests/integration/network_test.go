package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
)

func TestNetworkEndpoint(t *testing.T) {
	// Create handler
	handler := handlers.NewNetworkHandler()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/network", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Check response status
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP 200 OK")

	// Check content type
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response body
	var config protocol.NetworkConfig
	err := json.NewDecoder(w.Body).Decode(&config)
	assert.NoError(t, err, "Response should be valid JSON")

	// Verify response structure
	assert.NotNil(t, config.Interfaces, "Interfaces array should not be nil")

	// Log found interfaces
	t.Logf("Found %d network interfaces", len(config.Interfaces))

	for _, iface := range config.Interfaces {
		assert.NotEmpty(t, iface.Name, "Interface name should not be empty")
		assert.NotEmpty(t, iface.MACAddress, "Interface MAC should not be empty")
		assert.Contains(t, []string{"up", "down"}, iface.LinkState, "Link state should be 'up' or 'down'")
		assert.NotNil(t, iface.IPAddresses, "Addresses array should not be nil")

		t.Logf("  Interface: %s, MAC: %s, State: %s, Addresses: %d",
			iface.Name, iface.MACAddress, iface.LinkState, len(iface.IPAddresses))

		for _, addr := range iface.IPAddresses {
			assert.NotEmpty(t, addr.IP, "IP address should not be empty")
			assert.Greater(t, addr.Prefix, 0, "Prefix should be greater than 0")
			assert.Contains(t, []string{"ipv4", "ipv6"}, addr.Family, "Family should be 'ipv4' or 'ipv6'")

			t.Logf("    Address: %s/%d (%s)", addr.IP, addr.Prefix, addr.Family)
		}
	}
}

func TestNetworkEndpoint_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewNetworkHandler()

	// Test POST method
	req := httptest.NewRequest(http.MethodPost, "/network", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
