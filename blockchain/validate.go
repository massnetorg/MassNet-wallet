// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"time"

	"massnet.org/mass-wallet/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"
)

const (
	MaxSigOpsPerBlock = wire.MaxBlockPayload / 150 * txscript.MaxPubKeysPerMultiSig

	MaxTimeOffsetSeconds = 2 * 60 * 60

	MinCoinbaseScriptLen = 2

	MaxCoinbaseScriptLen = 100

	medianTimeBlocks = 11

	serializedHeightVersion = 2

	baseSubsidy = 50 * massutil.MaxwellPerMass

	CoinbaseMaturity = 1

	TransactionMaturity = 1

	FoundationAddr = "ms1qc9g5lnqduclq2kzjfcx9v6wqsg2zfwe2v3zt25"
)

var (
	coinbaseMaturity = int32(CoinbaseMaturity)

	zeroHash = &wire.Hash{}
)

// isNullOutpoint determines whether or not a previous transaction output point
// is set.
func isNullOutpoint(outpoint *wire.OutPoint) bool {
	if outpoint.Index == math.MaxUint32 && outpoint.Hash.IsEqual(zeroHash) {
		return true
	}
	return false
}

// IsCoinBaseTx determines whether or not a transaction is a coinbase.  A coinbase
// is a special transaction created by miners. This is represented in the block
// chain by a transaction with the first input that has a previous output transaction
// index set to the maximum value along with a zero hash.
//
// This function only differs from IsCoinBase in that it works with a raw wire
// transaction as opposed to a higher level util transaction.
func IsCoinBaseTx(msgTx *wire.MsgTx) bool {
	prevOut := &msgTx.TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 || !prevOut.Hash.IsEqual(zeroHash) {
		return false
	}
	return true
}

// IsCoinBaseTx determines whether or not a transaction is a coinbase.  A coinbase
// is a special transaction created by miners. This is represented in the block
// chain by a transaction with the first input that has a previous output transaction
// index set to the maximum value along with a zero hash.
//
// This function only differs from IsCoinBaseTx in that it works with a higher
// level util transaction as opposed to a raw wire transaction.
func IsCoinBase(tx *massutil.Tx) bool {
	return IsCoinBaseTx(tx.MsgTx())
}

// PkToAddress
func PkToAddress(pk *btcec.PublicKey, net *config.Params) (string, error) {
	var addressPubKeyStructs []*massutil.AddressPubKey
	pubKeySerial := pk.SerializeCompressed()
	addressPubKeyStruct, err := massutil.NewAddressPubKey(pubKeySerial, net)
	if err != nil {
		return "", err
	}

	addressPubKeyStructs = append(addressPubKeyStructs, addressPubKeyStruct)
	redeemScript, err := txscript.MultiSigScript(addressPubKeyStructs, 1)
	if err != nil {
		return "", err
	}

	scriptHash := massutil.Hash160(redeemScript)
	scriptHashStruct, err := massutil.NewAddressWitnessScriptHash(scriptHash, net)
	if err != nil {
		return "", err
	}

	address := scriptHashStruct.EncodeAddress()
	return address, nil
}

