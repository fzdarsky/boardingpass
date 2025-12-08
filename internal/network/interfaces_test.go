package network_test

import (
	"testing"

	"github.com/fzdarsky/boardingpass/internal/network"
	"github.com/stretchr/testify/assert"
)

func TestGetInterfaces(t *testing.T) {
	interfaces, err := network.GetInterfaces()

	assert.NoError(t, err, "GetInterfaces should not return an error")

	// Should have at least one non-loopback interface on any system
	t.Logf("Found %d network interfaces", len(interfaces))

	for _, iface := range interfaces {
		// Each interface should have required fields populated
		assert.NotEmpty(t, iface.Name, "Interface name should not be empty")
		assert.NotEmpty(t, iface.MACAddress, "Interface MAC should not be empty")
		assert.Contains(t, []string{"up", "down"}, iface.LinkState, "Link state should be 'up' or 'down'")

		t.Logf("Interface: %s, MAC: %s, State: %s, Addresses: %d",
			iface.Name, iface.MACAddress, iface.LinkState, len(iface.IPAddresses))

		// IPAddresses may be empty for interfaces without IP configuration
		for _, addr := range iface.IPAddresses {
			assert.NotEmpty(t, addr.IP, "IP address should not be empty")
			assert.Greater(t, addr.Prefix, 0, "Prefix should be greater than 0")
			assert.Contains(t, []string{"ipv4", "ipv6"}, addr.Family, "Family should be 'ipv4' or 'ipv6'")
		}
	}
}
