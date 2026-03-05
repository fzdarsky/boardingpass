// Package srp provides SRP-6a (Secure Remote Password) client-side implementation.
// This package implements RFC 5054 for zero-knowledge password authentication.
package srp

import (
	"crypto/sha256"
	"math/big"
	"strings"
)

// RFC 5054 2048-bit SRP Group Parameters
// These MUST match the server-side parameters exactly.
var (
	// N is the 2048-bit safe prime from RFC 5054 Appendix A
	N = initN()

	// G is the generator (always 2 for this group)
	G = big.NewInt(2)

	// K is the multiplier: k = H(N | g)
	K = computeK(N, G)
)

// initN initializes the N parameter (must match server exactly)
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

// padToN returns v's byte representation zero-padded to N's byte length.
// SRP-6a requires consistent padding of big integer hash inputs to ensure
// interoperability between implementations.
func padToN(v *big.Int) []byte {
	nLen := len(N.Bytes())
	vBytes := v.Bytes()
	if len(vBytes) >= nLen {
		return vBytes
	}
	padded := make([]byte, nLen)
	copy(padded[nLen-len(vBytes):], vBytes)
	return padded
}

// computeK computes the SRP-6a multiplier k = H(N, g).
// Note: g is hashed at its natural length (not padded to N), matching the
// secure-remote-password JS library behavior.
//
//nolint:gocritic // N is capitalized per RFC 5054 SRP-6a specification
func computeK(N, g *big.Int) *big.Int {
	hash := sha256.New()
	hash.Write(N.Bytes())
	hash.Write(g.Bytes())
	return new(big.Int).SetBytes(hash.Sum(nil))
}

// NormalizePassword strips all non-alphanumeric characters and lowercases,
// producing a canonical form for SRP key derivation. Both clients (CLI, mobile)
// and the service MUST apply this normalization before SRP to ensure identical
// password bytes regardless of input format (e.g. "94:C6:91:A8:18:EA",
// "94-c6-91-a8-18-ea", and "94C691A818EA" all normalize to "94c691a818ea").
func NormalizePassword(password string) string {
	var b strings.Builder
	b.Grow(len(password))
	for _, r := range password {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}