// checkCoinbase check the output of coinbase
func checkCoinbase(tx *massutil.Tx, db database.Db, pk *btcec.PublicKey, nextBlockHeight int32, net *config.Params) (bool, int64, error) {
	var value, reward int64

	for _, txIn := range tx.MsgTx().TxIn[1:] {
		txHash := txIn.PreviousOutPoint.Hash
		index := txIn.PreviousOutPoint.Index
		txList, err := db.FetchTxBySha(&txIn.PreviousOutPoint.Hash)
		if err != nil {
			return false, 0, err
		}
		txlast := txList[len(txList)-1]
		mtx := txlast.Tx

		blocksSincePrev := nextBlockHeight - txlast.Height
		if IsCoinBaseTx(mtx) {
			if blocksSincePrev < coinbaseMaturity {
				logging.CPrint(logging.ERROR, "tried to spemd coinbase before required mature",
					logging.LogFormat{
						"next block height": nextBlockHeight,
						"coinbase maturity": coinbaseMaturity,
					})
				return false, 0, errors.New("tried to spemd coinbase before required mature")
			}
		} else if blocksSincePrev < TransactionMaturity {
			logging.CPrint(logging.ERROR, "the txIn is not mature",
				logging.LogFormat{
					"TxHash": txHash,
					"Index":  index,
				})
			return false, 0, errors.New("the txIn is not mature")
		}

		address, err := PkToAddress(pk, net)
		if err != nil {
			return false, 0, err
		}
		addr, err := massutil.DecodeAddress(address, net)
		if err != nil {
			return false, 0, err
		}
		addrscript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return false, 0, err
		}
		if !bytes.Equal(mtx.TxOut[index].PkScript, addrscript) {
			str := fmt.Sprintf("collateral address is not correct")
			return false, 0, ruleError(ErrCollateralAddress, str)

		}
		dbSpentInfo := txlast.TxSpent

		if index > uint32(len(mtx.TxOut)-1) {
			str := fmt.Sprintf("index is not correct")
			return false, 0, ruleError(ErrIndex, str)
		}

		txOut := mtx.TxOut[index]
		if txOut == nil {
			str := fmt.Sprintf("Incheck:Output index: %d does not exist", index)
			return false, 0, ruleError(ErrTxOutNil, str)
		}

		if dbSpentInfo != nil && dbSpentInfo[index] {
			str := fmt.Sprint("tx has been spent")
			return false, 0, ruleError(ErrTxSpent, str)
		}
		value += mtx.TxOut[index].Value

	}

	address, err := massutil.DecodeAddress(FoundationAddr, net)
	if err != nil {
		return false, 0, err
	}
	switch address.(type) {
	case *massutil.AddressWitnessScriptHash:
	default:
		str := fmt.Sprintf("address is not correct")
		return false, 0, ruleError(ErrFoundationAddress, str)
	}

	pkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return false, 0, err
	}
	if !bytes.Equal(tx.MsgTx().TxOut[0].PkScript, pkScript) {
		str := fmt.Sprintf("Foundation address is not correct")
		return false, 0, ruleError(ErrFoundationAddress, str)

	}

	miner, foundation := CalcBlockSubsidy(nextBlockHeight, net, value)

	reward = miner + foundation

	return true, reward, nil
}

// SequenceLockActive determines if a transaction's sequence locks have been
// met, meaning that all the inputs of a given transaction have reached a
// height or time sufficient for their relative lock-time maturity.
func SequenceLockActive(sequenceLock *SequenceLock, blockHeight int32,
	medianTimePast time.Time) bool {
	if sequenceLock.Seconds >= medianTimePast.Unix() ||
		sequenceLock.BlockHeight >= blockHeight {
		return false
	}
	return true
}

// IsFinalizedTransaction determines whether or not a transaction is finalized.
func IsFinalizedTransaction(tx *massutil.Tx, blockHeight int32, blockTime time.Time) bool {
	msgTx := tx.MsgTx()

	lockTime := msgTx.LockTime
	if lockTime == 0 {
		return true
	}

	var blockTimeOrHeight int64
	if lockTime < txscript.LockTimeThreshold {
		blockTimeOrHeight = int64(blockHeight)
	} else {
		blockTimeOrHeight = blockTime.Unix()
	}
	if int64(lockTime) < blockTimeOrHeight {
		return true
	}

	for _, txIn := range msgTx.TxIn {
		if txIn.Sequence != math.MaxUint32 {
			return false
		}
	}
	return true
}

// CalcBlockSubsidy returns the subsidy amount a block at the provided height
// should have. This is mainly used for determining how much the coinbase for
// newly generated blocks awards as well as validating the coinbase for blocks
// has the expected value.
//
// The subsidy is halved every SubsidyHalvingInterval blocks.  Mathematically
// this is: baseSubsidy / 2^(height/subsidyHalvingInterval)
//

func calBlockSubsidy(value, miner, foundation, subsidy int64) (int64, int64) {
	if value >= 10*massutil.MaxwellPerMass {
		miner = subsidy * 8 / 10
		foundation = subsidy * 2 / 10
	} else if value < 10*massutil.MaxwellPerMass && value >= 5*massutil.MaxwellPerMass {
		miner = subsidy * 7 / 10
		foundation = subsidy * 3 / 10
	} else if value < 5*massutil.MaxwellPerMass && value >= 0 {
		miner = subsidy * 6 / 10
		foundation = subsidy * 4 / 10
	} else {
		logging.CPrint(logging.ERROR, "the value is a not-standard input!", logging.LogFormat{
			"value": value,
		})
		return -1, -1
	}
	return miner, foundation
}

