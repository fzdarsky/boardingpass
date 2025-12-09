package inventory

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const osReleasePath = "/etc/os-release"

// GetOSInfo detects OS distribution and version from /etc/os-release.
// Follows the os-release(5) specification from systemd.
func GetOSInfo() (protocol.OSInfo, error) {
	info := protocol.OSInfo{
		Distribution: "Unknown",
		Version:      "Unknown",
		FIPSEnabled:  false,
	}

	file, err := os.Open(osReleasePath)
	if err != nil {
		return info, fmt.Errorf("failed to open %s: %w", osReleasePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close %s: %w", osReleasePath, closeErr)
		}
	}()

	osData := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE or KEY="VALUE" format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		osData[key] = value
	}

	if err := scanner.Err(); err != nil {
		return info, fmt.Errorf("failed to read %s: %w", osReleasePath, err)
	}

	// Extract distribution name (prefer PRETTY_NAME, fallback to NAME)
	if name, ok := osData["PRETTY_NAME"]; ok {
		info.Distribution = name
	} else if name, ok := osData["NAME"]; ok {
		info.Distribution = name
	}

	// Extract version (prefer VERSION, fallback to VERSION_ID)
	if version, ok := osData["VERSION"]; ok {
		info.Version = version
	} else if versionID, ok := osData["VERSION_ID"]; ok {
		info.Version = versionID
	}

	// Check FIPS mode (will be set by GetFIPSStatus)
	info.FIPSEnabled = GetFIPSStatus()

	return info, nil
}
