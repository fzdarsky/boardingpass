package inventory_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/inventory"
	"github.com/stretchr/testify/assert"
)

func TestGetBoardInfo(t *testing.T) {
	// Basic smoke test - should not panic
	info, err := inventory.GetBoardInfo()

	assert.NoError(t, err, "GetBoardInfo should not return an error")

	// Board info should always have values (at minimum "Unknown")
	assert.NotEmpty(t, info.Manufacturer, "Manufacturer should not be empty")
	assert.NotEmpty(t, info.Model, "Model should not be empty")
	assert.NotEmpty(t, info.Serial, "Serial should not be empty")

	t.Logf("Board Info: Manufacturer=%s, Model=%s, Serial=%s",
		info.Manufacturer, info.Model, info.Serial)
}
