package inventory

import (
	"fmt"
	"os"
)

// GetHostname returns the system hostname.
func GetHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}
	return hostname, nil
}
