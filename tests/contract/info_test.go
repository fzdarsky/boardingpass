package contract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
)

// TestInfoContractValidation validates the /info endpoint response against OpenAPI schema.
func TestInfoContractValidation(t *testing.T) {
	// Create handler
	handler := handlers.NewInfoHandler()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Check HTTP status
	assert.Equal(t, http.StatusOK, w.Code, "Expected HTTP 200 OK")

	// Parse response
	var info protocol.SystemInfo
	err := json.NewDecoder(w.Body).Decode(&info)
	assert.NoError(t, err, "Response must be valid JSON")

	// Validate response structure matches OpenAPI schema

	// TPM field validation
	assert.NotNil(t, info.TPM, "tpm field is required")
	// TPM.Present is required
	if info.TPM.Present {
		// When present=true, manufacturer, model, version MAY be populated
		t.Logf("TPM present with details: Manufacturer=%v, Model=%v, Version=%v",
			ptrToString(info.TPM.Manufacturer),
			ptrToString(info.TPM.Model),
			ptrToString(info.TPM.Version))
	} else {
		// When present=false, manufacturer, model, version should be nil
		assert.Nil(t, info.TPM.Manufacturer, "Manufacturer should be nil when TPM not present")
		assert.Nil(t, info.TPM.Model, "Model should be nil when TPM not present")
		assert.Nil(t, info.TPM.Version, "Version should be nil when TPM not present")
	}

	// Board field validation
	assert.NotNil(t, info.Board, "board field is required")
	assert.NotEmpty(t, info.Board.Manufacturer, "board.manufacturer is required")
	assert.NotEmpty(t, info.Board.Model, "board.model is required")
	assert.NotEmpty(t, info.Board.Serial, "board.serial is required")

	// CPU field validation
	assert.NotNil(t, info.CPU, "cpu field is required")
	assert.NotEmpty(t, info.CPU.Architecture, "cpu.architecture is required")
	validArchitectures := []string{"x86_64", "aarch64", "armv7l", "386", "ppc64le", "s390x"}
	assert.Contains(t, validArchitectures, info.CPU.Architecture, "cpu.architecture must be a valid value")

	// OS field validation
	assert.NotNil(t, info.OS, "os field is required")
	assert.NotEmpty(t, info.OS.Distribution, "os.distribution is required")
	assert.NotEmpty(t, info.OS.Version, "os.version is required")
	// FIPSEnabled is a boolean, always has a value

	t.Logf("✓ /info endpoint response conforms to OpenAPI contract")
}

func ptrToString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
