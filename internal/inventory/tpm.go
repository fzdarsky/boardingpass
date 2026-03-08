// Package inventory provides system information extraction functionality.
// It includes TPM detection, board information, CPU architecture, OS details, and FIPS mode status.
package inventory

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

// firmwareTPMManufacturers are manufacturer IDs associated with firmware TPMs (fTPM/PTT).
var firmwareTPMManufacturers = map[string]bool{
	"Microsoft": true,
	"Intel":     true,
	"AMD":       true,
}

// discreteTPMManufacturers are manufacturer IDs associated with discrete TPM chips.
var discreteTPMManufacturers = map[string]bool{
	"Infineon":           true,
	"STMicroelectronics": true,
	"Nuvoton":            true,
	"Atmel":              true,
	"Broadcom":           true,
}

// vmVendorIndicators are strings found in DMI sys_vendor or product_name that indicate a VM.
var vmVendorIndicators = []string{
	"QEMU", "VMware", "VirtualBox", "Microsoft Virtual",
	"KVM", "Xen", "Parallels", "Hyper-V", "innotek",
}

// TPM2GetCapResult holds parsed output from tpm2_getcap properties-fixed.
type TPM2GetCapResult struct {
	Manufacturer string // Translated manufacturer name
	Model        string // Concatenated vendor strings
	SpecVersion  string // Family indicator (e.g., "2.0")
}

