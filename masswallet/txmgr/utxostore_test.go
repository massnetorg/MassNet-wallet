package txmgr

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/wire"
)

func TestGetPutWalletBalance(t *testing.T) {
	amt100, err := massutil.NewAmountFromUint(100)
	assert.Nil(t, err)
	amt1000, err := massutil.NewAmountFromUint(1000)
	assert.Nil(t, err)

	tests := []struct {
		name      string
		walletId  string
		amount    massutil.Amount
		expectAll map[string]massutil.Amount
		delete    bool
		putErr    error
		getErr    error
	}{
		{
			name:     "add valid 1",
			walletId: "ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp",
			amount:   amt100,
			expectAll: map[string]massutil.Amount{
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": amt100,
			},
		},
		{
			name:     "update valid 1",
			walletId: "ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp",
			amount:   amt1000,
			expectAll: map[string]massutil.Amount{
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": amt1000,
			},
		},
		{
			name:     "add valid 2",
			walletId: "ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz",
			amount:   amt100,
			expectAll: map[string]massutil.Amount{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": amt100,
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": amt1000,
			},
		},
		{
			name:     "add invalid 1",
			walletId: "ac10jv5xfkywm9fu2elcjyqyq4gyz6y",
			expectAll: map[string]massutil.Amount{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": amt100,
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": amt1000,
			},
			putErr: fmt.Errorf("putMinedBalance: short read (expected 42 bytes, read %d)", len("ac10jv5xfkywm9fu2elcjyqyq4gyz6y")),
			getErr: errors.New("not found"),
		},
		{
			name:     "delete valid 1",
			walletId: "ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp",
			expectAll: map[string]massutil.Amount{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": amt100,
			},
			delete: true,
		},
	}

	chainDb, chainDbTearDown, err := GetDb("TstWalletBalanceChainDb")
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer chainDbTearDown()

	s, walletDb, teardown, err := testTxStore("TstWalletBalance", chainDb)
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer teardown()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
				if test.delete {
					return s.utxoStore.RemoveMinedBalance(tx, test.walletId)
				}
				return s.utxoStore.UpdateMinedBalances(tx, map[string]massutil.Amount{
					test.walletId: test.amount,
				})
			})
			if !assert.Equal(t, test.putErr, err) {
				t.Fatal(err)
			}

			// get by id
			err = mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
				amt, err := s.utxoStore.GrossBalance(tx, test.walletId)
				if test.delete {
					if !assert.Equal(t, errors.New("not found"), err) {
						t.Fatal(err)
					}
					return nil
				}
				if err == nil {
					assert.Equal(t, test.amount.String(), amt.String())
				}
				return err
			})
			if !assert.Equal(t, test.getErr, err) {
				t.Fatal(err)
			}

			// get all
			err = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
				mp, err := s.utxoStore.FetchAllMinedBalance(tx)
				assert.Nil(t, err)
				assert.Equal(t, len(test.expectAll), len(mp))
				for wid, amt := range mp {
					v, exist := test.expectAll[wid]
					assert.True(t, exist)
					assert.Equal(t, v.String(), amt.String())
				}
				return nil
			})
			if !assert.Nil(t, err) {
				t.Fatal(err)
			}
		})
	}
}

