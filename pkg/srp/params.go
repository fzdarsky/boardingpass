// Package srp provides SRP-6a (Secure Remote Password) client-side implementation.
// This package implements RFC 5054 for zero-knowledge password authentication.
package srp

import (
	"crypto/sha256"
	"math/big"
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

// computeK computes the SRP-6a multiplier k = H(N | g)
//
//nolint:gocritic // N is capitalized per RFC 5054 SRP-6a specification
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
