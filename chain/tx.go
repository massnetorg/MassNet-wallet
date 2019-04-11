package chain

import (
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/massutil"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx *massutil.Tx) (bool, error) {
	if ok := c.blockChain.TxPool.HaveTransaction(tx.Hash()); ok {
		return false, nil
	}

	return c.blockChain.TxPool.ProcessTransaction(tx, true, false)
}
