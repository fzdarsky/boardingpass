package inventory

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const dmiBasePath = "/sys/class/dmi/id"

// GetFirmwareInfo extracts firmware information (BIOS/UEFI/U-Boot).
// On x86, reads from DMI tables. On ARM, attempts to read U-Boot version from device tree.
func GetFirmwareInfo() protocol.FirmwareInfo {
	info := protocol.FirmwareInfo{
		Vendor:  "Unknown",
		Version: "Unknown",
		Date:    "Unknown",
	}

	// Try DMI first (x86_64 systems with BIOS/UEFI)
	if _, err := os.Stat(dmiBasePath); err == nil {
		if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "bios_vendor")); err == nil {
			if v := filterPlaceholder(v); v != "" {
				info.Vendor = v
			}
		}
		if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "bios_version")); err == nil {
			if v := filterPlaceholder(v); v != "" {
				info.Version = v
			}
		}
		if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "bios_date")); err == nil {
			if v := filterPlaceholder(v); v != "" {
				info.Date = v
			}
		}
		return info
	}

	// Fallback: try U-Boot version from device tree (ARM systems)
	if v, err := readDMIFile("/proc/device-tree/chosen/u-boot,version"); err == nil {
		v = strings.TrimSpace(strings.TrimRight(v, "\x00"))
		if v != "" {
			info.Version = v
		}
	}

	return info
}

// GetProductInfo extracts product identity using fallback chain:
// product fields → board fields → device tree.
func GetProductInfo() (protocol.ProductInfo, error) {
	// Try DMI first (x86_64 systems)
	if _, err := os.Stat(dmiBasePath); err == nil {
		return readDMIProductInfo()
	}

	// Fallback to device tree for ARM/embedded systems
	return readDeviceTreeProductInfo()
}

// readDMIProductInfo reads product info from DMI with board fallbacks.
func readDMIProductInfo() (protocol.ProductInfo, error) {
	info := protocol.ProductInfo{
		Vendor:  "Unknown",
		Family:  "Unknown",
		Name:    "Unknown",
		Version: "Unknown",
		Serial:  "Unknown",
	}

	// Vendor: sys_vendor → board_vendor
	if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "sys_vendor")); err == nil {
		if v := filterPlaceholder(v); v != "" {
			info.Vendor = v
		}
	}
	if info.Vendor == "Unknown" {
		if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "board_vendor")); err == nil {
			if v := filterPlaceholder(v); v != "" {
				info.Vendor = v
			}
		}
	}

	// Family: product_family (no fallback)
	if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "product_family")); err == nil {
		if v := filterPlaceholder(v); v != "" {
			info.Family = v
		}
	}

	// Name: product_name → board_name
	if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "product_name")); err == nil {
		if v := filterPlaceholder(v); v != "" {
			info.Name = v
		}
	}
	if info.Name == "Unknown" {
		if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "board_name")); err == nil {
			if v := filterPlaceholder(v); v != "" {
				info.Name = v
			}
		}
	}

	// Version: product_version → board_version
	if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "product_version")); err == nil {
		if v := filterPlaceholder(v); v != "" {
			info.Version = v
		}
	}
	if info.Version == "Unknown" {
		if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "board_version")); err == nil {
			if v := filterPlaceholder(v); v != "" {
				info.Version = v
			}
		}
	}

	// Serial: product_serial → board_serial
	if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "product_serial")); err == nil {
		if v := filterPlaceholder(v); v != "" {
			info.Serial = v
		}
	}
	if info.Serial == "Unknown" {
		if v, err := readDMIFileWithFallback(filepath.Join(dmiBasePath, "board_serial")); err == nil {
			if v := filterPlaceholder(v); v != "" {
				info.Serial = v
			}
		}
	}

	return info, nil
}

// readDeviceTreeProductInfo reads product info from device tree (ARM systems).
func readDeviceTreeProductInfo() (protocol.ProductInfo, error) {
	info := protocol.ProductInfo{
		Vendor:  "Unknown",
		Family:  "Unknown",
		Name:    "Unknown",
		Version: "Unknown",
		Serial:  "Unknown",
	}

	deviceTreeBase := "/proc/device-tree"

	// Read model from device tree
	if model, err := readDMIFile(filepath.Join(deviceTreeBase, "model")); err == nil {
		model = strings.TrimSpace(strings.TrimRight(model, "\x00"))
		if model != "" {
			info.Name = model
			// Infer vendor from model string
			if strings.Contains(strings.ToLower(model), "raspberry pi") {
				info.Vendor = "Raspberry Pi Foundation"
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

	// Try alternative serial location (Raspberry Pi specific)
	if info.Serial == "Unknown" {
		if cpuinfo, err := readDMIFile("/proc/cpuinfo"); err == nil {
			for line := range strings.SplitSeq(cpuinfo, "\n") {
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

// filterPlaceholder trims whitespace and returns empty string for OEM placeholder values.
func filterPlaceholder(s string) string {
	s = strings.TrimSpace(s)
	if s == "To be filled by O.E.M." || s == "Default string" {
		return ""
	}
	return s
}

// readDMIFile reads a single line from a sysfs file.
func readDMIFile(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readDMIFileWithFallback reads a DMI sysfs file, falling back to sudo cat
// if os.ReadFile fails with a permission error. The sudo fallback is only
// attempted for files directly within /sys/class/dmi/id/.
func readDMIFileWithFallback(path string) (string, error) {
	cleanPath := filepath.Clean(path)

	data, err := os.ReadFile(cleanPath)
	if err == nil {
		return string(data), nil
	}

	if !errors.Is(err, os.ErrPermission) {
		return "", err
	}

	// Defense-in-depth: validate path is directly within DMI directory
	rel, relErr := filepath.Rel(dmiBasePath, cleanPath)
	if relErr != nil || strings.Contains(rel, "/") || strings.Contains(rel, "..") {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//nolint:gosec // G204: path is validated to be within /sys/class/dmi/id/
	out, sudoErr := exec.CommandContext(ctx, "sudo", "cat", cleanPath).Output()
	if sudoErr != nil {
		return "", err
	}

	return string(out), nil
}
