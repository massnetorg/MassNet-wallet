// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"testing"

	"github.com/massnetorg/MassNet-wallet/btcec"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/txscript"
	"github.com/massnetorg/MassNet-wallet/wire"
)

const defaultMinRelayTxFee = massutil.Amount(1000)

// TestCalcMinRequiredTxRelayFee tests the calcMinRequiredTxRelayFee API.
func TestCalcMinRequiredTxRelayFee(t *testing.T) {
	tests := []struct {
		name     string          // test description.
		size     int64           // Transaction size in bytes.
		relayFee massutil.Amount // minimum relay transaction fee.
		want     int64           // Expected fee.
	}{
		{
			// Ensure combination of size and fee that are less than 1000
			// produce a non-zero fee.
			"250 bytes with relay fee of 3",
			250,
			3,
			3,
		},
		{
			"100 bytes with default minimum relay fee",
			100,
			defaultMinRelayTxFee,
			100,
		},
		{
			"max standard tx size with default minimum relay fee",
			maxStandardTxSize,
			defaultMinRelayTxFee,
			100000,
		},
		{
			"max standard tx size with max Maxwell relay fee",
			maxStandardTxSize,
			massutil.MaxMaxwell,
			massutil.MaxMaxwell,
		},
		{
			"1500 bytes with 5000 relay fee",
			1500,
			5000,
			7500,
		},
		{
			"1500 bytes with 3000 relay fee",
			1500,
			3000,
			4500,
		},
		{
			"782 bytes with 5000 relay fee",
			782,
			5000,
			3910,
		},
		{
			"782 bytes with 3000 relay fee",
			782,
			3000,
			2346,
		},
		{
			"782 bytes with 2550 relay fee",
			782,
			2550,
			1994,
		},
	}

	for i, test := range tests {
		got := calcMinRequiredTxRelayFee(test.size, test.relayFee)
		if got != test.want {
			t.Errorf("index：%d, TestCalcMinRequiredTxRelayFee test '%s' "+
				"failed: got %v want %v", i, test.name, got,
				test.want)
			continue
		}
	}
}

// TestCheckPkScriptStandard tests the checkPkScriptStandard API.
func TestCheckPkScriptStandard(t *testing.T) {
	var pubKeys [][]byte
	for i := 0; i < 4; i++ {
		pk, err := btcec.NewPrivateKey(btcec.S256())
		if err != nil {
			t.Fatalf("TestCheckPkScriptStandard NewPrivateKey failed: %v",
				err)
			return
		}
		pubKeys = append(pubKeys, pk.PubKey().SerializeCompressed())
	}
	pubkey := []byte{75}
	tests := []struct {
		name       string // test description.
		script     *txscript.ScriptBuilder
		isStandard bool
	}{
		{
			"parse failure",
			txscript.NewScriptBuilder().
				AddData(pubkey).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"numPubKeys < 1",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_0).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"key1 and key2",
			txscript.NewScriptBuilder().AddOp(txscript.OP_2).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_2).AddOp(txscript.OP_CHECKMULTISIG),
			true,
		},
		{
			"key1 or key2",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_2).AddOp(txscript.OP_CHECKMULTISIG),
			true,
		},
		{
			"escrow",
			txscript.NewScriptBuilder().AddOp(txscript.OP_2).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddData(pubKeys[2]).
				AddOp(txscript.OP_3).AddOp(txscript.OP_CHECKMULTISIG),
			true,
		},
		{
			"one of four",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddData(pubKeys[2]).AddData(pubKeys[3]).
				AddOp(txscript.OP_4).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed1",
			txscript.NewScriptBuilder().AddOp(txscript.OP_3).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_2).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed2",
			txscript.NewScriptBuilder().AddOp(txscript.OP_2).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_3).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed3",
			txscript.NewScriptBuilder().AddOp(txscript.OP_0).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_2).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed4",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_0).AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed5",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]).
				AddOp(txscript.OP_CHECKMULTISIG),
			false,
		},
		{
			"malformed6",
			txscript.NewScriptBuilder().AddOp(txscript.OP_1).
				AddData(pubKeys[0]).AddData(pubKeys[1]),
			false,
		},
	}

	for _, test := range tests[2:] {
		script, err := test.script.Script()
		if err != nil {
			t.Fatalf("TestCheckPkScriptStandard test '%s' "+
				"failed: %v", test.name, err)
			continue
		}
		scriptClass := txscript.GetScriptClass(script)
		got := checkPkScriptStandard(script, scriptClass)
		if (test.isStandard && got != nil) ||
			(!test.isStandard && got == nil) {
			t.Log(got)
			t.Fatalf("TestCheckPkScriptStandard test '%s' failed",
				test.name)
			return
		}
	}

	for _, test := range tests[:2] {
		script, err := test.script.Script()
		if err != nil {
			t.Fatalf("TestCheckPkScriptStandard test '%s' "+
				"failed: %v", test.name, err)
			continue
		}

		got := checkPkScriptStandard(script, txscript.MultiSigTy)
		if (test.isStandard && got != nil) ||
			(!test.isStandard && got == nil) {
			t.Log(got)
			t.Fatalf("TestCheckPkScriptStandard test '%s' failed",
				test.name)
			return
		}
	}

}

// TestDust tests the isDust API.
func TestDust(t *testing.T) {
	pkScript := []byte{0, 20, 63, 144, 140, 149, 87, 108, 214, 24, 129, 250, 177, 187, 82, 157, 39, 239, 170, 233, 125, 124}

	tests := []struct {
		name     string // test description
		txOut    wire.TxOut
		relayFee massutil.Amount // minimum relay transaction fee.
		isDust   bool
	}{
		{
			// Any value is allowed with a zero relay fee.
			"zero value with zero relay fee",
			wire.TxOut{0, pkScript},
			0,
			false,
		},
		{
			// Zero value is dust with any relay fee"
			"zero value with very small tx fee",
			wire.TxOut{0, pkScript},
			1,
			true,
		},
		{
			"30 byte public key script with value 584",
			wire.TxOut{539, pkScript},
			1000,
			true,
		},
		{
			"30 byte public key script with value 585",
			wire.TxOut{540, pkScript},
			1000,
			false,
		},
		{
			// Maximum allowed value is never dust.
			"max Maxwell amount is never dust",
			wire.TxOut{massutil.MaxMaxwell, pkScript},
			massutil.MaxMaxwell,
			false,
		},
		{
			// Maximum int64 value causes overflow.
			"maximum int64 value",
			wire.TxOut{1<<63 - 1, pkScript},
			1<<63 - 1,
			true,
		},
		{
			// Unspendable pkScript due to an invalid public key
			// script.
			"unspendable pkScript",
			wire.TxOut{5000, []byte{0x01}},
			0, // no relay fee
			true,
		},
	}
	for i, test := range tests {
		res := isDust(&test.txOut, test.relayFee)
		if res != test.isDust {
			t.Fatalf("i:%d,Dust test '%s' failed: want %v got %v", i,
				test.name, test.isDust, res)
			continue
		}
	}
}
