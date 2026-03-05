package network_test

import (
	"strings"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/network"
	"github.com/stretchr/testify/assert"
)

const samplePCIIDs = `#
#	List of PCI ID's
#

8086  Intel Corporation
	1533  I210 Gigabit Network Connection
	a0f0  Wi-Fi 6 AX201 160MHz
14e4  Broadcom Inc. and subsidiaries
	43a0  BCM4360 802.11ac Dual Band Wireless Network Adapter
		1028 0015  DW1560

C 00  Unclassified device
	00  Non-VGA unclassified device
`

func TestParsePCIIDs(t *testing.T) {
	db := network.ParsePCIIDs(strings.NewReader(samplePCIIDs))

	// Vendors
	assert.Equal(t, "Intel Corporation", db.Vendors["8086"])
	assert.Equal(t, "Broadcom Inc. and subsidiaries", db.Vendors["14e4"])
	assert.Len(t, db.Vendors, 2)

	// Devices
	assert.Equal(t, "I210 Gigabit Network Connection", db.Devices["8086:1533"])
	assert.Equal(t, "Wi-Fi 6 AX201 160MHz", db.Devices["8086:a0f0"])
	assert.Equal(t, "BCM4360 802.11ac Dual Band Wireless Network Adapter", db.Devices["14e4:43a0"])
	assert.Len(t, db.Devices, 3)

	// Subdevice lines should NOT be parsed
	assert.Empty(t, db.Devices["1028:0015"])
	assert.Empty(t, db.Devices["14e4:1028"])
}

func TestParsePCIIDs_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVendors int
		wantDevices int
	}{
		{"empty", "", 0, 0},
		{"only comments", "# comment\n# another\n", 0, 0},
		{"vendor only", "8086  Intel Corporation\n", 1, 0},
		{"stops at class section", "8086  Intel\n\t1533  I210\nC 00  Unclassified\n0000  Fake Vendor\n", 1, 1},
		{"short line ignored", "80\n", 0, 0},
		{"missing double space", "8086 Intel\n", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := network.ParsePCIIDs(strings.NewReader(tt.input))
			assert.Len(t, db.Vendors, tt.wantVendors)
			assert.Len(t, db.Devices, tt.wantDevices)
		})
	}
}

func TestNormalizePCIID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0x8086", "8086"},
		{"0X8086", "8086"},
		{"8086", "8086"},
		{"0xA0F0", "a0f0"},
		{"", ""},
		{"  0x8086  ", "8086"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, network.NormalizePCIID(tt.input))
		})
	}
}

func TestLookupVendor(t *testing.T) {
	network.SetPCIDBForTesting(strings.NewReader(samplePCIIDs))
	t.Cleanup(network.ClearPCIDBForTesting)

	tests := []struct {
		name     string
		vendorID string
		want     string
	}{
		{"with 0x prefix", "0x8086", "Intel Corporation"},
		{"with 0X prefix", "0X8086", "Intel Corporation"},
		{"without prefix", "8086", "Intel Corporation"},
		{"uppercase hex", "0x14E4", "Broadcom Inc. and subsidiaries"},
		{"unknown vendor", "0xffff", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, network.LookupVendor(tt.vendorID))
		})
	}
}

func TestLookupDevice(t *testing.T) {
	network.SetPCIDBForTesting(strings.NewReader(samplePCIIDs))
	t.Cleanup(network.ClearPCIDBForTesting)

	tests := []struct {
		name     string
		vendorID string
		deviceID string
		want     string
	}{
		{"known device", "0x8086", "0x1533", "I210 Gigabit Network Connection"},
		{"another device", "0x8086", "0xa0f0", "Wi-Fi 6 AX201 160MHz"},
		{"broadcom device", "0x14e4", "0x43a0", "BCM4360 802.11ac Dual Band Wireless Network Adapter"},
		{"unknown device", "0x8086", "0xffff", ""},
		{"unknown vendor", "0xffff", "0x1533", ""},
		{"empty vendor", "", "0x1533", ""},
		{"empty device", "0x8086", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, network.LookupDevice(tt.vendorID, tt.deviceID))
		})
	}
}

func TestLookup_NoDB(t *testing.T) {
	network.ClearPCIDBForTesting()

	assert.Empty(t, network.LookupVendor("0x8086"))
	assert.Empty(t, network.LookupDevice("0x8086", "0x1533"))
}
