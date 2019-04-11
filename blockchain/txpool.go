// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"container/list"
	"fmt"
	"massnet.org/mass-wallet/consensus"
	"math"
	"sync"
	"time"

	"massnet.org/mass-wallet/config"

	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"
)

const (
	mempoolHeight = 0x7fffffff

	maxOrphanTxSize = 5000

	maxSigOpsPerTx = MaxSigOpsPerBlock / 5

	defaultBlockPrioritySize = 50000
)

// TxDesc is a descriptor containing a transaction in the mempool and the
// metadata we store about it.
type TxDesc struct {
	Tx               *massutil.Tx // Transaction.
	Added            time.Time    // Time when added to pool.
	Height           int32        // Blockheight when added to pool.
	Fee              int64        // Transaction fees.
	startingPriority float64      // Priority when added to the pool.
}

// TxPool is used as a source of transactions that need to be mined into
// blocks and relayed to other peers.  It is safe for concurrent access from
// multiple peers.
type TxPool struct {
	sync.RWMutex
	pool          map[wire.Hash]*TxDesc
	orphans       map[wire.Hash]*massutil.Tx
	orphansByPrev map[wire.Hash]map[wire.Hash]*massutil.Tx
	orphanTxPool  *OrphanTxPool
	addrindex     map[string]map[wire.Hash]struct{} // maps address to txs
	outpoints     map[wire.OutPoint]*massutil.Tx
	lastUpdated   time.Time // last time pool was updated
	pennyTotal    float64   // exponentially decaying total for penny spends.
	lastPennyUnix int64     // unix time of last ``penny spend''
	db            database.Db
	BlockChain    *BlockChain
	sigCache      *txscript.SigCache
	hashCache     *txscript.HashCache
	timeSource    MedianTimeSource
	NewTxCh       chan *massutil.Tx
}

// RemoveOrphan removes the passed orphan transaction from the orphan pool and
// previous orphan index.
//
// This function is safe for concurrent access.
func (tp *TxPool) RemoveOrphan(txHash *wire.Hash) {
	tp.Lock()
	tp.orphanTxPool.removeOrphan(txHash)
	tp.Unlock()
}

// isTransactionInPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (tp *TxPool) isTransactionInPool(hash *wire.Hash) bool {
	if _, exists := tp.pool[*hash]; exists {
		return true
	}

	return false
}

// IsTransactionInPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function is safe for concurrent access.
func (tp *TxPool) IsTransactionInPool(hash *wire.Hash) bool {
	tp.RLock()
	defer tp.RUnlock()

	return tp.isTransactionInPool(hash)
}

// IsOrphanInPool returns whether or not the passed transaction already exists
// in the orphan pool.
//
// This function is safe for concurrent access.
func (tp *TxPool) IsOrphanInPool(hash *wire.Hash) bool {
	tp.RLock()
	defer tp.RUnlock()

	return tp.orphanTxPool.isOrphanInPool(hash)
}

// haveTransaction returns whether or not the passed transaction already exists
// in the main pool or in the orphan pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (tp *TxPool) haveTransaction(hash *wire.Hash) bool {
	return tp.isTransactionInPool(hash) || tp.orphanTxPool.isOrphanInPool(hash)
}

// HaveTransaction returns whether or not the passed transaction already exists
// in the main pool or in the orphan pool.
//
// This function is safe for concurrent access.
func (tp *TxPool) HaveTransaction(hash *wire.Hash) bool {
	tp.RLock()
	defer tp.RUnlock()

	return tp.haveTransaction(hash)
}

// removeTransaction is the internal function which implements the public
// RemoveTransaction.  See the comment for RemoveTransaction for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (tp *TxPool) removeTransaction(tx *massutil.Tx, removeRedeemers bool) {
	txHash := tx.Hash()
	if removeRedeemers {
		for i := uint32(0); i < uint32(len(tx.MsgTx().TxOut)); i++ {
			outpoint := wire.NewOutPoint(txHash, i)
			if txRedeemer, exists := tp.outpoints[*outpoint]; exists {
				tp.removeTransaction(txRedeemer, true)
			}
		}
	}

	if txDesc, exists := tp.pool[*txHash]; exists {
		if config.AddrIndex {
			tp.removeTransactionFromAddrIndex(tx)
		}

		for _, txIn := range txDesc.Tx.MsgTx().TxIn {
			delete(tp.outpoints, txIn.PreviousOutPoint)
		}
		delete(tp.pool, *txHash)
		tp.lastUpdated = time.Now()
	}

}

