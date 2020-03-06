package txmgr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
)

func TestSetGetSyncedTo(t *testing.T) {
	tests := []struct {
		name string
		f    func(*TxStore, mwdb.DBTransaction, *massutil.Block) error
		num  int
		err  string
	}{
		{
			name: "valid Block 20",
			f: func(s *TxStore, ns mwdb.DBTransaction, block *massutil.Block) error {
				meta := &BlockMeta{
					Height:    block.Height(),
					Hash:      *block.Hash(),
					Timestamp: block.MsgBlock().Header.Timestamp,
				}
				return s.syncStore.SetSyncedTo(ns, meta)
			},
			num: 20,
			err: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chainDb, chainDbTearDown, err := GetDb("ChainTestDb")
			if !assert.Nil(t, err) {
				t.Fatal(err)
			}
			defer chainDbTearDown()

			s, walletDb, teardown, err := testTxStore("TstGetSetSyncedTo", chainDb)
			if !assert.Nil(t, err) {
				t.Fatal(err)
			}
			defer teardown()

			for i, blk := range blks200[0:test.num] {
				err := mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
					return test.f(s, tx, blk)
				})
				assert.Nil(t, err)

				mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
					meta, err := s.syncStore.SyncedTo(tx)
					assert.Nil(t, err)
					assert.Equal(t, uint64(i), meta.Height)
					assert.Equal(t, blk.Hash(), &meta.Hash)
					assert.True(t, blk.MsgBlock().Header.Timestamp.Equal(meta.Timestamp))
					return nil
				})
			}

			for i, blk := range blks200[0:test.num] {
				mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
					meta, err := s.syncStore.SyncedBlock(tx, uint64(i))
					assert.Nil(t, err)
					assert.Equal(t, uint64(i), meta.Height)
					assert.Equal(t, blk.Hash(), &meta.Hash)
					assert.True(t, blk.MsgBlock().Header.Timestamp.Equal(meta.Timestamp))
					return nil
				})
			}
		})
	}
}

func TestResetSyncedTo(t *testing.T) {
	tests := []struct {
		name   string
		before int
		after  int
		err    string
	}{
		{
			name:   "reset 20->10",
			before: 20,
			after:  10,
			err:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chainDb, chainDbTearDown, err := GetDb("ChainTestDb")
			if !assert.Nil(t, err) {
				t.Fatal(err)
			}
			defer chainDbTearDown()

			s, walletDb, teardown, err := testTxStore("TstResetSyncedTo", chainDb)
			if !assert.Nil(t, err) {
				t.Fatal(err)
			}
			defer teardown()

			for _, blk := range blks200[0:test.before] {
				err := mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
					meta := &BlockMeta{
						Height:    blk.Height(),
						Hash:      *blk.Hash(),
						Timestamp: blk.MsgBlock().Header.Timestamp,
					}
					return s.syncStore.SetSyncedTo(tx, meta)
				})
				assert.Nil(t, err)
			}
			mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
				meta, err := s.syncStore.SyncedTo(tx)
				assert.Nil(t, err)
				assert.Equal(t, uint64(19), meta.Height)
				return nil
			})

			err = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
				return s.syncStore.ResetSyncedTo(tx, 10)
			})
			assert.Nil(t, err)

			mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
				meta, err := s.syncStore.SyncedTo(tx)
				assert.Nil(t, err)
				assert.Equal(t, uint64(10), meta.Height)
				return nil
			})
		})
	}
}

func TestGetWalletStatus(t *testing.T) {
	tests := []struct {
		name         string
		walletId     string
		syncedHeight uint64
		expectAll    map[string]uint64
		delete       bool
		putErr       error
		getErr       error
	}{
		{
			name:         "add valid 1",
			walletId:     "ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp",
			syncedHeight: uint64(10),
			expectAll: map[string]uint64{
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": uint64(10),
			},
		},
		{
			name:         "update valid 1",
			walletId:     "ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp",
			syncedHeight: uint64(20),
			expectAll: map[string]uint64{
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": uint64(20),
			},
		},
		{
			name:         "add valid 2",
			walletId:     "ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz",
			syncedHeight: uint64(0),
			expectAll: map[string]uint64{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": uint64(0),
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": uint64(20),
			},
		},
		{
			name:     "add invalid 1",
			walletId: "ac10jv5xfkywm9fu2elcjyqyq4gyz6y",
			expectAll: map[string]uint64{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": uint64(0),
				"ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp": uint64(20),
			},
			putErr: fmt.Errorf("putWalletStatus expect 42 bytes key(acutal %d)", len("ac10jv5xfkywm9fu2elcjyqyq4gyz6y")),
			getErr: fmt.Errorf("readWalletStatus expect 42 bytes key(acutal %d)", len("ac10jv5xfkywm9fu2elcjyqyq4gyz6y")),
		},
		{
			name:     "delete valid 1",
			walletId: "ac10tcdcmxcatq0dp0ceucgdjc5m7azujzfenzzwfp",
			expectAll: map[string]uint64{
				"ac10jv5xfkywm9fu2elcjyqyq4gyz6yu6jzm7fq8fz": uint64(0),
			},
			delete: true,
		},
	}

	chainDb, chainDbTearDown, err := GetDb("ChainTestDb")
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer chainDbTearDown()

	s, walletDb, teardown, err := testTxStore("TstWalletStatus", chainDb)
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer teardown()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
				if test.delete {
					return s.syncStore.DeleteWalletStatus(tx, test.walletId)
				}
				return s.syncStore.PutWalletStatus(tx, &WalletStatus{
					WalletID:     test.walletId,
					SyncedHeight: test.syncedHeight,
				})
			})
			if !assert.Equal(t, test.putErr, err) {
				t.Fatal(err)
			}

			// get by id
			err = mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
				ws, err := s.syncStore.GetWalletStatus(tx, test.walletId)
				if test.delete {
					if !assert.Equal(t, errors.New("readWalletStatus expect 8 bytes value(acutal 0)"), err) {
						t.Fatal(err)
					}
					return nil
				}
				if err == nil {
					assert.Equal(t, test.syncedHeight, ws.SyncedHeight)
					assert.Equal(t, test.walletId, ws.WalletID)
				}
				return err
			})
			if !assert.Equal(t, test.getErr, err) {
				t.Fatal(err)
			}

			// get all
			err = mwdb.View(walletDb, func(tx mwdb.ReadTransaction) error {
				list, err := s.syncStore.GetAllWalletStatus(tx)
				assert.Nil(t, err)
				assert.Equal(t, len(test.expectAll), len(list))
				for _, ite := range list {
					v, exist := test.expectAll[ite.WalletID]
					assert.True(t, exist)
					assert.Equal(t, v, ite.SyncedHeight)
				}
				return nil
			})
			if !assert.Nil(t, err) {
				t.Fatal(err)
			}
		})
	}
}
