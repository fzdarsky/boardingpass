package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const sysClassNet = "/sys/class/net"

// ARPHRD_ETHER and other ARP hardware type constants from linux/if_arp.h.
const (
	arphrdEther     = 1   // Ethernet
	arphrdIEEE80211 = 801 // IEEE 802.11 (Wi-Fi)
)

// GetInterfaceType returns the interface type string based on sysfs data.
// Checks /sys/class/net/<name>/type (ARPHRD constant) and whether
// the interface has a wireless/ subdirectory.
func GetInterfaceType(name string) string {
	basePath := filepath.Join(sysClassNet, name)

	// Check for wireless interface (has wireless/ or phy80211/ subdir)
	if dirExists(filepath.Join(basePath, "wireless")) || dirExists(filepath.Join(basePath, "phy80211")) {
		return "wifi"
	}

	// Check for bridge
	if dirExists(filepath.Join(basePath, "bridge")) {
		return "bridge"
	}

	// Check for bond
	if dirExists(filepath.Join(basePath, "bonding")) {
		return "bond"
	}

	// Read ARPHRD type
	typeVal := readSysfsInt(filepath.Join(basePath, "type"))
	switch typeVal {
	case arphrdEther:
		// Check if it's a VLAN
		if fileExists(filepath.Join(basePath, "lower_device")) || isVLAN(name) {
			return "vlan"
		}
		// Check if it has a real device backing (PCI/USB) vs virtual
		if !dirExists(filepath.Join(basePath, "device")) {
			return "virtual"
		}
		return "ethernet"
	case arphrdIEEE80211:
		return "wifi"
	default:
		if dirExists(filepath.Join(basePath, "device")) {
			return "ethernet"
		}
		return "virtual"
	}
}

// GetInterfaceSpeed returns the link speed in Mbps from sysfs.
// Returns -1 if unavailable (virtual interfaces, unplugged cables).
func GetInterfaceSpeed(name string) int {
	speed := readSysfsInt(filepath.Join(sysClassNet, name, "speed"))
	if speed < 0 {
		return -1
	}
	return speed
}

// GetInterfaceCarrier returns whether a cable/link is detected.
func GetInterfaceCarrier(name string) bool {
	val := readSysfsString(filepath.Join(sysClassNet, name, "carrier"))
	return val == "1"
}

// GetInterfaceDriver returns the kernel driver name from the device/driver symlink.
func GetInterfaceDriver(name string) string {
	driverLink := filepath.Join(sysClassNet, name, "device", "driver")
	target, err := os.Readlink(driverLink)
	if err != nil {
		return ""
	}
	return filepath.Base(target)
}

// GetInterfaceVendor returns the PCI vendor ID (e.g. "0x8086") from sysfs.
func GetInterfaceVendor(name string) string {
	return readSysfsString(filepath.Join(sysClassNet, name, "device", "vendor"))
}

// GetInterfaceModel returns the PCI device ID (e.g. "0x1533") from sysfs.
func GetInterfaceModel(name string) string {
	return readSysfsString(filepath.Join(sysClassNet, name, "device", "device"))
}

// readSysfsString reads a single-line value from a sysfs file.
func readSysfsString(path string) string {
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is constructed from sysfs constants, not user input
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// readSysfsInt reads an integer value from a sysfs file.
// Returns -1 on any error.
func readSysfsInt(path string) int {
	s := readSysfsString(path)
	if s == "" {
		return -1
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return val
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// isVLAN checks if an interface name matches VLAN naming conventions.
func isVLAN(name string) bool {
	// Common patterns: eth0.100, vlan100
	if strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)
		if len(parts) == 2 {
			_, err := strconv.Atoi(parts[1])
			return err == nil
		}
	}
	if strings.HasPrefix(name, "vlan") {
		_, err := strconv.Atoi(name[4:])
		return err == nil
	}
	return false
}

// GetMetadata collects all sysfs metadata for a named interface.
func GetMetadata(name string) (ifaceType string, speed int, carrier bool, driver, vendor, model string) {
	ifaceType = GetInterfaceType(name)
	speed = GetInterfaceSpeed(name)
	carrier = GetInterfaceCarrier(name)
	driver = GetInterfaceDriver(name)
	vendor = GetInterfaceVendor(name)
	model = GetInterfaceModel(name)
	return
}

// FormatPCIID formats a PCI ID string, returning it as-is or empty on error.
func FormatPCIID(raw string) string {
	if raw == "" {
		return ""
	}
	// sysfs returns "0x8086\n" style values; already trimmed by readSysfsString
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		return raw
	}
	return fmt.Sprintf("0x%s", raw)
}
