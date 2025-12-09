package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// GetAddresses extracts IP addresses assigned to a network interface.
// Returns a list of IPAddress objects containing IPv4 and IPv6 addresses.
func GetAddresses(iface net.Interface) ([]protocol.IPAddress, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for interface %s: %w", iface.Name, err)
	}

	result := []protocol.IPAddress{}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP

		// Determine address family
		family := "ipv4"
		if ip.To4() == nil {
			family = "ipv6"
		}

		// Calculate prefix length
		ones, _ := ipNet.Mask.Size()

		// Format IP address
		ipStr := ip.String()

		// For IPv6, remove zone identifier if present (e.g., fe80::1%eth0 -> fe80::1)
		if family == "ipv6" {
			if idx := strings.Index(ipStr, "%"); idx != -1 {
				ipStr = ipStr[:idx]
			}
		}

		result = append(result, protocol.IPAddress{
			IP:     ipStr,
			Prefix: ones,
			Family: family,
		})
	}

	return result, nil
}