func TestPutGetAddress(t *testing.T) {
	tests := []struct {
		name      string
		puts      map[string]map[string]uint16
		expectAll map[string]map[string]uint16
		putErr    error
		getErr    error
	}{
		{
			name: "add standard address of wallet 1",
			puts: map[string]map[string]uint16{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": map[string]uint16{
					"ms1qq20yfsypqjuz305j2nhhu8khsj07mxfq2sa8ua685l2leayk02hrsk9kjvx": 0,
				},
			},
			expectAll: map[string]map[string]uint16{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": map[string]uint16{
					"ms1qq20yfsypqjuz305j2nhhu8khsj07mxfq2sa8ua685l2leayk02hrsk9kjvx": 0,
				},
			},
		},
		{
			name: "add staking address of wallet 1",
			puts: map[string]map[string]uint16{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": map[string]uint16{
					"ms1qp3x3hn3gc4umtsu7mqm0ddqsl7krergc6pdzg75t3uz82zfvdgjuqtgkkhk": 1,
				},
			},
			expectAll: map[string]map[string]uint16{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": map[string]uint16{
					"ms1qq20yfsypqjuz305j2nhhu8khsj07mxfq2sa8ua685l2leayk02hrsk9kjvx": 0,
					"ms1qp3x3hn3gc4umtsu7mqm0ddqsl7krergc6pdzg75t3uz82zfvdgjuqtgkkhk": 1,
				},
			},
		},
		{
			name: "add standard address of wallet 2",
			puts: map[string]map[string]uint16{
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": map[string]uint16{
					"ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl": 0,
				},
			},
			expectAll: map[string]map[string]uint16{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": map[string]uint16{
					"ms1qq20yfsypqjuz305j2nhhu8khsj07mxfq2sa8ua685l2leayk02hrsk9kjvx": 0,
					"ms1qp3x3hn3gc4umtsu7mqm0ddqsl7krergc6pdzg75t3uz82zfvdgjuqtgkkhk": 1,
				},
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": map[string]uint16{
					"ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl": 0,
				},
			},
		},
	}

	testDeleteWalletId := "ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz"
	testExpectAllAfterDelete := map[string]map[string]uint16{
		"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": map[string]uint16{
			"ms1qqehh47s0hvzrqqjl77ayj78yytstjkrsltcna343p8yg7ndskvveql4z3vl": 0,
		},
	}

	chainDb, chainDbTearDown, err := GetDb("TstAddressChainDb")
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer chainDbTearDown()

	s, walletDb, teardown, err := testTxStore("TstAddress", chainDb)
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer teardown()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
				for wid, mp := range test.puts {
					for addr, class := range mp {
						err := s.utxoStore.PutNewAddress(tx, wid, addr, class)
						if err != nil {
							return err
						}
					}
				}
				return nil
			})
			if !assert.Equal(t, test.putErr, err) {
				t.Fatal(err)
			}

			err = mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
				for wid, addrs := range test.expectAll {
					list, err := s.utxoStore.GetAddresses(tx, wid)
					if err != nil {
						return err
					}
					if !assert.True(t, len(list) == len(addrs)) {
						t.Fatal("expect all not equal")
					}
					for _, detail := range list {
						v, exist := addrs[detail.Address]
						if !exist {
							t.Fatal("address not found", detail.Address)
						}
						if v != detail.AddressClass {
							t.Fatal("address type mismatched", detail.AddressClass, v)
						}
					}
				}
				return nil
			})
			if !assert.Equal(t, test.getErr, err) {
				t.Fatal(err)
			}
		})
	}

	// test delete
	err = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
		return s.utxoStore.RemoveAddressByWalletId(tx, testDeleteWalletId)
	})
	assert.Nil(t, err)
	err = mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
		for wid, addrs := range testExpectAllAfterDelete {
			list, err := s.utxoStore.GetAddresses(tx, wid)
			assert.Nil(t, err)
			assert.True(t, len(list) == len(addrs))
			for _, detail := range list {
				v, exist := addrs[detail.Address]
				if !exist {
					t.Fatal("address not found", detail.Address)
				}
				if v != detail.AddressClass {
					t.Fatal("address type mismatched", detail.AddressClass, v)
				}
			}
		}
		return nil
	})
}

func simpleFilterTx(rec *TxRecord, tx *wire.MsgTx, txStore *TxStore,
	meta *BlockMeta, walletId string) (*TxRecord, error) {
	// check TxIn
	if !blockchain.IsCoinBaseTx(tx) {
		for i, txIn := range tx.TxIn {
			prevTx, err := txStore.chainFetcher.FetchTxBySha(&txIn.PreviousOutPoint.Hash)
			if err != nil {
				return nil, err
			}
			if prevTx == nil {
				return nil, errors.New("transaction not found on chain")
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

			rec.HasBindingIn = ps.IsBinding()
			rec.RelevantTxIn = append(rec.RelevantTxIn,
				&RelevantMeta{
					Index:    i,
					PkScript: ps,
					WalletId: walletId,
				})
		}
	}

	// check TxOut
	for i, txOut := range tx.TxOut {
		ps, err := utils.ParsePkScript(txOut.PkScript, txStore.chainParams)
		if err != nil {

			return nil, err
		}
		rec.HasBindingOut = ps.IsBinding()
		rec.RelevantTxOut = append(rec.RelevantTxOut,
			&RelevantMeta{
				Index:    i,
				PkScript: ps,
				WalletId: walletId,
			})
	}
	return rec, nil
}

func TestAddRelevantTx(t *testing.T) {
	chainDb, chainDbTearDown, err := GetDb("TstAddRelevantTxChainDb")
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer chainDbTearDown()

	s, walletDb, teardown, err := testTxStore("TstAddRelevant", chainDb)
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer teardown()

	numOutput := 0

	err = mwdb.Update(walletDb, func(ns mwdb.DBTransaction) error {
		wIds := s.ksmgr.ListKeystoreNames()
		err = s.ksmgr.UseKeystoreForWallet(wIds[0])
		if err != nil {
			return err
		}

		allMinedBalances := map[string]massutil.Amount{
			wIds[0]: massutil.ZeroAmount(),
		}
		totalOutput := massutil.ZeroAmount()
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
				for _, txout := range tx.MsgTx().TxOut {
					totalOutput, err = totalOutput.AddInt(txout.Value)
					if err != nil {
						return err
					}
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
		assert.Equal(t, totalOutput.String(), allMinedBalances[wIds[0]].String())
		return nil
	})
	assert.Nil(t, err)

	mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
		nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
		nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
		entries, err := fetchAllEntry(nsCredits)
		assert.Nil(t, err)
		assert.True(t, len(entries) == numOutput)
		entries, err = fetchAllEntry(nsUnspent)
		assert.Nil(t, err)
		assert.True(t, len(entries) == numOutput)
		return nil
	})
}
