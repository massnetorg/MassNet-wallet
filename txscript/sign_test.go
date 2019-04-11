// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
//
package txscript_test

import (
	"errors"
	"fmt"
	"massnet.org/mass-wallet/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wallet"
	"massnet.org/mass-wallet/wire"
	"os"
	"path/filepath"
)

//wallet path
const testWalletRoot = "data/testWallets"

func iniWallet(walletName string) (*wallet.Wallet, error) {
	walletPath := filepath.Join(testWalletRoot, walletName)
	_ = os.RemoveAll(walletPath)
	w, err := wallet.NewWallet(walletPath)
	if err != nil {
		return nil, err
	}
	return w, nil
}
func generatePubkey(w *wallet.Wallet, n int) error {
	seeds := w.RootPkExStrList
	if len(seeds) == 0 {
		err := errors.New("no seeds")
		if err != nil {
			return err
		}
	}

	// generate pubkey from seed
	_, _, err := w.GenerateChildKeysForSpace(w.RootPkExStrList[0], n)
	if err != nil {
		return err
	}
	return nil
}

func createP2wshScript(w *wallet.Wallet, nRequire int) ([]byte, error) {
	var pubKeys []*btcec.PublicKey
	m := len(w.RootPkExStrToChildPkMap[w.RootPkExStrList[0]])
	for i := 0; i < 2; i++ {
		m--
		pubKeys = append(pubKeys, w.RootPkExStrToChildPkMap[w.RootPkExStrList[0]][m])
	}
	//create 2-2 p2wsh
	Address, err := w.NewWitnessScriptAddress(pubKeys, nRequire, 0)
	if err != nil {
		return nil, err
	}
	DecodeAddress, err := massutil.DecodeAddress(Address, &config.ChainParams)
	if err != nil {
		return nil, err
	}
	pkScript, err := txscript.PayToAddrScript(DecodeAddress)
	if err != nil {
		return nil, err
	}
	return pkScript, nil
}

func checkScripts(msg string, tx *wire.MsgTx, idx int, witness wire.TxWitness, pkScript []byte, value int64) error {
	tx.TxIn[idx].Witness = witness
	vm, err := txscript.NewEngine(pkScript, tx, idx,
		txscript.ScriptBip16|txscript.ScriptVerifyDERSignatures, nil, nil, value)
	if err != nil {
		return fmt.Errorf("failed to make script engine for %s: %v",
			msg, err)
	}

	err = vm.Execute()
	if err != nil {
		return fmt.Errorf("invalid script signature for %s: %v", msg,
			err)
	}

	return nil
}

//sign and execute the script
func signAndCheck(msg string, tx *wire.MsgTx, idx int, pkScript []byte,
	hashType txscript.SigHashType, kdb txscript.GetSignDB, sdb txscript.ScriptDB,
	previousScript []byte, value int64) error {

	witness, err := txscript.SignTxOutputWit(&config.ChainParams, tx,
		idx, value, pkScript, hashType, kdb, sdb)
	if err != nil {
		return fmt.Errorf("failed to sign output %s: %v", msg, err)
	}

	return checkScripts(msg, tx, idx, witness, pkScript, value)
}
