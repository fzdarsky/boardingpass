package auth_test

import (
	"testing"
	"time"

	"github.com/fzdarsky/boardingpass/internal/auth"
)

func TestNewRateLimiter(t *testing.T) {
	rl := auth.NewRateLimiter()
	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}

	// Cleanup
	rl.Stop()
}

func TestRateLimiter_CheckLimit_NoAttempts(t *testing.T) {
	rl := auth.NewRateLimiter()
	defer rl.Stop()

	locked, retryAfter, err := rl.CheckLimit("192.168.1.100")
	if locked {
		t.Error("expected client not to be locked initially")
	}
	if retryAfter != 0 {
		t.Errorf("expected 0 retryAfter, got %v", retryAfter)
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRateLimiter_RecordFailure_ProgressiveDelays(t *testing.T) {
	rl := auth.NewRateLimiter()
	defer rl.Stop()

	clientIP := "192.168.1.100"

	// First failure: 1 second delay
	delay1 := rl.RecordFailure(clientIP)
	if delay1 != 1*time.Second {
		t.Errorf("expected 1s delay on first failure, got %v", delay1)
	}

	// Second failure: 2 second delay
	delay2 := rl.RecordFailure(clientIP)
	if delay2 != 2*time.Second {
		t.Errorf("expected 2s delay on second failure, got %v", delay2)
	}

	// Third failure: 5 second delay
	delay3 := rl.RecordFailure(clientIP)
	if delay3 != 5*time.Second {
		t.Errorf("expected 5s delay on third failure, got %v", delay3)
	}

	// Fourth+ failure: 60 second lockout
	delay4 := rl.RecordFailure(clientIP)
	if delay4 != 60*time.Second {
		t.Errorf("expected 60s lockout on fourth failure, got %v", delay4)
	}
}

func TestRateLimiter_CheckLimit_Locked(t *testing.T) {
	rl := auth.NewRateLimiter()
	defer rl.Stop()

	clientIP := "192.168.1.100"

	// Trigger lockout (4 failures)
	for i := 0; i < 4; i++ {
		rl.RecordFailure(clientIP)
	}

	// Client should be locked
	locked, retryAfter, err := rl.CheckLimit(clientIP)
	if !locked {
		t.Error("expected client to be locked after 4 failures")
	}
	if retryAfter <= 0 {
		t.Error("expected positive retryAfter duration")
	}
	if retryAfter > 60*time.Second {
		t.Errorf("expected retryAfter <= 60s, got %v", retryAfter)
	}
	if err != auth.ErrClientLocked {
		t.Errorf("expected ErrClientLocked, got %v", err)
	}
}

func TestRateLimiter_RecordSuccess(t *testing.T) {
	rl := auth.NewRateLimiter()
	defer rl.Stop()

	clientIP := "192.168.1.100"

	// Record some failures
	rl.RecordFailure(clientIP)
	rl.RecordFailure(clientIP)

	if rl.GetAttemptCount(clientIP) != 2 {
		t.Errorf("expected 2 attempts, got %d", rl.GetAttemptCount(clientIP))
	}

	// Record success (should clear attempts)
	rl.RecordSuccess(clientIP)

	if rl.GetAttemptCount(clientIP) != 0 {
		t.Errorf("expected 0 attempts after success, got %d", rl.GetAttemptCount(clientIP))
	}

	// Next failure should be treated as first failure (1s delay)
	delay := rl.RecordFailure(clientIP)
	if delay != 1*time.Second {
		t.Errorf("expected 1s delay after success, got %v", delay)
	}
}

func TestRateLimiter_GetAttemptCount(t *testing.T) {
	rl := auth.NewRateLimiter()
	defer rl.Stop()

	clientIP := "192.168.1.100"

	if rl.GetAttemptCount(clientIP) != 0 {
		t.Error("expected 0 attempts initially")
	}

	rl.RecordFailure(clientIP)
	if rl.GetAttemptCount(clientIP) != 1 {
		t.Errorf("expected 1 attempt, got %d", rl.GetAttemptCount(clientIP))
	}

	rl.RecordFailure(clientIP)
	if rl.GetAttemptCount(clientIP) != 2 {
		t.Errorf("expected 2 attempts, got %d", rl.GetAttemptCount(clientIP))
	}
}

func TestRateLimiter_ClearAttempts(t *testing.T) {
	rl := auth.NewRateLimiter()
	defer rl.Stop()

	clientIP := "192.168.1.100"

	// Record failures and trigger lockout
	for i := 0; i < 4; i++ {
		rl.RecordFailure(clientIP)
	}

	locked, _, _ := rl.CheckLimit(clientIP)
	if !locked {
		t.Error("expected client to be locked")
	}

	// Clear attempts (administrative unlock)
	rl.ClearAttempts(clientIP)

	locked, _, err := rl.CheckLimit(clientIP)
	if locked {
		t.Error("expected client not to be locked after clearing")
	}
	if err != nil {
		t.Errorf("unexpected error after clearing: %v", err)
	}

	if rl.GetAttemptCount(clientIP) != 0 {
		t.Errorf("expected 0 attempts after clearing, got %d", rl.GetAttemptCount(clientIP))
	}
}

func TestRateLimiter_MultipleClients(t *testing.T) {
	rl := auth.NewRateLimiter()
	defer rl.Stop()

	client1 := "192.168.1.100"
	client2 := "192.168.1.101"

	// Record failures for both clients
	rl.RecordFailure(client1)
	rl.RecordFailure(client2)
	rl.RecordFailure(client2)

	if rl.GetAttemptCount(client1) != 1 {
		t.Errorf("expected 1 attempt for client1, got %d", rl.GetAttemptCount(client1))
	}
	if rl.GetAttemptCount(client2) != 2 {
		t.Errorf("expected 2 attempts for client2, got %d", rl.GetAttemptCount(client2))
	}

	// Clear one client
	rl.RecordSuccess(client1)

	if rl.GetAttemptCount(client1) != 0 {
		t.Errorf("expected 0 attempts for client1 after success, got %d", rl.GetAttemptCount(client1))
	}
	if rl.GetAttemptCount(client2) != 2 {
		t.Errorf("expected 2 attempts for client2 (unchanged), got %d", rl.GetAttemptCount(client2))
	}
}

func TestRateLimiter_GetTrackedClientCount(t *testing.T) {
	rl := auth.NewRateLimiter()
	defer rl.Stop()

	if rl.GetTrackedClientCount() != 0 {
		t.Error("expected 0 tracked clients initially")
	}

	rl.RecordFailure("192.168.1.100")
	if rl.GetTrackedClientCount() != 1 {
		t.Errorf("expected 1 tracked client, got %d", rl.GetTrackedClientCount())
	}

	rl.RecordFailure("192.168.1.101")
	if rl.GetTrackedClientCount() != 2 {
		t.Errorf("expected 2 tracked clients, got %d", rl.GetTrackedClientCount())
	}

	rl.RecordSuccess("192.168.1.100")
	if rl.GetTrackedClientCount() != 1 {
		t.Errorf("expected 1 tracked client after success, got %d", rl.GetTrackedClientCount())
	}
}

func TestAttemptTracker_IsLocked(t *testing.T) {
	tracker := &auth.AttemptTracker{
		Count:       4,
		LastFailed:  time.Now(),
		LockedUntil: time.Now().Add(60 * time.Second),
	}

	if !tracker.IsLocked() {
		t.Error("expected tracker to be locked")
	}

	unlockedTracker := &auth.AttemptTracker{
		Count:       2,
		LastFailed:  time.Now(),
		LockedUntil: time.Time{}, // Zero time (not locked)
	}

	if unlockedTracker.IsLocked() {
		t.Error("expected tracker not to be locked")
	}
}

func TestAttemptTracker_TimeUntilUnlock(t *testing.T) {
	tracker := &auth.AttemptTracker{
		Count:       4,
		LastFailed:  time.Now(),
		LockedUntil: time.Now().Add(30 * time.Second),
	}

	duration := tracker.TimeUntilUnlock()
	if duration <= 0 {
		t.Error("expected positive duration until unlock")
	}
	if duration > 30*time.Second {
		t.Errorf("expected duration <= 30s, got %v", duration)
	}

	unlockedTracker := &auth.AttemptTracker{
		Count:       2,
		LastFailed:  time.Now(),
		LockedUntil: time.Time{},
	}

	duration = unlockedTracker.TimeUntilUnlock()
	if duration != 0 {
		t.Errorf("expected 0 duration for unlocked tracker, got %v", duration)
	}
}

func TestGetDelayForAttemptCount(t *testing.T) {
	tests := []struct {
		count    int
		expected time.Duration
	}{
		{0, 0},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 5 * time.Second},
		{4, 60 * time.Second},
		{5, 60 * time.Second},
		{100, 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.count+'0')), func(t *testing.T) {
			delay := auth.GetDelayForAttemptCount(tt.count)
			if delay != tt.expected {
				t.Errorf("count %d: expected delay %v, got %v", tt.count, tt.expected, delay)
			}
		})
	}
}

func TestFormatRetryAfter(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected int
	}{
		{0, 0},
		{1 * time.Second, 1},
		{5 * time.Second, 5},
		{60 * time.Second, 60},
		{1500 * time.Millisecond, 2}, // Rounds up
		{2100 * time.Millisecond, 3}, // Rounds up
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			result := auth.FormatRetryAfter(tt.duration)
			if result != tt.expected {
				t.Errorf("duration %v: expected %d seconds, got %d", tt.duration, tt.expected, result)
			}
		})
	}
}
