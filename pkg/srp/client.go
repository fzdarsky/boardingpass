package srp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
)

// Client represents the client-side state for an SRP-6a authentication session.
type Client struct {
	Username string
	Password string
	Salt     []byte
	a        *big.Int // Client ephemeral private value
	A        *big.Int // Client ephemeral public value
	B        *big.Int // Server ephemeral public value (received from server)
	S        *big.Int // Shared secret
	K        []byte   // Session key
	M1       []byte   // Client proof
	M2       []byte   // Server proof (for verification)
}

// NewClient creates a new SRP client for authentication.
func NewClient(username, password string) *Client {
	return &Client{
		Username: username,
		Password: password,
	}
}

// GenerateEphemeralKeypair generates the client's ephemeral private (a) and public (A) keys.
// A = g^a mod N
// Returns A as base64-encoded string for sending to server.
func (c *Client) GenerateEphemeralKeypair() (string, error) {
	// Generate ephemeral private value a (256 bits of entropy)
	aBytes := make([]byte, 32)
	if _, err := rand.Read(aBytes); err != nil {
		return "", fmt.Errorf("failed to generate random a: %w", err)
	}
	c.a = new(big.Int).SetBytes(aBytes)

	// Compute A = g^a % N
	c.A = new(big.Int).Exp(G, c.a, N)

	// Validate A (must not be 0 mod N)
	if new(big.Int).Mod(c.A, N).Cmp(big.NewInt(0)) == 0 {
		return "", fmt.Errorf("invalid A: A mod N == 0 (regenerate a)")
	}

	// Encode A for transmission
	ABase64 := base64.StdEncoding.EncodeToString(c.A.Bytes())
	return ABase64, nil
}

// SetServerResponse sets the server's response (salt and B) received during Init phase.
//
//nolint:gocritic // BBase64 is capitalized per RFC 5054 SRP-6a specification
func (c *Client) SetServerResponse(saltBase64, BBase64 string) error {
	// Decode salt
	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return fmt.Errorf("invalid salt encoding: %w", err)
	}
	c.Salt = salt

	// Decode server's B
	BBytes, err := base64.StdEncoding.DecodeString(BBase64)
	if err != nil {
		return fmt.Errorf("invalid B encoding: %w", err)
	}
	c.B = new(big.Int).SetBytes(BBytes)

	// Validate B (must not be 0 mod N)
	if new(big.Int).Mod(c.B, N).Cmp(big.NewInt(0)) == 0 {
		return fmt.Errorf("invalid B: B mod N == 0")
	}

	return nil
}

// ComputeSharedSecret computes the shared secret S and session key K.
// Client formula: S = (B - k*g^x)^(a + u*x) mod N
// K = H(S)
func (c *Client) ComputeSharedSecret() error {
	if c.a == nil || c.A == nil || c.B == nil || c.Salt == nil {
		return fmt.Errorf("must call GenerateEphemeralKeypair and SetServerResponse first")
	}

	// Derive private key x = H(salt | H(username | ":" | password))
	x := c.derivePrivateKey()

	// Compute u = H(A | B) - scrambling parameter
	u := c.computeU()

	// Compute S = (B - k*g^x)^(a + u*x) mod N

	// Step 1: g^x mod N
	gx := new(big.Int).Exp(G, x, N)

	// Step 2: k*g^x mod N
	kgx := new(big.Int).Mul(K, gx)
	kgx.Mod(kgx, N)

	// Step 3: B - k*g^x mod N
	base := new(big.Int).Sub(c.B, kgx)
	base.Mod(base, N)

	// Step 4: u*x
	ux := new(big.Int).Mul(u, x)

	// Step 5: a + u*x
	exponent := new(big.Int).Add(c.a, ux)

	// Step 6: (B - k*g^x)^(a + u*x) mod N
	c.S = new(big.Int).Exp(base, exponent, N)

	// Compute session key K = H(S)
	hash := sha256.New()
	hash.Write(c.S.Bytes())
	c.K = hash.Sum(nil)

	return nil
}

// derivePrivateKey derives the private key x from username, password, and salt.
// x = H(salt | H(username | ":" | password))
func (c *Client) derivePrivateKey() *big.Int {
	// H(username | ":" | password)
	innerHash := sha256.Sum256([]byte(c.Username + ":" + c.Password))

	// H(salt | innerHash)
	outerHash := sha256.New()
	outerHash.Write(c.Salt)
	outerHash.Write(innerHash[:])

	return new(big.Int).SetBytes(outerHash.Sum(nil))
}

// computeU computes the scrambling parameter u = H(A | B)
func (c *Client) computeU() *big.Int {
	hash := sha256.New()

	// Pad A and B to same length for consistent hashing
	maxLen := len(N.Bytes())
	ABytes := make([]byte, maxLen)
	BBytes := make([]byte, maxLen)

	ACopy := c.A.Bytes()
	BCopy := c.B.Bytes()

	copy(ABytes[maxLen-len(ACopy):], ACopy)
	copy(BBytes[maxLen-len(BCopy):], BCopy)

	hash.Write(ABytes)
	hash.Write(BBytes)

	return new(big.Int).SetBytes(hash.Sum(nil))
}

