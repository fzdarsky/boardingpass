package srp_test

import (
	"crypto/sha256"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/fzdarsky/boardingpass/pkg/srp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	assert.NotNil(t, client)
	assert.Equal(t, "testuser", client.Username)
	assert.Equal(t, "testpass", client.Password)
}

func TestClient_GenerateEphemeralKeypair(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Generate ephemeral keypair
	ABase64, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)
	assert.NotEmpty(t, ABase64)

	// Verify A was set internally
	require.NotNil(t, client.A)

	// Verify A is not zero mod N
	modN := new(big.Int).Mod(client.A, srp.N)
	assert.NotEqual(t, big.NewInt(0), modN, "A mod N should not be zero")

	// Verify base64 encoding is valid
	ABytes, err := base64.StdEncoding.DecodeString(ABase64)
	require.NoError(t, err)
	assert.NotEmpty(t, ABytes)
}

func TestClient_GenerateEphemeralKeypair_Uniqueness(t *testing.T) {
	client1 := srp.NewClient("testuser", "testpass")
	client2 := srp.NewClient("testuser", "testpass")

	A1, err := client1.GenerateEphemeralKeypair()
	require.NoError(t, err)

	A2, err := client2.GenerateEphemeralKeypair()
	require.NoError(t, err)

	// Each client should generate different ephemeral values
	assert.NotEqual(t, A1, A2, "different clients should generate different ephemeral values")
}

func TestClient_SetServerResponse(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Generate ephemeral keypair first
	_, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)

	// Create mock server response
	salt := base64.StdEncoding.EncodeToString([]byte("test-salt-12345"))

	// Generate mock server B (must be valid, not 0 mod N)
	b := big.NewInt(12345)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	// Set server response
	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	// Verify salt was set
	assert.NotNil(t, client.Salt)
	assert.NotEmpty(t, client.Salt)

	// Verify B was set
	require.NotNil(t, client.B)
	assert.Equal(t, B, client.B)
}

func TestClient_SetServerResponse_InvalidBase64(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	tests := []struct {
		name      string
		salt      string
		B         string
		wantError bool
		errMsg    string
	}{
		{
			name:      "invalid salt base64",
			salt:      "!!!invalid!!!",
			B:         base64.StdEncoding.EncodeToString([]byte("valid")),
			wantError: true,
			errMsg:    "invalid salt encoding",
		},
		{
			name:      "invalid B base64",
			salt:      base64.StdEncoding.EncodeToString([]byte("valid")),
			B:         "!!!invalid!!!",
			wantError: true,
			errMsg:    "invalid B encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.SetServerResponse(tt.salt, tt.B)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_SetServerResponse_InvalidB(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	salt := base64.StdEncoding.EncodeToString([]byte("test-salt"))

	// Create B that is 0 mod N (invalid)
	BInvalid := new(big.Int).Set(srp.N) // B = N, so B mod N = 0
	BInvalidBase64 := base64.StdEncoding.EncodeToString(BInvalid.Bytes())

	err := client.SetServerResponse(salt, BInvalidBase64)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid B: B mod N == 0")
}

func TestClient_ComputeSharedSecret(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Generate ephemeral keypair
	_, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)

	// Set mock server response
	salt := base64.StdEncoding.EncodeToString([]byte("test-salt-12345"))
	b := big.NewInt(67890)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	// Compute shared secret
	err = client.ComputeSharedSecret()
	require.NoError(t, err)

	// Verify S and K were set
	assert.NotNil(t, client.S, "shared secret S should be set")
	assert.NotNil(t, client.K, "session key K should be set")
	assert.Len(t, client.K, 32, "session key should be 32 bytes (SHA-256)")
}

func TestClient_ComputeSharedSecret_RequiresSetup(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Try to compute shared secret without setup
	err := client.ComputeSharedSecret()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must call GenerateEphemeralKeypair and SetServerResponse first")
}

func TestClient_ComputeClientProof(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Setup client
	_, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)

	salt := base64.StdEncoding.EncodeToString([]byte("test-salt-12345"))
	b := big.NewInt(67890)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	err = client.ComputeSharedSecret()
	require.NoError(t, err)

	// Compute client proof M1
	M1Base64, err := client.ComputeClientProof()
	require.NoError(t, err)
	assert.NotEmpty(t, M1Base64)

	// Verify M1 is valid base64
	M1, err := base64.StdEncoding.DecodeString(M1Base64)
	require.NoError(t, err)
	assert.Len(t, M1, 32, "M1 should be 32 bytes (SHA-256)")

	// Verify M1 was stored internally
	assert.NotNil(t, client.M1)
	assert.Equal(t, M1, client.M1)
}

func TestClient_ComputeClientProof_RequiresSharedSecret(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Try to compute client proof without computing shared secret
	_, err := client.ComputeClientProof()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must call ComputeSharedSecret first")
}

func TestClient_VerifyServerProof(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Setup and compute M1
	_, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)

	salt := base64.StdEncoding.EncodeToString([]byte("test-salt-12345"))
	b := big.NewInt(67890)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	err = client.ComputeSharedSecret()
	require.NoError(t, err)

	_, err = client.ComputeClientProof()
	require.NoError(t, err)

	// Compute expected M2 = H(A | M1 | K)
	hash := sha256.New()
	hash.Write(client.A.Bytes())
	hash.Write(client.M1)
	hash.Write(client.K)
	expectedM2 := hash.Sum(nil)
	expectedM2Base64 := base64.StdEncoding.EncodeToString(expectedM2)

	// Verify server proof
	err = client.VerifyServerProof(expectedM2Base64)
	assert.NoError(t, err)

	// Verify M2 was stored
	assert.NotNil(t, client.M2)
	assert.Equal(t, expectedM2, client.M2)
}

