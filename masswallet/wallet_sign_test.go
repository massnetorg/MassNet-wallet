// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
//
package masswallet

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"massnet.org/mass-wallet/consensus"
	"massnet.org/mass-wallet/database/memdb"
	"massnet.org/mass-wallet/massutil"

	"massnet.org/mass-wallet/masswallet/keystore"

	"massnet.org/mass-wallet/masswallet/txmgr"

	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/config"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	_ "massnet.org/mass-wallet/masswallet/db/ldb"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"
)

//wallet path
const (
	testWalletRoot = "testWallets"
	dbtype         = "leveldb"
	pubpass        = "123456"
	walletpass     = "11111111"
)

type wallet struct {
	mgr        *WalletManager
	walletName string
	mnemonic   string
	close      func()
	WitnessMap map[string][]byte
}

func iniServer() *mockServer {
	db, err := memdb.NewMemDb()
	if err != nil {
		panic(err)
	}
	return &mockServer{
		db: db,
	}
}

func iniWallet(walletName string) (*wallet, error) {
	walletPath := filepath.Join(testWalletRoot, walletName)
	fi, err := os.Stat(walletPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if fi != nil {
		os.RemoveAll(walletPath)
	}

	db, err := mwdb.CreateDB(dbtype, walletPath)
	if err != nil {
		return nil, err
	}

	cfg := &config.Config{
		Config: config.NewDefaultConfig(),
	}

	var w wallet
	w.mgr, err = NewWalletManager(iniServer(), db, cfg, &config.ChainParams, pubpass)
	if err != nil {
		db.Close()
		return nil, err
	}
	w.walletName, w.mnemonic, err = w.mgr.CreateWallet(walletpass, walletName, 128)
	if err != nil {
		db.Close()
		return nil, err
	}
	w.close = func() {
		db.Close()
		os.RemoveAll(testWalletRoot)
	}
	w.WitnessMap = make(map[string][]byte)
	return &w, nil
}

func generateAddress(w *wallet, n int) ([]*txmgr.AddressDetail, error) {
	_, err := w.mgr.UseWallet(w.walletName)
	if err != nil {
		return nil, err
	}

	addrs := make(map[string]struct{})
	for n > 0 {
		addr, err := w.mgr.NewAddress(0)
		if err != nil {
			return nil, err
		}
		addrs[addr] = struct{}{}
		n--
	}

	var result []*txmgr.AddressDetail
	details, err := w.mgr.GetAllAddressesWithPubkey()
	if err != nil {
		return nil, err
	}
	for _, detail := range details {
		if _, ok := addrs[detail.Address]; ok {
			result = append(result, detail)
		}
	}
	return result, nil
}

func createP2wshScript(w *wallet, nRequire, nTotal int) ([]byte, error) {

	addrs, err := generateAddress(w, nTotal)
	if err != nil {
		return nil, err
	}

	var pubKeys []*btcec.PublicKey
	for _, addr := range addrs {
		pubKeys = append(pubKeys, addr.PubKey)
	}
	redeemScript, witnessAddress, err := keystore.NewNonPersistentWitSAddrForBtcec(pubKeys, nRequire, massutil.AddressClassWitnessV0, &config.ChainParams)
	if err != nil {
		return nil, err
	}
	w.WitnessMap[witnessAddress.EncodeAddress()] = redeemScript
	pkScript, err := txscript.PayToAddrScript(witnessAddress)
	if err != nil {
		return nil, err
	}
	return pkScript, nil
}

func createP2wshStakingScript(w *wallet, nRequire, nTotal int, frozenPeriod uint64) ([]byte, error) {

	addrs, err := generateAddress(w, nTotal)
	if err != nil {
		return nil, err
	}

	var pubKeys []*btcec.PublicKey
	for _, addr := range addrs {
		pubKeys = append(pubKeys, addr.PubKey)
	}
	redeemScript, witnessAddress, err := keystore.NewNonPersistentWitSAddrForBtcec(pubKeys,
		nRequire, massutil.AddressClassWitnessStaking, &config.ChainParams)
	if err != nil {
		return nil, err
	}
	w.WitnessMap[witnessAddress.EncodeAddress()] = redeemScript
	pkScript, err := txscript.PayToStakingAddrScript(witnessAddress, frozenPeriod)
	if err != nil {
		return nil, err
	}
	return pkScript, nil
}

func createP2wshBindingScript(w *wallet, nRequire, nTotal int, pocPkHash []byte) ([]byte, error) {

	addrs, err := generateAddress(w, nTotal)
	if err != nil {
		return nil, err
	}

	var pubKeys []*btcec.PublicKey
	for _, addr := range addrs {
		pubKeys = append(pubKeys, addr.PubKey)
	}
	redeemScript, witnessAddress, err := keystore.NewNonPersistentWitSAddrForBtcec(pubKeys,
		nRequire, massutil.AddressClassWitnessV0, &config.ChainParams)
	if err != nil {
		return nil, err
	}
	w.WitnessMap[witnessAddress.EncodeAddress()] = redeemScript
	pkScript, err := txscript.PayToBindingScriptHashScript(witnessAddress.ScriptAddress(), pocPkHash)
	if err != nil {
		return nil, err
	}
	return pkScript, nil
}

func checkScripts(msg string, tx *wire.MsgTx, idx int, witness wire.TxWitness, pkScript []byte, value int64) error {
	tx.TxIn[idx].Witness = witness
	vm, err := txscript.NewEngine(pkScript, tx, idx,
		txscript.StandardVerifyFlags, nil, nil, value)
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
	hashCache := txscript.NewTxSigHashes(tx)

	witness, err := txscript.SignTxOutputWit(&config.ChainParams, tx,
		idx, value, pkScript, hashCache, hashType, kdb, sdb)
	if err != nil {
		return fmt.Errorf("failed to sign output %s: %v", msg, err)
	}

	return checkScripts(msg, tx, idx, witness, pkScript, value)
}

func TestAnyoneCanPay(t *testing.T) {
	wit0, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_0).AddOp(txscript.OP_DROP).Script()
	wit1, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_TRUE).Script()
	scriptHash := sha256.Sum256(wit1)
	pkscript, err := txscript.PayToWitnessScriptHashScript(scriptHash[:])
	if err != nil {
		t.FailNow()
	}
	tx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 0,
				},
				Witness:  wire.TxWitness{wit0, wit1},
				Sequence: wire.MaxTxInSequenceNum,
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value: 1,
			},
			{
				Value: 2,
			},
			{
				Value: 3,
			},
		},
		LockTime: 0,
	}

	hashCache := txscript.NewTxSigHashes(tx)
	vm, err := txscript.NewEngine(pkscript, tx, 0, txscript.StandardVerifyFlags, nil, hashCache, 3000000000)
	assert.Nil(t, err)
	if err == nil {
		err = vm.Execute()
		assert.Nil(t, err)
	}
}

