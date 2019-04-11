// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"

	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/wire"
)

// TxData contains contextual information about transactions such as which block
// they were found in and whether or not the outputs are spent.
type TxData struct {
	Tx          *massutil.Tx
	Hash        *wire.Hash
	BlockHeight int32
	Spent       []bool
	Err         error
}

// TxStore is used to store transactions needed by other transactions for things
// such as script validation and double spend prevention.  This also allows the
// transaction data to be treated as a view since it can contain the information
// from the point-of-view of different points in the chain.
type TxStore map[wire.Hash]*TxData

// connectTransactions updates the passed map by applying transaction and
// spend information for all the transactions in the passed block.  Only
// transactions in the passed map are updated.
func connectTransactions(txStore TxStore, block *massutil.Block) error {
	for _, tx := range block.Transactions() {
		msgTx := tx.MsgTx()
		if txD, exists := txStore[*tx.Hash()]; exists {
			txD.Tx = tx
			txD.BlockHeight = block.Height()
			txD.Spent = make([]bool, len(msgTx.TxOut))
			txD.Err = nil
		}

		if IsCoinBaseTx(msgTx) {
			continue
		} else {
			for _, txIn := range msgTx.TxIn {
				originHash := &txIn.PreviousOutPoint.Hash
				originIndex := txIn.PreviousOutPoint.Index
				if originTx, exists := txStore[*originHash]; exists {
					if originIndex > uint32(len(originTx.Spent)) {
						continue
					}
					originTx.Spent[originIndex] = true
				}
			}
		}

	}

	return nil
}

// disconnectTransactions updates the passed map by undoing transaction and
// spend information for all transactions in the passed block.  Only
// transactions in the passed map are updated.
func disconnectTransactions(txStore TxStore, block *massutil.Block) error {
	for _, tx := range block.Transactions() {
		if txD, exists := txStore[*tx.Hash()]; exists {
			txD.Tx = nil
			txD.BlockHeight = 0
			txD.Spent = nil
			txD.Err = database.ErrTxShaMissing
		}

		for _, txIn := range tx.MsgTx().TxIn {
			originHash := &txIn.PreviousOutPoint.Hash
			originIndex := txIn.PreviousOutPoint.Index
			originTx, exists := txStore[*originHash]
			if exists && originTx.Tx != nil && originTx.Err == nil {
				if originIndex > uint32(len(originTx.Spent)) {
					continue
				}
				originTx.Spent[originIndex] = false
			}
		}
	}

	return nil
}

// fetchTxStoreMain fetches transaction data about the provided set of
// transactions from the point of view of the end of the main chain.  It takes
// a flag which specifies whether or not fully spent transaction should be
// included in the results.
func fetchTxStoreMain(db database.Db, txSet map[wire.Hash]struct{}, includeSpent bool) TxStore {
	txStore := make(TxStore)
	if len(txSet) == 0 {
		return txStore
	}

	txList := make([]*wire.Hash, 0, len(txSet))
	for hash := range txSet {
		hashCopy := hash
		txStore[hash] = &TxData{Hash: &hashCopy, Err: database.ErrTxShaMissing}
		txList = append(txList, &hashCopy)
	}

	var txReplyList []*database.TxListReply
	if includeSpent {
		txReplyList = db.FetchTxByShaList(txList)
	} else {
		txReplyList = db.FetchUnSpentTxByShaList(txList)
	}
	for _, txReply := range txReplyList {
		txD, ok := txStore[*txReply.Sha]
		if !ok {
			continue
		}

		txD.Err = txReply.Err
		if txReply.Err == nil {
			txD.Tx = massutil.NewTx(txReply.Tx)
			txD.BlockHeight = txReply.Height
			txD.Spent = make([]bool, len(txReply.TxSpent))
			copy(txD.Spent, txReply.TxSpent)
		}
	}

	return txStore
}

// fetchTxStore fetches transaction data about the provided set of transactions
// from the point of view of the given node.  For example, a given node might
// be down a side chain where a transaction hasn't been spent from its point of
// view even though it might have been spent in the main chain (or another side
// chain).  Another scenario is where a transaction exists from the point of
// view of the main chain, but doesn't exist in a side chain that branches
// before the block that contains the transaction on the main chain.
func (b *BlockChain) fetchTxStore(node *blockNode, txSet map[wire.Hash]struct{}) (TxStore, error) {
	prevNode, err := b.getPrevNodeFromNode(node)
	if err != nil {
		return nil, err
	}

	if b.bestChain == nil || (prevNode != nil && prevNode.hash.IsEqual(b.bestChain.hash)) {
		txStore := fetchTxStoreMain(b.db, txSet, false)
		return txStore, nil
	}

	txStore := fetchTxStoreMain(b.db, txSet, true)

	detachNodes, attachNodes := b.getReorganizeNodes(prevNode)
	for e := detachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block, err := b.db.FetchBlockBySha(n.hash)
		if err != nil {
			return nil, err
		}

		disconnectTransactions(txStore, block)
	}

	if attachNodes.Len() == 0 {
		return txStore, nil
	}

	for e := attachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block, exists := b.blockCache[*n.hash]
		if !exists {
			return nil, fmt.Errorf("unable to find block %v in "+
				"side chain cache for transaction search",
				n.hash)
		}

		connectTransactions(txStore, block)
	}

	return txStore, nil
}

// fetchInputTransactions fetches the input transactions referenced by the
// transactions in the given block from its point of view.  See fetchTxList
// for more details on what the point of view entails.
func (b *BlockChain) fetchInputTransactions(node *blockNode, block *massutil.Block) (TxStore, error) {
	txInFlight := map[wire.Hash]int{}
	transactions := block.Transactions()
	for i, tx := range transactions {
		txInFlight[*tx.Hash()] = i
	}

	txNeededSet := make(map[wire.Hash]struct{})
	txStore := make(TxStore)
	for i, tx := range transactions[1:] {
		for _, txIn := range tx.MsgTx().TxIn {
			originHash := &txIn.PreviousOutPoint.Hash
			txD := &TxData{Hash: originHash, Err: database.ErrTxShaMissing}
			txStore[*originHash] = txD

			if inFlightIndex, ok := txInFlight[*originHash]; ok &&
				i >= inFlightIndex {

				originTx := transactions[inFlightIndex]
				txD.Tx = originTx
				txD.BlockHeight = node.height
				txD.Spent = make([]bool, len(originTx.MsgTx().TxOut))
				txD.Err = nil
			} else {
				txNeededSet[*originHash] = struct{}{}
			}
		}
	}

	txNeededStore, err := b.fetchTxStore(node, txNeededSet)
	if err != nil {
		return nil, err
	}

	for _, txD := range txNeededStore {
		txStore[*txD.Hash] = txD
	}

	return txStore, nil
}

// FetchTransactionStore fetches the input transactions referenced by the
// passed transaction from the point of view of the end of the main chain.  It
// also attempts to fetch the transaction itself so the returned TxStore can be
// examined for duplicate transactions.
func (b *BlockChain) FetchTransactionStore(tx *massutil.Tx, includeSpent bool) (TxStore, error) {
	txNeededSet := make(map[wire.Hash]struct{})
	txNeededSet[*tx.Hash()] = struct{}{}
	for _, txIn := range tx.MsgTx().TxIn {
		txNeededSet[txIn.PreviousOutPoint.Hash] = struct{}{}
	}

	txStore := fetchTxStoreMain(b.db, txNeededSet, includeSpent)
	return txStore, nil
}
