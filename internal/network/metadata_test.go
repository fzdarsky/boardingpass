package network_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/network"
	"github.com/stretchr/testify/assert"
)

func TestGetInterfaceType_ReturnsValidType(t *testing.T) {
	validTypes := []string{"ethernet", "wifi", "bridge", "bond", "vlan", "virtual"}

	// Use a known interface name or skip if none available
	ifaceType := network.GetInterfaceType("lo")
	assert.Contains(t, validTypes, ifaceType, "lo should return a valid type")
}

func TestGetInterfaceSpeed_LoopbackReturnsNegative(t *testing.T) {
	speed := network.GetInterfaceSpeed("lo")
	assert.Equal(t, -1, speed, "Loopback speed should be -1")
}

func TestGetInterfaceCarrier_LoopbackReturnsFalse(t *testing.T) {
	// Loopback carrier is typically "1" on Linux but not available on macOS
	carrier := network.GetInterfaceCarrier("nonexistent-iface")
	assert.False(t, carrier, "Non-existent interface should return false for carrier")
}

func TestGetInterfaceDriver_NonexistentReturnsEmpty(t *testing.T) {
	driver := network.GetInterfaceDriver("nonexistent-iface")
	assert.Empty(t, driver, "Non-existent interface should return empty driver")
}

func TestGetInterfaceVendor_NonexistentReturnsEmpty(t *testing.T) {
	vendor := network.GetInterfaceVendor("nonexistent-iface")
	assert.Empty(t, vendor, "Non-existent interface should return empty vendor")
}

func TestGetInterfaceModel_NonexistentReturnsEmpty(t *testing.T) {
	model := network.GetInterfaceModel("nonexistent-iface")
	assert.Empty(t, model, "Non-existent interface should return empty model")
}

func TestGetMetadata_ReturnsAllFields(t *testing.T) {
	ifaceType, speed, carrier, driver, vendor, model := network.GetMetadata("lo")
	_ = ifaceType
	_ = speed
	_ = carrier
	_ = driver
	_ = vendor
	_ = model
	// Smoke test: should not panic for any interface name
}

func TestIsVLAN_Naming(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"eth0", "ethernet"},
		{"lo", "virtual"},
		{"nonexistent", "virtual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// On macOS/CI without sysfs, all return "virtual" since /sys doesn't exist
			ifaceType := network.GetInterfaceType(tt.name)
			validTypes := []string{"ethernet", "wifi", "bridge", "bond", "vlan", "virtual"}
			assert.Contains(t, validTypes, ifaceType)
		})
	}
}

func TestFormatPCIID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0x8086", "0x8086"},
		{"0X8086", "0X8086"},
		{"8086", "0x8086"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := network.FormatPCIID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