func CalcBlockSubsidy(height int32, chainParams *config.Params, value int64) (int64, int64) {
	var miner int64
	var foundation int64

	if chainParams.SubsidyHalvingInterval == 0 {
		subsidy := baseSubsidy
		mReword, fReword := calBlockSubsidy(value, miner, foundation, int64(subsidy))
		return mReword, fReword
	}

	subsidy := int64(baseSubsidy >> uint(height/chainParams.SubsidyHalvingInterval))
	mReword, fReword := calBlockSubsidy(value, miner, foundation, int64(subsidy))
	return mReword, fReword

}

// CheckTransactionSanity performs some preliminary checks on a transaction to
// ensure it is sane.  These checks are context free.
func CheckTransactionSanity(tx *massutil.Tx) error {

	msgTx := tx.MsgTx()
	if len(msgTx.TxIn) == 0 {
		return ruleError(ErrNoTxInputs, "transaction has no inputs")
	}

	if len(msgTx.TxOut) == 0 {
		return ruleError(ErrNoTxOutputs, "transaction has no outputs")
	}

	serializedTxSize := tx.MsgTx().SerializeSize()

	if serializedTxSize > wire.MaxBlockPayload {
		str := fmt.Sprintf("serialized transaction is too big - got "+
			"%d, max %d", serializedTxSize, wire.MaxBlockPayload)
		return ruleError(ErrTxTooBig, str)
	}

	var totalMaxwell int64
	for _, txOut := range msgTx.TxOut {
		maxwell := txOut.Value
		if maxwell < 0 {
			str := fmt.Sprintf("transaction output has negative "+
				"value of %v", maxwell)
			return ruleError(ErrBadTxOutValue, str)
		}
		if maxwell > massutil.MaxMaxwell {
			str := fmt.Sprintf("transaction output value of %v is "+
				"higher than max allowed value of %v", maxwell,
				massutil.MaxMaxwell)
			return ruleError(ErrBadTxOutValue, str)
		}

		totalMaxwell += maxwell
		if totalMaxwell < 0 {
			str := fmt.Sprintf("total value of all transaction "+
				"outputs exceeds max allowed value of %v",
				massutil.MaxMaxwell)
			return ruleError(ErrBadTxOutValue, str)
		}
		if totalMaxwell > massutil.MaxMaxwell {
			str := fmt.Sprintf("total value of all transaction "+
				"outputs is %v which is higher than max "+
				"allowed value of %v", totalMaxwell,
				massutil.MaxMaxwell)
			return ruleError(ErrBadTxOutValue, str)
		}
	}

	existingTxOut := make(map[wire.OutPoint]struct{})
	for _, txIn := range msgTx.TxIn {
		if _, exists := existingTxOut[txIn.PreviousOutPoint]; exists {
			return ruleError(ErrDuplicateTxInputs, "transaction "+
				"contains duplicate inputs")
		}
		existingTxOut[txIn.PreviousOutPoint] = struct{}{}
	}

	if IsCoinBase(tx) {
		slen := msgTx.TxIn[0].Witness.SerializeSize()
		if slen < MinCoinbaseScriptLen || slen > MaxCoinbaseScriptLen {
			str := fmt.Sprintf("coinbase transaction script length "+
				"of %d is out of range (min: %d, max: %d)",
				slen, MinCoinbaseScriptLen, MaxCoinbaseScriptLen)
			return ruleError(ErrBadCoinbaseScriptLen, str)
		}
	} else {
		for _, txIn := range msgTx.TxIn {
			prevOut := &txIn.PreviousOutPoint
			if isNullOutpoint(prevOut) {
				return ruleError(ErrBadTxInput, "transaction "+
					"input refers to previous output that "+
					"is null")
			}
		}
	}

	return nil
}

func checkChainID(header *wire.BlockHeader, chainID wire.Hash) error {
	if !header.ChainID.IsEqual(&chainID) {
		str := fmt.Sprintf("block's chainID of %s is not equal to %s (genesis chainID)",
			header.ChainID.String(), chainID.String())
		return ruleError(ErrChainID, str)
	}
	return nil
}

func checkHeaderTimestamp(header *wire.BlockHeader) error {
	if !header.Timestamp.Equal(time.Unix(header.Timestamp.Unix(), 0)) {
		str := fmt.Sprintf("block timestamp of %v has a higher "+
			"precision than one second", header.Timestamp)
		return ruleError(ErrInvalidTime, str)
	}

	if time.Now().Add(12 * time.Second).Before(header.Timestamp) {
		str := fmt.Sprintf("block's timestamp of %d(%s) is too far in the future",
			header.Timestamp.Unix(), header.Timestamp.Format(time.RFC3339))
		return ruleError(ErrTimeTooNew, str)
	}

	return nil
}

