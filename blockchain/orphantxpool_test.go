package blockchain

import (
	"bytes"
	"testing"

	"github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/wire"
)

func TestAddOrphan(t *testing.T) {
	pool := newOrphanTxPool()
	tests := []struct {
		tx wire.MsgTx
	}{
		{
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{},
				TxOut:    []*wire.TxOut{},
				LockTime: 0,
				Payload:  []byte{},
			},
		},
		{
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{},
				TxOut:    []*wire.TxOut{},
				LockTime: 10,
				Payload:  []byte{},
			},
		},
		{
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{},
				TxOut:    []*wire.TxOut{},
				LockTime: 20,
				Payload:  []byte{},
			},
		},
	}
	config.MaxOrphanTxs = 2
	for _, test := range tests {
		pool.addOrphan(massutil.NewTx(&test.tx))
	}
	if len(pool.orphans) == 2 {
		t.Log("success to check limitNumOrphans")
	}
}

func TestMaybeAddOrphan(t *testing.T) {
	pool := newOrphanTxPool()
	msgtx := wire.MsgTx{
		Version: 1,
		TxIn:    []*wire.TxIn{},
		TxOut: []*wire.TxOut{{
			Value: 0,
			PkScript: bytes.Repeat([]byte{0x00},
				maxOrphanTxSize+1),
		}},
		LockTime: 0,
		Payload:  []byte{},
	}

	tx := massutil.NewTx(&msgtx)
	err := pool.maybeAddOrphan(tx)
	if err != nil {
		if err.Error() == "orphan transaction size of 5017 bytes is larger than max allowed size of 5000 bytes" {
			t.Log("success to reject a tx size > maxOrphanTxSize")
		} else {
			t.Error(err)
		}
	}

	pool.removeOrphan(tx.Hash())
}

func TestIsOrphanInPool(t *testing.T) {
	pool := newOrphanTxPool()
	msgtx := wire.MsgTx{
		Version: 1,
		TxIn:    []*wire.TxIn{},
		TxOut: []*wire.TxOut{{
			Value:    0,
			PkScript: bytes.Repeat([]byte{0x00}, 10),
		}},
		LockTime: 0,
		Payload:  []byte{},
	}

	tx := massutil.NewTx(&msgtx)
	err := pool.maybeAddOrphan(tx)
	if err != nil {
		t.Error(err)
	} else {
		if result := pool.isOrphanInPool(tx.Hash()); result {
			t.Log("success to find the tx in orphan pool")
		} else {
			t.Fail()
		}
	}
}