// removeTransactionFromAddrIndex removes the passed transaction from our
// address based index.
//
// This function MUST be called with the mempool lock held (for writes).
func (tp *TxPool) removeTransactionFromAddrIndex(tx *massutil.Tx) error {
	previousOutputScripts, _ := tp.fetchReferencedOutputScripts(tx)
	for _, pkScript := range previousOutputScripts {
		tp.removeScriptFromAddrIndex(pkScript, tx)
	}

	for _, txOut := range tx.MsgTx().TxOut {
		tp.removeScriptFromAddrIndex(txOut.PkScript, tx)
	}

	return nil
}

// removeScriptFromAddrIndex dissociates the address encoded by the
// passed pkScript from the passed tx in our address based tx index.
//
// This function MUST be called with the mempool lock held (for writes).
func (tp *TxPool) removeScriptFromAddrIndex(pkScript []byte, tx *massutil.Tx) error {
	_, addresses, _, _, err := txscript.ExtractPkScriptAddrs(pkScript,
		&config.ChainParams)
	if err != nil {
		logging.CPrint(logging.ERROR, "Unable to extract encoded addresses from script for addrindex",
			logging.LogFormat{
				"addrindex": err,
			})
		return err
	}
	for _, addr := range addresses {
		delete(tp.addrindex[addr.EncodeAddress()], *tx.Hash())
	}

	return nil
}

// RemoveTransaction removes the passed transaction from the mempool. If
// removeRedeemers flag is set, any transactions that redeem outputs from the
// removed transaction will also be removed recursively from the mempool, as
// they would otherwise become orphan.
//
// This function is safe for concurrent access.
func (tp *TxPool) RemoveTransaction(tx *massutil.Tx, removeRedeemers bool) {
	tp.Lock()
	defer tp.Unlock()

	tp.removeTransaction(tx, removeRedeemers)
}

// RemoveDoubleSpends removes all transactions which spend outputs spent by the
// passed transaction from the memory pool.  Removing those transactions then
// leads to removing all transactions which rely on them, recursively.  This is
// necessary when a block is connected to the main chain because the block may
// contain transactions which were previously unknown to the memory pool
//
// This function is safe for concurrent access.
func (tp *TxPool) RemoveDoubleSpends(tx *massutil.Tx) {
	tp.Lock()
	defer tp.Unlock()

	for _, txIn := range tx.MsgTx().TxIn {
		if txRedeemer, ok := tp.outpoints[txIn.PreviousOutPoint]; ok {
			if !txRedeemer.Hash().IsEqual(tx.Hash()) {
				tp.removeTransaction(txRedeemer, true)
			}
		}
	}
}

// addTransaction adds the passed transaction to the memory pool.  It should
// not be called directly as it doesn't perform any validation.  This is a
// helper for maybeAcceptTransaction.
//
// This function MUST be called with the mempool lock held (for writes).
func (tp *TxPool) addTransaction(tx *massutil.Tx, height int32, fee int64) error {
	if tp.pool == nil {
		return errors.New("the txpool is nil")
	}

	tp.pool[*tx.Hash()] = &TxDesc{
		Tx:     tx,
		Added:  time.Now(),
		Height: height,
		Fee:    fee,
	}

	for _, txIn := range tx.MsgTx().TxIn {
		tp.outpoints[txIn.PreviousOutPoint] = tx
	}
	tp.lastUpdated = time.Now()

	tp.NewTxCh <- tx

	if config.AddrIndex {
		err := tp.addTransactionToAddrIndex(tx)
		return err
	}
	return nil
}

// addTransactionToAddrIndex adds all addresses related to the transaction to
// our in-memory wallet index. Note that this wallet is only populated when
// we're running with the optional address index activated.
//
// This function MUST be called with the mempool lock held (for writes).
func (tp *TxPool) addTransactionToAddrIndex(tx *massutil.Tx) error {
	previousOutScripts, _ := tp.fetchReferencedOutputScripts(tx)
	for _, pkScript := range previousOutScripts {
		tp.indexScriptAddressToTx(pkScript, tx)
	}

	for _, txOut := range tx.MsgTx().TxOut {
		tp.indexScriptAddressToTx(txOut.PkScript, tx)
	}

	return nil
}