// CountSigOps returns the number of signature operations for all transaction
// input and output scripts in the provided transaction.  This uses the
// quicker, but imprecise, signature operation counting mechanism from
// txscript.
func CountSigOps(tx *massutil.Tx) int {
	msgTx := tx.MsgTx()
	if IsCoinBaseTx(msgTx) {
		return 0
	}
	totalSigOps := 0

	for _, txIn := range msgTx.TxIn {
		numSigOps := txscript.GetSigOpCount(txIn.Witness[len(txIn.Witness)-1])
		totalSigOps += numSigOps
	}

	for _, txOut := range msgTx.TxOut {
		numSigOps := txscript.GetSigOpCount(txOut.PkScript)
		totalSigOps += numSigOps
	}

	return totalSigOps
}

// checkBlockHeaderSanity performs some preliminary checks on a block header to
// ensure it is sane before continuing with processing.  These checks are
// context free.
func checkBlockHeaderSanity(header *wire.BlockHeader, chainID wire.Hash) (err error) {
	err = checkChainID(header, chainID)
	if err != nil {
		return
	}

	err = checkHeaderTimestamp(header)
	if err != nil {
		return
	}

	return nil
}

// Ensure the block timestamp is after the checkpoint timestamp.
func ensureCheckPointTime(blockHeader *wire.BlockHeader, checkpointBlock *massutil.Block) (bool, time.Time) {
	checkpointHeader := &checkpointBlock.MsgBlock().Header
	checkpointTime := checkpointHeader.Timestamp
	if blockHeader.Timestamp.Before(checkpointTime) {
		return false, checkpointTime
	}
	return true, checkpointTime
}

// checkBlockSanity performs some preliminary checks on a block to ensure it is
// sane before continuing with block processing.  These checks are context free.
func checkBlockSanity(block *massutil.Block, chainID wire.Hash) error {
	msgBlock := block.MsgBlock()
	header := &msgBlock.Header

	err := checkBlockHeaderSanity(header, chainID)
	if err != nil {
		logging.CPrint(logging.ERROR, "the err in checkBlockHeaderSanity", logging.LogFormat{
			"err": err,
		})
		return err
	}

	numTx := len(msgBlock.Transactions)
	if numTx == 0 {
		return ruleError(ErrNoTransactions, "block does not contain "+
			"any transactions")
	}

	if numTx > wire.MaxTxPerBlock {
		str := fmt.Sprintf("block contains too many transactions - "+
			"got %d, max %d", numTx, wire.MaxTxPerBlock)
		return ruleError(ErrTooManyTransactions, str)
	}

	serializedSize := msgBlock.SerializeSize()
	if serializedSize > wire.MaxBlockPayload {
		str := fmt.Sprintf("serialized block is too big - got %d, "+
			"max %d", serializedSize, wire.MaxBlockPayload)
		return ruleError(ErrBlockTooBig, str)
	}

	proposalMerkles := BuildMerkleTreeStoreForProposal(&block.MsgBlock().Proposals)
	calculatedProposalRoot := proposalMerkles[len(proposalMerkles)-1]
	if !header.ProposalRoot.IsEqual(calculatedProposalRoot) {
		str := fmt.Sprintf("block proposal root is invalid - block "+
			"header indicates %v, but calculated value is %v",
			header.ProposalRoot, calculatedProposalRoot)
		return ruleError(ErrBadProposalRoot, str)
	}

	transactions := block.Transactions()
	if !IsCoinBase(transactions[0]) {
		return ruleError(ErrFirstTxNotCoinbase, "first transaction in "+
			"block is not a coinbase")
	}

	for i, tx := range transactions[1:] {
		if IsCoinBase(tx) {
			str := fmt.Sprintf("block contains second coinbase at "+
				"index %d", i)
			return ruleError(ErrMultipleCoinbases, str)
		}
	}

	for _, tx := range transactions {
		err := CheckTransactionSanity(tx)
		if err != nil {
			return err
		}
	}

	merkles := BuildMerkleTreeStore(block.Transactions(), false)
	calculatedMerkleRoot := merkles[len(merkles)-1]
	if !header.TransactionRoot.IsEqual(calculatedMerkleRoot) {
		str := fmt.Sprintf("block merkle root is invalid - block "+
			"header indicates %v, but calculated value is %v",
			header.TransactionRoot, calculatedMerkleRoot)
		return ruleError(ErrBadMerkleRoot, str)
	}

	existingTxHashes := make(map[wire.Hash]struct{})
	for _, tx := range transactions {
		hash := tx.Hash()
		if _, exists := existingTxHashes[*hash]; exists {
			str := fmt.Sprintf("block contains duplicate "+
				"transaction %v", hash)
			return ruleError(ErrDuplicateTx, str)
		}
		existingTxHashes[*hash] = struct{}{}
	}

	totalSigOps := 0
	for _, tx := range transactions {
		lastSigOps := totalSigOps

		totalSigOps += CountSigOps(tx)
		if totalSigOps < lastSigOps || totalSigOps > MaxSigOpsPerBlock {
			str := fmt.Sprintf("block contains too many signature "+
				"operations - got %v, max %v", totalSigOps,
				MaxSigOpsPerBlock)
			return ruleError(ErrTooManySigOps, str)
		}
	}

	return nil
}

