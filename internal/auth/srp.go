// Package auth implements SRP-6a authentication, session management, and rate limiting.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/big"
)

// RFC 5054 2048-bit SRP Group Parameters
// N: 2048-bit safe prime
// g: Generator = 2
var (
	// srpN is the 2048-bit safe prime from RFC 5054 Appendix A
	srpN = initN()

	// srpG is the generator (always 2 for this group)
	srpG = big.NewInt(2)

	// srpK is the multiplier: k = H(N | g) - computed lazily
	srpK *big.Int
)

// initN initializes the N parameter
func initN() *big.Int {
	n := new(big.Int)
	n.SetString(
		"AC6BDB41324A9A9BF166DE5E1389582FAF72B6651987EE07FC3192943DB56050"+
			"A37329CBB4A099ED8193E0757767A13DD52312AB4B03310DCD7F48A9DA04FD50"+
			"E8083969EDB767B0CF6095179A163AB3661A05FBD5FAAAE82918A9962F0B93B8"+
			"55F97993EC975EEAA80D740ADBF4FF747359D041D5C33EA71D281E446B14773B"+
			"CA97B43A23FB801676BD207A436C6481F1D2B9078717461A5B9D32E688F87748"+
			"544523B524B0D57D5EA77A2775D2ECFA032CFBDBF52FB3786160279004E57AE6"+
			"AF874E7303CE53299CCC041C7BC308D82A5698F3A8D0C38271AE35F8E9DBFBB6"+
			"94B5C803D89F7AE435DE236D525F54759B65E372FCD68EF20FA7111F9E4AFF73", 16)
	return n
}

// computeK computes the SRP-6a multiplier k = H(N | g)
func computeK(N, g *big.Int) *big.Int {
	hash := sha256.New()

	// Pad N and g to same length for consistent hashing
	nBytes := N.Bytes()
	gBytes := make([]byte, len(nBytes))
	copy(gBytes[len(gBytes)-1:], g.Bytes())

	hash.Write(nBytes)
	hash.Write(gBytes)

	return new(big.Int).SetBytes(hash.Sum(nil))
}

// SRPServer represents the server-side state for an SRP-6a authentication session.
type SRPServer struct {
	Username string
	Salt     []byte // Raw salt bytes (not base64)
	Verifier *big.Int
	b        *big.Int // Server ephemeral private value
	B        *big.Int // Server ephemeral public value
	A        *big.Int // Client ephemeral public value (received during init)
	S        *big.Int // Shared secret
	K        []byte   // Session key
}

// NewSRPServer creates a new SRP server instance with the provided verifier.
func NewSRPServer(username string, saltBase64 string, verifier *big.Int) (*SRPServer, error) {
	// Decode salt
	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid salt encoding: %w", err)
	}

	return &SRPServer{
		Username: username,
		Salt:     salt,
		Verifier: verifier,
	}, nil
}

// Init handles the SRP-6a initialization phase.
// Client sends A (ephemeral public value), server generates b and B, returns B and salt.
func (s *SRPServer) Init(ABase64 string) (saltBase64, BBase64 string, err error) {
	// Get group parameters
	N, g, k := GetGroupParameters()

	// Decode client's A
	ABytes, err := base64.StdEncoding.DecodeString(ABase64)
	if err != nil {
		return "", "", fmt.Errorf("invalid A encoding: %w", err)
	}
	s.A = new(big.Int).SetBytes(ABytes)

	// Validate A (must not be 0 mod N)
	if new(big.Int).Mod(s.A, N).Cmp(big.NewInt(0)) == 0 {
		return "", "", fmt.Errorf("invalid A: A mod N == 0")
	}

	// Generate server ephemeral private value b (256 bits of entropy)
	bBytes := make([]byte, 32)
	if _, err := rand.Read(bBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random b: %w", err)
	}
	s.b = new(big.Int).SetBytes(bBytes)

	// Compute B = (k*v + g^b) % N
	// B = k*v % N
	kv := new(big.Int).Mul(k, s.Verifier)
	kv.Mod(kv, N)

	// g^b % N
	gb := new(big.Int).Exp(g, s.b, N)

	// B = (k*v + g^b) % N
	s.B = new(big.Int).Add(kv, gb)
	s.B.Mod(s.B, N)

	// Validate B (must not be 0 mod N)
	if new(big.Int).Mod(s.B, N).Cmp(big.NewInt(0)) == 0 {
		return "", "", fmt.Errorf("invalid B: B mod N == 0 (regenerate b)")
	}

	// Encode salt and B for response
	saltBase64 = base64.StdEncoding.EncodeToString(s.Salt)
	BBase64 = base64.StdEncoding.EncodeToString(s.B.Bytes())

	return saltBase64, BBase64, nil
}

