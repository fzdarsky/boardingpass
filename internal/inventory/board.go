package inventory

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const dmiBasePath = "/sys/class/dmi/id"

// GetBoardInfo extracts board information from DMI tables via /sys/class/dmi/id.
// Falls back to reading device tree on ARM devices if DMI is not available.
func GetBoardInfo() (protocol.BoardInfo, error) {
	// Try DMI first (x86_64 systems)
	if _, err := os.Stat(dmiBasePath); err == nil {
		return readDMIBoardInfo()
	}

	// Fallback to device tree for ARM/embedded systems
	return readDeviceTreeBoardInfo()
}

// readDMIBoardInfo reads board information from DMI tables.
func readDMIBoardInfo() (protocol.BoardInfo, error) {
	info := protocol.BoardInfo{
		Manufacturer: "Unknown",
		Model:        "Unknown",
		Serial:       "Unknown",
	}

	// Read manufacturer
	if mfr, err := readDMIFile(filepath.Join(dmiBasePath, "board_vendor")); err == nil {
		mfr = strings.TrimSpace(mfr)
		if mfr != "" && mfr != "To be filled by O.E.M." {
			info.Manufacturer = mfr
		}
	}

	// Try alternative manufacturer sources
	if info.Manufacturer == "Unknown" {
		if mfr, err := readDMIFile(filepath.Join(dmiBasePath, "sys_vendor")); err == nil {
			mfr = strings.TrimSpace(mfr)
			if mfr != "" && mfr != "To be filled by O.E.M." {
				info.Manufacturer = mfr
			}
		}
	}

	// Read model
	if model, err := readDMIFile(filepath.Join(dmiBasePath, "board_name")); err == nil {
		model = strings.TrimSpace(model)
		if model != "" && model != "To be filled by O.E.M." {
			info.Model = model
		}
	}

	// Try alternative model sources
	if info.Model == "Unknown" {
		if model, err := readDMIFile(filepath.Join(dmiBasePath, "product_name")); err == nil {
			model = strings.TrimSpace(model)
			if model != "" && model != "To be filled by O.E.M." {
				info.Model = model
			}
		}
	}

	// Read serial number
	if serial, err := readDMIFile(filepath.Join(dmiBasePath, "board_serial")); err == nil {
		serial = strings.TrimSpace(serial)
		if serial != "" && serial != "To be filled by O.E.M." {
			info.Serial = serial
		}
	}

	// Try alternative serial sources
	if info.Serial == "Unknown" {
		if serial, err := readDMIFile(filepath.Join(dmiBasePath, "product_serial")); err == nil {
			serial = strings.TrimSpace(serial)
			if serial != "" && serial != "To be filled by O.E.M." {
				info.Serial = serial
			}
		}
	}

	return info, nil
}

// readDeviceTreeBoardInfo reads board information from device tree (ARM systems).
func readDeviceTreeBoardInfo() (protocol.BoardInfo, error) {
	info := protocol.BoardInfo{
		Manufacturer: "Unknown",
		Model:        "Unknown",
		Serial:       "Unknown",
	}

	deviceTreeBase := "/proc/device-tree"

	// Read model from device tree
	if model, err := readDMIFile(filepath.Join(deviceTreeBase, "model")); err == nil {
		model = strings.TrimSpace(strings.TrimRight(model, "\x00"))
		if model != "" {
			info.Model = model
			// Try to extract manufacturer from model string
			// e.g., "Raspberry Pi 4 Model B" -> "Raspberry Pi Foundation"
			if strings.Contains(strings.ToLower(model), "raspberry pi") {
				info.Manufacturer = "Raspberry Pi Foundation"
			}
		}
	}

	// Read serial from device tree
	if serial, err := readDMIFile(filepath.Join(deviceTreeBase, "serial-number")); err == nil {
		serial = strings.TrimSpace(strings.TrimRight(serial, "\x00"))
		if serial != "" {
			info.Serial = serial
		}
	}

	// Try alternative serial location
	if info.Serial == "Unknown" {
		if serial, err := readDMIFile("/proc/cpuinfo"); err == nil {
			// Extract Serial from cpuinfo (Raspberry Pi specific)
			for line := range strings.SplitSeq(serial, "\n") {
				if strings.HasPrefix(line, "Serial") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						serial := strings.TrimSpace(parts[1])
						if serial != "" && serial != "0000000000000000" {
							info.Serial = serial
							break
						}
					}
				}
			}
		}
	}

	return info, nil
}

// readDMIFile reads a single line from a DMI sysfs file.
func readDMIFile(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