func TestClient_VerifyServerProof_Mismatch(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Setup and compute M1
	_, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)

	salt := base64.StdEncoding.EncodeToString([]byte("test-salt-12345"))
	b := big.NewInt(67890)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	err = client.ComputeSharedSecret()
	require.NoError(t, err)

	_, err = client.ComputeClientProof()
	require.NoError(t, err)

	// Use wrong M2
	wrongM2 := make([]byte, 32)
	wrongM2Base64 := base64.StdEncoding.EncodeToString(wrongM2)

	err = client.VerifyServerProof(wrongM2Base64)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server authentication failed")
}

func TestClient_VerifyServerProof_InvalidBase64(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Setup
	_, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)

	salt := base64.StdEncoding.EncodeToString([]byte("test-salt"))
	b := big.NewInt(67890)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	err = client.ComputeSharedSecret()
	require.NoError(t, err)

	_, err = client.ComputeClientProof()
	require.NoError(t, err)

	// Invalid base64
	err = client.VerifyServerProof("!!!invalid!!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid M2 encoding")
}

func TestClient_ClearSecrets(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Setup full authentication flow
	_, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)

	salt := base64.StdEncoding.EncodeToString([]byte("test-salt-12345"))
	b := big.NewInt(67890)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	err = client.ComputeSharedSecret()
	require.NoError(t, err)

	_, err = client.ComputeClientProof()
	require.NoError(t, err)

	// Clear secrets
	client.ClearSecrets()

	// Verify sensitive data is cleared
	assert.Equal(t, "", client.Password, "password should be cleared")
	assert.Nil(t, client.Salt, "salt should be cleared")
	assert.Nil(t, client.K, "session key should be cleared")
	assert.Nil(t, client.M1, "M1 should be cleared")
	assert.Nil(t, client.M2, "M2 should be cleared")
	assert.Nil(t, client.S, "shared secret should be cleared")
}

func TestClient_GetSessionKey(t *testing.T) {
	client := srp.NewClient("testuser", "testpass")

	// Setup and compute shared secret
	_, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)

	salt := base64.StdEncoding.EncodeToString([]byte("test-salt-12345"))
	b := big.NewInt(67890)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	err = client.ComputeSharedSecret()
	require.NoError(t, err)

	// Get session key
	K := client.GetSessionKey()
	assert.NotNil(t, K)
	assert.Len(t, K, 32)
	assert.Equal(t, client.K, K)
}

func TestComputeClientProofWithEphemeral(t *testing.T) {
	username := "testuser"
	password := "testpass"
	salt := base64.StdEncoding.EncodeToString([]byte("test-salt-12345"))

	// Generate test ephemeral values
	a := big.NewInt(12345)
	A := new(big.Int).Exp(srp.G, a, srp.N)

	b := big.NewInt(67890)
	B := new(big.Int).Exp(srp.G, b, srp.N)

	// Compute client proof
	M1, err := srp.ComputeClientProofWithEphemeral(username, password, salt, a, A, B)
	require.NoError(t, err)
	assert.NotNil(t, M1)
	assert.Len(t, M1, 32, "M1 should be 32 bytes (SHA-256)")
}

func TestComputeClientProofWithEphemeral_InvalidSalt(t *testing.T) {
	a := big.NewInt(12345)
	A := new(big.Int).Exp(srp.G, a, srp.N)
	B := new(big.Int).Exp(srp.G, big.NewInt(67890), srp.N)

	_, err := srp.ComputeClientProofWithEphemeral("user", "pass", "!!!invalid!!!", a, A, B)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid salt encoding")
}

func TestClient_FullAuthenticationFlow(t *testing.T) {
	// This test simulates a full client-side authentication flow

	username := "testuser"
	password := "testpass"

	client := srp.NewClient(username, password)

	// Step 1: Generate ephemeral keypair
	ABase64, err := client.GenerateEphemeralKeypair()
	require.NoError(t, err)
	assert.NotEmpty(t, ABase64)

	// Step 2: Receive server response (mock)
	salt := base64.StdEncoding.EncodeToString([]byte("server-salt-123456"))
	b := big.NewInt(98765)
	B := new(big.Int).Exp(srp.G, b, srp.N)
	BBase64 := base64.StdEncoding.EncodeToString(B.Bytes())

	err = client.SetServerResponse(salt, BBase64)
	require.NoError(t, err)

	// Step 3: Compute shared secret
	err = client.ComputeSharedSecret()
	require.NoError(t, err)
	assert.NotNil(t, client.K)

	// Step 4: Compute client proof M1
	M1Base64, err := client.ComputeClientProof()
	require.NoError(t, err)
	assert.NotEmpty(t, M1Base64)

	// Step 5: Verify server proof M2 (mock)
	hash := sha256.New()
	hash.Write(client.A.Bytes())
	hash.Write(client.M1)
	hash.Write(client.K)
	M2 := hash.Sum(nil)
	M2Base64 := base64.StdEncoding.EncodeToString(M2)

	err = client.VerifyServerProof(M2Base64)
	require.NoError(t, err)

	// Step 6: Clear secrets after authentication
	sessionKey := client.GetSessionKey()
	assert.Len(t, sessionKey, 32)

	client.ClearSecrets()
	assert.Equal(t, "", client.Password)
}
