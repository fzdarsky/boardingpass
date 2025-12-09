package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownManager handles graceful service shutdown.
type ShutdownManager struct {
	shutdownChan chan struct{}
	signalChan   chan os.Signal
	mu           sync.Mutex
	shutdown     bool
	stopped      bool
	reason       string
}

// NewShutdownManager creates a new shutdown manager.
func NewShutdownManager() *ShutdownManager {
	return &ShutdownManager{
		shutdownChan: make(chan struct{}, 1),
		signalChan:   make(chan os.Signal, 1),
		shutdown:     false,
	}
}

// Start begins listening for shutdown signals (SIGTERM, SIGINT).
// Returns a context that will be cancelled when shutdown is initiated.
func (sm *ShutdownManager) Start(ctx context.Context) context.Context {
	signal.Notify(sm.signalChan, syscall.SIGTERM, syscall.SIGINT)

	shutdownCtx, cancel := context.WithCancel(ctx)

	go func() {
		select {
		case sig := <-sm.signalChan:
			sm.mu.Lock()
			if !sm.shutdown {
				sm.shutdown = true
				sm.reason = fmt.Sprintf("received signal: %v", sig)
			}
			sm.mu.Unlock()
			cancel()

		case <-sm.shutdownChan:
			cancel()

		case <-ctx.Done():
			cancel()
		}
	}()

	return shutdownCtx
}

// Shutdown initiates a graceful shutdown with the given reason.
func (sm *ShutdownManager) Shutdown(reason string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.shutdown {
		return
	}

	sm.shutdown = true
	sm.reason = reason

	select {
	case sm.shutdownChan <- struct{}{}:
	default:
	}
}

// IsShutdown returns whether shutdown has been initiated.
func (sm *ShutdownManager) IsShutdown() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.shutdown
}

// Reason returns the reason for shutdown.
func (sm *ShutdownManager) Reason() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.reason
}

// Stop stops listening for signals and closes channels.
func (sm *ShutdownManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.stopped {
		return
	}

	sm.stopped = true
	signal.Stop(sm.signalChan)
	close(sm.signalChan)
	close(sm.shutdownChan)
}

// GracefulShutdown performs a graceful shutdown of the given shutdownFunc
// with a timeout. If the shutdown takes longer than timeout, it forces shutdown.
func GracefulShutdown(ctx context.Context, shutdownFunc func(context.Context) error, timeout time.Duration) error {
	// Create a context with timeout for the shutdown operation
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel to receive shutdown result
	done := make(chan error, 1)

	// Execute shutdown in a goroutine
	go func() {
		done <- shutdownFunc(shutdownCtx)
	}()

	// Wait for shutdown to complete or timeout
	select {
	case err := <-done:
		return err
	case <-shutdownCtx.Done():
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}
