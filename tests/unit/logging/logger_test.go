package logging_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_JSONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&stdout, &stderr)

	logger.Info("test message", map[string]any{
		"foo": "bar",
		"num": 42,
	})

	output := stdout.String()
	assert.NotEmpty(t, output)

	var entry map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &entry))

	assert.Equal(t, "info", entry["level"])
	assert.Equal(t, "test message", entry["message"])
	assert.NotEmpty(t, entry["timestamp"])

	fields, ok := entry["fields"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "bar", fields["foo"])
	assert.Equal(t, float64(42), fields["num"])
}

func TestLogger_HumanFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatHuman)
	logger.SetOutput(&stdout, &stderr)

	logger.Info("test message", map[string]any{
		"foo": "bar",
	})

	output := stdout.String()
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "foo=bar")
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  logging.LogLevel
		logFunc   func(*logging.Logger)
		shouldLog bool
	}{
		{
			name:      "debug logged when level is debug",
			logLevel:  logging.LevelDebug,
			logFunc:   func(l *logging.Logger) { l.Debug("test") },
			shouldLog: true,
		},
		{
			name:      "debug not logged when level is info",
			logLevel:  logging.LevelInfo,
			logFunc:   func(l *logging.Logger) { l.Debug("test") },
			shouldLog: false,
		},
		{
			name:      "info logged when level is info",
			logLevel:  logging.LevelInfo,
			logFunc:   func(l *logging.Logger) { l.Info("test") },
			shouldLog: true,
		},
		{
			name:      "warn logged when level is info",
			logLevel:  logging.LevelInfo,
			logFunc:   func(l *logging.Logger) { l.Warn("test") },
			shouldLog: true,
		},
		{
			name:      "error logged when level is error",
			logLevel:  logging.LevelError,
			logFunc:   func(l *logging.Logger) { l.Error("test") },
			shouldLog: true,
		},
		{
			name:      "info not logged when level is error",
			logLevel:  logging.LevelError,
			logFunc:   func(l *logging.Logger) { l.Info("test") },
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			logger := logging.New(tt.logLevel, logging.FormatJSON)
			logger.SetOutput(&stdout, &stderr)

			tt.logFunc(logger)

			if tt.shouldLog {
				assert.NotEmpty(t, stdout.String()+stderr.String())
			} else {
				assert.Empty(t, stdout.String()+stderr.String())
			}
		})
	}
}

func TestLogger_SecretRedaction(t *testing.T) {
	var stdout bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&stdout, &stdout)

	logger.Info("authentication attempt", map[string]any{
		"username": "testuser",
		"password": "super-secret",
		"token":    "Bearer abc123",
	})

	output := stdout.String()
	assert.NotContains(t, output, "super-secret")
	assert.NotContains(t, output, "abc123")
	assert.Contains(t, output, "[REDACTED]")
	assert.Contains(t, output, "testuser")
}

func TestLogger_ErrorToStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&stdout, &stderr)

	logger.Error("error message")

	assert.Empty(t, stdout.String())
	assert.NotEmpty(t, stderr.String())
	assert.Contains(t, stderr.String(), "error message")
}

func TestLogger_WithFields(t *testing.T) {
	var stdout bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&stdout, &stdout)

	contextLogger := logger.WithFields(map[string]any{
		"request_id": "req-123",
		"user_id":    "user-456",
	})

	contextLogger.Info("test message", map[string]any{
		"action": "login",
	})

	output := stdout.String()
	var entry map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &entry))

	fields, ok := entry["fields"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "req-123", fields["request_id"])
	assert.Equal(t, "user-456", fields["user_id"])
	assert.Equal(t, "login", fields["action"])
}

func TestRedactor_SensitiveKeys(t *testing.T) {
	redactor := logging.NewRedactor()

	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "redact password",
			input: map[string]any{
				"username": "testuser",
				"password": "secret123",
			},
			expected: map[string]any{
				"username": "testuser",
				"password": "[REDACTED]",
			},
		},
		{
			name: "redact SRP values",
			input: map[string]any{
				"A":  "client-ephemeral",
				"B":  "server-ephemeral",
				"M1": "client-proof",
				"M2": "server-proof",
			},
			expected: map[string]any{
				"A":  "[REDACTED]",
				"B":  "[REDACTED]",
				"M1": "[REDACTED]",
				"M2": "[REDACTED]",
			},
		},
		{
			name: "redact session token",
			input: map[string]any{
				"session_token": "abc123",
				"request_id":    "req-456",
			},
			expected: map[string]any{
				"session_token": "[REDACTED]",
				"request_id":    "req-456",
			},
		},
		{
			name: "redact nested fields",
			input: map[string]any{
				"user": map[string]any{
					"name":     "testuser",
					"password": "secret",
				},
			},
			expected: map[string]any{
				"user": map[string]any{
					"name":     "testuser",
					"password": "[REDACTED]",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.RedactFields(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactor_CustomKeys(t *testing.T) {
	redactor := logging.NewRedactor()
	redactor.AddSensitiveKey("custom_secret")

	input := map[string]any{
		"custom_secret": "value",
		"normal_field":  "visible",
	}

	result := redactor.RedactFields(input)

	assert.Equal(t, "[REDACTED]", result["custom_secret"])
	assert.Equal(t, "visible", result["normal_field"])
}

func TestRedactor_RedactString(t *testing.T) {
	redactor := logging.NewRedactor()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "redact password in URL",
			input:    "password=secret123",
			contains: "[REDACTED]",
		},
		{
			name:     "redact token in JSON",
			input:    `"token": "abc123"`,
			contains: "[REDACTED]",
		},
		{
			name:     "no redaction for normal text",
			input:    "normal log message",
			contains: "normal log message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.RedactString(tt.input)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestLogger_MergeFields(t *testing.T) {
	var stdout bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&stdout, &stdout)

	logger.Info("test",
		map[string]any{"key1": "value1"},
		map[string]any{"key2": "value2"},
	)

	output := stdout.String()
	var entry map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &entry))

	fields, ok := entry["fields"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value1", fields["key1"])
	assert.Equal(t, "value2", fields["key2"])
}

func TestLogger_EmptyFields(t *testing.T) {
	var stdout bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&stdout, &stdout)

	logger.Info("test message")

	output := stdout.String()
	var entry map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &entry))

	_, hasFields := entry["fields"]
	assert.False(t, hasFields, "fields should be omitted when empty")
}

func TestLogger_ConcurrentWrites(t *testing.T) {
	var stdout, stderr bytes.Buffer
	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	logger.SetOutput(&stdout, &stderr)

	done := make(chan bool)
	for i := range 100 {
		go func(i int) {
			logger.Info("concurrent message", map[string]any{
				"index": i,
			})
			done <- true
		}(i)
	}

	for range 100 {
		<-done
	}

	// Verify that output contains valid JSON lines
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	assert.Len(t, lines, 100)

	for _, line := range lines {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		assert.Equal(t, "info", entry["level"])
	}
}