// checkBlockHeaderContext peforms several validation checks on the block header
// which depend on its position within the block chain.
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: All checks except those involving comparing the header against
//    the checkpoints are not performed.
func (b *BlockChain) checkBlockHeaderContext(header *wire.BlockHeader, prevNode *blockNode, flags BehaviorFlags) error {

	if prevNode == nil {
		return nil
	}

	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
		currentHeight := prevNode.height + 1

		if uint64(currentHeight) != header.Height {
			str := "block height %d of block does not match the expected height of %d"
			str = fmt.Sprintf(str, header.Height, currentHeight)
			return ruleError(ErrBadBlockHeight, str)
		}

		if header.Timestamp.Unix() <= prevNode.timestamp.Unix() {
			str := "block timestamp of %v is not after expected %v"
			str = fmt.Sprintf(str, header.Timestamp, prevNode.timestamp)
			return ruleError(ErrTimeTooOld, str)
		}
	}

	blockHeight := prevNode.height + 1

	blockHash := header.BlockHash()
	if !b.verifyCheckpoint(blockHeight, &blockHash) {
		str := fmt.Sprintf("block at height %d does not match "+
			"checkpoint hash", blockHeight)
		return ruleError(ErrBadCheckpoint, str)
	}

	checkpointBlock, err := b.findPreviousCheckpoint()
	if err != nil {
		return err
	}
	if checkpointBlock != nil && blockHeight < checkpointBlock.Height() {
		str := fmt.Sprintf("block at height %d forks the main chain "+
			"before the previous checkpoint at height %d",
			blockHeight, checkpointBlock.Height())
		return ruleError(ErrForkTooOld, str)
	}

	return nil
}

// checkBlockContext peforms several validation checks on the block which depend
// on its position within the block chain.
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: The transaction are not checked to see if they are finalized
//    and the somewhat expensive BIP0034 validation is not performed.
//
// The flags are also passed to checkBlockHeaderContext.  See its documentation
// for how the flags modify its behavior.
func (b *BlockChain) checkBlockContext(block *massutil.Block, prevNode *blockNode, flags BehaviorFlags) error {
	if prevNode == nil {
		return nil
	}

	header := &block.MsgBlock().Header
	err := b.checkBlockHeaderContext(header, prevNode, flags)
	if err != nil {
		return err
	}

	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
		blockHeight := prevNode.height + 1

		blockTime, err := b.calcPastMedianTime(prevNode)
		if err != nil {
			return err
		}

		for _, tx := range block.Transactions() {
			if !IsFinalizedTransaction(tx, blockHeight,
				blockTime) {

				str := fmt.Sprintf("block contains unfinalized "+
					"transaction %v", tx.Hash())
				return ruleError(ErrUnfinalizedTx, str)

			}
		}

		coinbaseTx := block.Transactions()[0]
		err = checkSerializedHeight(coinbaseTx, blockHeight)
		if err != nil {
			return err
		}

		if err := ValidateWitnessCommitment(block); err != nil {
			return err
		}

		blockSize := block.MsgBlock().SerializeSize()
		if blockSize > wire.MaxBlockPayload {
			str := fmt.Sprintf("block's weight metric is "+
				"too high - got %v, max %v",
				blockSize, wire.MaxBlockPayload)
			return ruleError(ErrBlockSizeTooHigh, str)
		}

		err1 := CheckCoinbaseHeight(block)
		if err1 != nil {
			return err1
		}
	}

	return nil
}