// ComputeClientProof computes the client proof M1.
// M1 = H(H(N) XOR H(g) | H(username) | salt | A | B | K)
func (c *Client) ComputeClientProof() (string, error) {
	if c.K == nil {
		return "", fmt.Errorf("must call ComputeSharedSecret first")
	}

	// H(N)
	hashN := sha256.Sum256(N.Bytes())

	// H(g)
	hashG := sha256.Sum256(G.Bytes())

	// H(N) XOR H(g)
	hashNXorG := make([]byte, len(hashN))
	for i := range len(hashN) {
		hashNXorG[i] = hashN[i] ^ hashG[i]
	}

	// H(username)
	hashUsername := sha256.Sum256([]byte(c.Username))

	// Compute M1 = H(H(N) XOR H(g) | H(username) | salt | A | B | K)
	hash := sha256.New()
	hash.Write(hashNXorG)
	hash.Write(hashUsername[:])
	hash.Write(c.Salt)
	hash.Write(c.A.Bytes())
	hash.Write(c.B.Bytes())
	hash.Write(c.K)

	c.M1 = hash.Sum(nil)

	// Encode M1 for transmission
	M1Base64 := base64.StdEncoding.EncodeToString(c.M1)
	return M1Base64, nil
}

// VerifyServerProof verifies the server's proof M2.
// M2 = H(A | M1 | K)
// Returns nil if M2 matches, indicating successful mutual authentication.
//
//nolint:gocritic // M2Base64 is capitalized per RFC 5054 SRP-6a specification
func (c *Client) VerifyServerProof(M2Base64 string) error {
	if c.M1 == nil || c.K == nil {
		return fmt.Errorf("must call ComputeClientProof first")
	}

	// Decode server's M2
	serverM2, err := base64.StdEncoding.DecodeString(M2Base64)
	if err != nil {
		return fmt.Errorf("invalid M2 encoding: %w", err)
	}

	// Compute expected M2 = H(A | M1 | K)
	hash := sha256.New()
	hash.Write(c.A.Bytes())
	hash.Write(c.M1)
	hash.Write(c.K)
	expectedM2 := hash.Sum(nil)

	// Compare M2 values
	if len(serverM2) != len(expectedM2) {
		return fmt.Errorf("server authentication failed: M2 length mismatch")
	}

	// Use constant-time comparison to prevent timing attacks
	match := true
	for i := range expectedM2 {
		if serverM2[i] != expectedM2[i] {
			match = false
		}
	}

	if !match {
		return fmt.Errorf("server authentication failed: M2 mismatch")
	}

	c.M2 = serverM2
	return nil
}

// GetSessionKey returns the session key K (for testing purposes).
// In practice, the session token from the server should be used instead.
func (c *Client) GetSessionKey() []byte {
	return c.K
}

// ClearSecrets clears sensitive values from memory.
func (c *Client) ClearSecrets() {
	// Clear password
	c.Password = ""

	// Clear big integers
	if c.a != nil {
		c.a.SetInt64(0)
		c.a = nil
	}
	if c.S != nil {
		c.S.SetInt64(0)
		c.S = nil
	}

	// Clear byte slices
	if c.Salt != nil {
		for i := range c.Salt {
			c.Salt[i] = 0
		}
		c.Salt = nil
	}
	if c.K != nil {
		for i := range c.K {
			c.K[i] = 0
		}
		c.K = nil
	}
	if c.M1 != nil {
		for i := range c.M1 {
			c.M1[i] = 0
		}
		c.M1 = nil
	}
	if c.M2 != nil {
		for i := range c.M2 {
			c.M2[i] = 0
		}
		c.M2 = nil
	}
}

// ComputeClientProofWithEphemeral computes the client proof M1 given all the parameters.
// This is a helper function for testing purposes where you already have the ephemeral values.
// For normal client usage, use the Client struct methods instead.
//
//nolint:gocritic // a, A, B are capitalized per RFC 5054 SRP-6a specification
func ComputeClientProofWithEphemeral(username, password, saltBase64 string, a, A, B *big.Int) ([]byte, error) {
	// Decode salt
	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid salt encoding: %w", err)
	}

	// Derive private key x = H(salt | H(username | ":" | password))
	innerHash := sha256.Sum256([]byte(username + ":" + password))
	outerHash := sha256.New()
	outerHash.Write(salt)
	outerHash.Write(innerHash[:])
	x := new(big.Int).SetBytes(outerHash.Sum(nil))

	// Compute u = H(A | B)
	uHash := sha256.New()
	maxLen := len(N.Bytes())
	ABytes := make([]byte, maxLen)
	BBytes := make([]byte, maxLen)
	ACopy := A.Bytes()
	BCopy := B.Bytes()
	copy(ABytes[maxLen-len(ACopy):], ACopy)
	copy(BBytes[maxLen-len(BCopy):], BCopy)
	uHash.Write(ABytes)
	uHash.Write(BBytes)
	u := new(big.Int).SetBytes(uHash.Sum(nil))

	// Compute S = (B - k*g^x)^(a + u*x) mod N
	gx := new(big.Int).Exp(G, x, N)
	kgx := new(big.Int).Mul(K, gx)
	kgx.Mod(kgx, N)
	base := new(big.Int).Sub(B, kgx)
	base.Mod(base, N)
	ux := new(big.Int).Mul(u, x)
	exponent := new(big.Int).Add(a, ux)
	S := new(big.Int).Exp(base, exponent, N)

	// Compute session key K = H(S)
	kHash := sha256.New()
	kHash.Write(S.Bytes())
	sessionKey := kHash.Sum(nil)

	// Compute M1 = H(H(N) XOR H(g) | H(username) | salt | A | B | K)
	hashN := sha256.Sum256(N.Bytes())
	hashG := sha256.Sum256(G.Bytes())
	hashNXorG := make([]byte, len(hashN))
	for i := range len(hashN) {
		hashNXorG[i] = hashN[i] ^ hashG[i]
	}
	hashUsername := sha256.Sum256([]byte(username))

	m1Hash := sha256.New()
	m1Hash.Write(hashNXorG)
	m1Hash.Write(hashUsername[:])
	m1Hash.Write(salt)
	m1Hash.Write(A.Bytes())
	m1Hash.Write(B.Bytes())
	m1Hash.Write(sessionKey)

	return m1Hash.Sum(nil), nil
}
