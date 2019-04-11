// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"math"
	"runtime"

	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/txscript"
	"github.com/massnetorg/MassNet-wallet/wire"
)

// txValidateItem holds a transaction along with which input to validate.
type txValidateItem struct {
	txInIndex int
	txIn      *wire.TxIn
	tx        *massutil.Tx
	sigHashes *txscript.TxSigHashes
}

// txValidator provides a type which asynchronously validates transaction
// inputs.  It provides several channels for communication and a processing
// function that is intended to be in run multiple goroutines.
type txValidator struct {
	validateChan chan *txValidateItem
	quitChan     chan struct{}
	resultChan   chan error
	txStore      TxStore
	flags        txscript.ScriptFlags
	sigCache     *txscript.SigCache
	hashCache    *txscript.HashCache
}

// sendResult sends the result of a script pair validation on the internal
// result channel while respecting the quit channel.  The allows orderly
// shutdown when the validation process is aborted early due to a validation
// error in one of the other goroutines.
func (v *txValidator) sendResult(result error) {
	select {
	case v.resultChan <- result:
	case <-v.quitChan:
	}
}

// validateHandler consumes items to validate from the internal validate channel
// and returns the result of the validation on the internal result channel. It
// must be run as a goroutine.
func (v *txValidator) validateHandler() {
out:
	for {
		select {
		case txVI := <-v.validateChan:

			txIn := txVI.txIn
			originTxHash := &txIn.PreviousOutPoint.Hash
			originTx, exists := v.txStore[*originTxHash]
			if !exists || originTx.Err != nil || originTx.Tx == nil {
				str := fmt.Sprintf("unable to find input "+
					"transaction %v referenced from "+
					"transaction %v", originTxHash,
					txVI.tx.Hash())
				err := ruleError(ErrMissingTx, str)
				v.sendResult(err)
				break out
			}
			originMsgTx := originTx.Tx.MsgTx()

			originTxIndex := txIn.PreviousOutPoint.Index
			if originTxIndex >= uint32(len(originMsgTx.TxOut)) {
				str := fmt.Sprintf("out of bounds "+
					"input index %d in transaction %v "+
					"referenced from transaction %v",
					originTxIndex, originTxHash,
					txVI.tx.Hash())
				err := ruleError(ErrBadTxInput, str)
				v.sendResult(err)
				break out
			}

			witness := txIn.Witness
			pkScript := originMsgTx.TxOut[originTxIndex].PkScript
			inputAmount := originMsgTx.TxOut[originTxIndex].Value
			vm, err := txscript.NewEngine(pkScript, txVI.tx.MsgTx(),
				txVI.txInIndex, v.flags, v.sigCache, txVI.sigHashes,
				inputAmount)
			if err != nil {
				str := fmt.Sprintf("failed to parse input "+
					"%s:%d which references output %v - "+
					"%v (input witness %x, prev output script bytes %x)",
					txVI.tx.Hash(), txVI.txInIndex,
					txIn.PreviousOutPoint, err, witness, pkScript)
				err := ruleError(ErrScriptMalformed, str)
				v.sendResult(err)
				break out
			}

			if err := vm.Execute(); err != nil {
				str := fmt.Sprintf("failed to validate input "+
					"%s:%d which references output %v - "+
					"%v (input witness %x, prev output script bytes %x)",
					txVI.tx.Hash(), txVI.txInIndex,
					txIn.PreviousOutPoint, err, witness, pkScript)
				err := ruleError(ErrScriptValidation, str)
				v.sendResult(err)
				break out
			}

			v.sendResult(nil)

		case <-v.quitChan:
			break out
		}
	}
}

// Validate validates the scripts for all of the passed transaction inputs using
// multiple goroutines.
func (v *txValidator) Validate(items []*txValidateItem) error {
	if len(items) == 0 {
		return nil
	}

	maxGoRoutines := runtime.NumCPU() * 3
	if maxGoRoutines <= 0 {
		maxGoRoutines = 1
	}
	if maxGoRoutines > len(items) {
		maxGoRoutines = len(items)
	}

	for i := 0; i < maxGoRoutines; i++ {
		go v.validateHandler()
	}

	numInputs := len(items)
	currentItem := 0
	processedItems := 0
	for processedItems < numInputs {
		var validateChan chan *txValidateItem
		var item *txValidateItem
		if currentItem < numInputs {
			validateChan = v.validateChan
			item = items[currentItem]
		}

		select {
		case validateChan <- item:
			currentItem++

		case err := <-v.resultChan:
			processedItems++
			if err != nil {
				close(v.quitChan)
				return err
			}
		}
	}

	close(v.quitChan)
	return nil
}

