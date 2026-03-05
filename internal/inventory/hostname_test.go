package inventory_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHostname(t *testing.T) {
	hostname, err := inventory.GetHostname()
	require.NoError(t, err, "GetHostname should succeed")
	assert.NotEmpty(t, hostname, "Hostname should not be empty")

	t.Logf("Hostname: %s", hostname)
}