// fetchReferencedOutputScripts looks up and returns all the scriptPubKeys
// referenced by inputs of the passed transaction.
//
// This function MUST be called with the mempool lock held (for reads).
func (tp *TxPool) fetchReferencedOutputScripts(tx *massutil.Tx) ([][]byte, error) {
	txStore, _ := tp.FetchInputTransactions(tx, false)
	previousOutScripts := make([][]byte, 0, len(tx.MsgTx().TxIn))
	for _, txIn := range tx.MsgTx().TxIn {
		outPoint := txIn.PreviousOutPoint
		if txStore[outPoint.Hash].Err == nil {
			referencedOutPoint := txStore[outPoint.Hash].Tx.MsgTx().TxOut[outPoint.Index]
			previousOutScripts = append(previousOutScripts, referencedOutPoint.PkScript)
		}
	}
	return previousOutScripts, nil
}

// indexScriptByAddress alters our wallet index by indexing the payment wallet
// encoded by the passed scriptPubKey to the passed transaction.
//
// This function MUST be called with the mempool lock held (for writes).
func (tp *TxPool) indexScriptAddressToTx(pkScript []byte, tx *massutil.Tx) error {
	_, addresses, _, _, err := txscript.ExtractPkScriptAddrs(pkScript,
		&config.ChainParams)
	if err != nil {
		logging.CPrint(logging.ERROR, "Unable to extract encoded addresses from script for addrindex ",
			logging.LogFormat{
				"addrindex": err,
			})
		return err
	}

	for _, addr := range addresses {
		if tp.addrindex[addr.EncodeAddress()] == nil {
			tp.addrindex[addr.EncodeAddress()] = make(map[wire.Hash]struct{})
		}
		tp.addrindex[addr.EncodeAddress()][*tx.Hash()] = struct{}{}
	}

	return nil
}

// StartingPriority calculates the priority of this tx descriptor's underlying
// transaction relative to when it was first added to the mempool.  The result
// is lazily computed and then cached for subsequent function calls.
func (txD *TxDesc) StartingPriority(txStore TxStore) float64 {
	if txD.startingPriority != float64(0) {
		return txD.startingPriority
	}

	inputAge := calcInputValueAge(txD, txStore, txD.Height)
	txD.startingPriority = calcPriority(txD.Tx, inputAge)

	return txD.startingPriority
}

// CurrentPriority calculates the current priority of this tx descriptor's
// underlying transaction relative to the next block height.
func (txD *TxDesc) CurrentPriority(txStore TxStore, nextBlockHeight int32) float64 {
	inputAge := calcInputValueAge(txD, txStore, nextBlockHeight)
	return calcPriority(txD.Tx, inputAge)
}

// checkPoolDoubleSpend checks whether or not the passed transaction is
// attempting to spend coins already spent by other transactions in the pool.
// Note it does not check for double spends against transactions already in the
// main chain.
//
// This function MUST be called with the mempool lock held (for reads).
func (tp *TxPool) checkPoolDoubleSpend(tx *massutil.Tx) error {
	for _, txIn := range tx.MsgTx().TxIn {
		if txR, exists := tp.outpoints[txIn.PreviousOutPoint]; exists {
			logging.CPrint(logging.ERROR, "output already spent by transaction in the memory pool",
				logging.LogFormat{
					"output":      txIn.PreviousOutPoint,
					"transcation": txR.Hash(),
				})
			return txRuleError(wire.RejectDuplicate, "output already spent by transaction in the memory pool")
		}
	}

	return nil
}

// FetchInputTransactions fetches the input transactions referenced by the
// passed transaction.  First, it fetches from the main chain, then it tries to
// fetch any missing inputs from the transaction pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (tp *TxPool) FetchInputTransactions(tx *massutil.Tx, includeSpent bool) (TxStore, error) {
	txStore, err := tp.BlockChain.FetchTransactionStore(tx, includeSpent)
	if err != nil {
		return nil, err
	}

	for _, txD := range txStore {
		if txD.Err == database.ErrTxShaMissing || txD.Tx == nil {
			if poolTxDesc, exists := tp.pool[*txD.Hash]; exists {
				poolTx := poolTxDesc.Tx
				txD.Tx = poolTx
				txD.BlockHeight = mempoolHeight
				txD.Spent = make([]bool, len(poolTx.MsgTx().TxOut))
				txD.Err = nil
			}
		}
	}

	return txStore, nil
}

