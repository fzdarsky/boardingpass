// Package network provides network interface enumeration and IP address extraction.
package network

import (
	"fmt"
	"net"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// GetInterfaces enumerates all network interfaces on the system.
// Returns a list of NetworkInterface objects with details about each interface.
func GetInterfaces() ([]protocol.NetworkInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate network interfaces: %w", err)
	}

	result := make([]protocol.NetworkInterface, 0, len(ifaces))

	for _, iface := range ifaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := GetAddresses(iface)
		if err != nil {
			return nil, err
		}

		netIface := protocol.NetworkInterface{
			Name:        iface.Name,
			MACAddress:  iface.HardwareAddr.String(),
			LinkState:   GetLinkState(iface),
			IPAddresses: addrs,
		}
		result = append(result, netIface)
	}

	return result, nil
}
