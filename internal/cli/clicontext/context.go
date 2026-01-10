// Package clicontext provides global CLI context and state management.
package clicontext

import "sync"

// Global holds the global CLI context, including flags that affect all commands.
type Global struct {
	// AssumeYes automatically answers 'yes' to all prompts (non-interactive mode).
	// This is particularly useful for CI/CD pipelines and automated scripts.
	AssumeYes bool
}

var (
	globalContext = &Global{}
	mu            sync.RWMutex
)

// Set updates the global CLI context.
func Set(ctx *Global) {
	mu.Lock()
	defer mu.Unlock()
	globalContext = ctx
}

// Get returns the current global CLI context.
func Get() *Global {
	mu.RLock()
	defer mu.RUnlock()
	return globalContext
}

// AssumeYes returns whether the CLI is in assume-yes mode.
func AssumeYes() bool {
	mu.RLock()
	defer mu.RUnlock()
	return globalContext.AssumeYes
}

// SetAssumeYes sets the assume-yes flag.
func SetAssumeYes(value bool) {
	mu.Lock()
	defer mu.Unlock()
	globalContext.AssumeYes = value
}
