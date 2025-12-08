package inventory_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/inventory"
	"github.com/stretchr/testify/assert"
)

func TestGetTPMInfo(t *testing.T) {
	// Basic smoke test - should not panic
	info, err := inventory.GetTPMInfo()

	assert.NoError(t, err, "GetTPMInfo should not return an error")

	// TPM may or may not be present on test system
	if info.Present {
		// If TPM is present, manufacturer, model, and version may be populated
		t.Logf("TPM detected: Manufacturer=%v, Model=%v, Version=%v",
			ptrToString(info.Manufacturer),
			ptrToString(info.Model),
			ptrToString(info.Version))
	} else {
		// If no TPM, manufacturer, model, and version should be nil
		assert.Nil(t, info.Manufacturer, "Manufacturer should be nil when TPM not present")
		assert.Nil(t, info.Model, "Model should be nil when TPM not present")
		assert.Nil(t, info.Version, "Version should be nil when TPM not present")
	}
}

func ptrToString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
