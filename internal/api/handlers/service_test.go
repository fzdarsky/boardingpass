package handlers_test

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fzdarsky/boardingpass/internal/api/handlers"
	"github.com/fzdarsky/boardingpass/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSessionManager(t *testing.T) *auth.SessionManager {
	t.Helper()
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)
	return auth.NewSessionManager(secret, 30*time.Minute)
}

func TestServiceHandler_Unauthenticated(t *testing.T) {
	sm := newTestSessionManager(t)
	h := handlers.NewServiceHandler(sm)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "boardingpass", resp["service"])
	assert.Empty(t, resp["version"], "version should not be present for unauthenticated requests")
}

func TestServiceHandler_Authenticated(t *testing.T) {
	sm := newTestSessionManager(t)
	token, err := sm.CreateSession("testuser")
	require.NoError(t, err)

	h := handlers.NewServiceHandler(sm)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "boardingpass", resp["service"])
	assert.NotEmpty(t, resp["version"], "version should be present for authenticated requests")
}

func TestServiceHandler_InvalidToken(t *testing.T) {
	sm := newTestSessionManager(t)
	h := handlers.NewServiceHandler(sm)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "boardingpass", resp["service"])
	assert.Empty(t, resp["version"], "version should not be present for invalid tokens")
}

func TestServiceHandler_MethodNotAllowed(t *testing.T) {
	sm := newTestSessionManager(t)
	h := handlers.NewServiceHandler(sm)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
