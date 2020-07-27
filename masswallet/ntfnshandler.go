package masswallet

import (
	"bytes"
	"container/list"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"massnet.org/mass-wallet/blockchain"
	mdebug "massnet.org/mass-wallet/debug"
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
)

// NtfnsHandler ...
type NtfnsHandler struct {
	walletMgr *WalletManager

	memMtx         sync.Mutex
	mempool        map[wire.Hash]struct{}
	bestBlock      txmgr.BlockMeta
	expiredMempool map[uint64]map[wire.Hash]struct{}

	queueMsgTx chan *wire.MsgTx
	queueBlock chan *wire.MsgBlock

	taskChan *WalletTaskChan

	quit   chan struct{}
	quitWg sync.WaitGroup

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

	hasReadyWallet := false
	err = mwdb.View(h.walletMgr.db, func(rtx mwdb.ReadTransaction) (err error) {
		readyWallets, err := h.getReadyWallets(rtx)
		hasReadyWallet = len(readyWallets) > 0
		return err
	})
	if err != nil {
		logging.CPrint(logging.ERROR, "getReadyWallets error", logging.LogFormat{"err": err})
		return err
	}

	curHeight := syncHeight + 1
	if !hasReadyWallet && indexHeight > 2000 {
		for ; curHeight < indexHeight-2000; curHeight++ {
			sha, err := h.walletMgr.chainFetcher.FetchBlockShaByHeight(curHeight)
			if err != nil {
				logging.CPrint(logging.ERROR, "FetchBlockShaByHeight error",
					logging.LogFormat{"err": err, "height": curHeight})
				return err
			}
			h.bestBlock.Hash = *sha
			h.bestBlock.Height = curHeight
			// h.bestBlock.Timestamp = ? // not used
			blockMeta := &txmgr.BlockMeta{
				Hash:      *sha,
				Height:    curHeight,
				Timestamp: time.Now(), // not used
			}
			err = mwdb.Update(h.walletMgr.db, func(wtx mwdb.DBTransaction) error {
				return h.walletMgr.syncStore.SetSyncedTo(wtx, blockMeta)
			})
			if err != nil {
				logging.CPrint(logging.ERROR, "SetSyncedTo error",
					logging.LogFormat{"err": err, "height": curHeight, "block": sha})
				return err
			}
		}
	}

	for ; curHeight <= indexHeight; curHeight++ {
		blk, err := h.walletMgr.chainFetcher.FetchBlockByHeight(curHeight)
		if err != nil {
			logging.CPrint(logging.ERROR, "NtfnsHandler.Start(): FetchBlockByHeight error",
				logging.LogFormat{
					"height": curHeight,
					"err":    err,
				})
			return err
		}

		err = h.processConnectedBlock(blk)
		if err != nil {
			logging.CPrint(logging.ERROR, "NtfnsHandler.Start(): processConnectedBlock error",
				logging.LogFormat{
					"height": curHeight,
					"err":    err,
				})
			return err
		}
	}

	h.quitWg.Add(2)
	go handle(h)
	go worker(h)
	return nil
}

func (h *NtfnsHandler) Stop() {
	close(h.quit)
	h.quitWg.Wait()
	h.walletMgr.CloseDB()
}

