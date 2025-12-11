package auth_test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"

	"github.com/fzdarsky/boardingpass/internal/auth"
	"github.com/fzdarsky/boardingpass/pkg/srp"
)

func TestGetGroupParameters(t *testing.T) {
	N, g, k := auth.GetGroupParameters()

	if N == nil {
		t.Error("expected non-nil N")
	}
	if g == nil {
		t.Error("expected non-nil g")
	}
	if k == nil {
		t.Error("expected non-nil k")
	}

	// Verify g = 2
	if g.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("expected g = 2, got %v", g)
	}

	// Verify N is a large prime (basic check: > 0 and odd)
	if N.Cmp(big.NewInt(0)) <= 0 {
		t.Error("expected N > 0")
	}
	if N.Bit(0) != 1 {
		t.Error("expected N to be odd")
	}

	// Verify k = H(N | g)
	hash := sha256.New()
	nBytes := N.Bytes()
	gBytes := make([]byte, len(nBytes))
	copy(gBytes[len(gBytes)-1:], g.Bytes())
	hash.Write(nBytes)
	hash.Write(gBytes)
	expectedK := new(big.Int).SetBytes(hash.Sum(nil))

	if k.Cmp(expectedK) != 0 {
		t.Error("k does not match H(N | g)")
	}
}

