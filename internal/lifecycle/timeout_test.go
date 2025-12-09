package lifecycle

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewInactivityTracker(t *testing.T) {
	t.Run("with default timeout", func(t *testing.T) {
		tracker, err := NewInactivityTracker(0, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tracker.timeout != DefaultInactivityTimeout {
			t.Errorf("expected timeout %v, got %v", DefaultInactivityTimeout, tracker.timeout)
		}
	})

	t.Run("with custom timeout", func(t *testing.T) {
		customTimeout := 5 * time.Minute
		tracker, err := NewInactivityTracker(customTimeout, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tracker.timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, tracker.timeout)
		}
	})

	t.Run("rejects timeout below minimum", func(t *testing.T) {
		_, err := NewInactivityTracker(30*time.Second, nil)
		if err == nil {
			t.Error("expected error for timeout below minimum")
		}
	})

	t.Run("accepts minimum timeout", func(t *testing.T) {
		_, err := NewInactivityTracker(MinimumInactivityTimeout, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestInactivityTracker_RecordActivity(t *testing.T) {
	tracker, err := NewInactivityTracker(2*time.Minute, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	initialActivity := tracker.LastActivity()
	time.Sleep(10 * time.Millisecond)

	tracker.RecordActivity()
	newActivity := tracker.LastActivity()

	if !newActivity.After(initialActivity) {
		t.Error("expected activity time to be updated")
	}
}

func TestInactivityTracker_TimeUntilTimeout(t *testing.T) {
	timeout := 2 * time.Minute
	tracker, err := NewInactivityTracker(timeout, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Initial time until timeout should be close to the full timeout
	timeUntil := tracker.TimeUntilTimeout()
	if timeUntil < timeout-time.Second || timeUntil > timeout {
		t.Errorf("expected time until timeout ~%v, got %v", timeout, timeUntil)
	}

	// After recording activity, time should reset
	time.Sleep(100 * time.Millisecond)
	tracker.RecordActivity()

	newTimeUntil := tracker.TimeUntilTimeout()
	if newTimeUntil <= timeUntil-100*time.Millisecond {
		t.Error("expected time until timeout to reset after activity")
	}
}

func TestInactivityTracker_Start(t *testing.T) {
	t.Run("triggers shutdown on timeout", func(t *testing.T) {
		var shutdownCalled atomic.Bool

		shutdownFunc := func() {
			shutdownCalled.Store(true)
		}

		// Use short timeout for faster testing (500ms)
		tracker := newInactivityTrackerInternal(500*time.Millisecond, shutdownFunc)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go tracker.Start(ctx)

		// Wait for shutdown to be called
		time.Sleep(700 * time.Millisecond)

		if !shutdownCalled.Load() {
			t.Error("expected shutdown to be called after timeout")
		}
	})

	t.Run("activity prevents timeout", func(t *testing.T) {
		var shutdownCalled atomic.Bool

		shutdownFunc := func() {
			shutdownCalled.Store(true)
		}

		// Use short timeout for faster testing (500ms)
		tracker := newInactivityTrackerInternal(500*time.Millisecond, shutdownFunc)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go tracker.Start(ctx)

		// Record activity periodically to prevent timeout
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		done := make(chan bool)
		go func() {
			for i := 0; i < 4; i++ {
				<-ticker.C
				tracker.RecordActivity()
			}
			done <- true
		}()

		<-done

		if shutdownCalled.Load() {
			t.Error("shutdown should not be called when activity is ongoing")
		}

		tracker.Stop()
	})

	t.Run("stops when context is cancelled", func(t *testing.T) {
		var shutdownCalled atomic.Bool

		shutdownFunc := func() {
			shutdownCalled.Store(true)
		}

		tracker, err := NewInactivityTracker(10*time.Minute, shutdownFunc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		go tracker.Start(ctx)

		// Cancel context quickly
		time.Sleep(50 * time.Millisecond)
		cancel()

		// Wait a bit to ensure Start() has exited
		time.Sleep(100 * time.Millisecond)

		if shutdownCalled.Load() {
			t.Error("shutdown should not be called when context is cancelled")
		}
	})
}

func TestInactivityTracker_Stop(t *testing.T) {
	var shutdownCalled atomic.Bool

	shutdownFunc := func() {
		shutdownCalled.Store(true)
	}

	// Use short timeout for faster testing (500ms)
	tracker := newInactivityTrackerInternal(500*time.Millisecond, shutdownFunc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go tracker.Start(ctx)

	// Stop tracker before timeout
	time.Sleep(200 * time.Millisecond)
	tracker.Stop()

	// Wait past the original timeout
	time.Sleep(700 * time.Millisecond)

	if shutdownCalled.Load() {
		t.Error("shutdown should not be called after Stop()")
	}

	// Recording activity after stop should not panic
	tracker.RecordActivity()
}