// newTxValidator returns a new instance of txValidator to be used for
// validating transaction scripts asynchronously.
func newTxValidator(txStore TxStore, flags txscript.ScriptFlags, sigCache *txscript.SigCache, hashCache *txscript.HashCache) *txValidator {
	return &txValidator{
		validateChan: make(chan *txValidateItem),
		quitChan:     make(chan struct{}),
		resultChan:   make(chan error),
		txStore:      txStore,
		sigCache:     sigCache,
		hashCache:    hashCache,
		flags:        flags,
	}
}

// ValidateTransactionScripts validates the scripts for the passed transaction
// using multiple goroutines.
func ValidateTransactionScripts(tx *massutil.Tx, txStore TxStore, flags txscript.ScriptFlags, sigCache *txscript.SigCache, hashCache *txscript.HashCache) error {
	if !hashCache.ContainsHashes(tx.Hash()) {
		hashCache.AddSigHashes(tx.MsgTx())
	}

	var cachedHashes *txscript.TxSigHashes
	cachedHashes, _ = hashCache.GetSigHashes(tx.Hash())

	txIns := tx.MsgTx().TxIn
	txValItems := make([]*txValidateItem, 0, len(txIns))

	for txInIdx, txIn := range txIns {
		if txIn.PreviousOutPoint.Index == math.MaxUint32 {
			continue
		}

		txVI := &txValidateItem{
			txInIndex: txInIdx,
			txIn:      txIn,
			tx:        tx,
			sigHashes: cachedHashes,
		}
		txValItems = append(txValItems, txVI)
	}
	for _, txOut := range tx.MsgTx().TxOut {
		pkScript := txOut.PkScript

		if txscript.IsPayToLocktimeScriptHash(pkScript) {
			value := txOut.Value
			pops, class := txscript.GetScriptInfo(pkScript)
			height, _, err := txscript.GetParsedOpcode(class, pops)
			if err != nil {
				return err
			}
			if value < MinLockValue {
				str := fmt.Sprintf("Only more than %f value is supported in lockTX", MinLockValue)
				err := ruleError(ErrScriptValidation, str)
				return err
			}
			if height > wire.SequenceLockTimeIsSeconds || height < wire.MinLockHeight {
				str := fmt.Sprintf("The lock height should be more than %d height in lockTX", wire.MinLockHeight)
				err := ruleError(ErrScriptValidation, str)
				return err
			}
		}

	}

	validator := newTxValidator(txStore, flags, sigCache, hashCache)
	if err := validator.Validate(txValItems); err != nil {
		return err
	}

	return nil
}

// checkBlockScripts executes and validates the scripts for all transactions in
// the passed block.
func checkBlockScripts(block *massutil.Block, txStore TxStore,
	scriptFlags txscript.ScriptFlags, sigCache *txscript.SigCache, hashCache *txscript.HashCache) error {

	numInputs := 0
	for _, tx := range block.Transactions()[1:] {

		numInputs += len(tx.MsgTx().TxIn)
	}
	txValItems := make([]*txValidateItem, 0, numInputs)
	for _, tx := range block.Transactions()[1:] {

		hash := tx.Hash()
		if hashCache != nil && !hashCache.ContainsHashes(hash) {

			hashCache.AddSigHashes(tx.MsgTx())
		}
		var cachedHashes *txscript.TxSigHashes
		if true {
			if hashCache != nil {
				cachedHashes, _ = hashCache.GetSigHashes(hash)
			} else {
				cachedHashes = txscript.NewTxSigHashes(tx.MsgTx())
			}
		}

		for txInIdx, txIn := range tx.MsgTx().TxIn {
			if txIn.PreviousOutPoint.Index == math.MaxUint32 {
				continue
			}

			txVI := &txValidateItem{
				txInIndex: txInIdx,
				txIn:      txIn,
				tx:        tx,
				sigHashes: cachedHashes,
			}
			txValItems = append(txValItems, txVI)
		}
	}

	validator := newTxValidator(txStore, scriptFlags, sigCache, hashCache)
	if err := validator.Validate(txValItems); err != nil {
		return err
	}

	if hashCache != nil {
		for _, tx := range block.Transactions() {
			hashCache.PurgeSigHashes(tx.Hash())
		}
	}

	return nil
}
