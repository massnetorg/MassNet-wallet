package txmgr

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	_ "massnet.org/mass-wallet/masswallet/db/ldb"
	"massnet.org/mass-wallet/masswallet/keystore"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/wire"
)

const (
	keystoreBucket = "k"
	utxoBucket     = "u"
	txBucket       = "t"
	syncBucket     = "s"
)

func decodeHexStr(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func decodeHexTx(signedTx string) (*wire.MsgTx, error) {
	serializedTx, err := decodeHexStr(signedTx)
	if err != nil {
		return nil, err
	}
	var tx wire.MsgTx
	err = tx.SetBytes(serializedTx, wire.Packet)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func filterTx(rec *TxRecord, tx *wire.MsgTx, txStore *TxStore, meta *BlockMeta) (*TxRecord, error) {
	var err error
	cache := make(map[wire.Hash]*wire.MsgTx)
	// check TxIn
	if !blockchain.IsCoinBaseTx(tx) {
		for i, txIn := range tx.TxIn {
			prevTx := cache[txIn.PreviousOutPoint.Hash]
			if prevTx == nil {
				prevTx, err = txStore.chainFetcher.FetchTxBySha(&txIn.PreviousOutPoint.Hash)
				if err != nil {
					return nil, err
				}
				if prevTx == nil {
					fields := logging.LogFormat{
						"prevTx":      txIn.PreviousOutPoint.Hash.String(),
						"tx":          tx.TxHash().String(),
						"txInIndex":   i,
						"isUnminedTx": meta == nil,
					}
					if meta != nil {
						fields["block"] = meta.Hash.String()
						fields["height"] = meta.Height
						logging.CPrint(logging.INFO, "previous transaction of mined not found", fields)
						return nil, errors.New("maybe chain revoked")
					}
					// NOTE: just accept unmined tx whose all inputs are on the chain
					return nil, errors.New("invalid transaction")
				}

				if len(prevTx.TxOut) <= int(txIn.PreviousOutPoint.Index) {
					logging.CPrint(logging.ERROR, "TxIn out of range",
						logging.LogFormat{
							"tx":        tx.TxHash().String(),
							"txInIndex": i,
							"index":     txIn.PreviousOutPoint.Index,
							"range":     len(prevTx.TxOut) - 1,
						})
					return nil, errors.New("invalid transaction")
				}
				cache[txIn.PreviousOutPoint.Hash] = prevTx
			}

			pkScript := prevTx.TxOut[txIn.PreviousOutPoint.Index].PkScript
			ps, err := utils.ParsePkScript(pkScript, txStore.chainParams)
			if err != nil {
				logging.CPrint(logging.ERROR, "failed to parse txin pkscript",
					logging.LogFormat{
						"tx":        tx.TxHash().String(),
						"txInIndex": i,
						"err":       err,
					})
				return nil, err
			}

			ma, err := txStore.ksmgr.GetManagedAddressByScriptHash(ps.StdScriptAddress())
			if err != nil && err != keystore.ErrScriptHashNotFound {
				return nil, err
			}
			if ma != nil {
				//ready, err := txStore.CheckReady(ma.Account())
				//if err != nil {
				//	return  nil, err
				//}
				//if !ready {
				//	continue
				//}

				rec.HasBindingIn = ps.IsBinding()
				rec.RelevantTxIn = append(rec.RelevantTxIn,
					&RelevantMeta{
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

		ps, err := utils.ParsePkScript(txOut.PkScript, txStore.chainParams)
		if err != nil {

			return nil, err
		}
		ma, err := txStore.ksmgr.GetManagedAddressByScriptHash(ps.StdScriptAddress())
		if err != nil && err != keystore.ErrScriptHashNotFound {
			return nil, err
		}
		if ma != nil {
			//ready, err := h.walletMgr.CheckReady(ma.Account())
			//if err != nil {
			//	return false, nil, err
			//}
			//if !ready {
			//	continue
			//}
			rec.HasBindingOut = ps.IsBinding()
			rec.RelevantTxOut = append(rec.RelevantTxOut,
				&RelevantMeta{
					Index:        i,
					PkScript:     ps,
					WalletId:     ma.Account(),
					IsChangeAddr: ma.IsChangeAddr(),
				})
		}
	}
	return rec, nil
}

func TestRemoveRelevantTx(t *testing.T) {
	chainDb, chainDbTearDown, err := GetDb("TstRemoveRelevantTxChainDb")
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer chainDbTearDown()

	s, walletDb, teardown, err := testTxStore("TstRemoveRelevantTx", chainDb)
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer teardown()

	allAddresses := make(map[string][]byte)
	allTxHashes := make(map[wire.Hash]struct{})
	numOutput := 0
	numTx := 0

	var wIds []string

	err = mwdb.Update(walletDb, func(ns mwdb.DBTransaction) error {
		wIds = s.ksmgr.ListKeystoreNames()
		err = s.ksmgr.UseKeystoreForWallet(wIds[0])
		if err != nil {
			return err
		}

		allMinedBalances := map[string]massutil.Amount{
			wIds[0]: massutil.ZeroAmount(),
		}
		for i, block := range blks200[0:10] {
			if i == 0 {
				continue
			}
			blockMeta := &BlockMeta{
				block.MsgBlock().Header.Height,
				*block.Hash(),
				block.MsgBlock().Header.Timestamp,
			}
			for _, tx := range block.Transactions() {
				numOutput += len(tx.MsgTx().TxOut)
				numTx++
				allTxHashes[*tx.Hash()] = struct{}{}
				for _, txout := range tx.MsgTx().TxOut {
					ps, err := utils.ParsePkScript(txout.PkScript, s.chainParams)
					if err != nil {
						return err
					}
					allAddresses[ps.StdEncodeAddress()] = ps.StdScriptAddress()
				}

				rec, err := NewTxRecordFromMsgTx(tx.MsgTx(), time.Now())
				if err != nil {
					return err
				}
				rec, err = simpleFilterTx(rec, tx.MsgTx(), s, blockMeta, wIds[0])
				if err != nil {
					return err
				}
				err = s.AddRelevantTx(ns, allMinedBalances, rec, blockMeta)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	assert.Nil(t, err)

	// check before
	mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
		nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
		nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)
		entries, err := fetchAllEntry(nsCredits)
		assert.Nil(t, err)
		assert.True(t, len(entries) == numOutput)
		entries, err = fetchAllEntry(nsTxRecords)
		assert.Nil(t, err)
		assert.True(t, len(entries) == numTx)
		return nil
	})

	// remove
	err = mwdb.Update(walletDb, func(ns mwdb.DBTransaction) error {
		mam := keystore.NewMockAddrManager(wIds[0], allAddresses)
		rms, err := s.RemoveRelevantTx(ns, mam)
		for _, rm := range rms {
			if _, ok := allTxHashes[*rm]; !ok {
				return errors.New("unknown tx returned")
			}
			delete(allTxHashes, *rm)
		}
		return err
	})
	assert.Nil(t, err)

	// check after
	mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
		nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
		nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)
		entries, err := fetchAllEntry(nsCredits)
		assert.Nil(t, err)
		assert.Equal(t, len(entries), 0)
		entries, err = fetchAllEntry(nsTxRecords)
		assert.Nil(t, err)
		assert.Equal(t, len(entries), 0)
		return nil
	})
}

//
func TestExistUtxo(t *testing.T) {
	chainDb, chainDbTearDown, err := GetDb("TstExistUtoxChainDb")
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer chainDbTearDown()

	s, walletDb, teardown, err := testTxStore("TstExistUtxo", chainDb)
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer teardown()

	err = mwdb.Update(walletDb, func(ns mwdb.DBTransaction) error {
		wIds := s.ksmgr.ListKeystoreNames()
		err = s.ksmgr.UseKeystoreForWallet(wIds[0])
		if err != nil {
			return err
		}

		allMinedBalances := map[string]massutil.Amount{
			wIds[0]: massutil.ZeroAmount(),
		}
		for _, block := range blks200[0:10] {
			blockMeta := &BlockMeta{
				block.MsgBlock().Header.Height,
				*block.Hash(),
				block.MsgBlock().Header.Timestamp,
			}
			for _, tx := range block.Transactions() {
				rec, err := NewTxRecordFromMsgTx(tx.MsgTx(), time.Now())
				if err != nil {
					return err
				}
				rec, err = simpleFilterTx(rec, tx.MsgTx(), s, blockMeta, wIds[0])
				if err != nil {
					return err
				}
				err = s.AddRelevantTx(ns, allMinedBalances, rec, blockMeta)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	assert.Nil(t, err)

	mwdb.View(walletDb, func(ns mwdb.ReadTransaction) error {
		for j, block := range blks200[0:15] {
			for _, tx := range block.Transactions() {
				txhash := tx.Hash()
				for i, _ := range tx.MsgTx().TxOut {
					outPoint := wire.OutPoint{Hash: *txhash, Index: uint32(i)}
					f, err := s.ExistsUtxo(ns, &outPoint)
					if j >= 10 {
						assert.Equal(t, ErrNotFound, err)
					} else {
						assert.Nil(t, err)
						assert.NotNil(t, f)
						assert.False(t, f.Spent)
						assert.False(t, f.SpentByUnmined)
					}
				}
			}
		}
		return nil
	})
}
