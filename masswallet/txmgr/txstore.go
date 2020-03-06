package txmgr

import (
	"errors"
	"fmt"

	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/ifc"
	"massnet.org/mass-wallet/masswallet/keystore"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/wire"
)

//TxStore definition
type TxStore struct {
	chainParams  *config.Params
	chainFetcher ifc.ChainFetcher
	bucketMeta   *StoreBucketMeta

	utxoStore *UtxoStore
	syncStore *SyncStore
	ksmgr     *keystore.KeystoreManager
}

// NewTxStore ...
func NewTxStore(chainFetcher ifc.ChainFetcher, store mwdb.Bucket, utxoStore *UtxoStore,
	syncStore *SyncStore, ksmgr *keystore.KeystoreManager,
	bucketMeta *StoreBucketMeta, chainParams *config.Params) (t *TxStore, err error) {
	t = &TxStore{
		chainParams:  chainParams,
		chainFetcher: chainFetcher,
		bucketMeta:   bucketMeta,
		utxoStore:    utxoStore,
		syncStore:    syncStore,
		ksmgr:        ksmgr,
	}

	//bucketUnmined
	bucket, err := mwdb.GetOrCreateBucket(store, bucketUnmined)
	if err != nil {
		return nil, err
	}
	t.bucketMeta.nsUnmined = bucket.GetBucketMeta()

	// bucketTxRecords
	bucket, err = mwdb.GetOrCreateBucket(store, bucketTxRecords)
	if err != nil {
		return nil, err
	}
	t.bucketMeta.nsTxRecords = bucket.GetBucketMeta()

	//bucketBlocks
	bucket, err = mwdb.GetOrCreateBucket(store, bucketBlocks)
	if err != nil {
		return nil, err
	}
	t.bucketMeta.nsBlocks = bucket.GetBucketMeta()

	//bucketLGOutput
	bucket, err = mwdb.GetOrCreateBucket(store, bucketLGOutput)
	if err != nil {
		return nil, err
	}
	t.bucketMeta.nsLGHistory = bucket.GetBucketMeta()

	//bucketUnminedLGOutput
	bucket, err = mwdb.GetOrCreateBucket(store, bucketUnminedLGOutput)
	if err != nil {
		return nil, err
	}
	t.bucketMeta.nsUnminedLGHistory = bucket.GetBucketMeta()
	return
}

func checkAddress(addr string) error {
	return nil
}

func (s *TxStore) updateMinedBalance(tx mwdb.DBTransaction,
	allBalances map[string]massutil.Amount, rec *TxRecord, block *BlockMeta) error {

	spender := indexedIncidence{
		incidence: incidence{
			txHash: rec.Hash,
			block:  *block,
		},
	}

	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsDebits := tx.FetchBucket(s.bucketMeta.nsDebits)

	for _, rel := range rec.RelevantTxIn {
		prevOut := &rec.MsgTx.TxIn[rel.Index].PreviousOutPoint
		unspentKey, credKey, err := existsUnspent(nsUnspent, rel.WalletId, prevOut)
		if err != nil {
			return err
		}
		if credKey == nil {
			logging.CPrint(logging.ERROR, "unexpected: utxo related to input not found",
				logging.LogFormat{
					"tx":        rec.Hash.String(),
					"txInIndex": rel.Index,
					"wallet":    rel.WalletId,
					"block":     block,
					"prevTx":    prevOut.Hash.String(),
					"prevOut":   prevOut.Index,
				})
			return fmt.Errorf("unexpected error")
		}

		spender.index = uint32(rel.Index)
		amt, err := spendCredit(nsCredits, credKey, &spender)
		if err != nil {
			return err
		}

		err = putDebit(nsDebits, &rec.Hash, uint32(rel.Index), amt, block, credKey)
		if err != nil {
			return err
		}
		if err := deleteRawUnspent(nsUnspent, unspentKey); err != nil {
			return err
		}

		newBal, err := allBalances[rel.WalletId].Sub(amt)
		if err != nil {
			return err
		}
		allBalances[rel.WalletId] = newBal
	}
	return nil
}

// AddRelevantTx ...
func (s *TxStore) AddRelevantTx(tx mwdb.DBTransaction, allBalances map[string]massutil.Amount, rec *TxRecord, block *BlockMeta) error {
	err := s.InsertTx(tx, allBalances, rec, block)
	if err != nil {
		return err
	}

	return s.utxoStore.AddCredits(tx, allBalances, rec, block)
}

// InsertTx ...
func (s *TxStore) InsertTx(tx mwdb.DBTransaction, allBalances map[string]massutil.Amount, rec *TxRecord, block *BlockMeta) error {
	if block == nil {
		return s.insertMemPoolTx(tx, rec)
	}
	return s.insertMinedTx(tx, allBalances, rec, block)
}

