package inventory_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/inventory"
)

func TestGetFIPSStatus(t *testing.T) {
	// Basic smoke test - should not panic
	status := inventory.GetFIPSStatus()

	// FIPS status is a boolean - either true or false
	// On most test systems, it will be false
	t.Logf("FIPS Status: %v", status)
}
