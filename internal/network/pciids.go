package network

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"
)

// defaultPCIIDsPath is the standard location on RHEL/Fedora/CentOS.
const defaultPCIIDsPath = "/usr/share/hwdata/pci.ids"

// PCIDB holds parsed PCI vendor and device name mappings.
type PCIDB struct {
	// Vendors maps vendor ID (e.g., "8086") to vendor name (e.g., "Intel Corporation").
	Vendors map[string]string
	// Devices maps "vendorID:deviceID" (e.g., "8086:1533") to device name.
	Devices map[string]string
}

var (
	pciOnce sync.Once
	pciData *PCIDB
)

// ParsePCIIDs parses pci.ids formatted content from a reader.
// Vendor lines start at column 0 (e.g., "8086  Intel Corporation").
// Device lines start with a single tab (e.g., "\t1533  I210 Gigabit").
// Subdevice lines (double tab) and class sections ("C ") are skipped.
func ParsePCIIDs(r io.Reader) *PCIDB {
	db := &PCIDB{
		Vendors: make(map[string]string),
		Devices: make(map[string]string),
	}

	scanner := bufio.NewScanner(r)
	var currentVendor string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || line[0] == '#' {
			continue
		}

		// Stop at device class section
		if strings.HasPrefix(line, "C ") {
			break
		}

		// Double-tab: subdevice — skip
		if strings.HasPrefix(line, "\t\t") {
			continue
		}

		// Single-tab: device line
		if line[0] == '\t' {
			if currentVendor == "" {
				continue
			}
			rest := line[1:] // strip tab
			if id, name, ok := parsePCILine(rest); ok {
				db.Devices[currentVendor+":"+id] = name
			}
			continue
		}

		// No leading whitespace: vendor line
		if id, name, ok := parsePCILine(line); ok {
			db.Vendors[id] = name
			currentVendor = id
		}
	}

	return db
}

// parsePCILine extracts "XXXX  Name" from a line.
// Returns the 4-char hex ID, the name, and true on success.
func parsePCILine(line string) (id, name string, ok bool) {
	if len(line) < 6 {
		return "", "", false
	}
	// Format: "XXXX  Name..." (4 hex chars, two spaces, then name)
	id = line[:4]
	if line[4] != ' ' || line[5] != ' ' {
		return "", "", false
	}
	name = strings.TrimSpace(line[6:])
	if name == "" {
		return "", "", false
	}
	return id, name, true
}

// NormalizePCIID strips the "0x"/"0X" prefix and lowercases a PCI ID
// to match the pci.ids file format.
func NormalizePCIID(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "0x")
	raw = strings.TrimPrefix(raw, "0X")
	return strings.ToLower(raw)
}

// loadPCIIDs reads and parses the system pci.ids file.
func loadPCIIDs() {
	f, err := os.Open(defaultPCIIDsPath)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	pciData = ParsePCIIDs(f)
}

// LookupVendor returns the human-readable vendor name for a PCI vendor ID.
// The vendorID should be the raw sysfs value (e.g., "0x8086").
// Returns empty string if not found or if pci.ids is unavailable.
func LookupVendor(vendorID string) string {
	pciOnce.Do(loadPCIIDs)
	if pciData == nil {
		return ""
	}
	return pciData.Vendors[NormalizePCIID(vendorID)]
}

// LookupDevice returns the human-readable device name for a PCI vendor+device pair.
// Both IDs should be the raw sysfs values (e.g., "0x8086", "0x1533").
// Returns empty string if not found or if pci.ids is unavailable.
func LookupDevice(vendorID, deviceID string) string {
	pciOnce.Do(loadPCIIDs)
	if pciData == nil {
		return ""
	}
	vID := NormalizePCIID(vendorID)
	dID := NormalizePCIID(deviceID)
	return pciData.Devices[vID+":"+dID]
}

// SetPCIDBForTesting injects a parsed PCI database for testing.
// Consumes the sync.Once so loadPCIIDs is never called.
func SetPCIDBForTesting(r io.Reader) {
	pciOnce = sync.Once{}
	pciData = ParsePCIIDs(r)
	pciOnce.Do(func() {}) // consume Once to prevent loadPCIIDs
}

// ClearPCIDBForTesting resets the PCI database to nil.
// Consumes the sync.Once so loadPCIIDs is never called.
func ClearPCIDBForTesting() {
	pciOnce = sync.Once{}
	pciData = nil
	pciOnce.Do(func() {}) // consume Once to prevent loadPCIIDs
}
