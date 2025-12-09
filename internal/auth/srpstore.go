package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// srpSession holds an SRP server instance with expiry time.
type srpSession struct {
	server    *SRPServer
	expiresAt time.Time
}

// SRPStore stores SRP server instances between init and verify steps.
// It provides thread-safe storage with automatic cleanup of expired sessions.
type SRPStore struct {
	sessions map[string]*srpSession
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewSRPStore creates a new SRP server instance store with the given TTL.
// Sessions older than TTL will be automatically cleaned up.
func NewSRPStore(ttl time.Duration) *SRPStore {
	store := &SRPStore{
		sessions: make(map[string]*srpSession),
		ttl:      ttl,
	}

	// Start background cleanup goroutine
	go store.cleanupLoop()

	return store
}

// Store saves an SRP server instance and returns a session ID.
// The session ID should be returned to the client for use in the verify step.
func (s *SRPStore) Store(server *SRPServer) (string, error) {
	// Generate random session ID (16 bytes = 128 bits)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	sessionID := base64.URLEncoding.EncodeToString(idBytes)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = &srpSession{
		server:    server,
		expiresAt: time.Now().Add(s.ttl),
	}

	return sessionID, nil
}

// Retrieve fetches an SRP server instance by session ID.
// Returns nil if the session doesn't exist or has expired.
// The session is removed after retrieval (one-time use).
func (s *SRPStore) Retrieve(sessionID string) *SRPServer {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(session.expiresAt) {
		delete(s.sessions, sessionID)
		return nil
	}

	// Remove session after retrieval (one-time use)
	delete(s.sessions, sessionID)

	return session.server
}

// cleanupLoop periodically removes expired sessions.
func (s *SRPStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

// cleanup removes all expired sessions.
func (s *SRPStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.expiresAt) {
			delete(s.sessions, id)
		}
	}
}

// Count returns the number of active sessions (for testing/monitoring).
func (s *SRPStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}