//test the signTxOutputWit function
func TestSignTxOutputWit(t *testing.T) {
	// t.Parallel()
	w, err := iniWallet("txScriptWallet")
	if err != nil {
		t.Errorf("create wallet error : %v", err)
	}
	defer w.close()

	pkScript, err := createP2wshScript(w, 1, 1)
	getScript := txscript.ScriptClosure(func(addr massutil.Address) ([]byte, error) {
		// If keys were provided then we can only use the
		// redeem scripts provided with our inputs, too.

		script, _ := w.WitnessMap[addr.EncodeAddress()]

		return script, nil
	})
	getSign := txscript.SignClosure(func(pub *btcec.PublicKey, hash []byte) (*btcec.Signature, error) {
		return w.mgr.SignHash(pub, hash, []byte(walletpass))
	})
	if err != nil {
		t.Errorf("create wallet error : %v", err)
	}
	// make key
	// make script based on key.
	// sign with magic pixie dust.
	hashTypes := []txscript.SigHashType{
		txscript.SigHashAll,
		txscript.SigHashNone,
		txscript.SigHashSingle,
		txscript.SigHashAll | txscript.SigHashAnyOneCanPay,
		txscript.SigHashNone | txscript.SigHashAnyOneCanPay,
		txscript.SigHashSingle | txscript.SigHashAnyOneCanPay,
	}
	tx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 0,
				},
				Sequence: wire.MaxTxInSequenceNum,
			},
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 1,
				},
				Sequence: wire.MaxTxInSequenceNum,
			},
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 2,
				},
				Sequence: wire.MaxTxInSequenceNum,
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value: 1,
			},
			{
				Value: 2,
			},
			{
				Value: 3,
			},
		},
		LockTime: 0,
	}

	// p2wsh
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)
			//output value is 0
			var value = 0
			if err := signAndCheck(msg, tx, i, pkScript, hashType,
				getSign, getScript, nil, int64(value)); err != nil {
				t.Error(err)
				break
			}
		}
	}
}

