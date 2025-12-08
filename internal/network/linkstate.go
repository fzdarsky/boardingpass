package network

import (
	"net"
)

// GetLinkState determines the link state (up/down) of a network interface.
// Returns "up" if the interface is up and running, "down" otherwise.
func GetLinkState(iface net.Interface) string {
	// Check if interface is UP
	if iface.Flags&net.FlagUp != 0 {
		return "up"
	}
	return "down"
}