func (s *TxStore) insertMemPoolTx(tx mwdb.DBTransaction, rec *TxRecord) error {

	if blockchain.IsCoinBaseTx(&rec.MsgTx) {
		return fmt.Errorf("invalid unmined tx: coinbase")
	}

	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	um, err := existsRawUnmined(nsUnmined, rec.Hash[:])
	if err != nil {
		return err
	}
	if um != nil {
		return nil
	}

	logging.CPrint(logging.DEBUG, "Inserting unconfirmed transaction", logging.LogFormat{"tx": rec.Hash.String()})

	v, err := valueTxRecord(rec)
	if err != nil {
		return err
	}
	err = putRawUnmined(nsUnmined, rec.Hash[:], v)
	if err != nil {
		return err
	}

	return s.utxoStore.insertUnminedInputs(tx, rec)
}

func (s *TxStore) insertMinedTx(tx mwdb.DBTransaction, allBalances map[string]massutil.Amount, rec *TxRecord, block *BlockMeta) error {
	nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)
	if _, v := existsTxRecord(nsTxRecords, &rec.Hash, block); v != nil {
		return nil
	}
	var err error
	nsBlocks := tx.FetchBucket(s.bucketMeta.nsBlocks)
	blockKey, blockValue, err := existsBlockRecord(nsBlocks, block.Height)
	if err != nil {
		return err
	}
	if blockValue == nil {
		err = putBlockRecord(nsBlocks, block, &rec.Hash)
	} else {
		blockValue, err = appendRawBlockRecord(blockValue, &rec.Hash)
		if err != nil {
			return err
		}
		err = putRawBlockRecord(nsBlocks, blockKey, blockValue)
	}
	if err != nil {
		return err
	}
	if err := putTxRecord(nsTxRecords, rec, block); err != nil {
		return err
	}

	if err := s.updateMinedBalance(tx, allBalances, rec, block); err != nil {
		return err
	}

	// If this transaction previously existed within the store as unmined,
	// we'll need to remove it from the unmined bucket.
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	v, err := existsRawUnmined(nsUnmined, rec.Hash[:])
	if err != nil {
		return err
	}
	if v != nil {
		logging.VPrint(logging.INFO, "Marking unconfirmed transaction mined in block",
			logging.LogFormat{
				"tx":        rec.Hash.String(),
				"block":     block.Height,
				"blockHash": block.Hash.String(),
			})

		if err := s.utxoStore.deleteUnminedCredits(tx, rec); err != nil {
			return err
		}
		if err := deleteRawUnmined(nsUnmined, rec.Hash[:]); err != nil {
			return err
		}
	}

	return s.removeDoubleSpends(tx, rec)
}

func (s *TxStore) AddRelevantTxForImporting(tx mwdb.DBTransaction,
	allBalances map[string]massutil.Amount, rec *TxRecord, block *BlockMeta) error {

	err := s.insertMinedTxForImporting(tx, allBalances, rec, block)
	if err != nil {
		return err
	}

	return s.utxoStore.AddCredits(tx, allBalances, rec, block)
}

func (s *TxStore) insertMinedTxForImporting(tx mwdb.DBTransaction,
	allBalances map[string]massutil.Amount, rec *TxRecord, block *BlockMeta) error {
	nsBlocks := tx.FetchBucket(s.bucketMeta.nsBlocks)
	nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)

	_, v := existsTxRecord(nsTxRecords, &rec.Hash, block)
	exists := v != nil

	blockKey, blockValue, err := existsBlockRecord(nsBlocks, block.Height)
	if err != nil {
		return err
	}
	if blockValue == nil {
		if exists {
			// double check
			return errors.New("unexpected error: tx record exists but block record not")
		}
		err = putBlockRecord(nsBlocks, block, &rec.Hash)
	} else {
		blkHash, err := readBlockHashFromValue(blockValue)
		if err != nil {
			return err
		}
		if block.Hash != blkHash {
			logging.VPrint(logging.INFO, "chain reorg found during importing",
				logging.LogFormat{
					"exist":  blkHash.String(),
					"new":    block.Hash.String(),
					"height": block.Height,
				})
			return ErrChainReorg
		}
		if !exists {
			blockValue, err = appendRawBlockRecord(blockValue, &rec.Hash)
			if err != nil {
				return err
			}
			err = putRawBlockRecord(nsBlocks, blockKey, blockValue)
		}
	}
	if err != nil {
		return err
	}

	if !exists {
		if err := putTxRecord(nsTxRecords, rec, block); err != nil {
			return err
		}
	}

	if err := s.updateMinedBalance(tx, allBalances, rec, block); err != nil {
		return err
	}

	// double check
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	v, err = existsRawUnmined(nsUnmined, rec.Hash[:])
	if err != nil {
		logging.VPrint(logging.ERROR, "read unmined error",
			logging.LogFormat{
				"err":       err,
				"tx":        rec.Hash.String(),
				"block":     block.Height,
				"blockHash": block.Hash.String(),
			})
		return nil
	}
	if v != nil {
		logging.VPrint(logging.ERROR, "unexpected error: unmined tx exists",
			logging.LogFormat{
				"tx":        rec.Hash.String(),
				"block":     block.Height,
				"blockHash": block.Hash.String(),
			})
	}
	return nil
}

