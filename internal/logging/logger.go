// Package logging provides structured JSON logging with secret redaction for the BoardingPass service.
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log entry.
type LogLevel string

// Log severity levels.
const (
	// LevelDebug enables debug-level logging.
	LevelDebug LogLevel = "debug"
	// LevelInfo enables info-level logging.
	LevelInfo LogLevel = "info"
	// LevelWarn enables warn-level logging.
	LevelWarn LogLevel = "warn"
	// LevelError enables error-level logging.
	LevelError LogLevel = "error"
)

// LogFormat represents the output format for log entries.
type LogFormat string

// Log output formats.
const (
	// FormatJSON outputs logs as JSON (default).
	FormatJSON LogFormat = "json"
	// FormatHuman outputs logs in human-readable format.
	FormatHuman LogFormat = "human"
)

// Logger provides structured logging with secret redaction.
type Logger struct {
	level    LogLevel
	format   LogFormat
	redactor *Redactor
	stdout   io.Writer
	stderr   io.Writer
	mu       sync.Mutex
}

// logEntry represents a single log entry in JSON format.
type logEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// New creates a new Logger instance.
func New(level LogLevel, format LogFormat) *Logger {
	return &Logger{
		level:    level,
		format:   format,
		redactor: NewRedactor(),
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}
}

// SetOutput sets custom output writers for testing.
func (l *Logger) SetOutput(stdout, stderr io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stdout = stdout
	l.stderr = stderr
}

// Debug logs a debug-level message.
func (l *Logger) Debug(msg string, fields ...map[string]any) {
	l.log(LevelDebug, msg, mergeFields(fields...))
}

// DebugContext logs a debug-level message with context.
func (l *Logger) DebugContext(_ context.Context, msg string, fields ...map[string]any) {
	l.log(LevelDebug, msg, mergeFields(fields...))
}

// Info logs an info-level message.
func (l *Logger) Info(msg string, fields ...map[string]any) {
	l.log(LevelInfo, msg, mergeFields(fields...))
}

// InfoContext logs an info-level message with context.
func (l *Logger) InfoContext(_ context.Context, msg string, fields ...map[string]any) {
	l.log(LevelInfo, msg, mergeFields(fields...))
}

// Warn logs a warn-level message.
func (l *Logger) Warn(msg string, fields ...map[string]any) {
	l.log(LevelWarn, msg, mergeFields(fields...))
}

// WarnContext logs a warn-level message with context.
func (l *Logger) WarnContext(_ context.Context, msg string, fields ...map[string]any) {
	l.log(LevelWarn, msg, mergeFields(fields...))
}

// Error logs an error-level message.
func (l *Logger) Error(msg string, fields ...map[string]any) {
	l.log(LevelError, msg, mergeFields(fields...))
}

// ErrorContext logs an error-level message with context.
func (l *Logger) ErrorContext(_ context.Context, msg string, fields ...map[string]any) {
	l.log(LevelError, msg, mergeFields(fields...))
}

// log writes a log entry to the appropriate output stream.
func (l *Logger) log(level LogLevel, msg string, fields map[string]any) {
	if !l.shouldLog(level) {
		return
	}

	// Apply secret redaction to fields
	redactedFields := l.redactor.RedactFields(fields)

	entry := logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     string(level),
		Message:   msg,
		Fields:    redactedFields,
	}

	var output string
	if l.format == FormatJSON {
		output = l.formatJSON(entry)
	} else {
		output = l.formatHuman(entry)
	}

	l.write(level, output)
}

// shouldLog determines if a message at the given level should be logged.
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}

	return levels[level] >= levels[l.level]
}

// formatJSON formats a log entry as JSON.
func (l *Logger) formatJSON(entry logEntry) string {
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback if JSON marshaling fails
		return fmt.Sprintf(`{"timestamp":"%s","level":"error","message":"failed to marshal log entry: %s"}`,
			time.Now().UTC().Format(time.RFC3339), err.Error())
	}
	return string(data) + "\n"
}

// formatHuman formats a log entry in human-readable format.
func (l *Logger) formatHuman(entry logEntry) string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("[%s] %s: %s", entry.Timestamp, entry.Level, entry.Message))

	if len(entry.Fields) > 0 {
		// Format fields as key=value pairs
		for k, v := range entry.Fields {
			output.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
	}

	output.WriteString("\n")
	return output.String()
}

// write writes the formatted output to the appropriate stream.
func (l *Logger) write(level LogLevel, output string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	writer := l.stdout
	if level == LevelError {
		writer = l.stderr
	}

	_, _ = writer.Write([]byte(output))
}

// mergeFields merges multiple field maps into one.
func mergeFields(fields ...map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	merged := make(map[string]any)
	for _, f := range fields {
		maps.Copy(merged, f)
	}

	return merged
}

// WithFields creates a new logger with additional fields.
func (l *Logger) WithFields(fields map[string]any) *ContextLogger {
	return &ContextLogger{
		logger: l,
		fields: fields,
	}
}

// ContextLogger wraps a Logger with context-specific fields.
type ContextLogger struct {
	logger *Logger
	fields map[string]any
}

// Debug logs a debug-level message with context fields.
func (cl *ContextLogger) Debug(msg string, fields ...map[string]any) {
	allFields := mergeFields(append([]map[string]any{cl.fields}, fields...)...)
	cl.logger.Debug(msg, allFields)
}

// Info logs an info-level message with context fields.
func (cl *ContextLogger) Info(msg string, fields ...map[string]any) {
	allFields := mergeFields(append([]map[string]any{cl.fields}, fields...)...)
	cl.logger.Info(msg, allFields)
}

// Warn logs a warn-level message with context fields.
func (cl *ContextLogger) Warn(msg string, fields ...map[string]any) {
	allFields := mergeFields(append([]map[string]any{cl.fields}, fields...)...)
	cl.logger.Warn(msg, allFields)
}

// Error logs an error-level message with context fields.
func (cl *ContextLogger) Error(msg string, fields ...map[string]any) {
	allFields := mergeFields(append([]map[string]any{cl.fields}, fields...)...)
	cl.logger.Error(msg, allFields)
}
