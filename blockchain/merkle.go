// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"math"

	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/txscript"
	"github.com/massnetorg/MassNet-wallet/wire"

	"bytes"
	"fmt"
)

const (
	CoinbaseWitnessDataLen        = 32
	CoinbaseWitnessPkScriptLength = 38
)

var (
	WitnessMagicBytes = []byte{
		txscript.OP_RETURN,
		txscript.OP_DATA_36,
		0xaa,
		0x21,
		0xa9,
		0xed,
	}
)

// nextPowerOfTwo returns the next highest power of two from a given number if
// it is not already a power of two.  This is a helper function used during the
// calculation of a merkle tree.
func nextPowerOfTwo(n int) int {
	if n&(n-1) == 0 {
		return n
	}

	exponent := uint(math.Log2(float64(n))) + 1
	return 1 << exponent // 2^exponent
}

// HashMerkleBranches takes two hashes, treated as the left and right tree
// nodes, and returns the hash of their concatenation.  This is a helper
// function used to aid in the generation of a merkle tree.
func HashMerkleBranches(left *wire.Hash, right *wire.Hash) *wire.Hash {
	var sha [wire.HashSize * 2]byte
	copy(sha[:wire.HashSize], left[:])
	copy(sha[wire.HashSize:], right[:])

	newSha := wire.DoubleHashH(sha[:])
	return &newSha
}

// BuildMerkleTreeStore creates a merkle tree from a slice of transactions,
// stores it using a linear array, and returns a slice of the backing array.  A
// linear array was chosen as opposed to an actual tree structure since it uses
// about half as much memory.  The following describes a merkle tree and how it
// is stored in a linear array.
//
// A merkle tree is a tree in which every non-leaf node is the hash of its
// children nodes.  A diagram depicting how this works for transactions
// where h(x) is a double sha256 follows:
//
//	         root = h1234 = h(h12 + h34)
//	        /                           \
//	  h12 = h(h1 + h2)            h34 = h(h3 + h4)
//	   /            \              /            \
//	h1 = h(tx1)  h2 = h(tx2)    h3 = h(tx3)  h4 = h(tx4)
//
// The above stored as a linear array is as follows:
//
// 	[h1 h2 h3 h4 h12 h34 root]
//
// As the above shows, the merkle root is always the last element in the array.
//
// The number of inputs is not always a power of two which results in a
// balanced tree structure as above.  In that case, parent nodes with no
// children are also zero and parent nodes with only a single left node
// are calculated by concatenating the left node with itself before hashing.
// Since this function uses nodes that are pointers to the hashes, empty nodes
// will be nil.
//
// The additional bool parameter indicates if we are generating the merkle tree
// using witness transaction id's rather than regular transaction id's. This
// also presents an additional case wherein the wtxid of the coinbase transaction
// is the zeroHash.
func BuildMerkleTreeStore(transactions []*massutil.Tx, witness bool) []*wire.Hash {
	nextPoT := nextPowerOfTwo(len(transactions))
	arraySize := nextPoT*2 - 1
	merkles := make([]*wire.Hash, arraySize)

	for i, tx := range transactions {
		switch {
		case witness && i == 0:
			var zeroHash wire.Hash
			merkles[i] = &zeroHash
		case witness:
			wSha := tx.MsgTx().WitnessHash()
			merkles[i] = &wSha
		default:
			merkles[i] = tx.Hash()
		}

	}

	offset := nextPoT
	for i := 0; i < arraySize-1; i += 2 {
		switch {

		case merkles[i] == nil:
			merkles[offset] = nil

		case merkles[i+1] == nil:
			newHash := HashMerkleBranches(merkles[i], merkles[i])
			merkles[offset] = newHash

		default:
			newHash := HashMerkleBranches(merkles[i], merkles[i+1])
			merkles[offset] = newHash
		}
		offset++
	}

	return merkles
}

