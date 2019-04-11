// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"reflect"
	"testing"
)

// TestTx tests the MsgTx API.
func TestTx(t *testing.T) {
	var testRound = 100

	for i := 0; i < testRound; i++ {
		tx := mockTx()
		var wBuf bytes.Buffer
		err := tx.Serialize(&wBuf, DB)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		newTx := new(MsgTx)
		err = newTx.Deserialize(&wBuf, DB)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		// compare tx and newTx
		if !reflect.DeepEqual(tx, newTx) {
			t.Error("tx and newTx is not equal")
		}
	}
}

// TestTxOverflowErrors performs tests to ensure deserializing transactions
// which are intentionally crafted to use large values for the variable number
// of inputs and outputs are handled properly.  This could otherwise potentially
// be used as an attack vector.
func TestTxOverflowErrors(t *testing.T) {
	// Use protocol version 70001 and transaction version 1 specifically
	// here instead of the latest values because the test data is using
	// bytes encoded with those versions.
	pver := uint32(70001)
	txVer := uint32(1)

	tests := []struct {
		tx      MsgTx
		version uint32 // Transaction version
		err     error  // Expected error
	}{
		// Transaction that claims to have ~uint64(0) inputs.
		{
			MsgTx{
				Version: 1,
			}, txVer, nil,
		},

		// Transaction that claims to have ~uint64(0) outputs.
		{
			MsgTx{
				Version: 1,
			}, txVer, nil,
		},

		// Transaction that has an input with a signature script that
		// claims to have ~uint64(0) length.
		{
			MsgTx{
				Version: 1,
				TxIn: []*TxIn{
					{
						PreviousOutPoint: OutPoint{
							Hash: Hash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
								0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
								0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
								0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
							Index: 0xffffffff},
					},
				},
			}, txVer, nil,
		},

		// Transaction that has an output with a public key script
		// that claims to have ~uint64(0) length.
		{
			MsgTx{
				Version: 1,
				TxIn: []*TxIn{
					{
						PreviousOutPoint: OutPoint{
							Hash: Hash{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
								0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
								0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
								0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
							Index: 0xffffffff},
						Sequence: 0xffffffff,
					},
				},
				TxOut: []*TxOut{
					{
						Value: 0x7fffffffffffffff,
					},
				},
			}, pver, nil,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Decode from wire format.
		var tx MsgTx
		var wBuf bytes.Buffer
		err := tx.MassEncode(&wBuf, DB)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("MassDecode #%d wrong error got: %v, want: %v",
				i, err, reflect.TypeOf(test.err))
			continue
		}

		// Decode from wire format.
		rBuf := bytes.NewReader(wBuf.Bytes())
		err = tx.Deserialize(rBuf, DB)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("Deserialize #%d wrong error got: %v, want: %v",
				i, err, reflect.TypeOf(test.err))
			continue
		}
	}
}
