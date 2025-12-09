package lifecycle

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"
)

func TestNewShutdownManager(t *testing.T) {
	sm := NewShutdownManager()
	if sm == nil {
		t.Fatal("expected non-nil shutdown manager")
	}
	if sm.IsShutdown() {
		t.Error("expected shutdown to be false initially")
	}
}

func TestShutdownManager_Shutdown(t *testing.T) {
	sm := NewShutdownManager()

	if sm.IsShutdown() {
		t.Error("expected shutdown to be false initially")
	}

	sm.Shutdown("test shutdown")

	if !sm.IsShutdown() {
		t.Error("expected shutdown to be true after calling Shutdown()")
	}

	if sm.Reason() != "test shutdown" {
		t.Errorf("expected reason 'test shutdown', got '%s'", sm.Reason())
	}
}

func TestShutdownManager_Start(t *testing.T) {
	t.Run("cancels context on Shutdown()", func(t *testing.T) {
		sm := NewShutdownManager()
		ctx := context.Background()

		shutdownCtx := sm.Start(ctx)

		// Trigger shutdown
		sm.Shutdown("manual shutdown")

		// Wait for context to be cancelled
		select {
		case <-shutdownCtx.Done():
			// Context was cancelled as expected
		case <-time.After(1 * time.Second):
			t.Error("context was not cancelled within timeout")
		}

		sm.Stop()
	})

	t.Run("cancels context on signal", func(t *testing.T) {
		sm := NewShutdownManager()
		ctx := context.Background()

		shutdownCtx := sm.Start(ctx)

		// Send signal
		sm.signalChan <- syscall.SIGTERM

		// Wait for context to be cancelled
		select {
		case <-shutdownCtx.Done():
			// Context was cancelled as expected
		case <-time.After(1 * time.Second):
			t.Error("context was not cancelled within timeout")
		}

		if !sm.IsShutdown() {
			t.Error("expected shutdown to be true after signal")
		}

		reason := sm.Reason()
		if reason == "" {
			t.Error("expected reason to be set after signal")
		}

		sm.Stop()
	})

	t.Run("cancels context when parent context is cancelled", func(t *testing.T) {
		sm := NewShutdownManager()
		ctx, cancel := context.WithCancel(context.Background())

		shutdownCtx := sm.Start(ctx)

		// Cancel parent context
		cancel()

		// Wait for shutdown context to be cancelled
		select {
		case <-shutdownCtx.Done():
			// Context was cancelled as expected
		case <-time.After(1 * time.Second):
			t.Error("context was not cancelled within timeout")
		}

		sm.Stop()
	})
}

func TestShutdownManager_IdempotentShutdown(t *testing.T) {
	sm := NewShutdownManager()

	sm.Shutdown("first shutdown")
	firstReason := sm.Reason()

	// Try to shutdown again with different reason
	sm.Shutdown("second shutdown")
	secondReason := sm.Reason()

	if firstReason != secondReason {
		t.Errorf("expected reason to remain '%s', got '%s'", firstReason, secondReason)
	}
}

func TestGracefulShutdown(t *testing.T) {
	t.Run("successful shutdown", func(t *testing.T) {
		shutdownFunc := func(ctx context.Context) error {
			// Simulate some cleanup work
			time.Sleep(50 * time.Millisecond)
			return nil
		}

		ctx := context.Background()
		err := GracefulShutdown(ctx, shutdownFunc, 1*time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("shutdown with error", func(t *testing.T) {
		expectedErr := errors.New("shutdown error")

		shutdownFunc := func(ctx context.Context) error {
			return expectedErr
		}

		ctx := context.Background()
		err := GracefulShutdown(ctx, shutdownFunc, 1*time.Second)

		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("shutdown timeout", func(t *testing.T) {
		shutdownFunc := func(ctx context.Context) error {
			// Simulate slow shutdown that exceeds timeout
			time.Sleep(500 * time.Millisecond)
			return nil
		}

		ctx := context.Background()
		err := GracefulShutdown(ctx, shutdownFunc, 100*time.Millisecond)

		if err == nil {
			t.Error("expected timeout error")
		}

		if err != nil && err.Error() != "shutdown timed out after 100ms" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		shutdownFunc := func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := GracefulShutdown(ctx, shutdownFunc, 1*time.Second)

		if err == nil {
			t.Error("expected context cancellation error")
		}
	})
}

func TestShutdownManager_Stop(t *testing.T) {
	sm := NewShutdownManager()
	ctx := context.Background()

	_ = sm.Start(ctx)

	// Stop should not panic
	sm.Stop()

	// Calling Stop again should not panic
	sm.Stop()
}
