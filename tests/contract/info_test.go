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

	// Hostname field validation (new in enrollment flow)
	assert.NotEmpty(t, info.Hostname, "hostname field is required and must not be empty")

	// TPM field validation
	assert.NotNil(t, info.TPM, "tpm field is required")
	// TPM.Present is required
	if info.TPM.Present {
		// When present=true, type, spec_version, manufacturer, model MAY be populated
		t.Logf("TPM present: Type=%v, SpecVersion=%v, Manufacturer=%v, Model=%v",
			ptrToString(info.TPM.Type),
			ptrToString(info.TPM.SpecVersion),
			ptrToString(info.TPM.Manufacturer),
			ptrToString(info.TPM.Model))
		if info.TPM.Type != nil {
			validTypes := []string{"discrete", "firmware", "virtual"}
			assert.Contains(t, validTypes, *info.TPM.Type, "tpm.type must be a valid value")
		}
		if info.TPM.SpecVersion != nil {
			validVersions := []string{"1.2", "2.0"}
			assert.Contains(t, validVersions, *info.TPM.SpecVersion, "tpm.spec_version must be a valid value")
		}
	} else {
		// When present=false, all optional fields should be nil
		assert.Nil(t, info.TPM.Type, "Type should be nil when TPM not present")
		assert.Nil(t, info.TPM.SpecVersion, "SpecVersion should be nil when TPM not present")
		assert.Nil(t, info.TPM.Manufacturer, "Manufacturer should be nil when TPM not present")
		assert.Nil(t, info.TPM.Model, "Model should be nil when TPM not present")
	}

	// Firmware field validation
	assert.NotEmpty(t, info.Firmware.Vendor, "firmware.vendor is required")
	assert.NotEmpty(t, info.Firmware.Version, "firmware.version is required")
	assert.NotEmpty(t, info.Firmware.Date, "firmware.date is required")

	// Product field validation
	assert.NotEmpty(t, info.Product.Vendor, "product.vendor is required")
	assert.NotEmpty(t, info.Product.Name, "product.name is required")
	assert.NotEmpty(t, info.Product.Serial, "product.serial is required")

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
