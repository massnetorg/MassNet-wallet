package txmgr

import (
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"massnet.org/mass-wallet/consensus"

	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/keystore"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/wire"
)

type CreditIterationFilter func(*Credit) (stopIter, selectItem bool)

// UtxoStore ...
type UtxoStore struct {
	chainParams *config.Params
	ksmgr       *keystore.KeystoreManager
	bucketMeta  *StoreBucketMeta

	muUtxo sync.Mutex // only for ScriptAddressBalance and ScriptAddressUnspents
}

// NewUtxoStore ...
func NewUtxoStore(store mwdb.Bucket, ksmgr *keystore.KeystoreManager,
	bucketMeta *StoreBucketMeta, chainParams *config.Params) (s *UtxoStore, err error) {
	s = &UtxoStore{
		chainParams: chainParams,
		ksmgr:       ksmgr,
		bucketMeta:  bucketMeta,
	}
	// unspent
	bucket, err := mwdb.GetOrCreateBucket(store, bucketUnspent)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsUnspent = bucket.GetBucketMeta()
	// unmined inputs
	bucket, err = mwdb.GetOrCreateBucket(store, bucketUnminedInputs)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsUnminedInputs = bucket.GetBucketMeta()
	// unmined credits
	bucket, err = mwdb.GetOrCreateBucket(store, bucketUnminedCredits)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsUnminedCredits = bucket.GetBucketMeta()
	// mined balance
	bucket, err = mwdb.GetOrCreateBucket(store, bucketMinedBalance)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsMinedBalance = bucket.GetBucketMeta()
	// credits
	bucket, err = mwdb.GetOrCreateBucket(store, bucketCredits)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsCredits = bucket.GetBucketMeta()
	// debits
	bucket, err = mwdb.GetOrCreateBucket(store, bucketDebits)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsDebits = bucket.GetBucketMeta()

	//bucketAddresses
	bucket, err = mwdb.GetOrCreateBucket(store, bucketAddresses)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsAddresses = bucket.GetBucketMeta()
	return
}

func (s *UtxoStore) InitNewWallet(tx mwdb.DBTransaction, am *keystore.AddrManager) error {
	nsMinedBalance := tx.FetchBucket(s.bucketMeta.nsMinedBalance)
	return putMinedBalance(nsMinedBalance, am.Name(), massutil.ZeroAmount())
}

