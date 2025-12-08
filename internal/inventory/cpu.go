package inventory

import (
	"runtime"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

// GetCPUInfo detects the CPU architecture using runtime.GOARCH.
// Maps Go architecture strings to standardized values.
func GetCPUInfo() protocol.CPUInfo {
	arch := runtime.GOARCH

	// Map Go GOARCH values to standardized architecture names
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	case "arm":
		arch = "armv7l"
	}

	return protocol.CPUInfo{
		Architecture: arch,
	}
}
