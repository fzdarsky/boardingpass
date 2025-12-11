package integration_test

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/cli/client"
	"github.com/fzdarsky/boardingpass/internal/cli/session"
	"github.com/fzdarsky/boardingpass/pkg/srp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSRPAuthenticationFlow tests the full SRP authentication protocol flow.
// This validates that the client and server-side SRP implementations can complete
// a successful mutual authentication.
func TestSRPAuthenticationFlow(t *testing.T) {
	username := "testuser"
	password := "testpassword"

	// Client side: Create SRP client
	clientSRP := client.NewSRPClient(username, password)
	defer clientSRP.ClearSecrets()

	// Phase 1: Client generates ephemeral keypair
	ABase64, err := clientSRP.GenerateEphemeralKeypair()
	require.NoError(t, err)
	assert.NotEmpty(t, ABase64)

	// Server side: Generate salt and verifier (one-time setup)
	salt := base64.StdEncoding.EncodeToString([]byte("server-salt-123456"))
	saltBytes, _ := base64.StdEncoding.DecodeString(salt)

	// Compute verifier: v = g^x mod N, where x = H(salt | H(username:password))
	innerHash := sha256.Sum256([]byte(username + ":" + password))
	outerHash := sha256.New()
	outerHash.Write(saltBytes)
	outerHash.Write(innerHash[:])
	x := new(big.Int).SetBytes(outerHash.Sum(nil))
	verifier := new(big.Int).Exp(srp.G, x, srp.N)

	// Server side: Generate ephemeral b and compute B = k*v + g^b
	b := big.NewInt(87654)
	gExpB := new(big.Int).Exp(srp.G, b, srp.N)
	kv := new(big.Int).Mul(srp.K, verifier)
	B := new(big.Int).Add(kv, gExpB)
	B.Mod(B, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	// Phase 2: Client receives salt and B from server
	err = clientSRP.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	// Client computes shared secret
	err = clientSRP.ComputeSharedSecret()
	require.NoError(t, err)
	assert.NotNil(t, clientSRP.K, "session key should be computed")

	// Phase 3: Client computes proof M1
	M1Base64, err := clientSRP.ComputeClientProof()
	require.NoError(t, err)
	assert.NotEmpty(t, M1Base64)

	// Server side: Verify client proof M1
	// Decode client's A
	ABytes, err := base64.StdEncoding.DecodeString(ABase64)
	require.NoError(t, err)
	A := new(big.Int).SetBytes(ABytes)

	// Server computes u = H(A | B)
	uHash := sha256.New()
	maxLen := len(srp.N.Bytes())
	ABytesPadded := make([]byte, maxLen)
	BBytesPadded := make([]byte, maxLen)
	copy(ABytesPadded[maxLen-len(ABytes):], ABytes)
	copy(BBytesPadded[maxLen-len(B.Bytes()):], B.Bytes())
	uHash.Write(ABytesPadded)
	uHash.Write(BBytesPadded)
	u := new(big.Int).SetBytes(uHash.Sum(nil))

	// Server computes S = (A * v^u)^b mod N
	vu := new(big.Int).Exp(verifier, u, srp.N)
	Avu := new(big.Int).Mul(A, vu)
	Avu.Mod(Avu, srp.N)
	S := new(big.Int).Exp(Avu, b, srp.N)

	// Server computes session key K = H(S)
	serverKHash := sha256.New()
	serverKHash.Write(S.Bytes())
	serverK := serverKHash.Sum(nil)

	// Server computes expected M1
	hashN := sha256.Sum256(srp.N.Bytes())
	hashG := sha256.Sum256(srp.G.Bytes())
	hashNXorG := make([]byte, len(hashN))
	for i := range hashN {
		hashNXorG[i] = hashN[i] ^ hashG[i]
	}
	hashUsername := sha256.Sum256([]byte(username))

	serverM1Hash := sha256.New()
	serverM1Hash.Write(hashNXorG)
	serverM1Hash.Write(hashUsername[:])
	serverM1Hash.Write(saltBytes)
	serverM1Hash.Write(ABytes)
	serverM1Hash.Write(B.Bytes())
	serverM1Hash.Write(serverK)
	expectedM1 := serverM1Hash.Sum(nil)
	expectedM1Base64 := base64.StdEncoding.EncodeToString(expectedM1)

	// Verify M1 matches
	assert.Equal(t, expectedM1Base64, M1Base64, "client proof M1 should match server's computation")

	// Phase 4: Server computes proof M2 = H(A | M1 | K)
	serverM2Hash := sha256.New()
	serverM2Hash.Write(ABytes)
	clientM1, _ := base64.StdEncoding.DecodeString(M1Base64)
	serverM2Hash.Write(clientM1)
	serverM2Hash.Write(serverK)
	M2 := serverM2Hash.Sum(nil)
	M2Base64 := base64.StdEncoding.EncodeToString(M2)

	// Client verifies server proof M2
	err = clientSRP.VerifyServerProof(M2Base64)
	require.NoError(t, err)

	// Verify both sides have the same session key
	assert.Equal(t, serverK, clientSRP.K, "client and server should derive the same session key")

	// Clear secrets
	clientSRP.ClearSecrets()
	assert.Equal(t, "", clientSRP.Password, "password should be cleared")
	assert.Nil(t, clientSRP.K, "session key should be cleared")
}

// TestSRPAuthenticationFlow_WrongPassword tests that authentication fails with wrong password.
func TestSRPAuthenticationFlow_WrongPassword(t *testing.T) {
	correctPassword := "correctpassword"
	wrongPassword := "wrongpassword"
	username := "testuser"

	// Server setup with correct password
	salt := base64.StdEncoding.EncodeToString([]byte("server-salt-123456"))
	saltBytes, _ := base64.StdEncoding.DecodeString(salt)

	innerHash := sha256.Sum256([]byte(username + ":" + correctPassword))
	outerHash := sha256.New()
	outerHash.Write(saltBytes)
	outerHash.Write(innerHash[:])
	x := new(big.Int).SetBytes(outerHash.Sum(nil))
	verifier := new(big.Int).Exp(srp.G, x, srp.N)

	b := big.NewInt(87654)
	gExpB := new(big.Int).Exp(srp.G, b, srp.N)
	kv := new(big.Int).Mul(srp.K, verifier)
	B := new(big.Int).Add(kv, gExpB)
	B.Mod(B, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	// Client tries with wrong password
	clientSRP := client.NewSRPClient(username, wrongPassword)
	defer clientSRP.ClearSecrets()

	ABase64, err := clientSRP.GenerateEphemeralKeypair()
	require.NoError(t, err)

	err = clientSRP.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	err = clientSRP.ComputeSharedSecret()
	require.NoError(t, err)

	M1Base64, err := clientSRP.ComputeClientProof()
	require.NoError(t, err)

	// Server verifies M1 (should fail)
	ABytes, _ := base64.StdEncoding.DecodeString(ABase64)
	A := new(big.Int).SetBytes(ABytes)

	uHash := sha256.New()
	maxLen := len(srp.N.Bytes())
	ABytesPadded := make([]byte, maxLen)
	BBytesPadded := make([]byte, maxLen)
	copy(ABytesPadded[maxLen-len(ABytes):], ABytes)
	copy(BBytesPadded[maxLen-len(B.Bytes()):], B.Bytes())
	uHash.Write(ABytesPadded)
	uHash.Write(BBytesPadded)
	u := new(big.Int).SetBytes(uHash.Sum(nil))

	vu := new(big.Int).Exp(verifier, u, srp.N)
	Avu := new(big.Int).Mul(A, vu)
	Avu.Mod(Avu, srp.N)
	S := new(big.Int).Exp(Avu, b, srp.N)

	serverKHash := sha256.New()
	serverKHash.Write(S.Bytes())
	serverK := serverKHash.Sum(nil)

	// Compute expected M1 with server's session key
	hashN := sha256.Sum256(srp.N.Bytes())
	hashG := sha256.Sum256(srp.G.Bytes())
	hashNXorG := make([]byte, len(hashN))
	for i := range hashN {
		hashNXorG[i] = hashN[i] ^ hashG[i]
	}
	hashUsername := sha256.Sum256([]byte(username))

	serverM1Hash := sha256.New()
	serverM1Hash.Write(hashNXorG)
	serverM1Hash.Write(hashUsername[:])
	serverM1Hash.Write(saltBytes)
	serverM1Hash.Write(ABytes)
	serverM1Hash.Write(B.Bytes())
	serverM1Hash.Write(serverK)
	expectedM1 := serverM1Hash.Sum(nil)
	expectedM1Base64 := base64.StdEncoding.EncodeToString(expectedM1)

	// M1 should NOT match (wrong password leads to different session key)
	assert.NotEqual(t, expectedM1Base64, M1Base64, "M1 should not match with wrong password")
}

// TestSessionTokenPersistence tests that session tokens can be saved and loaded.
func TestSessionTokenPersistence(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	host := "test.boardingpass.local"
	port := 8443
	expectedToken := "test-session-token-123456789"

	// Create session store
	store, err := session.NewStore()
	require.NoError(t, err)

	// Save session token
	err = store.Save(host, port, expectedToken)
	require.NoError(t, err)

	// Verify token file exists with correct permissions
	tokenFile := getTokenFilename(t, store, host, port)
	info, err := os.Stat(tokenFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "session token file should have 0600 permissions")

	// Load session token
	loadedToken, err := store.Load(host, port)
	require.NoError(t, err)
	assert.Equal(t, expectedToken, loadedToken)

	// Delete session token
	err = store.Delete(host, port)
	require.NoError(t, err)

	// Verify token is deleted
	loadedToken, err = store.Load(host, port)
	require.NoError(t, err)
	assert.Equal(t, "", loadedToken, "token should be empty after deletion")
}

// TestMultipleServerSessions tests that multiple server sessions can coexist.
func TestMultipleServerSessions(t *testing.T) {
	setupTestEnv(t)

	store, err := session.NewStore()
	require.NoError(t, err)

	// Save tokens for different servers
	servers := []struct {
		host  string
		port  int
		token string
	}{
		{"server1.local", 8443, "token-for-server1"},
		{"server2.local", 8443, "token-for-server2"},
		{"server1.local", 9443, "token-for-server1-port9443"},
	}

	for _, srv := range servers {
		err := store.Save(srv.host, srv.port, srv.token)
		require.NoError(t, err)
	}

	// Verify each token can be loaded independently
	for _, srv := range servers {
		loaded, err := store.Load(srv.host, srv.port)
		require.NoError(t, err)
		assert.Equal(t, srv.token, loaded)
	}

	// Delete one token
	err = store.Delete("server1.local", 8443)
	require.NoError(t, err)

	// Verify only that token is deleted
	loaded, err := store.Load("server1.local", 8443)
	require.NoError(t, err)
	assert.Equal(t, "", loaded)

	// Other tokens should still exist
	loaded, err = store.Load("server2.local", 8443)
	require.NoError(t, err)
	assert.Equal(t, "token-for-server2", loaded)
}

// Helper functions

func setupTestEnv(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
}

func getTokenFilename(t *testing.T, store *session.Store, host string, port int) string {
	t.Helper()

	// Compute expected filename
	identifier := fmt.Sprintf("%s:%d", host, port)
	hash := sha256.Sum256([]byte(identifier))
	hashStr := fmt.Sprintf("%x", hash[:8])
	filename := fmt.Sprintf("session-%s.token", hashStr)

	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = os.Getenv("HOME")
	}
	cacheDir = filepath.Join(cacheDir, "boardingpass")

	return filepath.Join(cacheDir, filename)
}