//test the signTxOutputWit function
func TestSignStakingTxOutputWit(t *testing.T) {
	// t.Parallel()
	w, err := iniWallet("StakingTxScriptWallet")
	if err != nil {
		t.Errorf("create wallet error : %v", err)
	}
	defer w.close()

	pkScript, err := createP2wshStakingScript(w, 1, 1, consensus.MinFrozenPeriod)
	getScript := txscript.ScriptClosure(func(addr massutil.Address) ([]byte, error) {
		script, _ := w.WitnessMap[addr.EncodeAddress()]
		return script, nil
	})
	getSign := txscript.SignClosure(func(pub *btcec.PublicKey, hash []byte) (*btcec.Signature, error) {
		return w.mgr.SignHash(pub, hash, []byte(walletpass))
	})
	if err != nil {
		t.Errorf("create wallet error : %v", err)
	}
	// make key
	// make script based on key.
	// sign with magic pixie dust.
	hashTypes := []txscript.SigHashType{
		txscript.SigHashAll,
		txscript.SigHashNone,
		txscript.SigHashSingle,
		txscript.SigHashAll | txscript.SigHashAnyOneCanPay,
		txscript.SigHashNone | txscript.SigHashAnyOneCanPay,
		txscript.SigHashSingle | txscript.SigHashAnyOneCanPay,
	}
	tx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 0,
				},
				Sequence: consensus.MinFrozenPeriod + 1,
			},
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 1,
				},
				Sequence: consensus.MinFrozenPeriod + 1,
			},
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 2,
				},
				Sequence: consensus.MinFrozenPeriod + 1,
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value: 1,
			},
			{
				Value: 2,
			},
			{
				Value: 3,
			},
		},
		LockTime: 0,
	}

	// p2wsh
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)
			//output value is 0
			var value = 0
			if err := signAndCheck(msg, tx, i, pkScript, hashType,
				getSign, getScript, nil, int64(value)); err != nil {
				t.Error(err)
				break
			}
		}
	}
}

//test the signTxOutputWit function
func TestSignBindingTxOutputWit(t *testing.T) {
	// t.Parallel()
	w, err := iniWallet("BindingTxScriptWallet")
	if err != nil {
		t.Errorf("create wallet error : %v", err)
	}
	defer w.close()

	pocPkHash := []byte{
		12, 13, 14, 15, 116,
		12, 13, 14, 15, 116,
		12, 13, 14, 15, 116,
		12, 13, 14, 15, 116,
	}

	pkScript, err := createP2wshBindingScript(w, 1, 1, pocPkHash)
	getScript := txscript.ScriptClosure(func(addr massutil.Address) ([]byte, error) {
		script, _ := w.WitnessMap[addr.EncodeAddress()]
		return script, nil
	})
	getSign := txscript.SignClosure(func(pub *btcec.PublicKey, hash []byte) (*btcec.Signature, error) {
		return w.mgr.SignHash(pub, hash, []byte(walletpass))
	})
	if err != nil {
		t.Errorf("create wallet error : %v", err)
	}
	// make key
	// make script based on key.
	// sign with magic pixie dust.
	hashTypes := []txscript.SigHashType{
		txscript.SigHashAll,
		txscript.SigHashNone,
		txscript.SigHashSingle,
		txscript.SigHashAll | txscript.SigHashAnyOneCanPay,
		txscript.SigHashNone | txscript.SigHashAnyOneCanPay,
		txscript.SigHashSingle | txscript.SigHashAnyOneCanPay,
	}
	tx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 0,
				},
				Sequence: wire.MaxTxInSequenceNum,
			},
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 1,
				},
				Sequence: wire.MaxTxInSequenceNum,
			},
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  wire.Hash{},
					Index: 2,
				},
				Sequence: wire.MaxTxInSequenceNum,
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value: 1,
			},
			{
				Value: 2,
			},
			{
				Value: 3,
			},
		},
		LockTime: 0,
	}

	// p2wsh
	for _, hashType := range hashTypes {
		for i := range tx.TxIn {
			msg := fmt.Sprintf("%d:%d", hashType, i)
			//output value is 0
			var value = 0
			if err := signAndCheck(msg, tx, i, pkScript, hashType,
				getSign, getScript, nil, int64(value)); err != nil {
				t.Error(err)
				break
			}
		}
	}
}