// FetchTransaction returns the requested transaction from the transaction pool.
// This only fetches from the main transaction pool and does not include
// orphans.
//
// This function is safe for concurrent access.
func (tp *TxPool) FetchTransaction(txHash *wire.Hash) (*massutil.Tx, error) {
	tp.RLock()
	defer tp.RUnlock()

	if txDesc, exists := tp.pool[*txHash]; exists {
		return txDesc.Tx, nil
	}

	return nil, fmt.Errorf("transaction is not in the pool")
}

// FilterTransactionsByAddress returns all transactions currently in the
// mempool that either create an output to the passed address or spend a
// previously created ouput to the address.
func (tp *TxPool) FilterTransactionsByAddress(addr massutil.Address) ([]*massutil.Tx, error) {
	tp.RLock()
	defer tp.RUnlock()

	if txs, exists := tp.addrindex[addr.EncodeAddress()]; exists {
		addressTxs := make([]*massutil.Tx, 0, len(txs))
		for txHash := range txs {
			if tx, exists := tp.pool[txHash]; exists {
				addressTxs = append(addressTxs, tx.Tx)
			}
		}
		return addressTxs, nil
	}

	return nil, fmt.Errorf("address does not have any transactions in the pool")
}

// maybeAcceptTransaction is the internal function which implements the public
// MaybeAcceptTransaction.  See the comment for MaybeAcceptTransaction for
// more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (tp *TxPool) maybeAcceptTransaction(tx *massutil.Tx, isNew, rateLimit bool) ([]*wire.Hash, error) {
	txHash := tx.Hash()
	if tp.haveTransaction(txHash) {
		str := fmt.Sprintf("already have transaction %v", txHash)
		return nil, txRuleError(wire.RejectAlreadyExists, str)
	}

	err := CheckTransactionSanity(tx)
	if err != nil {
		if cerr, ok := err.(RuleError); ok {
			return nil, chainRuleError(cerr)
		}
		return nil, err
	}

	if IsCoinBase(tx) {
		str := fmt.Sprintf("transaction %v is an individual coinbase",
			txHash)
		return nil, txRuleError(wire.RejectInvalid, str)
	}

	if tx.MsgTx().LockTime > math.MaxInt32 {
		str := fmt.Sprintf("transaction %v has a lock time after "+
			"2038 which is not accepted yet", txHash)
		return nil, txRuleError(wire.RejectNonstandard, str)
	}

	_, curHeight, err := tp.db.NewestSha()
	if err != nil {
		return nil, err
	}
	nextBlockHeight := curHeight + 1
	node := tp.BlockChain.bestChain
	medianTimePast, err := tp.BlockChain.calcPastMedianTime(node)

	if !config.ChainParams.RelayNonStdTxs {
		minRelayTxFee, err := massutil.NewAmount(config.MinRelayTxFee)
		if err != nil {
			return nil, err
		}
		err = checkTransactionStandard(tx, nextBlockHeight,
			tp.timeSource, minRelayTxFee)
		if err != nil {
			rejectCode, found := extractRejectCode(err)
			if !found {
				rejectCode = wire.RejectNonstandard
			}
			str := fmt.Sprintf("transaction %v is not standard: %v",
				txHash, err)
			return nil, txRuleError(rejectCode, str)
		}
	}

	err = tp.checkPoolDoubleSpend(tx)
	if err != nil {
		return nil, err
	}

	txStore, _ := tp.FetchInputTransactions(tx, false)

	if txD, exists := txStore[*txHash]; exists && txD.Err == nil {

		for _, isOutputSpent := range txD.Spent {
			if !isOutputSpent {
				return nil, txRuleError(wire.RejectAlreadyExists,
					"transaction already exists")
			}
		}
	}

	delete(txStore, *txHash)

	var missingParents []*wire.Hash
	for _, txD := range txStore {
		if txD.Err == database.ErrTxShaMissing {
			missingParents = append(missingParents, txD.Hash)
		}
	}
	if len(missingParents) > 0 {
		return missingParents, nil
	}

	sequenceLock, err := tp.BlockChain.CalcSequenceLock(tx, txStore)
	if err != nil {
		if cerr, ok := err.(RuleError); ok {
			return nil, RuleError(cerr)
		}
		return nil, err
	}
	if !SequenceLockActive(sequenceLock, nextBlockHeight,
		medianTimePast) {
		return nil, txRuleError(wire.RejectNonstandard,
			"transaction's sequence locks on inputs not met")
	}

	txFee, err := CheckTransactionInputs(tx, nextBlockHeight, txStore)

	if err != nil {
		if cerr, ok := err.(RuleError); ok {
			return nil, chainRuleError(cerr)
		}
		return nil, err
	}

	if !config.ChainParams.RelayNonStdTxs {
		err := checkInputsStandard(tx, txStore)
		if err != nil {
			rejectCode, found := extractRejectCode(err)
			if !found {
				rejectCode = wire.RejectNonstandard
			}
			str := fmt.Sprintf("transaction %v has a non-standard "+
				"input: %v", txHash, err)
			return nil, txRuleError(rejectCode, str)
		}
	}

	numSigOps := CountSigOps(tx)
	if numSigOps > maxSigOpsPerTx {
		str := fmt.Sprintf("transaction %v has too many sigops: %d > %d",
			txHash, numSigOps, maxSigOpsPerTx)
		return nil, txRuleError(wire.RejectNonstandard, str)
	}

	serializedSize := int64(tx.MsgTx().SerializeSize())
	minRelayTxFee, err := massutil.NewAmount(config.MinRelayTxFee)
	if err != nil {
		return nil, err
	}
	minFee := calcMinRequiredTxRelayFee(serializedSize, minRelayTxFee)
	if serializedSize >= (defaultBlockPrioritySize-1000) && txFee < minFee {
		str := fmt.Sprintf("transaction %v has %d fees which is under "+
			"the required amount of %d", txHash, txFee,
			minFee)
		return nil, txRuleError(wire.RejectInsufficientFee, str)
	}

	if isNew && !config.NoRelayPriority && txFee < minFee {
		txD := &TxDesc{
			Tx:     tx,
			Added:  time.Now(),
			Height: curHeight,
			Fee:    txFee,
		}
		currentPriority := txD.CurrentPriority(txStore, nextBlockHeight)
		if currentPriority <= consensus.MinHighPriority {
			str := fmt.Sprintf("transaction %v has insufficient "+
				"priority (%g <= %g)", txHash,
				currentPriority, consensus.MinHighPriority)
			return nil, txRuleError(wire.RejectInsufficientFee, str)
		}
	}

	if rateLimit && txFee < minFee {
		nowUnix := time.Now().Unix()
		tp.pennyTotal *= math.Pow(1.0-1.0/600.0,
			float64(nowUnix-tp.lastPennyUnix))
		tp.lastPennyUnix = nowUnix

		if tp.pennyTotal >= config.FreeTxRelayLimit*10*1000 {
			str := fmt.Sprintf("transaction %v has been rejected "+
				"by the rate limiter due to low fees", txHash)
			return nil, txRuleError(wire.RejectInsufficientFee, str)
		}
		oldTotal := tp.pennyTotal

		tp.pennyTotal += float64(serializedSize)
		logging.CPrint(logging.TRACE, "rate limit", logging.LogFormat{
			"curTotal":  oldTotal,
			"nextTotal": tp.pennyTotal,
			"limit":     config.FreeTxRelayLimit * 10 * 1000,
		})
	}

	err = ValidateTransactionScripts(tx, txStore,
		txscript.StandardVerifyFlags, tp.sigCache,
		tp.hashCache)
	if err != nil {
		if cerr, ok := err.(RuleError); ok {
			return nil, chainRuleError(cerr)
		}
		return nil, err
	}

	err = tp.addTransaction(tx, curHeight, txFee)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// MaybeAcceptTransaction is the main workhorse for handling insertion of new