func (s *TxStore) removeDoubleSpends(tx mwdb.DBTransaction, rec *TxRecord) error {

	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)

	for _, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOutPoint
		prevOutKey := canonicalOutPoint(&prevOut.Hash, prevOut.Index)

		doubleSpendHashes := fetchUnminedInputSpendTxHashes(nsUnminedInputs, prevOutKey)
		for _, doubleSpendHash := range doubleSpendHashes {
			doubleSpendVal, err := existsRawUnmined(nsUnmined, doubleSpendHash[:])
			if err != nil {
				return err
			}

			if len(doubleSpendVal) == 0 {
				continue
			}

			var doubleSpend TxRecord
			doubleSpend.Hash = doubleSpendHash
			err = readRawTxRecordValue(doubleSpendVal, &doubleSpend)
			if err != nil {
				return err
			}

			logging.VPrint(logging.DEBUG, "Removing double spending transaction",
				logging.LogFormat{
					"tx": doubleSpend.Hash.String(),
				})

			if err := s.removeConflict(tx, &doubleSpend); err != nil {
				return err
			}
		}
	}

	// delete unmined inputs in case only the mined tx spend them
	return s.utxoStore.deleteUnminedInputs(tx, rec)
}

func (s *TxStore) removeConflict(tx mwdb.DBTransaction, rec *TxRecord) error {

	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)

	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		spenderHashes := fetchUnminedInputSpendTxHashes(nsUnminedInputs, k)
		for _, spenderHash := range spenderHashes {
			spenderVal, err := existsRawUnmined(nsUnmined, spenderHash[:])
			if err != nil {
				return err
			}

			if len(spenderVal) == 0 {
				continue
			}

			var spender TxRecord
			spender.Hash = spenderHash
			err = readRawTxRecordValue(spenderVal, &spender)
			if err != nil {
				return err
			}

			logging.VPrint(logging.DEBUG, "Transaction %v is part of a removed conflict "+
				"chain -- removing as well",
				logging.LogFormat{
					"tx": spender.Hash.String(),
				})
			if err := s.removeConflict(tx, &spender); err != nil {
				return err
			}
		}
		if err := deleteRawUnminedCredit(nsUnminedCredits, k); err != nil {
			return err
		}
	}

	// If this tx spends any previous credits (either mined or unmined), set
	// each unspent.  Mined transactions are only marked spent by having the
	// output in the unmined inputs bucket.
	if err := s.utxoStore.deleteUnminedInputs(tx, rec); err != nil {
		return err
	}
	if err := s.utxoStore.removeUnminedLGHistory(tx, rec); err != nil {
		return err
	}
	return deleteRawUnmined(nsUnmined, rec.Hash[:])
}

func (s *TxStore) ExistUnminedTx(tx mwdb.ReadTransaction, hash *wire.Hash) (mtx *wire.MsgTx, err error) {
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	var rec TxRecord
	v, err := existsRawUnmined(nsUnmined, hash[:])
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return nil, ErrNotFound
	}
	err = readRawTxRecordValue(v, &rec)
	if err != nil {
		return nil, err
	}
	return &rec.MsgTx, nil
}

// for current wallet
func (s *TxStore) ExistsTx(tx mwdb.ReadTransaction, out *wire.OutPoint) (mtx *wire.MsgTx, err error) {
	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)

	cred := credit{
		block: &BlockMeta{},
	}

	_, credKey, err := existsUnspent(nsUnspent, s.ksmgr.CurrentKeystore().Name(), out)
	if err != nil {
		return nil, err
	}

	found := false

	if credKey != nil { // unspent exists
		err = readRawCreditKey(credKey, &cred)
		if err != nil {
			return nil, err
		}
		found = true
	} else { // unspent not exists
		entries, err := getCreditsByTxHash(nsCredits, &out.Hash)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			err = readRawCreditKey(entry.Key, &cred)
			if err != nil {
				return nil, err
			}
			if cred.outPoint.Index == out.Index {
				found = true
				break
			}
		}
	}

	if found {
		_, v := existsTxRecord(nsTxRecords, &cred.outPoint.Hash, cred.block)
		if len(v) != 0 {
			var rec TxRecord
			if err = readRawTxRecordValue(v, &rec); err != nil {
				return nil, err
			}
			return &rec.MsgTx, nil
		} else { // double check
			logging.CPrint(logging.ERROR, "tx record not exists",
				logging.LogFormat{
					"tx":   out.Hash.String(),
					"vout": out.Index,
				})
			return nil, fmt.Errorf("unexpected error")
		}
	}
	return nil, ErrNotFound
}

