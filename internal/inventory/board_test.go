package inventory_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/inventory"
	"github.com/stretchr/testify/assert"
)

func TestGetProductInfo(t *testing.T) {
	// Basic smoke test - should not panic
	info, err := inventory.GetProductInfo()

	assert.NoError(t, err, "GetProductInfo should not return an error")

	// Product info should always have values (at minimum "Unknown")
	assert.NotEmpty(t, info.Vendor, "Vendor should not be empty")
	assert.NotEmpty(t, info.Name, "Name should not be empty")
	assert.NotEmpty(t, info.Serial, "Serial should not be empty")

	t.Logf("Product Info: Vendor=%s, Family=%s, Name=%s, Version=%s, Serial=%s",
		info.Vendor, info.Family, info.Name, info.Version, info.Serial)
}

func TestGetFirmwareInfo(t *testing.T) {
	info := inventory.GetFirmwareInfo()

	// On macOS (dev machines), neither DMI nor device tree exist, so all fields are "Unknown".
	// On Linux with DMI tables, BIOS fields should be populated.
	// On ARM with U-Boot, the version may be populated.
	assert.NotEmpty(t, info.Vendor, "Vendor should not be empty")
	assert.NotEmpty(t, info.Version, "Version should not be empty")
	assert.NotEmpty(t, info.Date, "Date should not be empty")

	t.Logf("Firmware Info: Vendor=%s, Version=%s, Date=%s",
		info.Vendor, info.Version, info.Date)
}
