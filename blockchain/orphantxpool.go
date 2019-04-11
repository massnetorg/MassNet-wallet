package blockchain

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/wire"
)

type OrphanTxPool struct {
	orphans       map[wire.Hash]*massutil.Tx
	orphansByPrev map[wire.Hash]map[wire.Hash]*massutil.Tx
}

func newOrphanTxPool() *OrphanTxPool {
	return &OrphanTxPool{
		orphans:       make(map[wire.Hash]*massutil.Tx),
		orphansByPrev: make(map[wire.Hash]map[wire.Hash]*massutil.Tx),
	}
}

// removeOrphan is the internal function which implements the public
// RemoveOrphan.  See the comment for RemoveOrphan for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (otp *OrphanTxPool) removeOrphan(txHash *wire.Hash) {
	tx, exists := otp.orphans[*txHash]
	if !exists {
		return
	}

	for _, txIn := range tx.MsgTx().TxIn {
		originTxHash := txIn.PreviousOutPoint.Hash
		if orphans, exists := otp.orphansByPrev[originTxHash]; exists {
			delete(orphans, *tx.Hash())

			if len(orphans) == 0 {
				delete(otp.orphansByPrev, originTxHash)
			}
		}
	}

	delete(otp.orphans, *txHash)
}

// ShaHashToBig converts a wire.Hash into a big.Int that can be used to
// perform math comparisons.
func ShaHashToBig(hash *wire.Hash) *big.Int {
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

// limitNumOrphans limits the number of orphan transactions by evicting a random
// orphan if adding a new one would cause it to overflow the max allowed.
//
// This function MUST be called with the mempool lock held (for writes).
func (otp *OrphanTxPool) limitNumOrphans() error {
	if len(otp.orphans)+1 > config.MaxOrphanTxs && config.MaxOrphanTxs > 0 {
		randHashBytes := make([]byte, wire.HashSize)
		_, err := rand.Read(randHashBytes)
		if err != nil {
			return err
		}
		randHashNum := new(big.Int).SetBytes(randHashBytes)

		var foundHash *wire.Hash
		for txHash := range otp.orphans {
			if foundHash == nil {
				foundHash = &txHash
			}
			txHashNum := ShaHashToBig(&txHash)
			if txHashNum.Cmp(randHashNum) > 0 {
				foundHash = &txHash
				break
			}
		}

		otp.removeOrphan(foundHash)
	}

	return nil
}

func (otp *OrphanTxPool) addOrphan(tx *massutil.Tx) {
	otp.limitNumOrphans()

	otp.orphans[*tx.Hash()] = tx
	for _, txIn := range tx.MsgTx().TxIn {
		originTxHash := txIn.PreviousOutPoint.Hash
		if _, exists := otp.orphansByPrev[originTxHash]; !exists {
			otp.orphansByPrev[originTxHash] =
				make(map[wire.Hash]*massutil.Tx)
		}
		otp.orphansByPrev[originTxHash][*tx.Hash()] = tx
	}

	logging.CPrint(logging.DEBUG, "Stored orphan transaction", logging.LogFormat{
		"orphan transcation":     tx.Hash(),
		"total orphan tx number": len(otp.orphans),
	})
}

func (otp *OrphanTxPool) maybeAddOrphan(tx *massutil.Tx) error {
	serializedLen := tx.MsgTx().SerializeSize()
	if serializedLen > maxOrphanTxSize {
		str := fmt.Sprintf("orphan transaction size of %d bytes is "+
			"larger than max allowed size of %d bytes",
			serializedLen, maxOrphanTxSize)
		return txRuleError(wire.RejectNonstandard, str)
	}

	otp.addOrphan(tx)

	return nil
}

// isOrphanInPool returns whether or not the passed transaction already exists
// in the orphan pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (otp *OrphanTxPool) isOrphanInPool(hash *wire.Hash) bool {
	if _, exists := otp.orphans[*hash]; exists {
		return true
	}

	return false
}