// CheckCoinbaseHeight checks whether block height in coinbase matches block
// height in header. We do not check *block's existence because this func
// is called in another func that *block exists.
func CheckCoinbaseHeight(block *massutil.Block) error {
	coinbaseTx := block.Transactions()[0]
	blockHeight := block.MsgBlock().Header.Height
	err := checkSerializedHeight(coinbaseTx, int32(blockHeight))
	if err != nil {
		return err
	}
	return nil
}

// ExtractCoinbaseHeight attempts to extract the height of the block from the
// scriptSig of a coinbase transaction.  Coinbase heights are only present in
// blocks of version 2 or later.  This was added as part of BIP0034.
func ExtractCoinbaseHeight(coinbaseTx *massutil.Tx) (int32, error) {
	payload := coinbaseTx.MsgTx().Payload
	if len(payload) < 1 {
		str := "the coinbase payload for blocks of " +
			"version %d or greater must start with the " +
			"length of the serialized block height, tag 1."
		str = fmt.Sprintf(str, serializedHeightVersion)
		return 0, ruleError(ErrMissingCoinbaseHeight, str)
	}

	opcode := int(payload[0])
	if opcode == txscript.OP_0 {
		return 0, nil
	}
	if opcode >= txscript.OP_1 && opcode <= txscript.OP_16 {
		return int32(opcode - (txscript.OP_1 - 1)), nil
	}

	serializedLen := int(payload[0])
	if len(payload[1:]) < serializedLen {
		str := "the coinbase signature script for blocks of " +
			"version %d or greater must start with the " +
			"serialized block height, tag 2."
		str = fmt.Sprintf(str, serializedLen)
		return 0, ruleError(ErrMissingCoinbaseHeight, str)
	}

	serializedHeightBytes := make([]byte, 8, 8)
	copy(serializedHeightBytes, payload[1:serializedLen+1])
	serializedHeight := binary.LittleEndian.Uint64(serializedHeightBytes)

	return int32(serializedHeight), nil
}

// checkSerializedHeight checks if the signature script in the passed
// transaction starts with the serialized block height of wantHeight.
func checkSerializedHeight(coinbaseTx *massutil.Tx, wantHeight int32) error {
	serializedHeight, err := ExtractCoinbaseHeight(coinbaseTx)
	if err != nil {
		return err
	}

	if serializedHeight != wantHeight {
		str := fmt.Sprintf("the coinbase signature script serialized "+
			"block height is %d when %d was expected",
			serializedHeight, wantHeight)
		return ruleError(ErrBadCoinbaseHeight, str)
	}
	return nil
}

// isTransactionSpent returns whether or not the provided transaction data
// describes a fully spent transaction.  A fully spent transaction is one where
// all outputs have been spent.
func isTransactionSpent(txD *TxData) bool {
	for _, isOutputSpent := range txD.Spent {
		if !isOutputSpent {
			return false
		}
	}
	return true
}

// checkDupTx ensures blocks do not contain duplicate transactions which
// 'overwrite' older transactions that are not fully spent.  This prevents an
// attack where a coinbase and all of its dependent transactions could be
// duplicated to effectively revert the overwritten transactions to a single
// confirmation thereby making them vulnerable to a double spend.
func (b *BlockChain) checkDupTx(node *blockNode, block *massutil.Block) error {
	fetchSet := make(map[wire.Hash]struct{})
	for _, tx := range block.Transactions() {
		fetchSet[*tx.Hash()] = struct{}{}
	}
	txResults, err := b.fetchTxStore(node, fetchSet)
	if err != nil {
		return err
	}

	for _, txD := range txResults {
		switch txD.Err {
		case database.ErrTxShaMissing:
			continue

		case nil:
			if !isTransactionSpent(txD) {
				str := fmt.Sprintf("tried to overwrite "+
					"transaction %v at block height %d "+
					"that is not fully spent", txD.Hash,
					txD.BlockHeight)
				return ruleError(ErrOverwriteTx, str)
			}

		default:
			return txD.Err
		}
	}

	return nil
}

