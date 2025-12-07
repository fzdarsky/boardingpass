package logging

import (
	"strings"
)

const redactedValue = "[REDACTED]"

// Redactor handles secret redaction in log fields.
type Redactor struct {
	sensitiveKeys map[string]bool
}

// NewRedactor creates a new Redactor with default sensitive keys.
func NewRedactor() *Redactor {
	return &Redactor{
		sensitiveKeys: map[string]bool{
			// Authentication & session
			"password":      true,
			"token":         true,
			"secret":        true,
			"key":           true,
			"session":       true,
			"session_token": true,
			"authorization": true,

			// SRP protocol values
			"proof":    true,
			"verifier": true,
			"salt":     true, // Salt can be logged in some contexts, but redact by default
			"a":        true, // SRP ephemeral client private
			"b":        true, // SRP ephemeral server private
			"m1":       true, // SRP client proof (lowercase to match case-insensitive check)
			"m2":       true, // SRP server proof (lowercase to match case-insensitive check)

			// Configuration & content
			"content":       true,
			"payload":       true,
			"config":        true,
			"bundle":        true,
			"file_content":  true,
			"configuration": true,

			// Credentials & secrets
			"api_key":     true,
			"access_key":  true,
			"secret_key":  true,
			"private_key": true,
			"cert":        true,
			"certificate": true,
			"tls_cert":    true,
			"tls_key":     true,
		},
	}
}

// AddSensitiveKey adds a custom key to the redaction list.
func (r *Redactor) AddSensitiveKey(key string) {
	r.sensitiveKeys[strings.ToLower(key)] = true
}

// RemoveSensitiveKey removes a key from the redaction list.
func (r *Redactor) RemoveSensitiveKey(key string) {
	delete(r.sensitiveKeys, strings.ToLower(key))
}

// RedactFields redacts sensitive values from a map of fields.
func (r *Redactor) RedactFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}

	redacted := make(map[string]any, len(fields))

	for k, v := range fields {
		if r.isSensitiveKey(k) {
			redacted[k] = redactedValue
		} else if nested, ok := v.(map[string]any); ok {
			// Recursively redact nested maps
			redacted[k] = r.RedactFields(nested)
		} else {
			redacted[k] = v
		}
	}

	return redacted
}

// RedactString redacts sensitive values from a string by checking for key patterns.
func (r *Redactor) RedactString(s string) string {
	// Simple pattern matching for common secrets in strings
	// This is a basic implementation - could be enhanced with regex patterns

	for key := range r.sensitiveKeys {
		// Look for patterns like "key=value" or "key: value"
		patterns := []string{
			key + "=",
			key + ": ",
			"\"" + key + "\":",
		}

		for _, pattern := range patterns {
			if strings.Contains(strings.ToLower(s), pattern) {
				// Found a potential secret - redact the whole line for safety
				return redactedValue
			}
		}
	}

	return s
}

// isSensitiveKey checks if a field key is marked as sensitive.
func (r *Redactor) isSensitiveKey(key string) bool {
	// Only check exact match (case-insensitive)
	// Substring matching was too aggressive and caught legitimate fields
	return r.sensitiveKeys[strings.ToLower(key)]
}
