// Package output provides output formatting utilities for the boarding CLI tool.
package output

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Format represents an output format.
type Format string

const (
	// FormatYAML represents YAML output format.
	FormatYAML Format = "yaml"
	// FormatJSON represents JSON output format.
	FormatJSON Format = "json"
)

// FormatData formats data according to the specified format.
// Returns the formatted output as a string.
func FormatData(data any, format Format) (string, error) {
	switch format {
	case FormatYAML:
		return formatYAML(data)
	case FormatJSON:
		return formatJSON(data)
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

// formatYAML formats data as YAML.
func formatYAML(data any) (string, error) {
	bytes, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to format as YAML: %w", err)
	}
	return string(bytes), nil
}

// formatJSON formats data as indented JSON.
func formatJSON(data any) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format as JSON: %w", err)
	}
	return string(bytes), nil
}

// ParseFormat parses a format string into a Format value.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "yaml", "yml":
		return FormatYAML, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid output format '%s': must be 'yaml' or 'json'", s)
	}
}