// CheckTransactionInputs performs a series of checks on the inputs to a
// transaction to ensure they are valid.
func CheckTransactionInputs(tx *massutil.Tx, txHeight int32, txStore TxStore) (int64, error) {
	if IsCoinBase(tx) {
		return 0, nil
	}
	txHash := tx.Hash()
	var totalMaxwellIn int64
	for _, txIn := range tx.MsgTx().TxIn {
		// Ensure the input is available.
		txInHash := &txIn.PreviousOutPoint.Hash
		originTx, exists := txStore[*txInHash]
		if !exists || originTx.Err != nil || originTx.Tx == nil {
			str := fmt.Sprintf("unable to find input transaction "+
				"%v for transaction %v", txInHash, txHash)
			return 0, ruleError(ErrMissingTx, str)
		}

		if IsCoinBase(originTx.Tx) {
			originHeight := originTx.BlockHeight
			blocksSincePrev := txHeight - originHeight
			if blocksSincePrev < coinbaseMaturity {
				str := fmt.Sprintf("tried to spend coinbase "+
					"transaction %v from height %v at "+
					"height %v before required maturity "+
					"of %v blocks", txInHash, originHeight,
					txHeight, coinbaseMaturity)
				return 0, ruleError(ErrImmatureSpend, str)
			}
		}

		originTxIndex := txIn.PreviousOutPoint.Index
		if originTxIndex >= uint32(len(originTx.Spent)) {
			str := fmt.Sprintf("out of bounds input index %d in "+
				"transaction %v referenced from transaction %v",
				originTxIndex, txInHash, txHash)
			return 0, ruleError(ErrBadTxInput, str)
		}
		if originTx.Spent[originTxIndex] {
			str := fmt.Sprintf("transaction %v tried to double "+
				"spend output %v", txHash, txIn.PreviousOutPoint)
			return 0, ruleError(ErrDoubleSpend, str)
		}

		originTxMaxwell := originTx.Tx.MsgTx().TxOut[originTxIndex].Value
		if originTxMaxwell < 0 {
			str := fmt.Sprintf("transaction output has negative "+
				"value of %v", originTxMaxwell)
			return 0, ruleError(ErrBadTxOutValue, str)
		}
		if originTxMaxwell > massutil.MaxMaxwell {
			str := fmt.Sprintf("transaction output value of %v is "+
				"higher than max allowed value of %v",
				originTxMaxwell, massutil.MaxMaxwell)
			return 0, ruleError(ErrBadTxOutValue, str)
		}

		lastMaxwellIn := totalMaxwellIn
		totalMaxwellIn += originTxMaxwell
		if totalMaxwellIn < lastMaxwellIn ||
			totalMaxwellIn > massutil.MaxMaxwell {
			str := fmt.Sprintf("total value of all transaction "+
				"inputs is %v which is higher than max "+
				"allowed value of %v", totalMaxwellIn,
				massutil.MaxMaxwell)
			return 0, ruleError(ErrBadTxOutValue, str)
		}

	}

	var totalMaxwellOut int64
	for _, txOut := range tx.MsgTx().TxOut {
		totalMaxwellOut += txOut.Value
	}

	if totalMaxwellIn < totalMaxwellOut {
		str := fmt.Sprintf("total value of all transaction inputs for "+
			"transaction %v is %v which is less than the amount "+
			"spent of %v", txHash, totalMaxwellIn, totalMaxwellOut)
		return 0, ruleError(ErrSpendTooHigh, str)
	}

	txFeeInMaxwell := totalMaxwellIn - totalMaxwellOut
	return txFeeInMaxwell, nil
}