// AddCredits ...
func (s *UtxoStore) AddCredits(tx mwdb.DBTransaction, allBalances map[string]massutil.Amount,
	rec *TxRecord, block *BlockMeta) error {

	if len(rec.RelevantTxOut) == 0 {
		return nil
	}

	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsAddresses := tx.FetchBucket(s.bucketMeta.nsAddresses)
	nsLGHistory := tx.FetchBucket(s.bucketMeta.nsLGHistory)
	nsUnminedLGHistory := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)

	isCoinBase := blockchain.IsCoinBaseTx(&rec.MsgTx)

	var addrRecord *addressRecord
	if block != nil {
		addrRecord = &addressRecord{
			blockHeight: block.Height,
		}
	} else {
		// unmined tx
		if isCoinBase {
			logging.CPrint(logging.ERROR, "unexpected coinbase tx",
				logging.LogFormat{
					"tx": rec.Hash.String(),
				})
			return fmt.Errorf("unexpected coinbase tx")
		}
		return s.addUnminedCredits(tx, rec)
	}

	// mined tx
	for _, rel := range rec.RelevantTxOut {
		maturity := rel.PkScript.Maturity()
		if isCoinBase {
			maturity = consensus.CoinbaseMaturity
		}

		index := uint32(rel.Index)

		k, v, err := existsCredit(nsCredits, &rec.Hash, index, block)
		if err != nil {
			return err
		}
		if v != nil {
			logging.CPrint(logging.WARN, "AddCredits: duplicated with mined",
				logging.LogFormat{
					"tx":    rec.Hash.String(),
					"index": index,
				})
			return fmt.Errorf("duplicated credit")
		}

		// record address
		addrRecord.addressClass = rel.PkScript.AddressClass()
		addrRecord.walletId = rel.WalletId
		if rel.PkScript.IsStaking() {
			addrRecord.encodeAddress = rel.PkScript.SecondEncodeAddress()
		} else {
			addrRecord.encodeAddress = rel.PkScript.StdEncodeAddress()
		}
		addrK, err := keyAddressRecord(addrRecord)
		if err != nil {
			return err
		}
		addrV, err := existsRawAddressRecord(nsAddresses, addrK)
		if err != nil {
			return err
		}
		if addrV == nil || readAddressHeight(addrV) == 0 {
			addrV = valueAddressRecord(addrRecord)
			err = putRawAddressRecord(nsAddresses, addrK, addrV)
			if err != nil {
				return err
			}
		}

		value := rec.MsgTx.TxOut[rel.Index].Value
		txOutAmt, err := massutil.NewAmountFromInt(value)
		if err != nil {
			return err
		}
		// logging.CPrint(logging.DEBUG, "Marking transaction output spendable",
		// 	logging.LogFormat{
		// 		"tx":     rec.Hash.String(),
		// 		"index":  rel.Index,
		// 		"amount": txOutAmt,
		// 	})

		cred := credit{
			outPoint: wire.OutPoint{
				Hash:  rec.Hash,
				Index: index,
			},
			block:  block,
			amount: txOutAmt,
			flags: UtxoFlags{
				Change: rel.IsChangeAddr,
			},
			maturity:   uint32(maturity),
			scriptHash: rel.PkScript.StdScriptAddress(),
			spentBy:    indexedIncidence{index: ^uint32(0)},
		}

		if rel.PkScript.IsStaking() {
			cred.flags.Class = ClassStakingUtxo
		} else if rel.PkScript.IsBinding() {
			cred.flags.Class = ClassBindingUtxo
		}

		v, err = valueUnspentCredit(&cred)
		if err != nil {
			return err
		}
		err = putRawCredit(nsCredits, k, v)
		if err != nil {
			return err
		}

		err = putUnspent(nsUnspent, rel.WalletId, &cred.outPoint, block)
		if err != nil {
			return err
		}

		newBal, err := allBalances[rel.WalletId].Add(txOutAmt)
		if err != nil {
			return err
		}
		allBalances[rel.WalletId] = newBal

	}

	// record staking/binding history
	for _, lout := range constructLGHistory(rec, false, block.Height) {
		err := deleteUnminedLGHistory(nsUnminedLGHistory, lout)
		if err != nil {
			return err
		}
		err = putLGHistory(nsLGHistory, lout)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *UtxoStore) addUnminedCredits(tx mwdb.DBTransaction, rec *TxRecord) error {
	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
	nsUnminedLGHistory := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)

	for _, rel := range rec.RelevantTxOut {
		maturity := rel.PkScript.Maturity()

		// check already exists
		k := canonicalOutPoint(&rec.Hash, uint32(rel.Index))
		v, err := existsRawUnminedCredit(nsUnminedCredits, k)
		if err != nil {
			return err
		}
		if v != nil {
			logging.CPrint(logging.INFO, "addUnminedCredits: duplicated with unmined",
				logging.LogFormat{
					"tx":    rec.Hash.String(),
					"txIdx": rel.Index,
				})
			return fmt.Errorf("duplicate unmined credit")
		}

		// check unspent exists
		unspentKey := canonicalUnspentKey(rel.WalletId, &rec.Hash, uint32(rel.Index))
		unspentVal, err := existsRawUnspent(nsUnspent, unspentKey)
		if err != nil {
			return err
		}
		if unspentVal != nil {
			logging.CPrint(logging.INFO, "addUnminedCredits: duplicated with mined",
				logging.LogFormat{
					"tx":    rec.Hash.String(),
					"txIdx": rel.Index,
				})
			return fmt.Errorf("expired unmined credit")
		}

		value := rec.MsgTx.TxOut[rel.Index].Value
		amount, err := massutil.NewAmountFromInt(value)
		if err != nil {
			return err
		}
		v, err = valueUnminedCredit(amount, rel.IsChangeAddr,
			uint32(maturity), rel.PkScript.StdScriptAddress(), rel.PkScript)
		if err != nil {
			return err
		}
		err = putRawUnminedCredit(nsUnminedCredits, k, v)
		if err != nil {
			return err
		}
	}
	for _, lout := range constructLGHistory(rec, false, 0) {
		err := putUnminedLGHistory(nsUnminedLGHistory, lout)
		if err != nil {
			return err
		}
	}
	return nil
}

// Temporarily not used. DO NOT DELETE!
//addUnminedInputsLGHistory inserts input history of staking/binding transaction
func (s *UtxoStore) addUnminedInputsLGHistory(tx mwdb.DBTransaction, rec *TxRecord) error {
	nsUnminedLGHistory := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)
	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)

	// check if utxo exists (mined or unmined)
	for _, rel := range rec.RelevantTxIn {
		prevOut := &rec.MsgTx.TxIn[rel.Index].PreviousOutPoint
		outPointKey := canonicalOutPoint(&prevOut.Hash, prevOut.Index)

		uCredit, err := existsRawUnminedCredit(nsUnminedCredits, outPointKey)
		if err != nil {
			return err
		}
		_, credKey, err := existsUnspent(nsUnspent, rel.WalletId, prevOut)
		if err != nil {
			return err
		}
		if uCredit == nil && credKey == nil {
			return fmt.Errorf("corresponding utxo of relevant input not found")
		}
		if uCredit != nil && credKey != nil { // debug code
			logging.CPrint(logging.ERROR, "both mined & unmined utxo found",
				logging.LogFormat{
					"tx":      rec.Hash.String(),
					"txIdx":   rel.Index,
					"prevTx":  prevOut.Hash.String(),
					"prevIdx": prevOut.Index,
				})
		}
	}
	for _, lout := range constructLGHistory(rec, true, 0) {
		err := putUnminedLGHistory(nsUnminedLGHistory, lout)
		if err != nil {
			return err
		}
	}
	return nil
}