// ExistsUtxo returns ErrNotFound if not exists
// for current wallet
func (s *TxStore) ExistsUtxo(tx mwdb.ReadTransaction, out *wire.OutPoint) (flags *UtxoFlags, err error) {
	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)

	cred := credit{
		block: &BlockMeta{},
	}

	// unspent exists
	uspKey, credKey, err := existsUnspent(nsUnspent, s.ksmgr.CurrentKeystore().Name(), out)
	if err != nil {
		return nil, err
	}
	if credKey != nil {
		credValue, err := existsRawCredit(nsCredits, credKey)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to fetch credit",
				logging.LogFormat{
					"error": err,
					"tx":    out.Hash.String(),
					"vout":  out.Index,
				})
			return nil, err
		}
		if credValue == nil { // double check
			logging.CPrint(logging.ERROR, "unspent exists but credit not found",
				logging.LogFormat{
					"tx":   out.Hash.String(),
					"vout": out.Index,
				})
			return nil, fmt.Errorf("unexpected error")
		}

		err = readCreditValue(credValue, &cred)
		if err != nil {
			return nil, err
		}
		if cred.flags.Spent { // double check
			logging.CPrint(logging.ERROR, "utxo exists but credit not",
				logging.LogFormat{
					"tx":   out.Hash.String(),
					"vout": out.Index,
				})
			return nil, fmt.Errorf("unexpected error")
		}
		cred.flags.SpentByUnmined = existsRawUnminedInput(nsUnminedInputs, uspKey) != nil
		return &cred.flags, nil
	}

	// unspent not exists
	entries, err := getCreditsByTxHash(nsCredits, &out.Hash)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		err = readRawCreditKey(entry.Key, &cred)
		if err != nil {
			return nil, err
		}
		if cred.outPoint.Index == out.Index {
			err = readCreditValue(entry.Value, &cred)
			if err != nil {
				return nil, err
			}
			if !cred.flags.Spent { // double check
				logging.CPrint(logging.ERROR, "utxo not exists but credit not spent",
					logging.LogFormat{
						"tx":   out.Hash.String(),
						"vout": out.Index,
					})
				return nil, fmt.Errorf("unexpected error")
			}
			cred.flags.SpentByUnmined = existsRawUnminedInput(nsUnminedInputs, uspKey) != nil
			return &cred.flags, nil
		}
	}

	// unmined credits
	if len(entries) == 0 {
		nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
		entries, err = getCreditsByTxHash(nsUnminedCredits, &out.Hash)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			err = readUnminedCreditKey(entry.Key, &cred)
			if err != nil {
				return nil, err
			}
			if cred.outPoint.Index == out.Index {
				err = readCreditValue(entry.Value, &cred)
				if err != nil {
					return nil, err
				}
				if cred.flags.Spent { // double check
					logging.CPrint(logging.ERROR, "found unmined credit spent",
						logging.LogFormat{
							"tx":   out.Hash.String(),
							"vout": out.Index,
						})
					return nil, fmt.Errorf("unexpected error")
				}
				cred.flags.SpentByUnmined = existsRawUnminedInput(nsUnminedInputs, uspKey) != nil
				cred.flags.IsUnmined = true
				return &cred.flags, nil
			}
		}
	}

	return nil, ErrNotFound
}

// Rollback ...
func (s *TxStore) Rollback(tx mwdb.DBTransaction, height uint64) error {
	return s.rollback(tx, height)
}

