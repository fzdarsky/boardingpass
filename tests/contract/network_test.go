package contract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
)

// TestNetworkContractValidation validates the /network endpoint response against OpenAPI schema.
func TestNetworkContractValidation(t *testing.T) {
	// Create handler
	handler := handlers.NewNetworkHandler()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/network", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Check HTTP status
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP 200 OK")

	// Parse response
	var config protocol.NetworkConfig
	err := json.NewDecoder(w.Body).Decode(&config)
	assert.NoError(t, err, "Response must be valid JSON")

	// Validate response structure matches OpenAPI schema

	// Interfaces field validation
	assert.NotNil(t, config.Interfaces, "interfaces field is required")
	assert.LessOrEqual(t, len(config.Interfaces), 32, "interfaces array must not exceed 32 items")

	// MAC address regex pattern from OpenAPI schema
	macPattern := regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)

	for _, iface := range config.Interfaces {
		// Interface name validation
		assert.NotEmpty(t, iface.Name, "interface.name is required")
		assert.Regexp(t, `^[a-zA-Z0-9]+$`, iface.Name, "interface.name must match pattern ^[a-zA-Z0-9]+$")

		// MAC address validation
		assert.NotEmpty(t, iface.MACAddress, "interface.mac is required")
		assert.Regexp(t, macPattern, iface.MACAddress, "interface.mac must be valid colon-separated hex format")

		// Link state validation
		assert.Contains(t, []string{"up", "down"}, iface.LinkState, "interface.link_state must be 'up' or 'down'")

		// Addresses validation
		assert.NotNil(t, iface.IPAddresses, "interface.addresses is required")

		for _, addr := range iface.IPAddresses {
			// IP address validation
			assert.NotEmpty(t, addr.IP, "address.ip is required")

			// Prefix validation
			assert.GreaterOrEqual(t, addr.Prefix, 0, "address.prefix must be >= 0")
			assert.LessOrEqual(t, addr.Prefix, 128, "address.prefix must be <= 128")

			// Family validation
			assert.Contains(t, []string{"ipv4", "ipv6"}, addr.Family, "address.family must be 'ipv4' or 'ipv6'")

			// Prefix range validation based on family
			if addr.Family == "ipv4" {
				assert.LessOrEqual(t, addr.Prefix, 32, "IPv4 prefix must be <= 32")
			}
		}

		t.Logf("Interface %s validated: MAC=%s, State=%s, Addresses=%d",
			iface.Name, iface.MACAddress, iface.LinkState, len(iface.IPAddresses))
	}

	t.Logf("✓ /network endpoint response conforms to OpenAPI contract")
}
