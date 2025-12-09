// Package provisioning provides atomic configuration bundle provisioning
// with rollback capabilities and path validation for the BoardingPass service.
package provisioning

import (
	"encoding/base64"
	"fmt"

	"github.com/fzdarsky/boardingpass/pkg/protocol"
)

const (
	// MaxBundleSize is the maximum total size of decoded content (10MB)
	MaxBundleSize = 10 * 1024 * 1024

	// MaxFileCount is the maximum number of files in a bundle
	MaxFileCount = 100
)

// ValidateBundle validates a configuration bundle for size and count constraints.
// Returns an error if the bundle exceeds limits or contains invalid data.
func ValidateBundle(bundle *protocol.ConfigBundle) error {
	if bundle == nil {
		return fmt.Errorf("bundle cannot be nil")
	}

	if len(bundle.Files) == 0 {
		return fmt.Errorf("bundle must contain at least one file")
	}

	if len(bundle.Files) > MaxFileCount {
		return fmt.Errorf("bundle contains %d files, maximum is %d", len(bundle.Files), MaxFileCount)
	}

	totalSize := int64(0)
	for i, file := range bundle.Files {
		if file.Path == "" {
			return fmt.Errorf("file at index %d has empty path", i)
		}

		if file.Content == "" {
			return fmt.Errorf("file %s has empty content", file.Path)
		}

		// Validate mode is within valid Unix permissions range (0-0777)
		if file.Mode < 0 || file.Mode > 0o777 {
			return fmt.Errorf("file %s has invalid mode %o, must be 0-0777", file.Path, file.Mode)
		}

		// Decode Base64 content to check validity and measure size
		decoded, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return fmt.Errorf("file %s has invalid Base64 content: %w", file.Path, err)
		}

		totalSize += int64(len(decoded))
		if totalSize > MaxBundleSize {
			return fmt.Errorf("bundle exceeds maximum size of %d bytes", MaxBundleSize)
		}
	}

	return nil
}

// DecodeFileContent decodes a Base64-encoded file content string.
// Returns the decoded bytes or an error if decoding fails.
func DecodeFileContent(content string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Base64 content: %w", err)
	}
	return decoded, nil
}
