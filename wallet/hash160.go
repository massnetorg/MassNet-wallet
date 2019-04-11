package wallet

import (
	"hash"

	"github.com/btcsuite/fastsha256"
	"github.com/btcsuite/golangcrypto/ripemd160"
)

// Calculate the hash of hasher over buf.
func calcHash(buf []byte, hasher hash.Hash) []byte {
	hasher.Write(buf)
	return hasher.Sum(nil)
}

// Hash160 calculates the hash ripemd160(sha256(b)).
func Hash160(buf []byte) []byte {
	return calcHash(calcHash(buf, fastsha256.New()), ripemd160.New())
}
