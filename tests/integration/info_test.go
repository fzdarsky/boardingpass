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

func TestInfoEndpoint(t *testing.T) {
	// Create handler
	handler := handlers.NewInfoHandler()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Check response status
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP 200 OK")

	// Check content type
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response body
	var info protocol.SystemInfo
	err := json.NewDecoder(w.Body).Decode(&info)
	assert.NoError(t, err, "Response should be valid JSON")

	// Verify required fields
	assert.NotEmpty(t, info.Firmware.Vendor, "Firmware vendor should not be empty")
	assert.NotEmpty(t, info.Product.Vendor, "Product vendor should not be empty")
	assert.NotEmpty(t, info.Product.Name, "Product name should not be empty")
	assert.NotEmpty(t, info.Product.Serial, "Product serial should not be empty")
	assert.NotEmpty(t, info.CPU.Architecture, "CPU architecture should not be empty")
	assert.NotEmpty(t, info.OS.Distribution, "OS distribution should not be empty")
	assert.NotEmpty(t, info.OS.Version, "OS version should not be empty")

	t.Logf("SystemInfo response: TPM Present=%v, Product=%s %s, CPU=%s, OS=%s %s, FIPS=%v",
		info.TPM.Present,
		info.Product.Vendor, info.Product.Name,
		info.CPU.Architecture,
		info.OS.Distribution, info.OS.Version,
		info.OS.FIPSEnabled)
}

func TestInfoEndpoint_Caching(t *testing.T) {
	// Create handler
	handler := handlers.NewInfoHandler()

	// Make first request
	req1 := httptest.NewRequest(http.MethodGet, "/info", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	var info1 protocol.SystemInfo
	err := json.NewDecoder(w1.Body).Decode(&info1)
	assert.NoError(t, err)

	// Make second request (should be cached)
	req2 := httptest.NewRequest(http.MethodGet, "/info", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var info2 protocol.SystemInfo
	err = json.NewDecoder(w2.Body).Decode(&info2)
	assert.NoError(t, err)

	// Results should be identical (from cache)
	assert.Equal(t, info1, info2, "Cached response should match first response")
}

func TestInfoEndpoint_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewInfoHandler()

	// Test POST method
	req := httptest.NewRequest(http.MethodPost, "/info", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
