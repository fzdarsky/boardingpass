package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrSessionNotFound is returned when a session token is not found in storage
	ErrSessionNotFound = errors.New("session token not found")

	// ErrSessionExpired is returned when a session token has expired
	ErrSessionExpired = errors.New("session token expired")

	// ErrSessionLimitExceeded is returned when maximum concurrent sessions limit is reached
	ErrSessionLimitExceeded = errors.New("session limit exceeded")
)

const (
	// DefaultSessionTTL is the default session token lifetime (30 minutes)
	DefaultSessionTTL = 30 * time.Minute

	// MinSessionTTL is the minimum allowed session TTL (5 minutes)
	MinSessionTTL = 5 * time.Minute

	// MaxConcurrentSessions is the maximum number of concurrent sessions allowed
	// This prevents session exhaustion attacks
	MaxConcurrentSessions = 10

	// TokenIDBytes is the number of random bytes in the token ID (32 bytes = 256 bits)
	TokenIDBytes = 32

	// CleanupInterval is how often expired sessions are cleaned up
	CleanupInterval = 1 * time.Minute
)

// Session represents an authenticated session with metadata.
type Session struct {
	Token     string    // Full token string (token_id.signature)
	Username  string    // Associated username
	CreatedAt time.Time // Session creation timestamp
	ExpiresAt time.Time // Session expiration timestamp
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// TimeUntilExpiry returns the duration until the session expires.
// Returns 0 if already expired.
func (s *Session) TimeUntilExpiry() time.Duration {
	if s.IsExpired() {
		return 0
	}
	return time.Until(s.ExpiresAt)
}

// SessionManager manages session tokens with in-memory storage and automatic cleanup.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session // key: token string
	secret   []byte              // HMAC secret key
	ttl      time.Duration       // Session time-to-live
	stopCh   chan struct{}       // Channel to stop cleanup goroutine
}

// NewSessionManager creates a new session manager with the given HMAC secret and TTL.
// The secret should be a cryptographically random value (recommended: 32 bytes).
// If ttl is less than MinSessionTTL, it will be set to MinSessionTTL.
func NewSessionManager(secret []byte, ttl time.Duration) *SessionManager {
	if ttl < MinSessionTTL {
		ttl = MinSessionTTL
	}

	sm := &SessionManager{
		sessions: make(map[string]*Session),
		secret:   secret,
		ttl:      ttl,
		stopCh:   make(chan struct{}),
	}

	// Start background cleanup goroutine
	go sm.cleanupExpiredSessions()

	return sm
}

// CreateSession creates a new session for the given username.
// Returns the session token string (format: token_id.signature) or an error.
func (sm *SessionManager) CreateSession(username string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check concurrent session limit
	if len(sm.sessions) >= MaxConcurrentSessions {
		return "", ErrSessionLimitExceeded
	}

	// Generate high-entropy token ID (32 bytes = 256 bits)
	tokenIDBytes := make([]byte, TokenIDBytes)
	if _, err := rand.Read(tokenIDBytes); err != nil {
		return "", fmt.Errorf("failed to generate token ID: %w", err)
	}
	tokenID := base64.URLEncoding.EncodeToString(tokenIDBytes)

	// Compute HMAC signature: HMAC-SHA256(token_id + username, secret)
	signature := sm.computeSignature(tokenID, username)

	// Construct token: token_id.signature
	token := tokenID + "." + signature

	// Create session metadata
	now := time.Now()
	session := &Session{
		Token:     token,
		Username:  username,
		CreatedAt: now,
		ExpiresAt: now.Add(sm.ttl),
	}

	// Store session
	sm.sessions[token] = session

	return token, nil
}

// ValidateSession validates a session token and returns the associated session.
// Returns ErrSessionNotFound if the token is not found.
// Returns ErrSessionExpired if the token has expired.
func (sm *SessionManager) ValidateSession(token string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Check if session exists
	session, exists := sm.sessions[token]
	if !exists {
		return nil, ErrSessionNotFound
	}

	// Check if session has expired
	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	// Verify HMAC signature to prevent tampering
	if !sm.verifySignature(token, session.Username) {
		return nil, errors.New("invalid session token signature")
	}

	return session, nil
}

// InvalidateSession removes a session from storage.
// This is useful for explicit logout or cleanup.
func (sm *SessionManager) InvalidateSession(token string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[token]; !exists {
		return ErrSessionNotFound
	}

	delete(sm.sessions, token)
	return nil
}

// GetSessionCount returns the current number of active sessions.
func (sm *SessionManager) GetSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.sessions)
}

// Stop stops the background cleanup goroutine.
// Should be called when shutting down the service.
func (sm *SessionManager) Stop() {
	close(sm.stopCh)
}

// computeSignature computes HMAC-SHA256 signature for a token.
// Signature = HMAC-SHA256(token_id + username, secret)
func (sm *SessionManager) computeSignature(tokenID, username string) string {
	h := hmac.New(sha256.New, sm.secret)
	h.Write([]byte(tokenID))
	h.Write([]byte(username))
	signature := h.Sum(nil)
	return base64.URLEncoding.EncodeToString(signature)
}

// verifySignature verifies the HMAC signature of a token.
func (sm *SessionManager) verifySignature(token, username string) bool {
	// Parse token into token_id and signature
	// Format: token_id.signature
	var tokenID, providedSignature string
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			tokenID = token[:i]
			providedSignature = token[i+1:]
			break
		}
	}

	if tokenID == "" || providedSignature == "" {
		return false
	}

	// Compute expected signature
	expectedSignature := sm.computeSignature(tokenID, username)

	// Constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(providedSignature), []byte(expectedSignature))
}

// cleanupExpiredSessions is a background goroutine that periodically removes expired sessions.
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.performCleanup()
		case <-sm.stopCh:
			return
		}
	}
}

// performCleanup removes all expired sessions from storage.
func (sm *SessionManager) performCleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for token, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, token)
		}
	}
}

// GenerateSessionSecret generates a cryptographically secure random secret
// for HMAC signing. Returns 32 bytes (256 bits) of random data.
// This should be called once at service startup.
func GenerateSessionSecret() ([]byte, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate session secret: %w", err)
	}
	return secret, nil
}