func (s *TxStore) rollback(tx mwdb.DBTransaction, height uint64) error {
	allMined, err := s.utxoStore.FetchAllMinedBalance(tx)
	if err != nil {
		return err
	}

	var coinBaseCredits []wire.OutPoint
	var heightsToRemove []uint64

	nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)
	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
	nsDebits := tx.FetchBucket(s.bucketMeta.nsDebits)
	nsBlocks := tx.FetchBucket(s.bucketMeta.nsBlocks)
	nsAddresses := tx.FetchBucket(s.bucketMeta.nsAddresses)
	nsLGHistory := tx.FetchBucket(s.bucketMeta.nsLGHistory)
	nsUnminedLGHistory := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)

	syncedTo, err := s.syncStore.SyncedTo(tx)
	if err != nil {
		return err
	}
	for curHeight := syncedTo.Height; height <= curHeight; curHeight-- {
		rbBlock, err := fetchBlockRecord(nsBlocks, curHeight)
		if err != nil {
			return err
		}
		if rbBlock == nil {
			continue
		}

		heightsToRemove = append(heightsToRemove, rbBlock.Height)

		for i := len(rbBlock.transactions) - 1; i >= 0; i-- {
			txHash := &rbBlock.transactions[i]

			_, recVal := existsTxRecord(nsTxRecords, txHash, &rbBlock.BlockMeta)
			if len(recVal) == 0 {
				logging.CPrint(logging.WARN, "tx record already deleted",
					logging.LogFormat{
						"tx":      txHash.String(),
						"height":  rbBlock.Height,
						"blkHash": rbBlock.Hash.String(),
						"blokTxs": rbBlock.transactions,
					})
				continue
			}
			var rec TxRecord
			rec.Hash = *txHash
			err = readRawTxRecordValue(recVal, &rec)
			if err != nil {
				return err
			}

			err = deleteTxRecord(nsTxRecords, txHash, &rbBlock.BlockMeta)
			if err != nil {
				return err
			}

			// Handle coinbase transactions specially since they are
			// not moved to the unconfirmed store.  A coinbase cannot
			// contain any debits, but all credits should be removed
			// and the mined balance decremented.
			if blockchain.IsCoinBaseTx(&rec.MsgTx) {
				op := wire.OutPoint{Hash: rec.Hash}
				for i, output := range rec.MsgTx.TxOut {
					op.Index = uint32(i)

					k, v, err := existsCredit(nsCredits, &rec.Hash, op.Index, &rbBlock.BlockMeta)
					if err != nil {
						return err
					}
					if v == nil {
						continue
					}

					err = deleteRawCredit(nsCredits, k)
					if err != nil {
						return err
					}

					coinBaseCredits = append(coinBaseCredits, op)

					ps, err := utils.ParsePkScript(output.PkScript, s.chainParams)
					if err != nil {
						logging.CPrint(logging.WARN, "unexpected: failed to parse txout pkscript",
							logging.LogFormat{
								"tx":         rec.Hash.String(),
								"txOutIndex": i,
								"err":        err,
							})
						return err
					}
					ma, err := s.ksmgr.GetManagedAddressByStdAddress(ps.StdEncodeAddress())
					if err != nil {
						logging.CPrint(logging.WARN, "unexpected: address not relevant",
							logging.LogFormat{
								"tx":         rec.Hash.String(),
								"txOutIndex": i,
								"err":        err,
							})
						continue
					}

					unspentKey, credKey, err := existsUnspent(nsUnspent, ma.Account(), &op)
					if err != nil {
						return err
					}
					if credKey != nil {
						err = deleteRawUnspent(nsUnspent, unspentKey)
						if err != nil {
							return err
						}

						amt, err := massutil.NewAmountFromInt(output.Value)
						if err != nil {
							return err
						}
						newBal, err := allMined[ma.Account()].Sub(amt)
						if err != nil {
							return err
						}
						allMined[ma.Account()] = newBal
					}

					// check and remove address
					addrRec := &addressRecord{
						walletId:     ma.Account(),
						addressClass: ps.AddressClass(),
					}
					if ps.IsStaking() {
						addrRec.encodeAddress = ps.SecondEncodeAddress()
					} else {
						addrRec.encodeAddress = ps.StdEncodeAddress()
					}
					addrKey, err := keyAddressRecord(addrRec)
					if err != nil {
						return err
					}
					addrVal, err := existsRawAddressRecord(nsAddresses, addrKey)
					if err != nil {
						return err
					}
					if addrVal == nil { // debug code
						logging.CPrint(logging.INFO, "address not exist",
							logging.LogFormat{
								"tx":      txHash.String(),
								"height":  rbBlock.Height,
								"blkHash": rbBlock.Hash.String(),
								"key":     addrKey,
								"vout":    i,
							})
					} else {
						if curHeight > 0 && readAddressHeight(addrVal) == curHeight {
							err = deleteRawAddressRecord(nsAddresses, addrKey)
							if err != nil {
								return err
							}
						}
					}
				}
				continue
			}

			err = putRawUnmined(nsUnmined, txHash[:], recVal)
			if err != nil {
				return err
			}

			// walletid -> (isWithdraw -> struct{})
			recWallets := make(map[string]map[bool]struct{})

			// non coinbase tx
			for i, input := range rec.MsgTx.TxIn {
				prevOut := &input.PreviousOutPoint
				err = putRawUnminedInput(nsUnminedInputs,
					canonicalOutPoint(&prevOut.Hash, prevOut.Index),
					rec.Hash[:])
				if err != nil {
					return err
				}

				// If this input is a debit, remove the debit
				// record and mark the credit that it spent as
				// unspent, incrementing the mined balance.
				debKey, credKey, err := existsDebit(nsDebits, &rec.Hash, uint32(i), &rbBlock.BlockMeta)
				if err != nil {
					return err
				}
				if debKey == nil {
					logging.CPrint(logging.WARN, "debit not exist",
						logging.LogFormat{
							"tx":     rec.Hash.String(),
							"vin":    i,
							"height": rbBlock.Height,
							"blk":    rbBlock.Hash.String(),
						})
					continue
				}
				err = deleteRawDebit(nsDebits, debKey)
				if err != nil {
					return err
				}

				// unspendRawCredit does not error in case the
				// no credit exists for this key, but this
				// behavior is correct.  Since blocks are
				// removed in increasing order, this credit
				// may have already been removed from a
				// previously removed transaction record in
				// this rollback.
				cred, err := unspendRawCredit(nsCredits, credKey)
				if err != nil {
					return err
				}
				if cred == nil {
					logging.CPrint(logging.WARN, "unspend non-existence credit",
						logging.LogFormat{
							"tx":        rec.Hash.String(),
							"vin":       i,
							"prevTx":    prevOut.Hash.String(),
							"prevIndex": prevOut.Index,
							"height":    rbBlock.Height,
							"blk":       rbBlock.Hash.String(),
						})
					continue
				}

				ma, err := s.ksmgr.GetManagedAddressByScriptHash(cred.scriptHash)
				if err != nil {
					if err == keystore.ErrScriptHashNotFound {
						logging.CPrint(logging.DEBUG, "address not relevant",
							logging.LogFormat{
								"tx":        rec.Hash.String(),
								"txInIndex": i,
								"height":    curHeight,
							})
						continue
					}
					return err
				}

				unspentVal, err := fetchNsUnspentValueFromRawCredit(credKey)
				if err != nil {
					return err
				}
				err = putRawUnspent(nsUnspent,
					canonicalUnspentKey(ma.Account(), &prevOut.Hash, prevOut.Index),
					unspentVal)
				if err != nil {
					return err
				}
				newBal, err := allMined[ma.Account()].Add(cred.amount)
				if err != nil {
					return err
				}
				allMined[ma.Account()] = newBal

				if cred.isStaking() || cred.isBinding() {
					m, ok := recWallets[ma.Account()]
					if !ok {
						m = make(map[bool]struct{})
						recWallets[ma.Account()] = m
					}
					m[true] = struct{}{}
				}
			}

			// txout
			for i, output := range rec.MsgTx.TxOut {
				k, v, err := existsCredit(nsCredits, &rec.Hash, uint32(i), &rbBlock.BlockMeta)
				if err != nil {
					return err
				}
				if v == nil {
					continue
				}

				err = deleteRawCredit(nsCredits, k)
				if err != nil {
					return err
				}

				unminedCredVal, err := valueUnminedCreditFromMined(v)
				if err != nil {
					return err
				}
				err = putRawUnminedCredit(nsUnminedCredits, canonicalOutPoint(&rec.Hash, uint32(i)), unminedCredVal)
				if err != nil {
					return err
				}

				ps, err := utils.ParsePkScript(output.PkScript, s.chainParams)
				if err != nil {
					logging.CPrint(logging.WARN, "unexpected: failed to parse txout pkscript",
						logging.LogFormat{
							"tx":         rec.Hash.String(),
							"txOutIndex": i,
							"height":     curHeight,
							"err":        err,
						})
					return err
				}
				ma, err := s.ksmgr.GetManagedAddressByStdAddress(ps.StdEncodeAddress())
				if err != nil {
					logging.CPrint(logging.DEBUG, "address not relevant",
						logging.LogFormat{
							"tx":         rec.Hash.String(),
							"txOutIndex": i,
							"height":     curHeight,
							"err":        err,
						})
					continue
				}

				unspentKey, credKey, err := existsUnspent(nsUnspent, ma.Account(),
					&wire.OutPoint{Hash: rec.Hash, Index: uint32(i)})
				if err != nil {
					return err
				}
				if credKey != nil {
					err = deleteRawUnspent(nsUnspent, unspentKey)
					if err != nil {
						return err
					}

					amt, err := massutil.NewAmountFromInt(output.Value)
					if err != nil {
						return err
					}
					newBal, err := allMined[ma.Account()].Sub(amt)
					if err != nil {
						return err
					}
					allMined[ma.Account()] = newBal
				}

				// check and delete address
				addrRec := &addressRecord{
					walletId:     ma.Account(),
					addressClass: ps.AddressClass(),
				}
				if ps.IsStaking() {
					addrRec.encodeAddress = ps.SecondEncodeAddress()
				} else {
					addrRec.encodeAddress = ps.StdEncodeAddress()
				}
				addrKey, err := keyAddressRecord(addrRec)
				if err != nil {
					return err
				}
				addrVal, err := existsRawAddressRecord(nsAddresses, addrKey)
				if err != nil {
					return err
				}
				if addrVal == nil { // debug code
					logging.CPrint(logging.INFO, "address not exist",
						logging.LogFormat{
							"tx":      txHash.String(),
							"height":  rbBlock.Height,
							"blkHash": rbBlock.Hash.String(),
							"key":     addrKey,
							"vout":    i,
						})
				} else {
					if curHeight > 0 && readAddressHeight(addrVal) == curHeight {
						err = deleteRawAddressRecord(nsAddresses, addrKey)
						if err != nil {
							return err
						}
					}
				}
				// check and delete lockt output
				if ps.IsStaking() || ps.IsBinding() {
					m, ok := recWallets[ma.Account()]
					if !ok {
						m = make(map[bool]struct{})
						recWallets[ma.Account()] = m
					}
					m[false] = struct{}{}
				}
			}

			// rollback mined history to unmined
			for walletId, m := range recWallets {
				for isWithdraw, _ := range m {
					for _, isBinding := range []bool{false, true} {
						lout := &lgTxHistory{
							walletId:    walletId,
							isBinding:   isBinding,
							isWithdraw:  isWithdraw,
							txhash:      rec.Hash,
							blockHeight: curHeight,
						}

						k := keyMinedLGHistory(lout)
						v, err := existsRawLGOutput(nsLGHistory, k)
						if err != nil {
							return err
						}
						if len(v) != 0 {
							err = deleteRawLGOutput(nsLGHistory, k)
							if err != nil {
								return err
							}
							err = putRawUnminedLGHistory(nsUnminedLGHistory, keyUnminedLGHistory(lout), v)
							if err != nil {
								return err
							}
						} else if len(m) == 2 {
							logging.CPrint(logging.WARN, "staking/binding history not found",
								logging.LogFormat{
									"tx":         rec.Hash.String(),
									"height":     curHeight,
									"wallet":     walletId,
									"isWithdraw": isWithdraw,
									"isBinding":  isBinding,
								})
						}
					}
				}
			}
		}
	}

	// delete block record
	for _, h := range heightsToRemove {
		err = deleteBlockRecord(nsBlocks, h)
		if err != nil {
			return err
		}
	}

	// remove coinbase credits
	for _, op := range coinBaseCredits {
		opKey := canonicalOutPoint(&op.Hash, op.Index)
		unminedSpendTxHashKeys := fetchUnminedInputSpendTxHashes(nsUnminedInputs, opKey)
		for _, unminedSpendTxHashKey := range unminedSpendTxHashKeys {
			unminedVal, err := existsRawUnmined(nsUnmined, unminedSpendTxHashKey[:])
			if err != nil {
				return err
			}
			// If the spending transaction spends multiple outputs
			// from the same transaction, we'll find duplicate
			// entries within the store, so it's possible we're
			// unable to find it if the conflicts have already been
			// removed in a previous iteration.
			if len(unminedVal) == 0 {
				continue
			}

			var unminedRec TxRecord
			unminedRec.Hash = unminedSpendTxHashKey
			err = readRawTxRecordValue(unminedVal, &unminedRec)
			if err != nil {
				return err
			}

			logging.CPrint(logging.DEBUG, "Transaction spends a removed coinbase output -- removing as well",
				logging.LogFormat{
					"tx": unminedRec.Hash.String(),
				})

			err = s.removeConflict(tx, &unminedRec)
			if err != nil {
				return err
			}
		}
	}

	return s.utxoStore.UpdateMinedBalances(tx, allMined)
}

