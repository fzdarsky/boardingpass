package inventory_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/inventory"
	"github.com/stretchr/testify/assert"
)

func TestGetOSInfo(t *testing.T) {
	info, err := inventory.GetOSInfo()

	assert.NoError(t, err, "GetOSInfo should not return an error")

	// Distribution and version should not be empty
	assert.NotEmpty(t, info.Distribution, "Distribution should not be empty")
	assert.NotEmpty(t, info.Version, "Version should not be empty")

	// FIPS enabled should be a boolean (always has a value)
	t.Logf("OS Info: Distribution=%s, Version=%s, FIPSEnabled=%v",
		info.Distribution, info.Version, info.FIPSEnabled)
}
