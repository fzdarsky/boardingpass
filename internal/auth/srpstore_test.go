package auth

import (
	"testing"
	"time"
)

func TestSRPStore_StoreAndRetrieve(t *testing.T) {
	store := NewSRPStore(5 * time.Minute)

	// Create a dummy SRP server
	server := &SRPServer{
		Username: "testuser",
	}

	// Store the server
	sessionID, err := store.Store(server)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	if sessionID == "" {
		t.Fatal("Store() returned empty session ID")
	}

	// Verify session count
	if count := store.Count(); count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}

	// Retrieve the server
	retrieved := store.Retrieve(sessionID)
	if retrieved == nil {
		t.Fatal("Retrieve() returned nil")
	}

	if retrieved.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", retrieved.Username)
	}

	// Verify session was deleted after retrieval (one-time use)
	if count := store.Count(); count != 0 {
		t.Errorf("Expected 0 sessions after retrieval, got %d", count)
	}

	// Verify second retrieval returns nil
	retrieved2 := store.Retrieve(sessionID)
	if retrieved2 != nil {
		t.Error("Second Retrieve() should return nil (one-time use)")
	}
}

func TestSRPStore_RetrieveInvalidSession(t *testing.T) {
	store := NewSRPStore(5 * time.Minute)

	// Try to retrieve non-existent session
	retrieved := store.Retrieve("invalid-session-id")
	if retrieved != nil {
		t.Error("Retrieve() should return nil for invalid session ID")
	}
}

func TestSRPStore_SessionExpiration(t *testing.T) {
	// Create store with very short TTL
	store := NewSRPStore(100 * time.Millisecond)

	server := &SRPServer{
		Username: "testuser",
	}

	sessionID, err := store.Store(server)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// Wait for session to expire
	time.Sleep(150 * time.Millisecond)

	// Try to retrieve expired session
	retrieved := store.Retrieve(sessionID)
	if retrieved != nil {
		t.Error("Retrieve() should return nil for expired session")
	}

	// Verify session was cleaned up
	if count := store.Count(); count != 0 {
		t.Errorf("Expected 0 sessions after expiration, got %d", count)
	}
}

func TestSRPStore_MultipleSessionsIsolation(t *testing.T) {
	store := NewSRPStore(5 * time.Minute)

	// Store multiple sessions
	server1 := &SRPServer{Username: "user1"}
	server2 := &SRPServer{Username: "user2"}
	server3 := &SRPServer{Username: "user3"}

	id1, _ := store.Store(server1)
	id2, _ := store.Store(server2)
	id3, _ := store.Store(server3)

	// Verify all three are stored
	if count := store.Count(); count != 3 {
		t.Errorf("Expected 3 sessions, got %d", count)
	}

	// Verify each session is isolated
	r1 := store.Retrieve(id1)
	if r1 == nil || r1.Username != "user1" {
		t.Error("Failed to retrieve session 1 correctly")
	}

	r2 := store.Retrieve(id2)
	if r2 == nil || r2.Username != "user2" {
		t.Error("Failed to retrieve session 2 correctly")
	}

	r3 := store.Retrieve(id3)
	if r3 == nil || r3.Username != "user3" {
		t.Error("Failed to retrieve session 3 correctly")
	}

	// Verify all sessions deleted after retrieval
	if count := store.Count(); count != 0 {
		t.Errorf("Expected 0 sessions after all retrievals, got %d", count)
	}
}

func TestSRPStore_AutomaticCleanup(t *testing.T) {
	// Create store with short TTL and fast cleanup
	store := NewSRPStore(50 * time.Millisecond)

	// Store multiple sessions
	for i := 0; i < 10; i++ {
		server := &SRPServer{Username: "testuser"}
		_, err := store.Store(server)
		if err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Verify all stored
	if count := store.Count(); count != 10 {
		t.Errorf("Expected 10 sessions, got %d", count)
	}

	// Wait for sessions to expire and cleanup to run
	// Cleanup runs every minute, but expiry is 50ms
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup by trying to retrieve
	// (or wait for the background cleanup, but that takes 1 minute)
	store.cleanup()

	// Verify sessions were cleaned up
	if count := store.Count(); count != 0 {
		t.Errorf("Expected 0 sessions after cleanup, got %d", count)
	}
}

func TestSRPStore_SessionIDUniqueness(t *testing.T) {
	store := NewSRPStore(5 * time.Minute)

	// Store multiple sessions and verify IDs are unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		server := &SRPServer{Username: "testuser"}
		id, err := store.Store(server)
		if err != nil {
			t.Fatalf("Store() failed: %v", err)
		}

		if ids[id] {
			t.Errorf("Duplicate session ID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != 100 {
		t.Errorf("Expected 100 unique session IDs, got %d", len(ids))
	}
}

func TestSRPStore_ConcurrentAccess(t *testing.T) {
	store := NewSRPStore(5 * time.Minute)

	// Test concurrent stores
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			server := &SRPServer{Username: "testuser"}
			_, err := store.Store(server)
			if err != nil {
				t.Errorf("Concurrent Store() failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if count := store.Count(); count != 10 {
		t.Errorf("Expected 10 sessions after concurrent stores, got %d", count)
	}
}