// free-standing transactions into a memory pool.  It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, detecting orphan transactions, and insertion into the memory pool.
//
// If the transaction is an orphan (missing parent transactions), the
// transaction is NOT added to the orphan pool, but each unknown referenced
// parent is returned.  Use ProcessTransaction instead if new orphans should
// be added to the orphan pool.
//
// This function is safe for concurrent access.
func (tp *TxPool) MaybeAcceptTransaction(tx *massutil.Tx, isNew, rateLimit bool) ([]*wire.Hash, error) {
	tp.Lock()
	defer tp.Unlock()

	return tp.maybeAcceptTransaction(tx, isNew, rateLimit)
}

// processOrphans is the internal function which implements the public
// ProcessOrphans.  See the comment for ProcessOrphans for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (tp *TxPool) processOrphans(hash *wire.Hash) {
	processHashes := list.New()
	processHashes.PushBack(hash)
	for processHashes.Len() > 0 {
		firstElement := processHashes.Remove(processHashes.Front())
		processHash := firstElement.(*wire.Hash)

		orphans, exists := tp.orphanTxPool.orphansByPrev[*processHash]
		if !exists || orphans == nil {
			continue
		}

		for _, tx := range orphans {
			orphanHash := tx.Hash()
			tp.orphanTxPool.removeOrphan(orphanHash)

			missingParents, err := tp.maybeAcceptTransaction(tx,
				true, true)
			if err != nil {
				logging.CPrint(logging.DEBUG, "Unable to move orphan transaction to mempool", logging.LogFormat{
					"orphan transaction": tx.Hash(),
					"err":                err,
				})
				continue
			}

			if len(missingParents) > 0 {
				tp.orphanTxPool.addOrphan(tx)
				continue
			}

			processHashes.PushBack(orphanHash)
		}
	}
}

