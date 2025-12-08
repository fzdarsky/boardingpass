package inventory_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/inventory"
	"github.com/stretchr/testify/assert"
)

func TestGetCPUInfo(t *testing.T) {
	info := inventory.GetCPUInfo()

	// Architecture should never be empty
	assert.NotEmpty(t, info.Architecture, "Architecture should not be empty")

	// Should be one of the expected values
	validArchitectures := []string{"x86_64", "aarch64", "armv7l", "386", "ppc64le", "s390x"}
	assert.Contains(t, validArchitectures, info.Architecture, "Architecture should be a known value")

	t.Logf("CPU Architecture: %s", info.Architecture)
}
