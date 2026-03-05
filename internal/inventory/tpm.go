// Package inventory provides system information extraction functionality.
// It includes TPM detection, board information, CPU architecture, OS details, and FIPS mode status.
package inventory

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const tpmSysPath = "/sys/class/tpm"

// tpmManufacturers maps 4-character ASCII TPM vendor IDs to human-readable names.
// Based on TCG TPM Vendor ID Registry.
// Reference: https://trustedcomputinggroup.org/wp-content/uploads/TCG_TPM_VendorIDRegistry_v1p06_r0p91_11july2021.pdf
var tpmManufacturers = map[string]string{
	"AMD\x00": "AMD",
	"ATML":    "Atmel",
	"BRCM":    "Broadcom",
	"HPE\x00": "HPE",
	"IBM\x00": "IBM",
	"IFX\x00": "Infineon",
	"INTC":    "Intel",
	"LEN\x00": "Lenovo",
	"MSFT":    "Microsoft",
	"NSM ":    "National Semiconductor",
	"NTZ\x00": "Nationz",
	"NTC\x00": "Nuvoton",
	"QCOM":    "Qualcomm",
	"SMSC":    "SMSC",
	"STM ":    "STMicroelectronics",
	"SMSN":    "Samsung",
	"SNS\x00": "Sinosun",
	"TXN\x00": "Texas Instruments",
}

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

	// Fallback: use tpm_version_major (available since Linux 5.6)
	if info.Version == nil {
		if versionMajor, err := readTPMFile(filepath.Join(tpmSysPath, tpmDevice, "tpm_version_major")); err == nil {
			switch strings.TrimSpace(versionMajor) {
			case "2":
				v := "2.0"
				info.Version = &v
			case "1":
				v := "1.2"
				info.Version = &v
			}
		}
	}

	// Read manufacturer from caps file (contains "Manufacturer: 0xHHHHHHHH")
	if caps, err := readTPMFile(filepath.Join(devicePath, "caps")); err == nil {
		if mfr := ParseManufacturerFromCaps(caps); mfr != "" {
			info.Manufacturer = &mfr
		}
	}

	// Fallback: try direct manufacturer file
	if info.Manufacturer == nil {
		if mfr, err := readTPMFile(filepath.Join(devicePath, "manufacturer")); err == nil {
			mfr = strings.TrimSpace(mfr)
			if mfr != "" {
				// Try to translate if it looks like a hex ID
				if translated := TranslateManufacturerID(mfr); translated != "" {
					info.Manufacturer = &translated
				} else {
					info.Manufacturer = &mfr
				}
			}
		}
	}

	// Fallback: extract manufacturer from modalias ACPI device ID (e.g., MSFT0101 → Microsoft)
	if info.Manufacturer == nil {
		if modalias, err := readTPMFile(filepath.Join(devicePath, "modalias")); err == nil {
			if mfr := ParseManufacturerFromModalias(modalias); mfr != "" {
				info.Manufacturer = &mfr
			}
		}
	}

	// Read model - try description first, then extract from modalias
	if desc, err := readTPMFile(filepath.Join(devicePath, "description")); err == nil {
		desc = strings.TrimSpace(desc)
		if desc != "" && desc != "TPM" {
			info.Model = &desc
		}
	}

	// Fallback: extract meaningful model from modalias
	if info.Model == nil {
		if modalias, err := readTPMFile(filepath.Join(devicePath, "modalias")); err == nil {
			if model := ParseModelFromModalias(modalias); model != "" {
				info.Model = &model
			}
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

// ParseManufacturerFromCaps extracts and translates the manufacturer ID from the caps file.
// The caps file contains lines like "Manufacturer: 0x53544d20" (hex-encoded ASCII).
func ParseManufacturerFromCaps(caps string) string {
	re := regexp.MustCompile(`Manufacturer:\s*(0x[0-9a-fA-F]+)`)
	matches := re.FindStringSubmatch(caps)
	if len(matches) < 2 {
		return ""
	}

	hexStr := strings.TrimPrefix(matches[1], "0x")
	return TranslateManufacturerID(hexStr)
}

// TranslateManufacturerID converts a hex manufacturer ID to a human-readable name.
// The hex string represents 4 ASCII characters (e.g., "53544d20" = "STM ").
func TranslateManufacturerID(hexStr string) string {
	// Ensure we have exactly 8 hex characters (4 bytes)
	hexStr = strings.TrimSpace(hexStr)
	if len(hexStr) != 8 {
		return ""
	}

	// Decode hex to bytes
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return ""
	}

	// Convert to ASCII string
	ascii := string(bytes)

	// Look up in manufacturer table
	if name, ok := tpmManufacturers[ascii]; ok {
		return name
	}

	// If not found, return the ASCII representation with trailing nulls/spaces removed
	return strings.TrimRight(ascii, "\x00 ")
}

// ParseManufacturerFromModalias extracts the manufacturer from the ACPI device ID in modalias.
// For example, "acpi:MSFT0101:" → "Microsoft", "acpi:INTC0102:" → "Intel".
func ParseManufacturerFromModalias(modalias string) string {
	modalias = strings.TrimSpace(modalias)

	// Extract from acpi:XXXX: format
	if rest, found := strings.CutPrefix(modalias, "acpi:"); found {
		deviceID := strings.TrimSuffix(rest, ":")
		if len(deviceID) >= 4 {
			// First 4 characters are the vendor ID (e.g., MSFT, INTC, AMD_)
			vendorID := deviceID[:4]
			switch vendorID {
			case "MSFT":
				return "Microsoft"
			case "INTC":
				return "Intel"
			case "AMD_":
				return "AMD"
			}
		}
	}

	return ""
}

// ParseModelFromModalias extracts a more readable model identifier from the modalias string.
// Modalias format is typically "acpi:MSFT0101:" or "platform:tpm_crb".
func ParseModelFromModalias(modalias string) string {
	modalias = strings.TrimSpace(modalias)
	if modalias == "" {
		return ""
	}

	// Extract the device identifier from acpi:XXXX: format
	if rest, found := strings.CutPrefix(modalias, "acpi:"); found {
		// rest is "MSFT0101:" or similar
		deviceID := strings.TrimSuffix(rest, ":")
		if deviceID == "" {
			return ""
		}
		// MSFT0101 is Microsoft firmware TPM
		switch {
		case strings.HasPrefix(deviceID, "MSFT"):
			return "Firmware TPM (fTPM)"
		case strings.HasPrefix(deviceID, "INTC"):
			return "Intel Platform Trust Technology (PTT)"
		case strings.HasPrefix(deviceID, "AMD"):
			return "AMD Platform Security Processor (fTPM)"
		default:
			return deviceID
		}
	}

	// Extract from platform:XXX format
	if device, found := strings.CutPrefix(modalias, "platform:"); found {
		device = strings.TrimSuffix(device, ":")
		if device == "tpm_crb" {
			return "Command Response Buffer (CRB)"
		}
		return device
	}

	return ""
}
