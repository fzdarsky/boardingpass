package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/lifecycle"
	"github.com/fzdarsky/boardingpass/internal/logging"
	"github.com/fzdarsky/boardingpass/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteHandler_POST_Success(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	sentinelPath := filepath.Join(tempDir, "issued")

	shutdownCalled := false
	shutdownReason := ""

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewCompleteHandler(sentinelPath, func(reason string) {
		shutdownCalled = true
		shutdownReason = reason
	}, logger)

	req := httptest.NewRequest(http.MethodPost, "/complete", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Validate response
	assert.Equal(t, http.StatusOK, w.Code)

	var response protocol.CompleteResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "shutting_down", response.Status)
	assert.Equal(t, sentinelPath, response.SentinelFile)

	// Verify sentinel file was created
	_, err = os.Stat(sentinelPath)
	assert.NoError(t, err, "sentinel file should exist")

	// Verify shutdown was triggered
	assert.True(t, shutdownCalled, "shutdown function should have been called")
	assert.Equal(t, "provisioning completed", shutdownReason)
}

func TestCompleteHandler_POST_MethodNotAllowed(t *testing.T) {
	tempDir := t.TempDir()
	sentinelPath := filepath.Join(tempDir, "issued")

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewCompleteHandler(sentinelPath, nil, logger)

	// Try GET instead of POST
	req := httptest.NewRequest(http.MethodGet, "/complete", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCompleteHandler_POST_IdempotentSentinelCreation(t *testing.T) {
	tempDir := t.TempDir()
	sentinelPath := filepath.Join(tempDir, "issued")

	logger := logging.New(logging.LevelInfo, logging.FormatJSON)
	handler := handlers.NewCompleteHandler(sentinelPath, nil, logger)

	// First call
	req1 := httptest.NewRequest(http.MethodPost, "/complete", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second call (sentinel file already exists)
	req2 := httptest.NewRequest(http.MethodPost, "/complete", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestSentinelFileIntegration(t *testing.T) {
	tempDir := t.TempDir()
	sentinelPath := filepath.Join(tempDir, "issued")

	sentinel := lifecycle.NewSentinel(sentinelPath)

	// Initially should not exist
	exists, err := sentinel.Exists()
	require.NoError(t, err)
	assert.False(t, exists)

	// Create sentinel file
	err = sentinel.Create()
	require.NoError(t, err)

	// Now should exist
	exists, err = sentinel.Exists()
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify file has content
	content, err := os.ReadFile(sentinelPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "BoardingPass provisioning completed")
}

func TestInactivityTrackerIntegration(t *testing.T) {
	// Skip in short mode due to timeout duration
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	var shutdownCalled atomic.Bool
	shutdownFunc := func() {
		shutdownCalled.Store(true)
	}

	tracker, err := lifecycle.NewInactivityTracker(lifecycle.MinimumInactivityTimeout, shutdownFunc)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	go tracker.Start(ctx)

	// Wait for timeout to trigger
	time.Sleep(lifecycle.MinimumInactivityTimeout + 5*time.Second)

	assert.True(t, shutdownCalled.Load(), "inactivity timeout should trigger shutdown")
}

func TestShutdownManagerIntegration(t *testing.T) {
	sm := lifecycle.NewShutdownManager()
	ctx := context.Background()

	shutdownCtx := sm.Start(ctx)

	// Trigger shutdown manually
	sm.Shutdown("test shutdown")

	// Context should be cancelled
	select {
	case <-shutdownCtx.Done():
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("context was not cancelled within timeout")
	}

	assert.True(t, sm.IsShutdown())
	assert.Equal(t, "test shutdown", sm.Reason())

	sm.Stop()
}
