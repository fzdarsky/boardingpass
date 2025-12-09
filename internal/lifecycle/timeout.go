package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	// DefaultInactivityTimeout is the default duration before the service
	// automatically shuts down due to inactivity.
	DefaultInactivityTimeout = 10 * time.Minute

	// MinimumInactivityTimeout is the minimum allowed timeout duration.
	MinimumInactivityTimeout = 1 * time.Minute
)

// InactivityTracker tracks the last activity time and triggers shutdown
// after a period of inactivity.
type InactivityTracker struct {
	timeout      time.Duration
	lastActivity time.Time
	mu           sync.RWMutex
	shutdownFunc func()
	timer        *time.Timer
	stopped      bool
}

// NewInactivityTracker creates a new inactivity tracker.
// If timeout is 0, uses DefaultInactivityTimeout.
// If timeout is less than MinimumInactivityTimeout, returns an error.
func NewInactivityTracker(timeout time.Duration, shutdownFunc func()) (*InactivityTracker, error) {
	if timeout == 0 {
		timeout = DefaultInactivityTimeout
	}
	if timeout < MinimumInactivityTimeout {
		return nil, fmt.Errorf("timeout must be at least %v, got %v", MinimumInactivityTimeout, timeout)
	}

	return newInactivityTrackerInternal(timeout, shutdownFunc), nil
}

// newInactivityTrackerInternal creates a tracker without minimum validation.
// Used internally and by tests that need shorter timeouts.
func newInactivityTrackerInternal(timeout time.Duration, shutdownFunc func()) *InactivityTracker {
	if timeout == 0 {
		timeout = DefaultInactivityTimeout
	}

	return &InactivityTracker{
		timeout:      timeout,
		lastActivity: time.Now(),
		shutdownFunc: shutdownFunc,
		stopped:      false,
	}
}

// Start begins monitoring for inactivity.
// Should be called in a goroutine.
func (t *InactivityTracker) Start(ctx context.Context) {
	t.mu.Lock()
	t.timer = time.NewTimer(t.timeout)
	t.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			t.mu.Lock()
			if t.timer != nil {
				t.timer.Stop()
			}
			t.stopped = true
			t.mu.Unlock()
			return

		case <-t.timer.C:
			t.mu.RLock()
			timeSinceActivity := time.Since(t.lastActivity)
			stopped := t.stopped
			t.mu.RUnlock()

			if stopped {
				return
			}

			if timeSinceActivity >= t.timeout {
				// Inactivity timeout reached, trigger shutdown
				if t.shutdownFunc != nil {
					t.shutdownFunc()
				}
				return
			}

			// Reset timer for remaining time
			t.mu.Lock()
			remaining := t.timeout - timeSinceActivity
			if t.timer != nil {
				t.timer.Reset(remaining)
			}
			t.mu.Unlock()
		}
	}
}

// RecordActivity records that activity has occurred, resetting the inactivity timer.
func (t *InactivityTracker) RecordActivity() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return
	}

	t.lastActivity = time.Now()

	// Reset the timer to the full timeout duration
	if t.timer != nil {
		t.timer.Stop()
		t.timer.Reset(t.timeout)
	}
}

// Stop stops the inactivity tracker.
func (t *InactivityTracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.stopped = true
	if t.timer != nil {
		t.timer.Stop()
	}
}

// LastActivity returns the time of the last recorded activity.
func (t *InactivityTracker) LastActivity() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastActivity
}

// TimeUntilTimeout returns the duration until the inactivity timeout will trigger.
func (t *InactivityTracker) TimeUntilTimeout() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	elapsed := time.Since(t.lastActivity)
	if elapsed >= t.timeout {
		return 0
	}
	return t.timeout - elapsed
}