// ExtractWitnessCommitment attempts to locate, and return the witness
// commitment for a block. The witness commitment is of the form:
// SHA256(witness root || witness nonce). The function additionally returns a
// boolean indicating if the witness root was located within any of the txOut's
// in the passed transaction. The witness commitment is stored as the data push
// for an OP_RETURN with special magic bytes to aide in location.
func ExtractWitnessCommitment(tx *massutil.Tx) ([]byte, bool) {
	if !IsCoinBase(tx) {
		return nil, false
	}

	msgTx := tx.MsgTx()
	for i := len(msgTx.TxOut) - 1; i >= 0; i-- {
		pkScript := msgTx.TxOut[i].PkScript
		if len(pkScript) >= CoinbaseWitnessPkScriptLength &&
			bytes.HasPrefix(pkScript, WitnessMagicBytes) {
			start := len(WitnessMagicBytes)
			end := CoinbaseWitnessPkScriptLength
			return msgTx.TxOut[i].PkScript[start:end], true
		}
	}

	return nil, false
}

// ValidateWitnessCommitment validates the witness commitment (if any) found
// within the coinbase transaction of the passed block.
func ValidateWitnessCommitment(blk *massutil.Block) error {
	if len(blk.Transactions()) == 0 {
		str := "cannot validate witness commitment of block without " +
			"transactions"
		return ruleError(ErrNoTransactions, str)
	}

	coinbaseTx := blk.Transactions()[0]
	if len(coinbaseTx.MsgTx().TxIn) == 0 {
		return ruleError(ErrNoTxInputs, "transaction has no inputs")
	}

	witnessCommitment, _ := ExtractWitnessCommitment(coinbaseTx)

	coinbaseWitness := coinbaseTx.MsgTx().TxIn[0].Witness
	if len(coinbaseWitness) != 1 {
		str := fmt.Sprintf("the coinbase transaction has %d items in "+
			"its witness stack when only one is allowed",
			len(coinbaseWitness))
		return ruleError(ErrInvalidWitnessCommitment, str)
	}
	witnessNonce := coinbaseWitness[0]
	if len(witnessNonce) != CoinbaseWitnessDataLen {
		str := fmt.Sprintf("the coinbase transaction witness nonce "+
			"has %d bytes when it must be %d bytes",
			len(witnessNonce), CoinbaseWitnessDataLen)
		return ruleError(ErrInvalidWitnessCommitment, str)
	}

	witnessMerkleTree := BuildMerkleTreeStore(blk.Transactions(), true)
	witnessMerkleRoot := witnessMerkleTree[len(witnessMerkleTree)-1]

	var witnessPreimage [wire.HashSize * 2]byte
	copy(witnessPreimage[:], witnessMerkleRoot[:])
	copy(witnessPreimage[wire.HashSize:], witnessNonce)

	computedCommitment := wire.DoubleHashB(witnessPreimage[:])
	if !bytes.Equal(computedCommitment, witnessCommitment) {
		str := fmt.Sprintf("witness commitment does not match: "+
			"computed %v, coinbase includes %v", computedCommitment,
			witnessCommitment)
		return ruleError(ErrWitnessCommitmentMismatch, str)
	}

	return nil
}

func BuildMerkleTreeStoreForProposal(proposals *wire.ProposalArea) []*wire.Hash {
	nextPoT := nextPowerOfTwo(int(proposals.AllCount))
	var arraySize int
	if nextPoT == 0 {
		arraySize = 1
	} else {
		arraySize = nextPoT*2 - 1
	}
	merkles := make([]*wire.Hash, arraySize)

	var length int
	if proposals.PunishmentCount == 0 {
		length = int(proposals.AllCount) + 1
	} else {
		length = int(proposals.AllCount)
	}

	for i := 0; i < length; i++ {
		if proposals.PunishmentCount == 0 {
			ph := wire.NewPlaceHolder()
			p := wire.NewProposalFromPlaceHolder(ph)
			sha := p.Hash()
			merkles[i] = &sha
		} else {
			for _, punishment := range proposals.PunishmentArea {
				sha := punishment.Hash()
				merkles[i] = &sha
			}
		}
		for _, other := range proposals.OtherArea {
			sha := other.Hash()
			merkles[i] = &sha
		}
	}

	offset := nextPoT
	for i := 0; i < arraySize-1; i += 2 {
		switch {
		case merkles[i] == nil:
			merkles[offset] = nil

		case merkles[i+1] == nil:
			newHash := HashMerkleBranches(merkles[i], merkles[i])
			merkles[offset] = newHash

		default:
			newHash := HashMerkleBranches(merkles[i], merkles[i+1])
			merkles[offset] = newHash
		}
		offset++
	}

	return merkles
}
