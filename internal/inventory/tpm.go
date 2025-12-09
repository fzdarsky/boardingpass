// Package inventory provides system information extraction functionality.
// It includes TPM detection, board information, CPU architecture, OS details, and FIPS mode status.
package inventory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const tpmSysPath = "/sys/class/tpm"

// GetTPMInfo extracts TPM information from /sys/class/tpm.
// Returns TPMInfo with Present=false if no TPM is detected.
func GetTPMInfo() (protocol.TPMInfo, error) {
	info := protocol.TPMInfo{
		Present: false,
	}

	// Check if /sys/class/tpm exists
	entries, err := os.ReadDir(tpmSysPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No TPM present
			return info, nil
		}
		return info, fmt.Errorf("failed to read TPM directory: %w", err)
	}

	// Look for TPM device entries (typically tpm0, tpm1, etc.)
	var tpmDevice string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "tpm") {
			tpmDevice = entry.Name()
			break
		}
	}

	if tpmDevice == "" {
		// No TPM device found
		return info, nil
	}

	info.Present = true
	devicePath := filepath.Join(tpmSysPath, tpmDevice, "device")

	// Read TPM version
	if version, err := readTPMFile(filepath.Join(devicePath, "caps")); err == nil {
		// Extract version from caps file
		// Caps file typically contains "TCG version: 1.2" or "TPM 2.0"
		if strings.Contains(version, "2.0") {
			v := "2.0"
			info.Version = &v
		} else if strings.Contains(version, "1.2") {
			v := "1.2"
			info.Version = &v
		}
	}

	// Try alternative version detection via description
	if info.Version == nil {
		if desc, err := readTPMFile(filepath.Join(devicePath, "description")); err == nil {
			if strings.Contains(desc, "2.0") {
				v := "2.0"
				info.Version = &v
			} else if strings.Contains(desc, "1.2") {
				v := "1.2"
				info.Version = &v
			}
		}
	}

	// Read manufacturer - try multiple sources
	if mfr, err := readTPMFile(filepath.Join(devicePath, "manufacturer")); err == nil {
		mfr = strings.TrimSpace(mfr)
		if mfr != "" {
			info.Manufacturer = &mfr
		}
	}

	// Try alternative manufacturer location
	if info.Manufacturer == nil {
		if mfr, err := readTPMFile(filepath.Join(tpmSysPath, tpmDevice, "tpm_version_major")); err == nil {
			// This might contain manufacturer ID
			mfr = strings.TrimSpace(mfr)
			if mfr != "" {
				info.Manufacturer = &mfr
			}
		}
	}

	// Read model/device name
	if model, err := readTPMFile(filepath.Join(devicePath, "modalias")); err == nil {
		model = strings.TrimSpace(model)
		if model != "" {
			info.Model = &model
		}
	}

	return info, nil
}

// readTPMFile reads a single line from a sysfs file and returns it as a trimmed string.
func readTPMFile(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