func (s *TxStore) removableTxForRemoveWallet(msgTx *wire.MsgTx, scriptHashSet map[string]struct{}) (bool, error) {

	for i, txout := range msgTx.TxOut {
		ps, err := utils.ParsePkScript(txout.PkScript, s.chainParams)
		if err != nil {
			if err == utils.ErrUnsupportedScript {
				continue
			}
			logging.CPrint(logging.ERROR, "failed to parse pkscript",
				logging.LogFormat{
					"tx":         msgTx.TxHash().String(),
					"txOutIndex": i,
					"err":        err,
				})
			return false, err
		}
		if _, ok := scriptHashSet[string(ps.StdScriptAddress())]; !ok {
			_, err := s.ksmgr.GetManagedAddressByStdAddress(ps.StdEncodeAddress())
			if err != nil {
				continue
			}
			return false, nil
		}
	}
	return true, nil
}

func (s *TxStore) checkBlockRecordAfterTxRemoved(nsBlocks mwdb.Bucket, blkDeleted map[uint64]map[wire.Hash]struct{}) error {

	for height, hashes := range blkDeleted {
		k, v, err := existsBlockRecord(nsBlocks, height)
		if err != nil {
			return err
		}

		if v == nil {
			logging.CPrint(logging.WARN, "block record not found",
				logging.LogFormat{
					"height": height,
				})
			continue
		}

		blkRec := &blockRecord{}
		err = readRawBlockRecord(k, v, blkRec)
		if err != nil {
			return err
		}
		newBlkTxs := make([]wire.Hash, 0)
		for _, v := range blkRec.transactions {
			_, ok := hashes[v]
			if !ok {
				newBlkTxs = append(newBlkTxs, v)
			}
		}
		if len(newBlkTxs) == 0 {
			if err := deleteBlockRecord(nsBlocks, height); err != nil {
				return err
			}
		} else {
			if err := updateBlockRecord(nsBlocks, &blkRec.BlockMeta, newBlkTxs); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *TxStore) RemoveRelevantTx(tx mwdb.DBTransaction, addrmgr *keystore.AddrManager /* , syncHeight uint64 */) ([]*wire.Hash, error) {
	mas := addrmgr.ManagedAddresses()
	if len(mas) == 0 {
		return nil, nil
	}
	scriptHashes := make([][]byte, 0)
	scriptHashSet := make(map[string]struct{})
	for _, ma := range mas {
		scriptHashes = append(scriptHashes, ma.ScriptAddress())
		scriptHashSet[string(ma.ScriptAddress())] = struct{}{}
	}

	var rec TxRecord
	deletedTx := make([]*wire.Hash, 0)
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	nsBlocks := tx.FetchBucket(s.bucketMeta.nsBlocks)
	nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)

	// unmined tx
	unminedHashes, err := s.utxoStore.removeRelevantUnminedCredit(tx, scriptHashSet)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to remove relevant unmined credit",
			logging.LogFormat{
				"wallet": addrmgr.Name(),
			})
		return nil, err
	}

	for hash := range unminedHashes {
		v, err := existsRawUnmined(nsUnmined, hash[:])
		if err != nil {
			return nil, err
		}
		if len(v) == 0 {
			logging.CPrint(logging.WARN, "unmined not found",
				logging.LogFormat{
					"tx": hash.String(),
				})
			continue
		}
		err = readRawTxRecordValue(v, &rec)
		if err != nil {
			return nil, err
		}
		removable, err := s.removableTxForRemoveWallet(&rec.MsgTx, scriptHashSet)
		if err != nil {
			return nil, err
		}
		if removable {
			err = deleteRawUnmined(nsUnmined, hash[:])
			if err != nil {
				return nil, err
			}
			cpHash := hash
			deletedTx = append(deletedTx, &cpHash)
		}
	}

	// mined tx
	hash2Height, err := s.utxoStore.removeRelevantCredit(tx, scriptHashSet)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to remove relevant credit",
			logging.LogFormat{
				"wallet": addrmgr.Name(),
			})
		return nil, err
	}

	blkDeleted := make(map[uint64]map[wire.Hash]struct{})
	for hash, txheight := range hash2Height {
		item, err := fetchRawTxRecordByHashHeight(nsTxRecords, &hash, txheight)
		if err != nil {
			return nil, err
		}
		if item == nil {
			logging.CPrint(logging.WARN, "tx record not found",
				logging.LogFormat{
					"tx":     hash.String(),
					"height": txheight,
				})
			continue
		}
		err = readRawTxRecordValue(item.Value, &rec)
		if err != nil {
			return nil, err
		}
		removable, err := s.removableTxForRemoveWallet(&rec.MsgTx, scriptHashSet)
		if err != nil {
			return nil, err
		}
		if removable {
			err = deleteRawTxRecord(nsTxRecords, item.Key)
			if err != nil {
				return nil, err
			}

			// check & remove blockrecord
			height, _, err := readTxRecordKey(item.Key)
			if err != nil {
				return nil, err
			}
			s, ok := blkDeleted[height]
			if !ok {
				s = make(map[wire.Hash]struct{})
				blkDeleted[height] = s
			}
			s[hash] = struct{}{}
			cpHash := hash
			deletedTx = append(deletedTx, &cpHash)
		}
	}
	err = s.checkBlockRecordAfterTxRemoved(nsBlocks, blkDeleted)
	if err != nil {
		return nil, err
	}

	return deletedTx, nil

}