// GetTPMInfo extracts TPM information from sysfs and tpm2-tools.
// Returns TPMInfo with Present=false if no TPM is detected.
func GetTPMInfo() (protocol.TPMInfo, error) {
	info := protocol.TPMInfo{
		Present: false,
	}

	// Check if /sys/class/tpm exists
	entries, err := os.ReadDir(tpmSysPath)
	if err != nil {
		if os.IsNotExist(err) {
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
		return info, nil
	}

	info.Present = true
	devicePath := filepath.Join(tpmSysPath, tpmDevice, "device")

	// --- SpecVersion detection (multi-fallback) ---

	// Priority 1: tpm_version_major (available since Linux 5.6)
	if versionMajor, err := readTPMFile(filepath.Join(tpmSysPath, tpmDevice, "tpm_version_major")); err == nil {
		switch strings.TrimSpace(versionMajor) {
		case "2":
			v := "2.0"
			info.SpecVersion = &v
		case "1":
			v := "1.2"
			info.SpecVersion = &v
		}
	}

	// Priority 2: caps file (TPM 1.2 style)
	if info.SpecVersion == nil {
		if caps, err := readTPMFile(filepath.Join(devicePath, "caps")); err == nil {
			if strings.Contains(caps, "2.0") {
				v := "2.0"
				info.SpecVersion = &v
			} else if strings.Contains(caps, "1.2") {
				v := "1.2"
				info.SpecVersion = &v
			}
		}
	}

	// Priority 3: description file
	if info.SpecVersion == nil {
		if desc, err := readTPMFile(filepath.Join(devicePath, "description")); err == nil {
			if strings.Contains(desc, "2.0") {
				v := "2.0"
				info.SpecVersion = &v
			} else if strings.Contains(desc, "1.2") {
				v := "1.2"
				info.SpecVersion = &v
			}
		}
	}

	// Read modalias (used for manufacturer, model, and type detection)
	var modalias string
	if m, err := readTPMFile(filepath.Join(devicePath, "modalias")); err == nil {
		modalias = m
	}

	// --- Manufacturer detection (multi-fallback) ---

	// Priority 1: caps file (contains "Manufacturer: 0xHHHHHHHH")
	if caps, err := readTPMFile(filepath.Join(devicePath, "caps")); err == nil {
		if mfr := ParseManufacturerFromCaps(caps); mfr != "" {
			info.Manufacturer = &mfr
		}
	}

	// Priority 2: direct manufacturer file
	if info.Manufacturer == nil {
		if mfr, err := readTPMFile(filepath.Join(devicePath, "manufacturer")); err == nil {
			mfr = strings.TrimSpace(mfr)
			if mfr != "" {
				if translated := TranslateManufacturerID(mfr); translated != "" {
					info.Manufacturer = &translated
				} else {
					info.Manufacturer = &mfr
				}
			}
		}
	}

	// --- Model detection (sysfs first) ---

	// Priority 1: description file (if non-empty and not generic "TPM")
	if desc, err := readTPMFile(filepath.Join(devicePath, "description")); err == nil {
		desc = strings.TrimSpace(desc)
		if desc != "" && desc != "TPM" {
			info.Model = &desc
		}
	}

	// --- tpm2_getcap fallback for manufacturer, model, and spec version ---

	getcapResult, getcapErr := runTPM2GetCap()
	if getcapErr == nil {
		parsed := ParseTPM2GetCap(getcapResult)

		// Use tpm2_getcap as fallback for manufacturer
		if info.Manufacturer == nil && parsed.Manufacturer != "" {
			info.Manufacturer = &parsed.Manufacturer
		}

		// Use tpm2_getcap as fallback for model (vendor strings)
		if info.Model == nil && parsed.Model != "" {
			info.Model = &parsed.Model
		}

		// Use tpm2_getcap as fallback for spec version
		if info.SpecVersion == nil && parsed.SpecVersion != "" {
			info.SpecVersion = &parsed.SpecVersion
		}

		// If tpm2_getcap succeeded, it's definitely TPM 2.0
		if info.SpecVersion == nil {
			v := "2.0"
			info.SpecVersion = &v
		}
	}

	// --- Remaining sysfs fallbacks ---

	// Manufacturer fallback: extract from modalias ACPI device ID
	if info.Manufacturer == nil && modalias != "" {
		if mfr := ParseManufacturerFromModalias(modalias); mfr != "" {
			info.Manufacturer = &mfr
		}
	}

	// Model fallback: extract from modalias
	if info.Model == nil && modalias != "" {
		if model := ParseModelFromModalias(modalias); model != "" {
			info.Model = &model
		}
	}

	// --- Type detection ---
	var manufacturer string
	if info.Manufacturer != nil {
		manufacturer = *info.Manufacturer
	}
	info.Type = detectTPMType(manufacturer, modalias)

	return info, nil
}

// runTPM2GetCap runs tpm2_getcap properties-fixed via sudo and returns stdout.
func runTPM2GetCap() (string, error) {
	// Find tpm2_getcap binary
	var tpm2Path string
	for _, p := range []string{"/usr/sbin/tpm2_getcap", "/usr/bin/tpm2_getcap"} {
		if _, err := os.Stat(p); err == nil {
			tpm2Path = p
			break
		}
	}
	if tpm2Path == "" {
		return "", fmt.Errorf("tpm2_getcap not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//nolint:gosec // G204: tpm2Path is from a fixed allow-list, not user input
	out, err := exec.CommandContext(ctx, "sudo", tpm2Path, "properties-fixed").Output()
	if err != nil {
		return "", fmt.Errorf("tpm2_getcap failed: %w", err)
	}

	return string(out), nil
}

// ParseTPM2GetCap parses the output of tpm2_getcap properties-fixed.
// Extracts manufacturer, model (vendor strings), and spec version (family indicator).
func ParseTPM2GetCap(output string) TPM2GetCapResult {
	var result TPM2GetCapResult

	// Parse TPM2_PT_MANUFACTURER
	if raw := extractGetCapRaw("TPM2_PT_MANUFACTURER", output); raw != "" {
		// raw is like "0x53544d20" — strip "0x" prefix and translate
		hexStr := strings.TrimPrefix(raw, "0x")
		result.Manufacturer = TranslateManufacturerID(hexStr)
	}

	// Parse TPM2_PT_VENDOR_STRING_1 through _4 and concatenate
	var vendorParts []string
	for i := 1; i <= 4; i++ {
		key := fmt.Sprintf("TPM2_PT_VENDOR_STRING_%d", i)
		if val := extractGetCapValue(key, output); val != "" {
			// Trim null characters and whitespace
			cleaned := strings.TrimRight(val, "\x00 ")
			if cleaned != "" {
				vendorParts = append(vendorParts, cleaned)
			}
		}
	}
	if len(vendorParts) > 0 {
		result.Model = strings.Join(vendorParts, "")
	}

	// Parse TPM2_PT_FAMILY_INDICATOR for spec version
	if val := extractGetCapValue("TPM2_PT_FAMILY_INDICATOR", output); val != "" {
		result.SpecVersion = val
	}

	return result
}

// extractGetCapRaw extracts the raw hex value for a property from tpm2_getcap output.
// Format: "TPM2_PT_MANUFACTURER:\n  raw: 0x53544d20\n  value: ..."
func extractGetCapRaw(property, output string) string {
	re := regexp.MustCompile(property + `:\s*\n\s*raw:\s*(0x[0-9a-fA-F]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// extractGetCapValue extracts the quoted value string for a property from tpm2_getcap output.
// Format: "TPM2_PT_VENDOR_STRING_1:\n  raw: 0x...\n  value: \"SLB9\""
func extractGetCapValue(property, output string) string {
	re := regexp.MustCompile(property + `:\s*\n\s*raw:\s*0x[0-9a-fA-F]+\s*\n\s*value:\s*"([^"]*)"`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// detectTPMType classifies the TPM as "discrete", "firmware", or "virtual".
// Returns nil if the type cannot be determined.
func detectTPMType(manufacturer, modalias string) *string {
	// Check 1: VM detection via DMI sys_vendor and product_name
	if isVirtualMachine() {
		t := "virtual"
		return &t
	}

	// Check 2: manufacturer-based classification
	if manufacturer != "" {
		if firmwareTPMManufacturers[manufacturer] {
			t := "firmware"
			return &t
		}
		if discreteTPMManufacturers[manufacturer] {
			t := "discrete"
			return &t
		}
	}

	// Check 3: modalias ACPI prefix
	if modalias != "" {
		if rest, found := strings.CutPrefix(modalias, "acpi:"); found {
			deviceID := strings.TrimSuffix(rest, ":")
			if len(deviceID) >= 4 {
				prefix := deviceID[:4]
				switch prefix {
				case "MSFT", "INTC", "AMD_":
					t := "firmware"
					return &t
				}
			}
		}
	}

	return nil
}

// isVirtualMachine checks DMI fields to determine if running in a virtual machine.
func isVirtualMachine() bool {
	for _, field := range []string{"sys_vendor", "product_name"} {
		if data, err := readDMIFile(filepath.Join(dmiBasePath, field)); err == nil {
			value := strings.TrimSpace(data)
			for _, indicator := range vmVendorIndicators {
				if strings.Contains(value, indicator) {
					return true
				}
			}
		}
	}
	return false
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