// Verify handles the SRP-6a verification phase.
// Client sends M1 (proof), server validates and computes M2.
func (s *SRPServer) Verify(M1Base64 string) (M2Base64 string, err error) {
	if s.A == nil || s.B == nil {
		return "", fmt.Errorf("init must be called before verify")
	}

	// Decode client's M1
	M1, err := base64.StdEncoding.DecodeString(M1Base64)
	if err != nil {
		return "", fmt.Errorf("invalid M1 encoding: %w", err)
	}

	// Compute shared secret S and session key K
	if err := s.computeSharedSecret(); err != nil {
		return "", fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// Compute expected M1
	expectedM1 := s.computeM1()

	// Compare M1 using constant-time comparison (prevent timing attacks)
	if subtle.ConstantTimeCompare(M1, expectedM1) != 1 {
		return "", fmt.Errorf("authentication failed: invalid proof M1")
	}

	// Compute M2 = H(A | M1 | K)
	M2 := s.computeM2(M1)
	M2Base64 = base64.StdEncoding.EncodeToString(M2)

	return M2Base64, nil
}

// computeSharedSecret computes the shared secret S and session key K.
// S = (A * v^u)^b % N
// K = H(S)
func (s *SRPServer) computeSharedSecret() error {
	N, _, _ := GetGroupParameters()

	// Compute u = H(A | B)
	u := s.computeU()

	// Compute S = (A * v^u)^b % N
	// Step 1: v^u % N
	vu := new(big.Int).Exp(s.Verifier, u, N)

	// Step 2: A * v^u % N
	avu := new(big.Int).Mul(s.A, vu)
	avu.Mod(avu, N)

	// Step 3: (A * v^u)^b % N
	s.S = new(big.Int).Exp(avu, s.b, N)

	// Compute session key K = H(S)
	hash := sha256.New()
	hash.Write(s.S.Bytes())
	s.K = hash.Sum(nil)

	return nil
}

// computeU computes the scrambling parameter u = H(A | B)
func (s *SRPServer) computeU() *big.Int {
	N, _, _ := GetGroupParameters()
	hash := sha256.New()

	// Pad A and B to same length for consistent hashing
	maxLen := len(N.Bytes())
	ABytes := make([]byte, maxLen)
	BBytes := make([]byte, maxLen)

	ACopy := s.A.Bytes()
	BCopy := s.B.Bytes()

	copy(ABytes[maxLen-len(ACopy):], ACopy)
	copy(BBytes[maxLen-len(BCopy):], BCopy)

	hash.Write(ABytes)
	hash.Write(BBytes)

	return new(big.Int).SetBytes(hash.Sum(nil))
}

// computeM1 computes the client proof M1 = H(H(N) XOR H(g) | H(username) | salt | A | B | K)
func (s *SRPServer) computeM1() []byte {
	N, g, _ := GetGroupParameters()

	// H(N)
	hashN := sha256.Sum256(N.Bytes())

	// H(g)
	hashG := sha256.Sum256(g.Bytes())

	// H(N) XOR H(g)
	hashNXorG := make([]byte, len(hashN))
	for i := 0; i < len(hashN); i++ {
		hashNXorG[i] = hashN[i] ^ hashG[i]
	}

	// H(username)
	hashUsername := sha256.Sum256([]byte(s.Username))

	// Compute M1 = H(H(N) XOR H(g) | H(username) | salt | A | B | K)
	hash := sha256.New()
	hash.Write(hashNXorG)
	hash.Write(hashUsername[:])
	hash.Write(s.Salt)
	hash.Write(s.A.Bytes())
	hash.Write(s.B.Bytes())
	hash.Write(s.K)

	return hash.Sum(nil)
}

// computeM2 computes the server proof M2 = H(A | M1 | K)
func (s *SRPServer) computeM2(M1 []byte) []byte {
	hash := sha256.New()
	hash.Write(s.A.Bytes())
	hash.Write(M1)
	hash.Write(s.K)

	return hash.Sum(nil)
}

// GetSessionKey returns the computed session key K.
// Should only be called after successful Verify().
func (s *SRPServer) GetSessionKey() []byte {
	return s.K
}

// ClearSecrets clears sensitive values from memory.
func (s *SRPServer) ClearSecrets() {
	if s.b != nil {
		s.b.SetInt64(0)
		s.b = nil
	}
	if s.S != nil {
		s.S.SetInt64(0)
		s.S = nil
	}
	if s.K != nil {
		for i := range s.K {
			s.K[i] = 0
		}
		s.K = nil
	}
}

// GetGroupParameters returns the SRP group parameters (N, g, k) for testing or client use.
func GetGroupParameters() (N, g, k *big.Int) {
	// Lazy initialize k if not already done
	if srpK == nil {
		srpK = computeK(srpN, srpG)
	}
	return srpN, srpG, srpK
}