// ProcessOrphans determines if there are any orphans which depend on the passed
// transaction hash (it is possible that they are no longer orphans) and
// potentially accepts them to the memory pool.  It repeats the process for the
// newly accepted transactions (to detect further orphans which may no longer be
// orphans) until there are no more.
//
// This function is safe for concurrent access.
func (tp *TxPool) ProcessOrphans(hash *wire.Hash) {
	tp.Lock()
	tp.processOrphans(hash)
	tp.Unlock()
}

// ProcessTransaction is the main workhorse for handling insertion of new
// free-standing transactions into the memory pool.  It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, orphan transaction handling, and insertion into the memory pool.
//
// This function is safe for concurrent access.
func (tp *TxPool) ProcessTransaction(tx *massutil.Tx, allowOrphan, rateLimit bool) (bool, error) {
	tp.Lock()
	defer tp.Unlock()

	missingParents, err := tp.maybeAcceptTransaction(tx, true, rateLimit)
	if err != nil {
		return false, err
	}

	var isOrphan bool

	if len(missingParents) == 0 {

		isOrphan = false
		tp.processOrphans(tx.Hash())
	} else {

		isOrphan = true
		if !allowOrphan {
			str := fmt.Sprintf("orphan transaction %v references "+
				"outputs of unknown or fully-spent "+
				"transaction %v", tx.Hash(), missingParents[0])
			return isOrphan, txRuleError(wire.RejectDuplicate, str)
		}

		err := tp.orphanTxPool.maybeAddOrphan(tx)
		if err != nil {
			return isOrphan, err
		}
	}

	return isOrphan, nil
}

// Count returns the number of transactions in the main pool.  It does not
// include the orphan pool.
//
// This function is safe for concurrent access.
func (tp *TxPool) Count() int {
	tp.RLock()
	defer tp.RUnlock()

	return len(tp.pool)
}

// TxShas returns a slice of hashes for all of the transactions in the memory
// pool.
//
// This function is safe for concurrent access.
func (tp *TxPool) TxShas() []*wire.Hash {
	tp.RLock()
	defer tp.RUnlock()

	hashes := make([]*wire.Hash, len(tp.pool))
	i := 0
	for hash := range tp.pool {
		hashCopy := hash
		hashes[i] = &hashCopy
		i++
	}

	return hashes
}

// TxDescs returns a slice of descriptors for all the transactions in the pool.
// The descriptors are to be treated as read only.
//
// This function is safe for concurrent access.
func (tp *TxPool) TxDescs() []*TxDesc {
	tp.RLock()
	defer tp.RUnlock()

	descs := make([]*TxDesc, len(tp.pool))
	i := 0
	for _, desc := range tp.pool {
		descs[i] = desc
		i++
	}

	return descs
}

// LastUpdated returns the last time a transaction was added to or removed from
// the main pool.  It does not include the orphan pool.
//
// This function is safe for concurrent access.
func (tp *TxPool) LastUpdated() time.Time {
	tp.RLock()
	defer tp.RUnlock()

	return tp.lastUpdated
}

// newTxMemPool returns a new memory pool for validating and storing standalone
// transactions until they are mined into a block.
func NewTxMemPool(db database.Db, bc *BlockChain, sigCache *txscript.SigCache, hashCache *txscript.HashCache, timeSource MedianTimeSource) *TxPool {
	memPool := &TxPool{
		db:           db,
		sigCache:     sigCache,
		hashCache:    hashCache,
		timeSource:   timeSource,
		BlockChain:   bc,
		pool:         make(map[wire.Hash]*TxDesc),
		orphanTxPool: newOrphanTxPool(),
		outpoints:    make(map[wire.OutPoint]*massutil.Tx),
	}
	if config.AddrIndex {
		memPool.addrindex = make(map[string]map[wire.Hash]struct{})
	}
	return memPool
}
