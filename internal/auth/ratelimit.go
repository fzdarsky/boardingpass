package auth

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded is returned when a client is rate-limited
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrClientLocked is returned when a client is locked out due to too many failed attempts
	ErrClientLocked = errors.New("client locked out")
)

const (
	// Progressive delay thresholds
	delay1stFailure = 1 * time.Second
	delay2ndFailure = 2 * time.Second
	delay3rdFailure = 5 * time.Second
	lockoutDuration = 60 * time.Second

	// CleanupThreshold is how long to keep attempt trackers for inactive clients
	CleanupThreshold = 5 * time.Minute

	// CleanupIntervalRateLimit is how often to clean up inactive client trackers
	CleanupIntervalRateLimit = 2 * time.Minute
)

// AttemptTracker tracks authentication attempts for a single client IP.
type AttemptTracker struct {
	Count       int       // Number of consecutive failed attempts
	LastFailed  time.Time // Timestamp of last failed attempt
	LockedUntil time.Time // Timestamp when lockout expires (zero if not locked)
}

// IsLocked returns true if the client is currently locked out.
func (at *AttemptTracker) IsLocked() bool {
	return time.Now().Before(at.LockedUntil)
}

// TimeUntilUnlock returns the duration until the lockout expires.
// Returns 0 if not locked.
func (at *AttemptTracker) TimeUntilUnlock() time.Duration {
	if !at.IsLocked() {
		return 0
	}
	return time.Until(at.LockedUntil)
}

// RateLimiter implements progressive delay brute force protection.
// Delays increase with each failed attempt: 1s, 2s, 5s, then 60s lockout.
type RateLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*AttemptTracker // key: client IP address
	stopCh   chan struct{}              // Channel to stop cleanup goroutine
}

// NewRateLimiter creates a new rate limiter with background cleanup.
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string]*AttemptTracker),
		stopCh:   make(chan struct{}),
	}

	// Start background cleanup goroutine
	go rl.cleanupInactiveClients()

	return rl
}

// CheckLimit checks if a client IP is currently rate-limited or locked out.
// Returns (locked, retryAfter, error):
//   - locked: true if client is locked out (3+ failures)
//   - retryAfter: duration to wait before next attempt
//   - error: ErrClientLocked if locked, ErrRateLimitExceeded if rate-limited, nil otherwise
func (rl *RateLimiter) CheckLimit(clientIP string) (locked bool, retryAfter time.Duration, err error) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	tracker, exists := rl.attempts[clientIP]
	if !exists {
		// No attempts yet, allow request
		return false, 0, nil
	}

	// Check if client is locked out (either by time-based lock or failure count)
	if tracker.IsLocked() {
		retryAfter := tracker.TimeUntilUnlock()
		return true, retryAfter, ErrClientLocked
	}

	// Check if client has 3+ failures (should be locked on 4th+ attempt)
	if tracker.Count >= 3 {
		// Apply lockout for 4th+ attempt
		retryAfter := lockoutDuration
		return true, retryAfter, ErrClientLocked
	}

	// No active lockout, allow request
	return false, 0, nil
}

// RecordFailure records a failed authentication attempt for a client IP.
// Returns the delay duration that should be enforced before the next attempt.
func (rl *RateLimiter) RecordFailure(clientIP string) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	tracker, exists := rl.attempts[clientIP]
	if !exists {
		tracker = &AttemptTracker{}
		rl.attempts[clientIP] = tracker
	}

	// Increment failure count
	tracker.Count++
	tracker.LastFailed = time.Now()

	// Determine delay based on failure count
	var delay time.Duration
	switch tracker.Count {
	case 1:
		// First failure: 1 second delay
		delay = delay1stFailure
	case 2:
		// Second failure: 2 second delay
		delay = delay2ndFailure
	case 3:
		// Third failure: 5 second delay
		delay = delay3rdFailure
	default:
		// 4+ failures: 60 second lockout
		tracker.LockedUntil = time.Now().Add(lockoutDuration)
		delay = lockoutDuration
	}

	return delay
}

// RecordSuccess records a successful authentication for a client IP.
// This clears the failure count and removes any lockout.
func (rl *RateLimiter) RecordSuccess(clientIP string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Clear all attempt tracking for this client
	delete(rl.attempts, clientIP)
}

// GetAttemptCount returns the current failure count for a client IP.
// Returns 0 if no failures have been recorded.
func (rl *RateLimiter) GetAttemptCount(clientIP string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	tracker, exists := rl.attempts[clientIP]
	if !exists {
		return 0
	}
	return tracker.Count
}

// ClearAttempts manually clears all attempt tracking for a client IP.
// This can be used for administrative unlock.
func (rl *RateLimiter) ClearAttempts(clientIP string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, clientIP)
}

// GetTrackedClientCount returns the number of clients currently being tracked.
func (rl *RateLimiter) GetTrackedClientCount() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return len(rl.attempts)
}

// Stop stops the background cleanup goroutine.
// Should be called when shutting down the service.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// cleanupInactiveClients is a background goroutine that removes attempt trackers
// for clients that have been inactive for CleanupThreshold duration.
func (rl *RateLimiter) cleanupInactiveClients() {
	ticker := time.NewTicker(CleanupIntervalRateLimit)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.performCleanup()
		case <-rl.stopCh:
			return
		}
	}
}

// performCleanup removes attempt trackers for inactive clients.
func (rl *RateLimiter) performCleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-CleanupThreshold)

	for clientIP, tracker := range rl.attempts {
		// Remove if:
		// 1. Lockout has expired AND last failure was > CleanupThreshold ago, OR
		// 2. Not locked and last failure was > CleanupThreshold ago
		if tracker.LastFailed.Before(cutoff) && !tracker.IsLocked() {
			delete(rl.attempts, clientIP)
		}
	}
}

// GetDelayForAttemptCount returns the delay duration for a given attempt count.
// This is useful for testing or displaying rate limit information.
func GetDelayForAttemptCount(count int) time.Duration {
	switch count {
	case 1:
		return delay1stFailure
	case 2:
		return delay2ndFailure
	case 3:
		return delay3rdFailure
	default:
		if count >= 4 {
			return lockoutDuration
		}
		return 0
	}
}

// FormatRetryAfter formats a duration as seconds for use in HTTP Retry-After header.
// Returns the number of seconds rounded up to the nearest integer.
func FormatRetryAfter(d time.Duration) int {
	seconds := int(d.Seconds())
	if d.Nanoseconds()%int64(time.Second) > 0 {
		seconds++ // Round up
	}
	return seconds
}