// checkConnectBlock performs several checks to confirm connecting the passed
// block to the main chain (including whatever reorganization might be necessary
// to get this node to the main chain) does not violate any rules.
//
// The CheckConnectBlock function makes use of this function to perform the
// bulk of its work.  The only difference is this function accepts a node which
// may or may not require reorganization to connect it to the main chain whereas
// CheckConnectBlock creates a new node which specifically connects to the end
// of the current main chain and then calls this function with that node.
//
// See the comments for CheckConnectBlock for some examples of the type of
// checks performed by this function.
func (b *BlockChain) checkConnectBlock(node *blockNode, block *massutil.Block) error {
	if node.hash.IsEqual(config.ChainParams.GenesisHash) && b.bestChain == nil {
		return nil
	}

	err := b.checkDupTx(node, block)
	if err != nil {
		return err
	}

	txInputStore, err := b.fetchInputTransactions(node, block)
	if err != nil {
		return err
	}

	transactions := block.Transactions()
	totalSigOps := 0
	for _, tx := range transactions {
		numsigOps := CountSigOps(tx)
		lastSigops := totalSigOps
		totalSigOps += numsigOps
		if totalSigOps < lastSigops || totalSigOps > MaxSigOpsPerBlock {
			str := fmt.Sprintf("block contains too many "+
				"signature operations - got %v, max %v",
				totalSigOps, MaxSigOpsPerBlock)
			return ruleError(ErrTooManySigOps, str)
		}
	}

	var totalFees int64
	for _, tx := range transactions {
		txFee, err := CheckTransactionInputs(tx, node.height, txInputStore)
		if err != nil {
			return err
		}

		lastTotalFees := totalFees
		totalFees += txFee
		if totalFees < lastTotalFees {
			return ruleError(ErrBadFees, "total fees for block "+
				"overflows accumulator")
		}
	}

	var totalMaxwellOut int64
	for _, txOut := range transactions[0].MsgTx().TxOut {
		totalMaxwellOut += txOut.Value
	}

	headPubkey := block.MsgBlock().Header.PubKey
	if headPubkey != nil && !reflect.DeepEqual(headPubkey, wire.NewEmptyPoCPublicKey()) {
		coinbaseValidate, reward, err := checkCoinbase(transactions[0], b.db, headPubkey, node.height, &config.ChainParams)
		if coinbaseValidate != true || err != nil {
			return err
		}
		expectedMaxwellOut := reward + totalFees

		if totalMaxwellOut > expectedMaxwellOut {
			str := fmt.Sprintf("coinbase transaction for block pays %v "+
				"which is more than expected value of %v",
				totalMaxwellOut, expectedMaxwellOut)
			return ruleError(ErrBadCoinbaseValue, str)
		}
	}

	checkpoint := b.LatestCheckpoint()
	runScripts := !b.noVerify
	if checkpoint != nil && uint64(node.height) <= checkpoint.Height {
		runScripts = false
	}

	prevNode, err := b.getPrevNodeFromNode(node)
	if err != nil {
		logging.CPrint(logging.ERROR, "getPrevNodeFromNode", logging.LogFormat{
			"err": err,
		})
		return err
	}

	var scriptFlags txscript.ScriptFlags

	blockHeader := &block.MsgBlock().Header
	if blockHeader.Version >= 3 && b.isMajorityVersion(3, prevNode,
		config.ChainParams.BlockEnforceNumRequired) {

		scriptFlags |= txscript.ScriptVerifyDERSignatures
	}

	if blockHeader.Version >= 4 && b.isMajorityVersion(4, prevNode,
		config.ChainParams.BlockEnforceNumRequired) {

		scriptFlags |= txscript.ScriptVerifyCheckLockTimeVerify
	}

	scriptFlags |= txscript.ScriptVerifyCheckSequenceVerify

	medianTime, err := b.CalcPastMedianTime()

	for _, tx := range block.Transactions() {
		sequenceLock, err := b.calcSequenceLock(node, tx, txInputStore)
		if err != nil {
			return err
		}
		if !SequenceLockActive(sequenceLock, node.height,
			medianTime) {
			str := fmt.Sprintf("block contains " +
				"transaction whose input sequence " +
				"locks are not met")
			return ruleError(ErrUnfinalizedTx, str)
		}
	}

	if runScripts {
		err := checkBlockScripts(block, txInputStore, scriptFlags, b.sigCache, b.hashCache)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckConnectBlock performs several checks to confirm connecting the passed
// block to the main chain does not violate any rules.  An example of some of
// the checks performed are ensuring connecting the block would not cause any
// duplicate transaction hashes for old transactions that aren't already fully
// spent, double spends, exceeding the maximum allowed signature operations
// per block, invalid values in relation to the expected block subsidy, or fail
// transaction script validation.
//
// This function is NOT safe for concurrent access.
func (b *BlockChain) CheckConnectBlock(block *massutil.Block) error {
	prevNode := b.bestChain
	h256 := sha256.New()
	h256.Write(bytesCombine(block.MsgBlock().Header.Previous[:], block.MsgBlock().Header.Challenge.Bytes()))
	blkSha := h256.Sum(nil)
	sha := wire.Hash{}
	for i := range sha {
		sha[i] = blkSha[i]
	}
	newNode := newBlockNode(&block.MsgBlock().Header, &sha, block.Height())
	if prevNode != nil {
		newNode.parent = prevNode
		newNode.capSum.Add(prevNode.capSum, newNode.capSum)
	}

	return b.checkConnectBlock(newNode, block)
}

func bytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}
