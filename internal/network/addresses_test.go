package network_test

import (
	"net"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/network"
	"github.com/stretchr/testify/assert"
)

func TestGetAddresses(t *testing.T) {
	// Get all interfaces to test with real interface
	ifaces, err := net.Interfaces()
	assert.NoError(t, err, "Failed to get network interfaces")

	// Find first non-loopback interface
	var testIface net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 {
			testIface = iface
			break
		}
	}

	if testIface.Name == "" {
		t.Skip("No non-loopback interface found for testing")
	}

	addresses, err := network.GetAddresses(testIface)
	assert.NoError(t, err, "GetAddresses should not return an error")

	t.Logf("Interface %s has %d addresses", testIface.Name, len(addresses))

	for _, addr := range addresses {
		assert.NotEmpty(t, addr.IP, "IP address should not be empty")
		assert.Greater(t, addr.Prefix, 0, "Prefix should be greater than 0")
		assert.Contains(t, []string{"ipv4", "ipv6"}, addr.Family, "Family should be 'ipv4' or 'ipv6'")

		t.Logf("  Address: %s/%d (%s)", addr.IP, addr.Prefix, addr.Family)
	}
}