func constructLGHistory(rec *TxRecord, parseInputs bool, height uint64) []*lgTxHistory {

	l := make(map[string]*lgTxHistory) // staking
	g := make(map[string]*lgTxHistory) // binding
	result := make([]*lgTxHistory, 0)

	items := rec.RelevantTxIn
	if !parseInputs {
		items = rec.RelevantTxOut
	}

	for _, rel := range items {
		var (
			lout *lgTxHistory
			ok   bool
		)
		if rel.PkScript.IsStaking() {
			if lout, ok = l[rel.WalletId]; !ok {
				lout = &lgTxHistory{
					walletId:    rel.WalletId,
					txhash:      rec.Hash,
					isWithdraw:  parseInputs,
					isBinding:   false,
					blockHeight: height,
				}
				l[rel.WalletId] = lout
				result = append(result, lout)
			}
		}

		if rel.PkScript.IsBinding() {
			if lout, ok = g[rel.WalletId]; !ok {
				lout = &lgTxHistory{
					walletId:    rel.WalletId,
					txhash:      rec.Hash,
					isWithdraw:  parseInputs,
					isBinding:   true,
					blockHeight: height,
				}
				g[rel.WalletId] = lout
				result = append(result, lout)
			}
		}
		if lout == nil {
			continue
		}

		lout.indexes = append(lout.indexes, uint32(rel.Index))
	}
	logging.CPrint(logging.DEBUG, "constructLGHistory",
		logging.LogFormat{
			"tx":      rec.Hash.String(),
			"staking": len(l),
			"binding": len(g),
			"total":   len(result),
		})
	return result
}

