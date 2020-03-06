package pocutil

import (
	"encoding/binary"
)

// PoCValue represents the value type of PoC Record.
type PoCValue uint64

const (
	// PoCPrefixLen represents the byte length of PoCPrefix.
	PoCPrefixLen = 4
)

var (
	// PoCPrefix is the prefix for PoC Calculation.
	PoCPrefix = []byte("MASS")
)

// P calculates MASSSHA256(PoCPrefix || PubKeyHash || X).CutByBitLength(bitLength),
// it takes in x as PoCValue.
func P(x PoCValue, bl int, pubKeyHash Hash) PoCValue {
	var xb [8]byte
	binary.LittleEndian.PutUint64(xb[:], uint64(x))

	return PB(xb[:], bl, pubKeyHash)
}

// PB calculates MASSSHA256(PoCPrefix || PubKeyHash || X).CutByBitLength(bitLength),
// it takes in x as Byte Slice.
func PB(x []byte, bl int, pubKeyHash Hash) PoCValue {
	x = PoCBytes(x, bl)
	raw := make([]byte, PoCPrefixLen+32+8)
	copy(raw[:], PoCPrefix[:])
	copy(raw[PoCPrefixLen:], pubKeyHash[:])
	copy(raw[PoCPrefixLen+32:], x)

	return CutHash(MASSSHA256(raw[:]), bl)
}

// F calculates MASSSHA256(PoCPrefix || PubKeyHash || X || XP).CutByBitLength(bitLength),
// it takes in (x, xp) as PoCValue.
func F(x, xp PoCValue, bl int, pubKeyHash Hash) PoCValue {
	var xb, xpb [8]byte
	binary.LittleEndian.PutUint64(xb[:], uint64(x))
	binary.LittleEndian.PutUint64(xpb[:], uint64(xp))

	return FB(xb[:], xpb[:], bl, pubKeyHash)
}

// FB calculates MASSSHA256(PoCPrefix || PubKeyHash || X || XP).CutByBitLength(bitLength),
// it takes in (x, xp) as Byte Slice.
func FB(x, xp []byte, bl int, pubKeyHash Hash) PoCValue {
	x, xp = PoCBytes(x, bl), PoCBytes(xp, bl)
	raw := make([]byte, PoCPrefixLen+32+8*2)
	copy(raw[:], PoCPrefix[:])
	copy(raw[PoCPrefixLen:], pubKeyHash[:])
	copy(raw[PoCPrefixLen+32:], x)
	copy(raw[PoCPrefixLen+32+8:], xp)

	return CutHash(MASSSHA256(raw[:]), bl)
}
