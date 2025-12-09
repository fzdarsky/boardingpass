package inventory

import (
	"os"
	"strings"
)

const fipsModePath = "/proc/sys/crypto/fips_enabled"

// GetFIPSStatus checks if FIPS 140-3 mode is enabled.
// Returns true if /proc/sys/crypto/fips_enabled contains "1".
func GetFIPSStatus() bool {
	data, err := os.ReadFile(fipsModePath)
	if err != nil {
		// FIPS mode file doesn't exist or can't be read - not enabled
		return false
	}

	value := strings.TrimSpace(string(data))
	return value == "1"
}