// rollback / insertMinedTx
func (s *UtxoStore) removeUnminedLGHistory(tx mwdb.DBTransaction, rec *TxRecord) error {

	lout := &lgTxHistory{
		txhash:     rec.Hash,
		isWithdraw: false,
	}

	nsUnminedLGHistory := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)
	for i, output := range rec.MsgTx.TxOut {
		ps, err := utils.ParsePkScript(output.PkScript, s.chainParams)
		if err != nil {
			if err == utils.ErrUnsupportedScript {
				continue
			}
			logging.CPrint(logging.DEBUG, "unexpected: failed to parse txout pkscript",
				logging.LogFormat{
					"tx":         rec.Hash.String(),
					"txOutIndex": i,
					"err":        err,
				})
			return err
		}
		if ps.IsStaking() || ps.IsBinding() {
			ma, err := s.ksmgr.GetManagedAddressByStdAddress(ps.StdEncodeAddress())
			if err != nil {
				continue
			}
			lout.walletId = ma.Account()
			lout.isBinding = ps.IsBinding()
			err = deleteUnminedLGHistory(nsUnminedLGHistory, lout)
			if err != nil {
				return err
			}
		}
	}

	lout.isWithdraw = true
	cache := make(map[wire.Hash]*wire.MsgTx)
	for i, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOutPoint

		prevTx := cache[prevOut.Hash]
		if prevTx == nil {

			um, err := existsRawUnmined(nsUnmined, prevOut.Hash[:])
			if err != nil {
				return err
			}
			if um == nil {
				item, err := fetchLatestRawTxRecordOfHash(nsTxRecords, &prevOut.Hash)
				if err != nil {
					return err
				}
				if item != nil {
					um = item.Value
				}
			}
			if len(um) == 0 {
				continue
			}
			var rec TxRecord
			err = readRawTxRecordValue(um, &rec)
			if err != nil {
				return err
			}
			prevTx = &rec.MsgTx
			cache[prevOut.Hash] = prevTx
		}

		txout := prevTx.TxOut[prevOut.Index]
		ps, err := utils.ParsePkScript(txout.PkScript, s.chainParams)
		if err != nil {
			if err == utils.ErrUnsupportedScript {
				continue
			}
			logging.CPrint(logging.DEBUG, "failed to parse pkscript",
				logging.LogFormat{
					"tx":        rec.Hash.String(),
					"txInIndex": i,
				})
			return err
		}
		if ps.IsStaking() || ps.IsBinding() {
			ma, err := s.ksmgr.GetManagedAddressByStdAddress(ps.StdEncodeAddress())
			if err != nil {
				continue
			}
			lout.walletId = ma.Account()
			lout.isBinding = ps.IsBinding()
			err = deleteUnminedLGHistory(nsUnminedLGHistory, lout)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *UtxoStore) removeRelevantCredit(tx mwdb.DBTransaction,
	scriptHashSet map[string]struct{}) (map[wire.Hash]uint64, error) {
	if len(scriptHashSet) == 0 {
		return nil, nil
	}
	nsDebits := tx.FetchBucket(s.bucketMeta.nsDebits)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)

	iter := nsCredits.NewIterator(nil)
	defer iter.Release()

	cred := credit{
		block: &BlockMeta{},
	}
	txs := make(map[wire.Hash]uint64)

	for iter.Next() {
		itKey, itValue := iter.Key(), iter.Value()
		err := readRawCreditKey(itKey, &cred)
		if err != nil {
			return nil, err
		}
		err = readCreditValue(itValue, &cred)
		if err != nil {
			return nil, err
		}
		if _, ok := scriptHashSet[string(cred.scriptHash)]; ok {
			err = deleteRawCredit(nsCredits, itKey)
			if err != nil {
				return nil, err
			}
			logging.CPrint(logging.DEBUG, "delete relevant credit",
				logging.LogFormat{
					"tx":    cred.outPoint.Hash.String(),
					"index": cred.outPoint.Index,
				})
			k := canonicalOutPoint(&cred.outPoint.Hash, cred.outPoint.Index)
			_ = deleteRawUnminedInput(nsUnminedInputs, k)

			if cred.flags.Spent {

				debitKey := readCreditSpender(itValue)
				if debitKey != nil {
					err = deleteRawDebit(nsDebits, debitKey)
					if err != nil {
						return nil, err
					}
				} else {
					// double check
					logging.CPrint(logging.ERROR, "debit missing",
						logging.LogFormat{
							"tx":    cred.outPoint.Hash.String(),
							"index": cred.outPoint.Index,
						})
					return nil, fmt.Errorf("unexpected error")
				}
			}

			txs[cred.outPoint.Hash] = cred.block.Height
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return txs, nil
}

func (s *UtxoStore) removeRelevantUnminedCredit(tx mwdb.DBTransaction,
	scriptHashSet map[string]struct{}) (map[wire.Hash]struct{}, error) {
	if len(scriptHashSet) == 0 {
		return nil, nil
	}

	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)

	iter := nsUnminedCredits.NewIterator(nil)
	defer iter.Release()

	var cred credit
	txs := make(map[wire.Hash]struct{})
	for iter.Next() {
		itKey, itValue := iter.Key(), iter.Value()
		err := readUnminedCreditKey(itKey, &cred)
		if err != nil {
			return nil, err
		}
		err = readCreditValue(itValue, &cred)
		if err != nil {
			return nil, err
		}
		if _, ok := scriptHashSet[string(cred.scriptHash)]; ok {
			err = deleteRawUnminedCredit(nsUnminedCredits, itKey)
			if err != nil {
				return nil, err
			}
			logging.CPrint(logging.DEBUG, "delete relevant unmined credit",
				logging.LogFormat{
					"tx":    cred.outPoint.Hash.String(),
					"index": cred.outPoint.Index,
				})
			k := canonicalOutPoint(&cred.outPoint.Hash, cred.outPoint.Index)
			_ = deleteRawUnminedInput(nsUnminedInputs, k)
			txs[cred.outPoint.Hash] = struct{}{}
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return txs, nil
}

func (s *UtxoStore) RemoveAddressByWalletId(tx mwdb.DBTransaction, walletId string) error {
	if len(walletId) == 0 {
		return nil
	}
	nsAddresses := tx.FetchBucket(s.bucketMeta.nsAddresses)
	return deleteByPrefix(nsAddresses, []byte(walletId))
}

func (s *UtxoStore) RemoveUnspentByWalletId(tx mwdb.DBTransaction, walletId string) error {
	if len(walletId) == 0 {
		return nil
	}
	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	return deleteByPrefix(nsUnspent, []byte(walletId))
}

func (s *UtxoStore) RemoveLGHistoryByWalletId(tx mwdb.DBTransaction, walletId string) error {
	if len(walletId) == 0 {
		return nil
	}
	nsLGHistory := tx.FetchBucket(s.bucketMeta.nsLGHistory)
	nsUnminedLGHistory := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)
	err := deleteByPrefix(nsLGHistory, []byte(walletId))
	if err != nil {
		return err
	}
	return deleteByPrefix(nsUnminedLGHistory, []byte(walletId))
}

func (s *UtxoStore) PutNewAddress(tx mwdb.DBTransaction, walletId, addr string, addrClass uint16) error {
	addrRecord := &addressRecord{
		walletId:      walletId,
		encodeAddress: addr,
		addressClass:  addrClass,
		blockHeight:   0,
	}
	k, err := keyAddressRecord(addrRecord)
	if err != nil {
		return err
	}
	nsAddresses := tx.FetchBucket(s.bucketMeta.nsAddresses)
	return putRawAddressRecord(nsAddresses, k, valueAddressRecord(addrRecord))
}

func (s *UtxoStore) GetAddresses(tx mwdb.ReadTransaction, walletId string) ([]*AddressDetail, error) {
	nsAddresses := tx.FetchBucket(s.bucketMeta.nsAddresses)
	return fetchAddressesByWalletId(nsAddresses, walletId)
}

func (s *UtxoStore) insertUnminedInputs(tx mwdb.DBTransaction, rec *TxRecord) error {
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)

	for _, rel := range rec.RelevantTxIn {
		prevOut := &rec.MsgTx.TxIn[rel.Index].PreviousOutPoint
		k := canonicalOutPoint(&prevOut.Hash, prevOut.Index)
		err := putRawUnminedInput(nsUnminedInputs, k, rec.Hash[:])
		if err != nil {
			return err
		}
	}
	// return s.addUnminedInputsLGHistory(tx, rec) // NOTE: Not record withdrawal at present
	return nil
}

func (s *UtxoStore) deleteUnminedCredits(tx mwdb.DBTransaction, rec *TxRecord) error {
	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
	for i := range rec.MsgTx.TxOut {
		k := canonicalOutPoint(&rec.Hash, uint32(i))
		if err := deleteRawUnminedCredit(nsUnminedCredits, k); err != nil {
			return err
		}
	}
	return nil
}