func TestNewSRPServer(t *testing.T) {
	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	verifier := new(big.Int).SetInt64(12345) // Dummy verifier

	server, err := auth.NewSRPServer("testuser", salt, verifier)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if server == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestNewSRPServer_InvalidSalt(t *testing.T) {
	verifier := new(big.Int).SetInt64(12345)

	_, err := auth.NewSRPServer("testuser", "invalid-base64!!!", verifier)
	if err == nil {
		t.Error("expected error for invalid salt")
	}
	if !contains(err.Error(), "invalid salt encoding") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSRPServer_Init(t *testing.T) {
	N, g, _ := auth.GetGroupParameters()

	// Generate client ephemeral value A
	a := generateRandomBigInt(256)
	A := new(big.Int).Exp(g, a, N)
	ABase64 := base64.StdEncoding.EncodeToString(A.Bytes())

	// Create server with verifier
	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	password := "test-password"
	verifier, err := auth.ComputeVerifier("testuser", salt, password, N, g)
	if err != nil {
		t.Fatal(err)
	}

	server, err := auth.NewSRPServer("testuser", salt, verifier)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize SRP session
	returnedSalt, BBase64, err := server.Init(ABase64)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if returnedSalt != salt {
		t.Errorf("expected salt %q, got %q", salt, returnedSalt)
	}

	if BBase64 == "" {
		t.Error("expected non-empty B")
	}

	// Verify B is valid base64
	BBytes, err := base64.StdEncoding.DecodeString(BBase64)
	if err != nil {
		t.Errorf("B is not valid base64: %v", err)
	}

	B := new(big.Int).SetBytes(BBytes)

	// Verify B is in valid range (0 < B < N)
	if B.Cmp(big.NewInt(0)) <= 0 {
		t.Error("expected B > 0")
	}
	if B.Cmp(N) >= 0 {
		t.Error("expected B < N")
	}

	// Verify B mod N != 0
	if new(big.Int).Mod(B, N).Cmp(big.NewInt(0)) == 0 {
		t.Error("B mod N should not be 0")
	}
}

func TestSRPServer_Init_InvalidA(t *testing.T) {
	N, _, _ := auth.GetGroupParameters()

	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	verifier := new(big.Int).SetInt64(12345)

	server, err := auth.NewSRPServer("testuser", salt, verifier)
	if err != nil {
		t.Fatal(err)
	}

	// Test with A = 0 (invalid)
	A := big.NewInt(0)
	ABase64 := base64.StdEncoding.EncodeToString(A.Bytes())

	_, _, err = server.Init(ABase64)
	if err == nil {
		t.Error("expected error for A = 0")
	}
	if !contains(err.Error(), "invalid A") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Test with A = N (A mod N == 0, invalid)
	ABase64 = base64.StdEncoding.EncodeToString(N.Bytes())
	_, _, err = server.Init(ABase64)
	if err == nil {
		t.Error("expected error for A = N")
	}
}

func TestSRPServer_Init_InvalidBase64(t *testing.T) {
	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	verifier := new(big.Int).SetInt64(12345)

	server, err := auth.NewSRPServer("testuser", salt, verifier)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = server.Init("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestSRPServer_Verify(t *testing.T) {
	N, g, _ := auth.GetGroupParameters()

	// Client generates ephemeral keypair
	a := generateRandomBigInt(256)
	A := new(big.Int).Exp(g, a, N)

	// Server setup
	username := "testuser"
	password := "test-password"
	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	verifier, err := auth.ComputeVerifier(username, salt, password, N, g)
	if err != nil {
		t.Fatal(err)
	}

	server, err := auth.NewSRPServer(username, salt, verifier)
	if err != nil {
		t.Fatal(err)
	}

	// Server init
	ABase64 := base64.StdEncoding.EncodeToString(A.Bytes())
	_, BBase64, err := server.Init(ABase64)
	if err != nil {
		t.Fatal(err)
	}

	BBytes, _ := base64.StdEncoding.DecodeString(BBase64)
	B := new(big.Int).SetBytes(BBytes)

	// Client computes shared secret
	M1 := computeClientProof(username, salt, password, a, A, B)
	M1Base64 := base64.StdEncoding.EncodeToString(M1)

	// Server verify
	M2Base64, err := server.Verify(M1Base64)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if M2Base64 == "" {
		t.Error("expected non-empty M2")
	}

	// Verify M2 is valid base64
	_, err = base64.StdEncoding.DecodeString(M2Base64)
	if err != nil {
		t.Errorf("M2 is not valid base64: %v", err)
	}
}

func TestSRPServer_Verify_InvalidProof(t *testing.T) {
	N, g, _ := auth.GetGroupParameters()

	a := generateRandomBigInt(256)
	A := new(big.Int).Exp(g, a, N)

	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	password := "test-password"
	verifier, err := auth.ComputeVerifier("testuser", salt, password, N, g)
	if err != nil {
		t.Fatal(err)
	}

	server, err := auth.NewSRPServer("testuser", salt, verifier)
	if err != nil {
		t.Fatal(err)
	}

	ABase64 := base64.StdEncoding.EncodeToString(A.Bytes())
	_, _, err = server.Init(ABase64)
	if err != nil {
		t.Fatal(err)
	}

	// Send invalid proof
	invalidM1 := []byte("invalid-proof-data")
	M1Base64 := base64.StdEncoding.EncodeToString(invalidM1)

	_, err = server.Verify(M1Base64)
	if err == nil {
		t.Error("expected error for invalid proof")
	}
	if !contains(err.Error(), "authentication failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSRPServer_Verify_BeforeInit(t *testing.T) {
	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	verifier := new(big.Int).SetInt64(12345)

	server, err := auth.NewSRPServer("testuser", salt, verifier)
	if err != nil {
		t.Fatal(err)
	}

	// Try to verify without calling Init first
	M1Base64 := base64.StdEncoding.EncodeToString([]byte("proof"))
	_, err = server.Verify(M1Base64)
	if err == nil {
		t.Error("expected error when verify called before init")
	}
	if !contains(err.Error(), "init must be called before verify") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSRPServer_ClearSecrets(t *testing.T) {
	N, g, _ := auth.GetGroupParameters()

	a := generateRandomBigInt(256)
	A := new(big.Int).Exp(g, a, N)

	salt := base64.StdEncoding.EncodeToString([]byte("testsalt"))
	password := "test-password"
	verifier, err := auth.ComputeVerifier("testuser", salt, password, N, g)
	if err != nil {
		t.Fatal(err)
	}

	server, err := auth.NewSRPServer("testuser", salt, verifier)
	if err != nil {
		t.Fatal(err)
	}

	ABase64 := base64.StdEncoding.EncodeToString(A.Bytes())
	_, BBase64, err := server.Init(ABase64)
	if err != nil {
		t.Fatal(err)
	}

	BBytes, _ := base64.StdEncoding.DecodeString(BBase64)
	B := new(big.Int).SetBytes(BBytes)

	M1 := computeClientProof("testuser", salt, password, a, A, B)
	M1Base64 := base64.StdEncoding.EncodeToString(M1)

	_, err = server.Verify(M1Base64)
	if err != nil {
		t.Fatal(err)
	}

	// Get session key before clearing
	K := server.GetSessionKey()
	if K == nil {
		t.Fatal("expected non-nil session key")
	}

	// Clear secrets
	server.ClearSecrets()

	// Session key should be cleared (all zeros or nil)
	K2 := server.GetSessionKey()
	if K2 != nil {
		allZeros := true
		for _, b := range K2 {
			if b != 0 {
				allZeros = false
				break
			}
		}
		if !allZeros {
			t.Error("expected session key to be cleared")
		}
	}
}

// Helper function: compute client-side M1 proof for testing
//
//nolint:gocritic // a, A, B are capitalized per RFC 5054 SRP-6a specification
func computeClientProof(username, saltBase64, password string, a, A, B *big.Int) []byte {
	// Use the shared SRP client implementation
	M1, err := srp.ComputeClientProofWithEphemeral(username, password, saltBase64, a, A, B)
	if err != nil {
		panic(fmt.Sprintf("failed to compute client proof: %v", err))
	}
	return M1
}

// Helper: generate random big int with specified bit length
func generateRandomBigInt(bits int) *big.Int {
	bytes := make([]byte, bits/8)
	rand.Read(bytes)
	return new(big.Int).SetBytes(bytes)
}