func handle(h *NtfnsHandler) {
	defer Recover()
	defer h.quitWg.Done()

	logging.CPrint(logging.INFO, "NtfnsHandler started", logging.LogFormat{})

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

func (h *NtfnsHandler) onRelevantBlockConnected(tx mwdb.DBTransaction, readyWallets map[string]struct{},
	blockMeta *txmgr.BlockMeta, relevantTxs []*txmgr.TxRecord) error {

	if len(relevantTxs) == 0 {
		return nil
	}

	all, err := h.walletMgr.utxoStore.FetchAllMinedBalance(tx)
	if err != nil {
		logging.VPrint(logging.ERROR, "FetchAllMinedBalance error", logging.LogFormat{"err": err})
		return err
	}
	walletBalances := make(map[string]massutil.Amount)
	for walletId, bal := range all {
		if _, ok := readyWallets[walletId]; ok {
			walletBalances[walletId] = bal
		}
	}
	if mdebug.DevMode() {
		if len(walletBalances) == 0 {
			logging.CPrint(logging.FATAL, "unexpected empty walletBalances", logging.LogFormat{
				"readyWalletes": readyWallets,
				"all":           all,
			})
		}
	}

	for _, rec := range relevantTxs {
		if err := h.walletMgr.txStore.AddRelevantTx(tx, walletBalances, rec, blockMeta); err != nil {
			logging.VPrint(logging.ERROR, "Cannot add relevant block",
				logging.LogFormat{
					"tx":  rec.Hash.String(),
					"err": err,
				})
			return err
		}
	}

	err = h.walletMgr.utxoStore.UpdateMinedBalances(tx, walletBalances)
	if err != nil {
		logging.VPrint(logging.ERROR, "UpdateMinedBalances error", logging.LogFormat{"err": err})
	}
	return err
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

func (h *NtfnsHandler) filterTx(tx *wire.MsgTx, blockMeta *txmgr.BlockMeta,
	recInCurBlk map[wire.Hash]*txmgr.TxRecord,
	readyWallets map[string]struct{}) (bool, *txmgr.TxRecord, error) {

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
				bro, ok := recInCurBlk[txIn.PreviousOutPoint.Hash]
				if ok {
					prevTx = &bro.MsgTx
				} else {
					// For connected block, it's unnecessary to go on checking
					// if no output created by previous hash.
					exist := false
					mwdb.View(h.walletMgr.db, func(rtx mwdb.ReadTransaction) error {
						exist = h.walletMgr.utxoStore.ExistCreditFromTx(rtx, &txIn.PreviousOutPoint.Hash)
						return nil
					})
					if !exist {
						continue
					}
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
				if _, ok := readyWallets[ma.Account()]; !ok {
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
			if _, ok := readyWallets[ma.Account()]; !ok {
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

func (h *NtfnsHandler) filterBlock(dbtx mwdb.DBTransaction, readyWallets map[string]struct{},
	block *wire.MsgBlock, addedExpireMempool map[uint64]map[wire.Hash]struct{}) (err error) {

	blockMeta := &txmgr.BlockMeta{
		Hash:      block.BlockHash(),
		Height:    block.Header.Height,
		Timestamp: block.Header.Timestamp,
	}
	blockMeta.Loc, err = h.walletMgr.chainFetcher.FetchBlockLocByHeight(blockMeta.Height)
	if err != nil {
		logging.CPrint(logging.WARN, "FetchBlockLocByHeight error",
			logging.LogFormat{
				"err":    err,
				"height": blockMeta.Height,
			})
		return err
	}
	if !bytes.Equal(blockMeta.Loc.Hash[:], blockMeta.Hash[:]) {
		logging.CPrint(logging.WARN, "block hash mismatch", logging.LogFormat{"height": blockMeta.Height})
		return ErrMaybeChainRevoked
	}
	txLocs, err := massutil.NewBlock(block).TxLoc()
	if err != nil {
		logging.CPrint(logging.ERROR, "get tx locs error", logging.LogFormat{"height": blockMeta.Height, "err": err})
		return err
	}

	var relevantTxs []*txmgr.TxRecord
	confirmedTxs := make(map[wire.Hash]struct{})
	if len(readyWallets) > 0 {
		recInCurBlk := make(map[wire.Hash]*txmgr.TxRecord)
		for i, tx := range block.Transactions {
			isRelevant, rec, err := h.filterTx(tx, blockMeta, recInCurBlk, readyWallets)
			if err != nil {
				logging.CPrint(logging.WARN, "Unable to filter transaction",
					logging.LogFormat{
						tx.TxHash().String(): err,
					})
				return err
			}

			if isRelevant {
				txLoc := txLocs[i]
				rec.TxLoc = &txLoc
				relevantTxs = append(relevantTxs, rec)
				confirmedTxs[rec.Hash] = struct{}{}
			}
		}
	}

	addedExpireMempool[block.Header.Height] = confirmedTxs

	err = h.onRelevantBlockConnected(dbtx, readyWallets, blockMeta, relevantTxs)
	if err != nil {
		logging.VPrint(logging.ERROR, "onRelevantBlockConnected error",
			logging.LogFormat{
				"block":  blockMeta.Hash.String(),
				"height": blockMeta.Height,
				"err":    err,
			})
		return err
	}
	return h.walletMgr.syncStore.SetSyncedTo(dbtx, blockMeta)
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
	readyWallets, err := h.getReadyWallets(dbtx)
	if err != nil {
		return err
	}
	for blocksToConnect.Front() != nil {
		block := blocksToConnect.Front().Value.(*wire.MsgBlock)

		if err := h.filterBlock(dbtx, readyWallets, block, addedExpireMempool); err != nil {
			return err
		}

		blocksToConnect.Remove(blocksToConnect.Front())
	}

	return nil
}

func worker(h *NtfnsHandler) {
	defer Recover()
	defer h.quitWg.Done()

	mwdb.View(h.walletMgr.db, func(tx mwdb.ReadTransaction) error {
		wss, err := h.walletMgr.syncStore.GetAllWalletStatus(tx)
		if err != nil {
			return err
		}
		h.taskChan = NewWalletTaskChan(len(wss))
		for _, ws := range wss {
			logging.CPrint(logging.DEBUG, "wallet status",
				logging.LogFormat{
					"syncedheight": ws.SyncedHeight,
					"walletId":     ws.WalletID,
					"ready":        ws.Ready(),
					"removed":      ws.IsRemoved(),
					"best":         h.bestBlock.Height,
				})
			// remove
			if ws.IsRemoved() {
				h.taskChan.PushRemove(ws.WalletID)
				logging.CPrint(logging.INFO, "restart removing", logging.LogFormat{"walletId": ws.WalletID})
				continue
			}

			// import
			if !ws.Ready() {
				h.taskChan.PushImport(ws.WalletID)
				logging.CPrint(logging.INFO, "restart importing", logging.LogFormat{"walletId": ws.WalletID})
			}
		}
		return nil
	})

	for {
		select {
		case <-h.quit:
			logging.CPrint(logging.INFO, "NtfnsHandler worker stopped")
			return
		case task := <-h.taskChan.C:
			switch task.taskType {
			case WalletTaskImport:
				fin, err := h.asyncImport(task.walletId)
				if err != nil {
					logging.CPrint(logging.ERROR, "asyncImport error", logging.LogFormat{
						"walletId": task.walletId,
						"err":      err,
					})
					if err == txmgr.ErrUnexpectedCreditNotFound {
						// TODO: mark status failed
						fin = true
					}
				}
				if !fin {
					h.taskChan.PushImport(task.walletId)
					continue
				}
				logging.CPrint(logging.INFO, "asyncImport finish", logging.LogFormat{"walletId": task.walletId})

			case WalletTaskRemove:
				err := h.asyncRemove(task.walletId)
				if err != nil {
					logging.CPrint(logging.ERROR, "asyncRemove failed", logging.LogFormat{
						"walletId": task.walletId,
						"err":      err,
					})
					if err != ErrTaskAbort {
						h.taskChan.PushRemove(task.walletId)
					}
					continue
				}
				logging.CPrint(logging.INFO, "asyncRemove finish", logging.LogFormat{"walletId": task.walletId})
			}
		}
	}
}

func (h *NtfnsHandler) asyncImport(walletId string) (finish bool, err error) {
	addrmgr, err := h.walletMgr.ksmgr.GetAddrManagerByAccountID(walletId)
	if err != nil {
		// could only be ErrAccountNotFound, but unexpected
		return true, err
	}

	mas := addrmgr.ManagedAddresses()
	relatedHashes := make([][]byte, 0)
	for _, ma := range mas {
		relatedHashes = append(relatedHashes, ma.ScriptAddress())
	}

	h.suspend(false, "[asyncImport] run", logging.LogFormat{"walletId": walletId})
	defer func() {
		h.resume(false, "[asyncImport] stop", logging.LogFormat{"walletId": walletId, "finish": finish})
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
		logging.VPrint(logging.DEBUG, "[asyncImport] range",
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
			blockMeta := &txmgr.BlockMeta{
				Hash:      header.BlockHash(),
				Height:    header.Height,
				Timestamp: header.Timestamp,
			}
			blockMeta.Loc, err = h.walletMgr.chainFetcher.FetchBlockLocByHeight(blockMeta.Height)
			if err != nil {
				logging.CPrint(logging.WARN, "FetchBlockLocByHeight error",
					logging.LogFormat{
						"err":    err,
						"height": blockMeta.Height,
					})
				return err
			}
			if !bytes.Equal(blockMeta.Loc.Hash[:], blockMeta.Hash[:]) {
				logging.CPrint(logging.WARN, "block hash mismatch", logging.LogFormat{"height": blockMeta.Height})
				return ErrMaybeChainRevoked
			}

			added := make([]wire.Hash, 0)
			for _, txloc := range txlocs {
				msg, err := fetcher.FetchTxByLoc(height, txloc)
				if err != nil {
					return err
				}

				rec, err := h.filterTxForImporting(msg, blockMeta, addrmgr)
				if err != nil {
					return err
				}
				if rec == nil {
					logging.CPrint(logging.ERROR, "unexpected error, tx is not relevant",
						logging.LogFormat{
							"tx":     rec.Hash.String(),
							"block":  blockMeta.Hash.String(),
							"height": blockMeta.Height,
						})
					return fmt.Errorf("unexpected error: tx is not relevant")
				}
				rec.TxLoc = txloc

				err = h.walletMgr.txStore.AddRelevantTxForImporting(dbtx, allBalances, rec, blockMeta)
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

func (h *NtfnsHandler) asyncRemove(walletId string) error {
	am, err := h.walletMgr.ksmgr.GetAddrManagerByAccountID(walletId)
	if err != nil {
		logging.CPrint(logging.ERROR, "unexpected error", logging.LogFormat{"err": err, "walletId": walletId})
		return nil
	}

	h.suspend(true, "[asyncRemove-1] deleting balance, address, staking/binding histories", logging.LogFormat{"walletId": walletId})
	err = mwdb.Update(h.walletMgr.db, func(wtx mwdb.DBTransaction) error {
		err := h.walletMgr.utxoStore.RemoveUnspentByWalletId(wtx, walletId)
		if err != nil {
			logging.CPrint(logging.ERROR, "RemoveUnspentByWalletId error", logging.LogFormat{"err": err})
			return err
		}
		err = h.walletMgr.utxoStore.RemoveAddressByWalletId(wtx, walletId)
		if err != nil {
			logging.CPrint(logging.ERROR, "RemoveAddressByWalletId error", logging.LogFormat{"err": err})
			return err
		}
		err = h.walletMgr.utxoStore.RemoveGameHistoryByWalletId(wtx, walletId)
		if err != nil {
			logging.CPrint(logging.ERROR, "RemoveGameHistoryByWalletId error", logging.LogFormat{"err": err})
			return err
		}

		return h.walletMgr.utxoStore.RemoveMinedBalance(wtx, walletId)
	})
	h.resume(true, "[asyncRemove-1] stop", logging.LogFormat{"walletId": walletId})
	if err != nil {
		logging.CPrint(logging.ERROR, "[asyncRemove-1] failed", logging.LogFormat{"err": err})
		return err
	}

	for {
		select {
		case <-h.quit:
			return ErrTaskAbort
		default:
			h.suspend(true, "[asyncRemove-2] deleting credits, keystore", logging.LogFormat{"walletId": walletId})
			finish := false
			var removedTx []*wire.Hash
			err := mwdb.Update(h.walletMgr.db, func(wtx mwdb.DBTransaction) (err error) {
				removedTx, finish, err = h.walletMgr.txStore.RemoveRelevantTx(wtx, am)
				if finish {
					err = h.walletMgr.syncStore.DeleteWalletStatus(wtx, walletId)
					if err == nil {
						_, err = h.walletMgr.ksmgr.DeleteKeystore(wtx, walletId)
						if err != nil {
							logging.CPrint(logging.ERROR, "DeleteKeystore error", logging.LogFormat{"err": err})
						}
					} else {
						logging.CPrint(logging.ERROR, "DeleteWalletStatus error", logging.LogFormat{"err": err})
					}
				}
				return err
			})
			h.resume(true, "[asyncRemove-2] stop", logging.LogFormat{"walletId": walletId})
			if err != nil {
				logging.CPrint(logging.ERROR, "[asyncRemove-2] failed", logging.LogFormat{"err": err})
				if finish {
					mwdb.View(h.walletMgr.db, func(rtx mwdb.ReadTransaction) error {
						h.walletMgr.ksmgr.UpdateManagedKeystores(rtx, walletId)
						return nil
					})
				}
				return err
			}
			h.RemoveMempoolTx(removedTx)

			if finish {
				return nil
			}
		}
	}
}

func (h *NtfnsHandler) OnImportWallet(walletId string) {
	h.taskChan.PushImport(walletId)
}

func (h *NtfnsHandler) OnRemoveWallet(walletId string) error {
	err := mwdb.Update(h.walletMgr.db, func(wtx mwdb.DBTransaction) error {
		ws, err := h.walletMgr.syncStore.GetWalletStatus(wtx, walletId)
		if err != nil {
			return err
		}
		if !ws.Ready() {
			return ErrWalletUnready
		}
		return h.walletMgr.syncStore.MarkDeleteWallet(wtx, walletId)
	})
	if err != nil {
		return err
	}
	h.taskChan.PushRemove(walletId)
	return nil
}

func (h *NtfnsHandler) IsWorkerBusy() bool {
	return h.taskChan.IsBusy()
}

func (h *NtfnsHandler) RemoveMempoolTx(txs []*wire.Hash) {
	h.memMtx.Lock()
	for _, hash := range txs {
		delete(h.mempool, *hash)
	}
	h.memMtx.Unlock()
}

func (h *NtfnsHandler) processConnectedBlock(newBlock *wire.MsgBlock) error {

	h.memMtx.Lock()
	bestBlock := h.bestBlock
	h.memMtx.Unlock()

	rollbackBlock := make(map[uint64]struct{})
	addedExpireMempool := make(map[uint64]map[wire.Hash]struct{})
	err := mwdb.Update(h.walletMgr.db, func(tx mwdb.DBTransaction) error {
		if newBlock.Header.Previous == bestBlock.Hash {
			readyWallets, err := h.getReadyWallets(tx)
			if err != nil {
				return err
			}
			if err := h.filterBlock(tx, readyWallets, newBlock, addedExpireMempool); err != nil {
				logging.CPrint(logging.ERROR, "failed to filter block", logging.LogFormat{"err": err})
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
	knownBestHeight := h.walletMgr.ChainIndexerSyncedHeight()
	bestPeer := h.walletMgr.server.SyncManager().BestPeer()
	if bestPeer != nil && bestPeer.Height > knownBestHeight {
		knownBestHeight = bestPeer.Height
	}
	if syncHeight, _ := h.walletMgr.SyncedTo(); syncHeight < knownBestHeight-1 {
		return nil
	}

	logging.CPrint(logging.INFO, "recv tx",
		logging.LogFormat{
			"tx": tx.TxHash().String(),
		})
	var readyWallets map[string]struct{}
	err := mwdb.View(h.walletMgr.db, func(rtx mwdb.ReadTransaction) (err error) {
		readyWallets, err = h.getReadyWallets(rtx)
		return
	})
	if err != nil {
		return err
	}
	if _, _, err := h.filterTx(tx, nil, nil, readyWallets); err != nil {
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

func (h *NtfnsHandler) suspend(log bool, msg string, fields logging.LogFormat) {
	h.sigSuspend <- struct{}{}
	if log {
		logging.VPrint(logging.INFO, msg, fields)
	}
}

func (h *NtfnsHandler) resume(log bool, msg string, fields logging.LogFormat) {
	h.sigResume <- struct{}{}
	if log {
		logging.VPrint(logging.INFO, msg, fields)
	}
}

func (h *NtfnsHandler) getReadyWallets(rtx mwdb.ReadTransaction) (map[string]struct{}, error) {
	readyWallets := make(map[string]struct{})
	for _, name := range h.walletMgr.ksmgr.ListKeystoreNames() {
		ws, err := h.walletMgr.syncStore.GetWalletStatus(rtx, name)
		if err != nil {
			return nil, err
		}
		if ws.Ready() && !ws.IsRemoved() {
			readyWallets[name] = struct{}{}
		}
	}
	return readyWallets, nil
}

func Recover() {
	err := recover()
	if err != nil {
		logging.CPrint(logging.FATAL, "panic", logging.LogFormat{
			"err":   err,
			"stack": string(debug.Stack()),
		})
	}
}