func (s *UtxoStore) deleteUnminedInputs(tx mwdb.DBTransaction, rec *TxRecord) error {
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)
	for _, input := range rec.MsgTx.TxIn {
		prevOut := &input.PreviousOutPoint
		k := canonicalOutPoint(&prevOut.Hash, prevOut.Index)
		if err := deleteRawUnminedInput(nsUnminedInputs, k); err != nil {
			return err
		}
	}
	return nil
}

func (s *UtxoStore) UpdateMinedBalances(tx mwdb.DBTransaction, balances map[string]massutil.Amount) error {
	nsMinedBalance := tx.FetchBucket(s.bucketMeta.nsMinedBalance)
	for walletId, amt := range balances {
		err := putMinedBalance(nsMinedBalance, walletId, amt)
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchAllMinedBalance ...
func (s *UtxoStore) FetchAllMinedBalance(tx mwdb.DBTransaction) (map[string]massutil.Amount, error) {
	nsMinedBalance := tx.FetchBucket(s.bucketMeta.nsMinedBalance)
	entries, err := fetchMinedBalance(nsMinedBalance, "")
	if err != nil {
		return nil, fmt.Errorf("error to fetch mined balance: %v)", err)
	}

	m := make(map[string]massutil.Amount)
	for _, entry := range entries {
		if len(entry.Value) != 8 {
			return nil, fmt.Errorf("balance: short read (expected 8 bytes, read %v)", len(entry.Value))
		}
		amount, err := massutil.NewAmountFromUint(binary.BigEndian.Uint64(entry.Value))
		if err != nil {
			return nil, err
		}
		m[string(entry.Key)] = amount
	}
	return m, nil
}

func (s *UtxoStore) RemoveMinedBalance(tx mwdb.DBTransaction, account string) error {
	nsMinedBalance := tx.FetchBucket(s.bucketMeta.nsMinedBalance)
	return deleteMinedBalance(nsMinedBalance, account)
}

func (s *UtxoStore) GrossBalance(tx mwdb.ReadTransaction, keystoreName string) (massutil.Amount, error) {
	nsMinedBalance := tx.FetchBucket(s.bucketMeta.nsMinedBalance)
	entries, err := fetchMinedBalance(nsMinedBalance, keystoreName)
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	if len(entries) == 0 {
		return massutil.ZeroAmount(), ErrNotFound
	}
	amount, err := massutil.NewAmountFromUint(binary.BigEndian.Uint64(entries[0].Value))
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	return amount, nil
}

// ScriptAddressBalance query balance by ManagedAddress.ScriptAddress
func (s *UtxoStore) WalletBalance(tx mwdb.ReadTransaction, addrMgr *keystore.AddrManager,
	minConf uint32, syncHeight uint64) (*BalanceDetail, error) {
	ret := &BalanceDetail{
		Total:               massutil.ZeroAmount(),
		Spendable:           massutil.ZeroAmount(),
		WithdrawableStaking: massutil.ZeroAmount(),
		WithdrawableBinding: massutil.ZeroAmount(),
	}

	filteredScripts := make(map[string]struct{})
	for _, ma := range addrMgr.ManagedAddresses() {
		filteredScripts[string(ma.ScriptAddress())] = struct{}{}
	}
	if len(filteredScripts) == 0 {
		return ret, nil
	}
	bal, err := s.ScriptAddressBalance(tx, filteredScripts, minConf, syncHeight)
	if err != nil {
		return nil, fmt.Errorf("error to get address Balance: %v", err)
	}
	for _, v := range bal {
		ret.Total, err = ret.Total.Add(v.Total)
		if err != nil {
			return nil, err
		}
		ret.Spendable, err = ret.Spendable.Add(v.Spendable)
		if err != nil {
			return nil, err
		}
		ret.WithdrawableStaking, err = ret.WithdrawableStaking.Add(v.WithdrawableStaking)
		if err != nil {
			return nil, err
		}
		ret.WithdrawableBinding, err = ret.WithdrawableBinding.Add(v.WithdrawableBinding)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// ScriptAddressBalance scripts -- script address in string format
func (s *UtxoStore) ScriptAddressBalance(tx mwdb.ReadTransaction, scripts map[string]struct{},
	minConf uint32, syncHeight uint64) (map[string]*BalanceDetail, error) {

	s.muUtxo.Lock()
	defer s.muUtxo.Unlock()

	ret := make(map[string]*BalanceDetail) // script address -> balance
	for script := range scripts {
		ret[script] = &BalanceDetail{
			Total:               massutil.ZeroAmount(),
			Spendable:           massutil.ZeroAmount(),
			WithdrawableStaking: massutil.ZeroAmount(),
			WithdrawableBinding: massutil.ZeroAmount(),
		}
	}

	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)

	iter := nsUnspent.NewIterator(mwdb.BytesPrefix([]byte(s.ksmgr.CurrentKeystore().Name())))
	defer iter.Release()

	cred := &credit{
		block: &BlockMeta{},
	}
	for iter.Next() {
		itKey, itValue := iter.Key(), iter.Value()
		err := readCanonicalUnspentKey(itKey, &cred.outPoint)
		if err != nil {
			return nil, err
		}
		err = readBlockOfUnspent(itValue, cred.block)
		if err != nil {
			return nil, err
		}

		_, credValue, err := existsCredit(nsCredits, &cred.outPoint.Hash, cred.outPoint.Index, cred.block)
		if err != nil {
			return nil, err
		}
		if credValue == nil {
			inspect, err := existsRawUnspent(nsUnspent, iter.Key())
			if err != nil {
				return nil, err
			}
			if inspect == nil {
				// due to block rollback
				continue
			}
			logging.VPrint(logging.ERROR, "unpent exists but credit not",
				logging.LogFormat{
					"tx":      cred.outPoint.Hash.String(),
					"index":   cred.outPoint.Index,
					"height":  cred.block.Height,
					"blkHash": cred.block.Hash.String(),
				})
			continue
		}
		//check if it is the queried address
		err = readCreditValue(credValue, cred)
		if err != nil {
			return nil, err
		}
		// utxo amount will be 0 when no use transactions and coinbase subsidy is 0 too.
		if cred.amount.IsZero() {
			continue
		}
		balance, ok := ret[string(cred.scriptHash)]
		if !ok {
			continue
		}

		confs := syncHeight - cred.block.Height + 1
		if confs >= uint64(minConf) {
			balance.Total, err = balance.Total.Add(cred.amount)
			if err != nil {
				return nil, err
			}
			if confs >= uint64(cred.maturity) /* && !cred.flags.SpentByUnmined */ {
				if cred.isBinding() {
					balance.WithdrawableBinding, err = balance.WithdrawableBinding.Add(cred.amount)
					if err != nil {
						return nil, err
					}
				} else if cred.isStaking() {
					balance.WithdrawableStaking, err = balance.WithdrawableStaking.Add(cred.amount)
					if err != nil {
						return nil, err
					}
				} else {
					balance.Spendable, err = balance.Spendable.Add(cred.amount)
					if err != nil {
						return nil, err
					}
				}

			}
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return ret, nil
}

// AddressUnspents returns all spendable UTXOs of specified addresses, including those spent by unmined transaction
// return scriptHash->*Credit
func (s *UtxoStore) ScriptAddressUnspents(tx mwdb.ReadTransaction, scriptAddrs map[string]struct{},
	syncHeight uint64, filter CreditIterationFilter) (map[string][]*Credit, error) {

	s.muUtxo.Lock()
	defer s.muUtxo.Unlock()

	ret := make(map[string][]*Credit) // script address -> credits
	for scriptAddr := range scriptAddrs {
		ret[scriptAddr] = make([]*Credit, 0, 100)
	}

	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)

	var op wire.OutPoint
	var block BlockMeta

	iter := nsUnspent.NewIterator(mwdb.BytesPrefix([]byte(s.ksmgr.CurrentKeystore().Name())))
	defer iter.Release()

	for iter.Next() {
		itKey, itValue := iter.Key(), iter.Value()
		err := readCanonicalUnspentKey(itKey, &op)
		if err != nil {
			return nil, err
		}

		err = readBlockOfUnspent(itValue, &block)
		if err != nil {
			return nil, err
		}

		_, credValue, err := existsCredit(nsCredits, &op.Hash, op.Index, &block)
		if err != nil {
			return nil, err
		}
		if credValue == nil {
			inspect, err := existsRawUnspent(nsUnspent, iter.Key())
			if err != nil {
				return nil, err
			}
			if inspect == nil {
				// due to block rollback
				continue
			}
			logging.VPrint(logging.ERROR, "unpent exists but credit not",
				logging.LogFormat{
					"tx":      op.Hash.String(),
					"index":   op.Index,
					"height":  block.Height,
					"blkHash": block.Hash.String(),
				})
			continue
		}
		var cred credit
		err = readCreditValue(credValue, &cred)
		if err != nil {
			return nil, err
		}
		// utxo amount will be 0 when no use transactions and coinbase subsidy is 0 too.
		if cred.amount.IsZero() {
			continue
		}
		credits, ok := ret[string(cred.scriptHash)]
		if !ok {
			continue
		}
		cred.flags.SpentByUnmined = existsRawUnminedInput(nsUnminedInputs, itKey) != nil

		item := &Credit{
			OutPoint:      op,
			BlockMeta:     block,
			Amount:        cred.amount,
			Maturity:      cred.maturity,
			Confirmations: uint32(syncHeight - block.Height + 1),
			Flags:         cred.flags,
			ScriptHash:    cred.scriptHash,
		}

		stopIter, selItem := filter(item)
		if selItem {
			ret[string(cred.scriptHash)] = append(credits, item)
		}
		if stopIter {
			break
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *UtxoStore) GetUnminedStakingHistoryDetail(tx mwdb.ReadTransaction, addrMgr *keystore.AddrManager) ([]*StakingHistoryDetail, error) {
	nsUnminedLGHistory := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)
	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
	entries, err := fetchRawLGHistoryByWalletId(nsUnminedLGHistory, addrMgr.Name(), outputStaking)
	if err != nil {
		return nil, err
	}

	ret := make([]*StakingHistoryDetail, 0)
	for _, entry := range entries {
		lout := &lgTxHistory{}
		err := readLGHistory(true, entry.Key, entry.Value, lout)
		if err != nil {
			return nil, err
		}
		if lout.isWithdraw {
			continue
		}

		for _, index := range lout.indexes {
			k := canonicalOutPoint(&lout.txhash, index)
			credV, err := existsRawUnminedCredit(nsUnminedCredits, k)
			if err != nil {
				return nil, err
			}
			if credV == nil { // debug code
				// maybe credit has been mined during this query
				logging.CPrint(logging.WARN, "history: unmined credit not found",
					logging.LogFormat{
						"tx":     lout.txhash.String(),
						"vout":   index,
						"wallet": lout.walletId,
					})
				continue
			}
			maturity, scripthash, err := fetchRawCreditMaturityScriptHash(credV)
			if err != nil {
				return nil, err
			}
			addr, err := massutil.NewAddressStakingScriptHash(scripthash, s.chainParams)
			if err != nil {
				return nil, err
			}
			amt, _, err := fetchRawCreditAmountSpent(credV)
			if err != nil {
				return nil, err
			}
			det := &StakingHistoryDetail{
				TxHash: lout.txhash,
				Index:  index,
				Utxo: StakingUtxo{
					Hash:         lout.txhash,
					Index:        index,
					Address:      addr.EncodeAddress(),
					FrozenPeriod: maturity - 1,
					Amount:       amt,
				},
			}
			ret = append(ret, det)
		}

	}
	return ret, nil
}

func (s *UtxoStore) GetStakingHistoryDetail(tx mwdb.ReadTransaction, addrMgr *keystore.AddrManager) ([]*StakingHistoryDetail, error) {
	nsLGHistory := tx.FetchBucket(s.bucketMeta.nsLGHistory)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)
	entries, err := fetchRawLGHistoryByWalletId(nsLGHistory, addrMgr.Name(), outputStaking)
	if err != nil {
		return nil, err
	}

	ret := make([]*StakingHistoryDetail, 0)
	for _, entry := range entries {
		lout := &lgTxHistory{}
		err := readLGHistory(false, entry.Key, entry.Value, lout)
		if err != nil {
			return nil, err
		}
		if lout.isWithdraw {
			continue
		}
		if lout.blockHeight == 0 { // debug code
			logging.CPrint(logging.ERROR, "block height is 0",
				logging.LogFormat{
					"tx":      lout.txhash.String(),
					"wallet":  lout.walletId,
					"binding": lout.isBinding,
				})
			continue
		}

		indexToCredit, err := getCreditsByTxHashHeight(nsCredits, &lout.txhash, lout.blockHeight)
		if err != nil {
			return nil, err
		}

		for _, index := range lout.indexes {
			credit, ok := indexToCredit[index]
			if !ok {
				logging.CPrint(logging.WARN, "credit not found",
					logging.LogFormat{
						"tx":      lout.txhash.String(),
						"index":   index,
						"height":  lout.blockHeight,
						"wallet":  lout.walletId,
						"binding": lout.isBinding,
					})
				continue
			}
			maturity, scripthash, err := fetchRawCreditMaturityScriptHash(credit.Value)
			if err != nil {
				return nil, err
			}
			addr, err := massutil.NewAddressStakingScriptHash(scripthash, s.chainParams)
			if err != nil {
				return nil, err
			}
			amt, spent, err := fetchRawCreditAmountSpent(credit.Value)
			if err != nil {
				return nil, err
			}
			det := &StakingHistoryDetail{
				TxHash:      lout.txhash,
				Index:       index,
				BlockHeight: lout.blockHeight,
				Utxo: StakingUtxo{
					Hash:         lout.txhash,
					Index:        index,
					Spent:        spent,
					Address:      addr.EncodeAddress(),
					FrozenPeriod: maturity - 1,
					Amount:       amt,
				},
			}
			if !det.Utxo.Spent {
				k := canonicalOutPoint(&det.TxHash, det.Index)
				det.Utxo.SpentByUnmined = existsRawUnminedInput(nsUnminedInputs, k) != nil
			}
			ret = append(ret, det)
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return (ret[i].BlockHeight + uint64(ret[i].Utxo.FrozenPeriod)) >
			(ret[j].BlockHeight + uint64(ret[j].Utxo.FrozenPeriod))
	})
	return ret, nil
}

func (s *UtxoStore) GetUnminedBindingHistoryDetail(tx mwdb.ReadTransaction, addrMgr *keystore.AddrManager) ([]*BindingHistoryDetail, error) {
	nsUnminedLGHistory := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)
	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)

	entries, err := fetchRawLGHistoryByWalletId(nsUnminedLGHistory, addrMgr.Name(), outputBinding)
	if err != nil {
		return nil, err
	}

	ret := make([]*BindingHistoryDetail, 0)
	for _, entry := range entries {
		lout := &lgTxHistory{}
		err := readLGHistory(true, entry.Key, entry.Value, lout)
		if err != nil {
			return nil, err
		}

		// query unmined MsgTx
		unmined, err := existsRawUnmined(nsUnmined, lout.txhash[:])
		if err != nil {
			return nil, err
		}
		if len(unmined) == 0 {
			// maybe tx has been mined, otherwise data error
			logging.CPrint(logging.WARN, "binding history: unmined tx not found",
				logging.LogFormat{
					"tx":     lout.txhash.String(),
					"wallet": lout.walletId,
				})
			continue
		}
		rec := &TxRecord{}
		err = readRawTxRecordValue(unmined, rec)
		if err != nil {
			return nil, err
		}

		// deposit tx
		// if !lout.isWithdraw {
		for _, index := range lout.indexes {
			k := canonicalOutPoint(&lout.txhash, index)
			credV, err := existsRawUnminedCredit(nsUnminedCredits, k)
			if err != nil {
				return nil, err
			}
			if credV == nil { // double check
				// maybe credit has been mined during this query
				logging.CPrint(logging.WARN, "binding history: unmined credit not found",
					logging.LogFormat{
						"tx":     lout.txhash.String(),
						"vout":   index,
						"wallet": lout.walletId,
					})
				continue
			}
			// amount
			amt, err := massutil.NewAmountFromInt(rec.MsgTx.TxOut[index].Value)
			if err != nil {
				return nil, err
			}
			// parse script
			script, err := utils.ParsePkScript(rec.MsgTx.TxOut[index].PkScript, s.chainParams)
			if err != nil {
				return nil, err
			}

			det := &BindingHistoryDetail{
				TxHash: lout.txhash,
				Index:  index,
				Utxo: BindingUtxo{
					Hash:           lout.txhash,
					Index:          index,
					Amount:         amt,
					HolderAddress:  script.StdEncodeAddress(),
					BindingAddress: script.SecondEncodeAddress(),
				},
				MsgTx: &rec.MsgTx,
			}
			ret = append(ret, det)
		}
	}
	return ret, nil
}

func (s *UtxoStore) GetBindingHistoryDetail(tx mwdb.ReadTransaction, addrMgr *keystore.AddrManager) ([]*BindingHistoryDetail, error) {
	nsLGHistory := tx.FetchBucket(s.bucketMeta.nsLGHistory)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)

	entries, err := fetchRawLGHistoryByWalletId(nsLGHistory, addrMgr.Name(), outputBinding)
	if err != nil {
		return nil, err
	}

	ret := make([]*BindingHistoryDetail, 0)
	for _, entry := range entries {
		lout := &lgTxHistory{}
		err := readLGHistory(false, entry.Key, entry.Value, lout)
		if err != nil {
			return nil, err
		}
		if lout.blockHeight == 0 { // debug code
			logging.CPrint(logging.ERROR, "block height is 0",
				logging.LogFormat{
					"tx":       lout.txhash.String(),
					"wallet":   lout.walletId,
					"withdraw": lout.isWithdraw,
				})
			continue
		}

		rec := &TxRecord{}
		rawRec, err := fetchRawTxRecordByTxHashHeight(nsTxRecords, &lout.txhash, lout.blockHeight)
		if err != nil {
			return nil, err
		}
		if len(rawRec) == 0 { // debug code
			logging.CPrint(logging.ERROR, "tx record not found",
				logging.LogFormat{
					"tx":       lout.txhash.String(),
					"wallet":   lout.walletId,
					"height":   lout.blockHeight,
					"withdraw": lout.isWithdraw,
				})
			continue
		}
		err = readRawTxRecordValue(rawRec, rec)
		if err != nil {
			return nil, err
		}

		// deposit
		// if !lout.isWithdraw {
		indexToCredit, err := getCreditsByTxHashHeight(nsCredits, &lout.txhash, lout.blockHeight)
		if err != nil {
			return nil, err
		}

		for _, index := range lout.indexes {
			credit, ok := indexToCredit[index]
			if !ok {
				logging.CPrint(logging.WARN, "credit not found",
					logging.LogFormat{
						"tx":     lout.txhash.String(),
						"index":  index,
						"height": lout.blockHeight,
						"wallet": lout.walletId,
					})
				continue
			}
			amt, spent, err := fetchRawCreditAmountSpent(credit.Value)
			if err != nil {
				return nil, err
			}
			// parse script
			script, err := utils.ParsePkScript(rec.MsgTx.TxOut[index].PkScript, s.chainParams)
			if err != nil {
				return nil, err
			}
			det := &BindingHistoryDetail{
				TxHash:      lout.txhash,
				Index:       index,
				BlockHeight: lout.blockHeight,
				Utxo: BindingUtxo{
					Hash:           lout.txhash,
					Index:          index,
					Amount:         amt,
					Spent:          spent,
					HolderAddress:  script.StdEncodeAddress(),
					BindingAddress: script.SecondEncodeAddress(),
				},
				MsgTx: &rec.MsgTx,
			}
			if !det.Utxo.Spent {
				k := canonicalOutPoint(&det.TxHash, det.Index)
				det.Utxo.SpentByUnmined = existsRawUnminedInput(nsUnminedInputs, k) != nil
			}
			ret = append(ret, det)
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].BlockHeight > ret[j].BlockHeight
	})
	return ret, nil
}
