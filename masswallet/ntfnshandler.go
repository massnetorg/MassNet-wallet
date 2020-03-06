package masswallet

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/keystore"
	"massnet.org/mass-wallet/masswallet/txmgr"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/wire"
)

// const ...
const (
	// MaxUnconfirmedBlocks = 128
	MaxMemPoolExpire = uint64(1024)

	MaxImportingTask = 3
)

// NtfnsHandler ...
type NtfnsHandler struct {
	walletMgr *WalletManager

	memMtx         sync.Mutex
	mempool        map[wire.Hash]struct{}
	bestBlock      txmgr.BlockMeta
	expiredMempool map[uint64]map[wire.Hash]struct{}

	queueMsgTx     chan *wire.MsgTx
	queueBlock     chan *wire.MsgBlock
	queueImporting chan *keystore.AddrManager

	quit chan struct{}

	sigSuspend chan struct{}
	sigResume  chan struct{}
}

// NewNtfnsHandler ...
func NewNtfnsHandler(w *WalletManager) (*NtfnsHandler, error) {
	h := &NtfnsHandler{
		walletMgr:      w,
		mempool:        make(map[wire.Hash]struct{}),
		expiredMempool: make(map[uint64]map[wire.Hash]struct{}),

		queueMsgTx: make(chan *wire.MsgTx, 1024),
		queueBlock: make(chan *wire.MsgBlock, 1024),

		queueImporting: make(chan *keystore.AddrManager, MaxImportingTask*2),

		sigSuspend: make(chan struct{}),
		sigResume:  make(chan struct{}),
		quit:       make(chan struct{}),
	}
	var syncedTo *txmgr.BlockMeta
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) (err error) {
		syncedTo, err = w.syncStore.SyncedTo(tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	h.bestBlock = *syncedTo
	return h, nil
}

func (h *NtfnsHandler) Start() error {

	syncHeight, err := h.walletMgr.SyncedTo()
	if err != nil {
		return err
	}
	indexHeight := h.walletMgr.ChainIndexerSyncedHeight()

	logging.CPrint(logging.INFO, "NtfnsHandler start...",
		logging.LogFormat{
			"syncHeight":    syncHeight,
			"indexedHeight": indexHeight,
		})
	for i := syncHeight + 1; i <= indexHeight; i++ {
		blk, err := h.walletMgr.chainFetcher.FetchBlockByHeight(i)
		if err != nil {
			logging.CPrint(logging.ERROR, "NtfnsHandler.Start(): FetchBlockByHeight error",
				logging.LogFormat{
					"i":   i,
					"err": err,
				})
			return err
		}

		err = h.processConnectedBlock(blk)
		if err != nil {
			logging.CPrint(logging.ERROR, "NtfnsHandler.Start(): syncing error",
				logging.LogFormat{
					"blkHeight": i,
				})
			return err
		}
	}

	go handle(h)
	go handleImportingTask(h)
	return nil
}

func (h *NtfnsHandler) Stop() {
	close(h.quit)
}

func handle(h *NtfnsHandler) {
	logging.CPrint(logging.INFO, "start handling", logging.LogFormat{})

	for {
		select {
		case <-h.quit:
			logging.CPrint(logging.INFO, "NtfnsHandler stopped", logging.LogFormat{})
			return

		case <-h.sigSuspend:
			<-h.sigResume

		case block := <-h.queueBlock:
			err := h.processConnectedBlock(block)
			if err != nil {
				logging.CPrint(logging.WARN, "processConnectedBlock error", logging.LogFormat{
					"hash":   block.Header.BlockHash().String(),
					"height": block.Header.Height,
					"err":    err,
				})
			}

		case tx := <-h.queueMsgTx:
			err := h.proccessReceivedTx(tx)
			if err != nil {
				logging.CPrint(logging.WARN, "proccessReceivedTx error", logging.LogFormat{
					"hash": tx.TxHash().String(),
					"err":  err,
				})
			}
		}
	}
}

func (h *NtfnsHandler) onRelevantTx(rec *txmgr.TxRecord) error {
	err := mwdb.Update(h.walletMgr.db, func(tx mwdb.DBTransaction) error {
		return h.walletMgr.txStore.AddRelevantTx(tx, nil, rec, nil)
	})
	if err != nil {
		logging.VPrint(logging.ERROR, "Cannot add relevant transaction",
			logging.LogFormat{
				"tx":  rec.Hash.String(),
				"err": err,
			})
	}
	return err
}

func (h *NtfnsHandler) onRelevantBlockConnected(tx mwdb.DBTransaction, blockMeta *txmgr.BlockMeta,
	relevantTxs []*txmgr.TxRecord) error {

	walletBalances, err := h.walletMgr.utxoStore.FetchAllMinedBalance(tx)
	if err != nil {
		logging.VPrint(logging.ERROR, "FetchAllMinedBalance error",
			logging.LogFormat{
				"block":  blockMeta.Hash.String(),
				"height": blockMeta.Height,
				"err":    err,
			})
		return err
	}

	for _, rec := range relevantTxs {
		if err := h.walletMgr.txStore.AddRelevantTx(tx, walletBalances, rec, blockMeta); err != nil {
			logging.VPrint(logging.ERROR, "Cannot add relevant block",
				logging.LogFormat{
					"block":       blockMeta.Hash.String(),
					"transaction": rec.Hash.String(),
					"err":         err,
				})
			return err
		}
	}

	err = h.walletMgr.utxoStore.UpdateMinedBalances(tx, walletBalances)
	if err != nil {
		logging.VPrint(logging.ERROR, "UpdateMinedBalances error",
			logging.LogFormat{
				"block":  blockMeta.Hash.String(),
				"height": blockMeta.Height,
				"err":    err,
			})
		return err
	}

	return h.walletMgr.syncStore.SetSyncedTo(tx, blockMeta)
}

func (h *NtfnsHandler) filterTxForImporting(tx *wire.MsgTx, blockMeta *txmgr.BlockMeta,
	importingAddrMgr *keystore.AddrManager) (*txmgr.TxRecord, error) {
	rec, err := txmgr.NewTxRecordFromMsgTx(tx, time.Now())
	if err != nil {
		logging.VPrint(logging.ERROR, "Cannot create transaction record",
			logging.LogFormat{
				"tx":  tx.TxHash().String(),
				"err": err,
			})
		return nil, err
	}

	cache := make(map[wire.Hash]*wire.MsgTx)
	// check TxIn
	if !blockchain.IsCoinBaseTx(tx) {
		for i, txIn := range tx.TxIn {
			prevTx := cache[txIn.PreviousOutPoint.Hash]
			if prevTx == nil {
				prevTx, err = h.walletMgr.chainFetcher.FetchLastTxUntilHeight(&txIn.PreviousOutPoint.Hash, blockMeta.Height)
				if err != nil {
					return nil, err
				}
				if prevTx == nil {
					logging.CPrint(logging.WARN, "previous transaction not found", logging.LogFormat{
						"tx":        tx.TxHash().String(),
						"txInIndex": i,
						"block":     blockMeta.Hash.String(),
						"height":    blockMeta.Height,
					})
					return nil, ErrImportingContinuable
				}
				cache[txIn.PreviousOutPoint.Hash] = prevTx
			}

			pkScript := prevTx.TxOut[txIn.PreviousOutPoint.Index].PkScript
			ps, err := utils.ParsePkScript(pkScript, h.walletMgr.chainParams)
			if err != nil {
				if err == utils.ErrUnsupportedScript {
					continue
				}
				logging.CPrint(logging.ERROR, "unexpected error",
					logging.LogFormat{
						"tx":        tx.TxHash().String(),
						"txInIndex": i,
						"err":       err,
					})
				return nil, err
			}
			ma, _ := importingAddrMgr.Address(ps.StdEncodeAddress())
			if ma != nil {
				rec.HasBindingIn = ps.IsBinding()
				rec.RelevantTxIn = append(rec.RelevantTxIn,
					&txmgr.RelevantMeta{
						Index:        i,
						PkScript:     ps,
						WalletId:     ma.Account(),
						IsChangeAddr: ma.IsChangeAddr(),
					})
			}
		}
	}

	// check TxOut
	for i, txOut := range tx.TxOut {
		ps, err := utils.ParsePkScript(txOut.PkScript, h.walletMgr.chainParams)
		if err != nil {
			if err == utils.ErrUnsupportedScript {
				continue
			}
			logging.CPrint(logging.ERROR, "unexpected error",
				logging.LogFormat{
					"tx":         tx.TxHash().String(),
					"txOutIndex": i,
					"err":        err,
				})
			return nil, err
		}
		ma, _ := importingAddrMgr.Address(ps.StdEncodeAddress())
		if ma != nil {
			rec.HasBindingOut = ps.IsBinding()
			rec.RelevantTxOut = append(rec.RelevantTxOut,
				&txmgr.RelevantMeta{
					Index:        i,
					PkScript:     ps,
					WalletId:     ma.Account(),
					IsChangeAddr: ma.IsChangeAddr(),
				})
		}
	}

	if len(rec.RelevantTxIn) == 0 && len(rec.RelevantTxOut) == 0 {
		return nil, nil
	}

	if rec.HasBindingIn && rec.HasBindingOut {
		return nil, ErrBothBinding
	}
	return rec, nil
}

func (h *NtfnsHandler) filterTx(tx *wire.MsgTx, blockMeta *txmgr.BlockMeta, recInCurBlk map[wire.Hash]*txmgr.TxRecord) (bool, *txmgr.TxRecord, error) {

	rec, err := txmgr.NewTxRecordFromMsgTx(tx, time.Now())
	if err != nil {
		logging.VPrint(logging.ERROR, "Cannot create transaction record for tx",
			logging.LogFormat{
				rec.Hash.String(): err,
			})
		return false, nil, err
	}
	if blockMeta != nil {
		recInCurBlk[rec.Hash] = rec
	}

	h.memMtx.Lock()
	_, ok := h.mempool[rec.Hash]
	h.memMtx.Unlock()
	if ok && blockMeta == nil {
		return false, nil, nil
	}

	cache := make(map[wire.Hash]*wire.MsgTx)
	// check TxIn
	if !blockchain.IsCoinBaseTx(tx) {
		for i, txIn := range tx.TxIn {
			prevTx := cache[txIn.PreviousOutPoint.Hash]

			// tx in current block
			if prevTx == nil && blockMeta != nil {
				r, ok := recInCurBlk[txIn.PreviousOutPoint.Hash]
				if ok {
					prevTx = &r.MsgTx
				}
			}

			// mined/unmined tx
			if prevTx == nil {
				prevTx, err = h.walletMgr.chainFetcher.FetchTxBySha(&txIn.PreviousOutPoint.Hash)
				if err != nil {
					return false, nil, err
				}
				if prevTx == nil {
					prevTx, err = h.walletMgr.existsUnminedTx(&txIn.PreviousOutPoint.Hash)
					if err != nil {
						if err != txmgr.ErrNotFound {
							return false, nil, err
						}
					}
				}
			}

			if prevTx == nil {
				fields := logging.LogFormat{
					"tx":          rec.Hash.String(),
					"txInIndex":   i,
					"isUnminedTx": blockMeta == nil,
				}
				if blockMeta != nil {
					fields["block"] = blockMeta.Hash.String()
					fields["height"] = blockMeta.Height
					logging.CPrint(logging.DEBUG, "previous transaction of mined not found", fields)
					return false, nil, ErrMaybeChainRevoked
				}
				logging.CPrint(logging.WARN, "previous transaction of unmined not found", fields)
				// NOTE: just accept unmined tx whose all inputs are on the chain
				return false, nil, ErrInvalidTx
			} else {
				if len(prevTx.TxOut) <= int(txIn.PreviousOutPoint.Index) {
					logging.CPrint(logging.ERROR, "TxIn out of range",
						logging.LogFormat{
							"tx":        rec.Hash.String(),
							"txInIndex": i,
							"index":     txIn.PreviousOutPoint.Index,
							"range":     len(prevTx.TxOut) - 1,
						})
					return false, nil, ErrInvalidTx
				}
				cache[txIn.PreviousOutPoint.Hash] = prevTx
			}

			pkScript := prevTx.TxOut[txIn.PreviousOutPoint.Index].PkScript
			ps, err := utils.ParsePkScript(pkScript, h.walletMgr.chainParams)
			if err != nil {
				if err == utils.ErrUnsupportedScript {
					continue
				}
				logging.CPrint(logging.ERROR, "failed to parse txin pkscript",
					logging.LogFormat{
						"tx":        rec.Hash.String(),
						"txInIndex": i,
						"err":       err,
					})
				return false, nil, err
			}

			ma, err := h.walletMgr.ksmgr.GetManagedAddressByScriptHash(ps.StdScriptAddress())
			if err != nil && err != keystore.ErrScriptHashNotFound {
				return false, nil, err
			}
			if ma != nil {
				ready, err := h.walletMgr.CheckReady(ma.Account())
				if err != nil {
					return false, nil, err
				}
				if !ready {
					continue
				}

				rec.HasBindingIn = ps.IsBinding()
				rec.RelevantTxIn = append(rec.RelevantTxIn,
					&txmgr.RelevantMeta{
						Index:        i,
						PkScript:     ps,
						WalletId:     ma.Account(),
						IsChangeAddr: ma.IsChangeAddr(),
					})
			}
		}
	}

	// check TxOut
	for i, txOut := range tx.TxOut {

		ps, err := utils.ParsePkScript(txOut.PkScript, h.walletMgr.chainParams)
		if err != nil {
			if err == utils.ErrUnsupportedScript {
				continue
			}
			logging.CPrint(logging.ERROR, "failed to parse txout pkscript",
				logging.LogFormat{
					"tx":         rec.Hash.String(),
					"txOutIndex": i,
					"err":        err,
				})
			return false, nil, err
		}
		ma, err := h.walletMgr.ksmgr.GetManagedAddressByScriptHash(ps.StdScriptAddress())
		if err != nil && err != keystore.ErrScriptHashNotFound {
			return false, nil, err
		}
		if ma != nil {
			ready, err := h.walletMgr.CheckReady(ma.Account())
			if err != nil {
				return false, nil, err
			}
			if !ready {
				continue
			}
			rec.HasBindingOut = ps.IsBinding()
			rec.RelevantTxOut = append(rec.RelevantTxOut,
				&txmgr.RelevantMeta{
					Index:        i,
					PkScript:     ps,
					WalletId:     ma.Account(),
					IsChangeAddr: ma.IsChangeAddr(),
				})
		}

	}

	if len(rec.RelevantTxIn) == 0 && len(rec.RelevantTxOut) == 0 {
		return false, nil, nil
	}

	if rec.HasBindingIn && rec.HasBindingOut {
		return false, nil, ErrBothBinding
	}

	if blockMeta == nil {
		if err := h.onRelevantTx(rec); err != nil {
			return false, nil, err
		}
		h.memMtx.Lock()
		h.mempool[rec.Hash] = struct{}{}
		h.memMtx.Unlock()
	}

	return true, rec, nil
}

func (h *NtfnsHandler) filterBlock(dbtx mwdb.DBTransaction,
	block *wire.MsgBlock, addedExpireMempool map[uint64]map[wire.Hash]struct{}) error {

	blockMeta := &txmgr.BlockMeta{
		Hash:      block.BlockHash(),
		Height:    block.Header.Height,
		Timestamp: block.Header.Timestamp,
	}

	var relevantTxs []*txmgr.TxRecord
	confirmedTxs := make(map[wire.Hash]struct{})
	recInCurBlk := make(map[wire.Hash]*txmgr.TxRecord)
	for _, tx := range block.Transactions {
		isRelevant, rec, err := h.filterTx(tx, blockMeta, recInCurBlk)
		if err != nil {
			logging.CPrint(logging.WARN, "Unable to filter transaction",
				logging.LogFormat{
					tx.TxHash().String(): err,
				})
			return err
		}

		if isRelevant {
			relevantTxs = append(relevantTxs, rec)
			confirmedTxs[rec.Hash] = struct{}{}
		}
	}

	addedExpireMempool[block.Header.Height] = confirmedTxs

	return h.onRelevantBlockConnected(dbtx, blockMeta, relevantTxs)
}

func (h *NtfnsHandler) disconnectBlock(tx mwdb.DBTransaction, height uint64) error {
	if height == 0 {
		return fmt.Errorf("genesis block cannot be disconnected")
	}
	syncedTo, err := h.walletMgr.syncStore.SyncedTo(tx)
	if err != nil {
		return err
	}
	if height > syncedTo.Height {
		return nil
	}

	err = h.walletMgr.txStore.Rollback(tx, height)
	if err != nil {
		return err
	}

	resetHeight := height - 1
	err = h.walletMgr.syncStore.ResetSyncedTo(tx, resetHeight)
	if err != nil {
		return err
	}
	wss, err := h.walletMgr.syncStore.GetAllWalletStatus(tx)
	if err != nil {
		return err
	}

	for _, ws := range wss {
		if ws.Ready() {
			continue
		}

		if ws.SyncedHeight > resetHeight {
			ws.SyncedHeight = resetHeight
			err = h.walletMgr.syncStore.PutWalletStatus(tx, ws)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *NtfnsHandler) reorg(dbtx mwdb.DBTransaction, currentBest txmgr.BlockMeta, newBest *wire.MsgBlock,
	rollbackBlock map[uint64]struct{}, addedExpireMempool map[uint64]map[wire.Hash]struct{}) error {

	blocksToConnect := list.New()

	// step 1. align height
	for currentBest.Height < newBest.Header.Height {
		blocksToConnect.PushFront(newBest)

		var err error
		// tmp := &newBest.Header
		newBest, err = h.getBlock(&newBest.Header.Previous)
		if err != nil {
			return err
		}
		if newBest == nil {
			return ErrMaybeChainRevoked
		}
	}

	if currentBest.Hash != newBest.BlockHash() {

		for currentBest.Height > newBest.Header.Height {
			err := h.disconnectBlock(dbtx, currentBest.Height)
			if err != nil {
				return err
			}
			rollbackBlock[currentBest.Height] = struct{}{}
			currentBest.Height--
			// currentBest.Hash = header.Previous
		}

		bm, err := h.walletMgr.syncStore.SyncedBlock(dbtx, currentBest.Height)
		if err != nil {
			return err
		}
		if bm == nil {
			return fmt.Errorf("synced block not found for height: %d", currentBest.Height)
		}
		currentBest = *bm

		if currentBest.Hash != newBest.BlockHash() {

			currentPrev, err := h.walletMgr.syncStore.SyncedBlock(dbtx, currentBest.Height-1)
			if err != nil {
				return err
			}
			if currentPrev == nil {
				return fmt.Errorf("prev synced block not found, current: %v", currentBest)
			}

			newTailBlock := newBest
			for newTailBlock.Header.Previous != currentPrev.Hash {
				if err = h.disconnectBlock(dbtx, currentPrev.Height+1); err != nil {
					return err
				}
				rollbackBlock[currentPrev.Height+1] = struct{}{}
				blocksToConnect.PushFront(newTailBlock)

				currentPrev, err = h.walletMgr.syncStore.SyncedBlock(dbtx, currentPrev.Height-1)
				if err != nil {
					return err
				}
				if currentPrev == nil {
					return fmt.Errorf("prev synced block not found, current: %v", currentPrev)
				}

				// tmp := &newTailBlock.Header
				newTailBlock, err = h.getBlock(&newTailBlock.Header.Previous)
				if err != nil {
					return err
				}
				if newTailBlock == nil {
					return ErrMaybeChainRevoked
				}
			}
			if err = h.disconnectBlock(dbtx, currentPrev.Height+1); err != nil {
				return err
			}

			rollbackBlock[currentPrev.Height+1] = struct{}{}
			blocksToConnect.PushFront(newTailBlock)
		}
	}
	// step 3. connect
	for blocksToConnect.Front() != nil {
		block := blocksToConnect.Front().Value.(*wire.MsgBlock)

		if err := h.filterBlock(dbtx, block, addedExpireMempool); err != nil {
			return err
		}

		blocksToConnect.Remove(blocksToConnect.Front())
	}

	return nil
}

func handleImportingTask(h *NtfnsHandler) {
	err := mwdb.View(h.walletMgr.db, func(tx mwdb.ReadTransaction) error {
		wss, err := h.walletMgr.syncStore.GetAllWalletStatus(tx)
		if err != nil {
			return err
		}
		for _, ws := range wss {
			logging.CPrint(logging.DEBUG, "wallet status before reload importing",
				logging.LogFormat{
					"syncedheight": ws.SyncedHeight,
					"walletId":     ws.WalletID,
					"ready":        ws.Ready(),
					"best":         h.bestBlock.Height,
				})

			// double check
			am, err := h.walletMgr.ksmgr.GetAddrManagerByAccountID(ws.WalletID)
			if err != nil {
				logging.CPrint(logging.FATAL, "wallet not found",
					logging.LogFormat{
						"err":      err,
						"walletId": ws.WalletID,
					})
			}

			if !ws.Ready() {
				h.queueImporting <- am
				logging.CPrint(logging.INFO, "reload importing task",
					logging.LogFormat{
						"walletId": ws.WalletID,
					})
			}
		}
		return nil
	})
	if err != nil {
		logging.CPrint(logging.FATAL, "reload importing task failed",
			logging.LogFormat{
				"err": err,
			})
	}

	for {
		select {
		case <-h.quit:
			h.walletMgr.CloseDB()
			return
		case am := <-h.queueImporting:
			fin, err := h.innerImport(am)
			if err != nil {
				logging.CPrint(logging.ERROR, "innerImport error",
					logging.LogFormat{
						"err":      err,
						"walletId": am.Name(),
					})
			}
			if !fin {
				h.queueImporting <- am
			}
		}
	}
}

func (h *NtfnsHandler) innerImport(addrmgr *keystore.AddrManager) (finish bool, err error) {
	mas := addrmgr.ManagedAddresses()
	relatedHashes := make([][]byte, 0)
	for _, ma := range mas {
		relatedHashes = append(relatedHashes, ma.ScriptAddress())
	}

	h.suspend("start innerImport", logging.LogFormat{"walletId": addrmgr.Name()})
	defer func() {
		h.resume("done innerImport", logging.LogFormat{"walletId": addrmgr.Name(), "finish": finish})
	}()

	stop := uint64(0)
	heightAdded := make(map[uint64][]wire.Hash, 0)
	fetcher := h.walletMgr.chainFetcher
	err = mwdb.Update(h.walletMgr.db, func(dbtx mwdb.DBTransaction) error {
		// sync status
		ws, err := h.walletMgr.syncStore.GetWalletStatus(dbtx, addrmgr.Name())
		if err != nil {
			return err
		}
		// balances
		addrMgrBalance, err := h.walletMgr.utxoStore.GrossBalance(dbtx, addrmgr.Name())
		if err != nil {
			logging.VPrint(logging.ERROR, "FetchAllMinedBalance error", logging.LogFormat{"err": err})
			return err
		}
		allBalances := map[string]massutil.Amount{addrmgr.Name(): addrMgrBalance}

		stop = ws.SyncedHeight + 1000
		if stop > h.bestBlock.Height {
			stop = h.bestBlock.Height
		}
		result, err := fetcher.FetchScriptHashRelatedTx(relatedHashes, ws.SyncedHeight+1, stop+1, h.walletMgr.chainParams)
		if err != nil {
			return err
		}
		logging.CPrint(logging.DEBUG, "innerImport: range",
			logging.LogFormat{
				"start":    ws.SyncedHeight + 1,
				"stop":     stop + 1,
				"walletId": addrmgr.Name(),
			})
		for _, height := range result.Heights() {
			txlocs := result.Get(height)
			if len(txlocs) == 0 {
				continue
			}
			header, err := fetcher.FetchBlockHeaderByHeight(height)
			if err != nil {
				return err
			}
			if header == nil {
				logging.CPrint(logging.WARN, "header not found on chain, maybe chain forks",
					logging.LogFormat{
						"blockheight": height,
					})
				return ErrImportingContinuable
			}

			// insertTx
			blockmeta := &txmgr.BlockMeta{
				Hash:      header.BlockHash(),
				Height:    header.Height,
				Timestamp: header.Timestamp,
			}

			added := make([]wire.Hash, 0)
			for _, txloc := range txlocs {
				msg, err := fetcher.FetchTxByLoc(height, txloc)
				if err != nil {
					return err
				}
				if msg == nil {
					logging.CPrint(logging.WARN, "tx not found on chain, maybe chain forks",
						logging.LogFormat{
							"blockheight": header.Height,
							"txlen":       txloc.TxLen,
							"txstart":     txloc.TxStart,
						})
					return ErrImportingContinuable
				}

				rec, err := h.filterTxForImporting(msg, blockmeta, addrmgr)
				if err != nil {
					return err
				}
				if rec == nil {
					logging.CPrint(logging.ERROR, "unexpected error, tx is not relevant",
						logging.LogFormat{
							"tx":     rec.Hash.String(),
							"block":  blockmeta.Hash.String(),
							"height": blockmeta.Height,
						})
					return fmt.Errorf("unexpected error: tx is not relevant")
				}

				err = h.walletMgr.txStore.AddRelevantTxForImporting(dbtx, allBalances, rec, blockmeta)
				if err != nil {
					if err == txmgr.ErrChainReorg {
						return ErrImportingContinuable
					}
					return err
				}
				added = append(added, msg.TxHash())
			}

			if h.bestBlock.Height > MaxMemPoolExpire && height <= h.bestBlock.Height-MaxMemPoolExpire {
				continue
			}
			heightAdded[height] = added
		}

		ws.SyncedHeight = stop
		if stop == h.bestBlock.Height {
			ws.SyncedHeight = txmgr.WalletSyncedDone
			finish = true
			logging.CPrint(logging.INFO, "innerImport: finish",
				logging.LogFormat{
					"walletId":  addrmgr.Name(),
					"finHeight": stop,
				})
		}

		err = h.walletMgr.utxoStore.UpdateMinedBalances(dbtx, allBalances)
		if err != nil {
			logging.VPrint(logging.ERROR, "UpdateMinedBalances error", logging.LogFormat{"err": err})
			return err
		}

		return h.walletMgr.syncStore.PutWalletStatus(dbtx, ws)
	})
	if err != nil {
		return false, err
	}

	// update mem pol
	for height, added := range heightAdded {
		m, ok := h.expiredMempool[height]
		if !ok {
			m = make(map[wire.Hash]struct{})
		}
		for _, hash := range added {
			m[hash] = struct{}{}
		}
		h.expiredMempool[height] = m
	}

	return finish, nil
}

func (h *NtfnsHandler) pushImportingTask(addrmgr *keystore.AddrManager) {
	h.queueImporting <- addrmgr
}

func (h *NtfnsHandler) exceedImportingLimit() bool {
	return len(h.queueImporting) >= MaxImportingTask
}

func (h *NtfnsHandler) RemoveMempoolTx(txs []*wire.Hash) {
	h.memMtx.Lock()
	for _, hash := range txs {
		delete(h.mempool, *hash)
	}
	h.memMtx.Unlock()
}

// Must call NtfnsHandler.suspend() before openning database transaction and NtfnsHandler.resume() after removing done
func (h *NtfnsHandler) onKeystoreRemove(tx mwdb.DBTransaction, addrmgr *keystore.AddrManager) ([]*wire.Hash, error) {

	deletedHash := make([]*wire.Hash, 0)
	err := func() error {
		deletedTx, err := h.walletMgr.txStore.RemoveRelevantTx(tx, addrmgr /* , h.bestBlock.Height */)
		if err != nil {
			return err
		}
		err = h.walletMgr.utxoStore.RemoveUnspentByWalletId(tx, addrmgr.Name())
		if err != nil {
			return err
		}
		err = h.walletMgr.utxoStore.RemoveAddressByWalletId(tx, addrmgr.Name())
		if err != nil {
			return err
		}
		err = h.walletMgr.utxoStore.RemoveLGHistoryByWalletId(tx, addrmgr.Name())
		if err != nil {
			return err
		}

		err = h.walletMgr.utxoStore.RemoveMinedBalance(tx, addrmgr.Name())
		if err != nil {
			return err
		}

		for _, hash := range deletedTx {
			deletedHash = append(deletedHash, hash)
		}
		return nil
	}()

	if err != nil {
		return nil, err
	}

	return deletedHash, nil
}

func (h *NtfnsHandler) processConnectedBlock(newBlock *wire.MsgBlock) error {

	h.memMtx.Lock()
	bestBlock := h.bestBlock
	h.memMtx.Unlock()

	rollbackBlock := make(map[uint64]struct{})
	addedExpireMempool := make(map[uint64]map[wire.Hash]struct{})
	err := mwdb.Update(h.walletMgr.db, func(tx mwdb.DBTransaction) error {
		if newBlock.Header.Previous == bestBlock.Hash {
			if err := h.filterBlock(tx, newBlock, addedExpireMempool); err != nil {
				logging.CPrint(logging.ERROR, "failed to filter block",
					logging.LogFormat{
						"err": err,
					})
				return err
			}
			return nil
		}

		err := h.reorg(tx, bestBlock, newBlock, rollbackBlock, addedExpireMempool)
		if err != nil {
			if err != ErrMaybeChainRevoked {
				logging.CPrint(logging.ERROR, "failed to reorg",
					logging.LogFormat{
						"err":     err,
						"height":  newBlock.Header.Height,
						"blkHash": newBlock.BlockHash().String(),
					})
			} else {
				logging.CPrint(logging.INFO, "failed to reorg",
					logging.LogFormat{
						"err":     err,
						"height":  newBlock.Header.Height,
						"blkHash": newBlock.BlockHash().String(),
					})
			}
		}
		return err
	})
	if err == nil {
		h.memMtx.Lock()
		// process rollback
		for height := range rollbackBlock {
			if blk, ok := h.expiredMempool[height]; ok {
				for txHash := range blk {
					h.mempool[txHash] = struct{}{}
				}
			}
			delete(h.expiredMempool, height)
		}

		// process add
		for height, confirmedTxs := range addedExpireMempool {
			h.expiredMempool[height] = confirmedTxs
			if height > MaxMemPoolExpire {
				if oldBlock, ok := h.expiredMempool[height-MaxMemPoolExpire]; ok {
					for txHash := range oldBlock {
						delete(h.mempool, txHash)
					}
					delete(h.expiredMempool, height-MaxMemPoolExpire)
				}
			}
		}
		// update bestblock
		bestBlock.Hash = newBlock.BlockHash()
		bestBlock.Height = newBlock.Header.Height
		bestBlock.Timestamp = newBlock.Header.Timestamp
		h.bestBlock = bestBlock
		h.memMtx.Unlock()
	}

	return err
}

func (h *NtfnsHandler) proccessReceivedTx(tx *wire.MsgTx) error {
	logging.CPrint(logging.INFO, "filter broadcast transaction",
		logging.LogFormat{
			"tx": tx.TxHash().String(),
		})
	if _, _, err := h.filterTx(tx, nil, nil); err != nil {
		logging.CPrint(logging.WARN, "Unable to filter transaction",
			logging.LogFormat{
				"tx":  tx.TxHash().String(),
				"err": err,
			})
		return err
	}
	return nil
}

func (h *NtfnsHandler) getBlock(hash *wire.Hash) (*wire.MsgBlock, error) {
	return h.walletMgr.chainFetcher.FetchBlockBySha(hash)
}

func (h *NtfnsHandler) OnBlockConnected(newBlock *wire.MsgBlock) error {
	if newBlock.Header.Height%1000 == 0 {
		logging.CPrint(logging.INFO, "OnBlockConnected",
			logging.LogFormat{
				"hash":   newBlock.BlockHash().String(),
				"height": newBlock.Header.Height,
			})
	}
	h.queueBlock <- newBlock
	return nil
}

func (h *NtfnsHandler) OnTransactionReceived(tx *wire.MsgTx) error {
	h.queueMsgTx <- tx
	return nil
}

func (h *NtfnsHandler) suspend(msg string, fields logging.LogFormat) {
	h.sigSuspend <- struct{}{}
	logging.CPrint(logging.DEBUG, "suspend: "+msg, fields)
}

func (h *NtfnsHandler) resume(msg string, fields logging.LogFormat) {
	h.sigResume <- struct{}{}
	logging.CPrint(logging.DEBUG, "resume: "+msg, fields)
}
