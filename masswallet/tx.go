package masswallet

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"massnet.org/mass-wallet/masswallet/keystore"

	pb "massnet.org/mass-wallet/api/proto"

	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"

	"errors"
	"sort"

	"massnet.org/mass-wallet/masswallet/ifc"
	"massnet.org/mass-wallet/masswallet/txmgr"
)

const (
	TxHistoryMax = 500
	BlockBatch   = 500
)

type TxIn struct {
	TxId string
	Vout uint32
}

type StakingTxOut struct {
	Address      string
	FrozenPeriod uint32
	Amount       massutil.Amount
}

type BindingOutput struct {
	HolderAddress  string
	BindingAddress string
	Amount         massutil.Amount
}

func (w *WalletManager) constructTxIn(Inputs []*TxIn, LockTime uint64) (*wire.MsgTx, error) {
	am := w.ksmgr.CurrentKeystore()
	if am == nil {
		return nil, ErrNoWalletInUse
	}
	mtxReturn := &wire.MsgTx{}
	for _, input := range Inputs {
		txHash, err := wire.NewHashFromStr(input.TxId)
		if err != nil {
			logging.CPrint(logging.ERROR, "exist err in constructTxin", logging.LogFormat{"err": err.Error()})
			return nil, ErrShaHashFromStr
		}

		prevOut := wire.NewOutPoint(txHash, input.Vout)
		txIn := wire.NewTxIn(prevOut, nil)
		if LockTime != 0 {
			txIn.Sequence = wire.MaxTxInSequenceNum - 1
		}
		mtx, err := w.existsMsgTx(&txIn.PreviousOutPoint)
		if err == txmgr.ErrNotFound {
			logging.CPrint(logging.INFO, "mined prev tx not found, check unmined tx", logging.LogFormat{})
			mtx, err = w.existsUnminedTx(&txIn.PreviousOutPoint.Hash)
		}
		if err != nil {
			logging.CPrint(logging.ERROR, "check prev tx failed", logging.LogFormat{"err": err})
			return nil, ErrInvalidParameter
		}

		txOut := mtx.TxOut[txIn.PreviousOutPoint.Index]
		pks, err := utils.ParsePkScript(txOut.PkScript, w.chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "ParsePkScript error",
				logging.LogFormat{
					"err": err,
				})
			return nil, ErrInvalidParameter
		}
		_, err = am.Address(pks.StdEncodeAddress())
		if err != nil {
			return nil, ErrNoAddressInWallet
		}
		if pks.IsStaking() {
			txIn.Sequence = pks.Maturity()
		}

		mtxReturn.AddTxIn(txIn)
	}
	return mtxReturn, nil
}

func constructTxOut(amounts map[string]massutil.Amount, mtx *wire.MsgTx) (*wire.MsgTx, error) {
	// Add all transaction outputs to the transaction after performing
	// some validity checks.
	for encodedAddr, amount := range amounts {
		// Ensure amount is in the valid range for monetary amounts.
		if amount.IsZero() || amount.Cmp(massutil.MaxAmount()) > 0 {
			return nil, ErrInvalidAmount
		}

		// Decode the provided wallet.
		addr, err := massutil.DecodeAddress(encodedAddr, &config.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to decode address", logging.LogFormat{
				"err: ": err,
			})
			return nil, ErrFailedDecodeAddress
		}

		// Ensure the wallet is one of the supported types and that
		// the network encoded with the wallet matches the network the
		// server is currently on.
		if !massutil.IsWitnessV0Address(addr) {
			logging.CPrint(logging.ERROR, "Invalid address or key", logging.LogFormat{"address": encodedAddr})
			return nil, ErrInvalidAddress
		}
		if !addr.IsForNet(&config.ChainParams) {
			return nil, ErrNet
		}

		// create a new script which pays to the provided wallet.
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to create pkScript", logging.LogFormat{})
			return nil, ErrCreatePkScript
		}

		txOut := wire.NewTxOut(amount.IntValue(), pkScript)
		mtx.AddTxOut(txOut)
	}
	return mtx, nil
}

func constructStakingTxOut(outputs []*StakingTxOut, mtx *wire.MsgTx) error {
	// Add all transaction outputs to the transaction after performing
	// some validity checks.
	for _, output := range outputs {
		// Ensure amount is in the valid range for monetary amounts.
		if output.Amount.IsZero() || output.Amount.Cmp(massutil.MaxAmount()) > 0 {
			return ErrInvalidAmount
		}

		// Ensure the wallet is one of the supported types and that
		// the network encoded with the wallet matches the network the
		// server is currently on.
		addr, err := massutil.DecodeAddress(output.Address, &config.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to decode address", logging.LogFormat{
				"err: ":   err,
				"address": output.Address,
			})
			return ErrFailedDecodeAddress
		}
		if !addr.IsForNet(&config.ChainParams) {
			return ErrNet
		}

		if !massutil.IsWitnessStakingAddress(addr) {
			logging.CPrint(logging.ERROR, "Invalid lock address", logging.LogFormat{"address": output.Address})
			return ErrInvalidStakingAddress
		}

		pkScript, err := txscript.PayToStakingAddrScript(addr, uint64(output.FrozenPeriod))
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to create pkScript", logging.LogFormat{
				"err: ":   err,
				"address": output.Address,
			})
			return ErrCreatePkScript
		}

		txOut := wire.NewTxOut(output.Amount.IntValue(), pkScript)
		mtx.AddTxOut(txOut)
	}
	return nil
}

func constructBindingTxOut(outputs []*BindingOutput, mtx *wire.MsgTx) error {
	// Add all transaction outputs to the transaction after performing
	// some validity checks.
	for _, output := range outputs {
		// Ensure amount is in the valid range for monetary amounts.
		if output.Amount.IsZero() || output.Amount.Cmp(massutil.MaxAmount()) > 0 {
			return ErrInvalidAmount
		}

		// Decode the provided wallet.
		HolderAddress, err := massutil.DecodeAddress(output.HolderAddress, &config.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to decode address", logging.LogFormat{
				"err: ":   err,
				"address": output.HolderAddress,
			})
			return ErrFailedDecodeAddress
		}
		BindingAddress, err := massutil.DecodeAddress(output.BindingAddress, &config.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to decode address", logging.LogFormat{
				"err: ":   err,
				"address": output.BindingAddress,
			})
			return ErrFailedDecodeAddress
		}
		// Ensure the wallet is one of the supported types and that
		// the network encoded with the wallet matches the network the
		// server is currently on.
		if !massutil.IsWitnessV0Address(HolderAddress) {
			logging.CPrint(logging.ERROR, "Invalid HolderAddress or key", logging.LogFormat{"address": HolderAddress.String()})
			return ErrInvalidAddress
		}
		if !HolderAddress.IsForNet(&config.ChainParams) {
			return ErrNet
		}

		if !massutil.IsAddressPubKeyHash(BindingAddress) {
			logging.CPrint(logging.ERROR, "Invalid BindingAddress or key", logging.LogFormat{"address": BindingAddress.String()})
			return ErrInvalidAddress
		}
		if !BindingAddress.IsForNet(&config.ChainParams) {
			return ErrNet
		}
		// Create a new script which pays to the provided wallet.
		pkScript, err := txscript.PayToBindingScriptHashScript(HolderAddress.ScriptAddress(), BindingAddress.ScriptAddress())
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to create pkScript", logging.LogFormat{})
			return ErrCreatePkScript
		}

		txOut := wire.NewTxOut(output.Amount.IntValue(), pkScript)
		mtx.AddTxOut(txOut)

	}
	return nil
}

// messageToHex serializes a message to the wire protocol encoding using the
// latest protocol version and returns a hex-encoded string of the result.
func messageToHex(msg wire.Message) (string, error) {
	var buf bytes.Buffer
	if _, err := msg.Encode(&buf, wire.Packet); err != nil {
		return "", ErrEncode
	}

	return hex.EncodeToString(buf.Bytes()), nil
}

func (w *WalletManager) EstimateTxFee(Amounts map[string]massutil.Amount, LockTime uint64, userTxFee massutil.Amount, from string) (
	msgTx *wire.MsgTx, fee massutil.Amount, err error) {

	addrs, err := w.prepareFromAddresses(from)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}

	msgTx = wire.NewMsgTx()
	// construct output
	for encodedAddr, amount := range Amounts {
		txOut, err := amountToTxOut(encodedAddr, amount)
		if err != nil {
			return nil, massutil.ZeroAmount(), err
		}
		msgTx.AddTxOut(txOut)
	}
	u, err := w.autoConstructTxInAndChangeTxOut(msgTx, LockTime, addrs, userTxFee, defaultChangeAddress)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}
	return msgTx, u, err
}

func (w *WalletManager) EstimateStakingTxFee(outputs []*StakingTxOut, LockTime uint64, userTxFee massutil.Amount,
	fromAddr, changeAddr string) (msgTx *wire.MsgTx, fee massutil.Amount, err error) {

	addrs, err := w.prepareFromAddresses(fromAddr)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}

	msgTx = wire.NewMsgTx()
	err = constructStakingTxOut(outputs, msgTx)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}

	u, err := w.autoConstructTxInAndChangeTxOut(msgTx, LockTime, addrs, userTxFee, changeAddr)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}
	return msgTx, u, err
}

func (w *WalletManager) EstimateBindingTxFee(outputs []*BindingOutput, LockTime uint64, userTxFee massutil.Amount,
	fromAddr, changeAddr string) (msgTx *wire.MsgTx, fee massutil.Amount, err error) {

	addrs, err := w.prepareFromAddresses(fromAddr)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}

	msgTx = wire.NewMsgTx()
	err = constructBindingTxOut(outputs, msgTx)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}

	u, err := w.autoConstructTxInAndChangeTxOut(msgTx, LockTime, addrs, userTxFee, changeAddr)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}
	return msgTx, u, err
}

func amountToTxOut(encodedAddr string, amount massutil.Amount) (*wire.TxOut, error) {
	if amount.IsZero() {
		logging.CPrint(logging.ERROR, "invalid output amount",
			logging.LogFormat{
				"err": ErrInvalidAmount,
			})
		return nil, ErrInvalidAmount
	}

	// Decode the provided wallet.
	addr, err := massutil.DecodeAddress(encodedAddr, &config.ChainParams)
	if err != nil {
		logging.CPrint(logging.ERROR, "decode address failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	if !addr.IsForNet(&config.ChainParams) {
		logging.CPrint(logging.ERROR, "address is not in the net",
			logging.LogFormat{
				"err":     ErrNet,
				"address": encodedAddr,
			})
		return nil, ErrNet
	}

	// Ensure the wallet is one of the supported types and that
	// the network encoded with the wallet matches the network the
	// server is currently on.
	if !massutil.IsWitnessV0Address(addr) {
		logging.CPrint(logging.ERROR, "Invalid address or key", logging.LogFormat{"address": encodedAddr})
		return nil, ErrInvalidAddress
	}

	// create a new script which pays to the provided wallet.
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to create pkScript",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	txOut := wire.NewTxOut(amount.IntValue(), pkScript)

	return txOut, nil
}

//
func (w *WalletManager) estimateSignedSize(utxos []*txmgr.Credit, TxOutLen int) (int64, error) {
	signedSize := 0
	for _, utx := range utxos {
		txidx := utx.OutPoint.Index
		outPoint := &utx.OutPoint
		txIn := wire.NewTxIn(outPoint, nil)

		mtx, err := w.existsMsgTx(&txIn.PreviousOutPoint)
		if err != nil {
			return 0, err
		}

		pkScript := mtx.TxOut[txidx].PkScript
		_, addrs, _, _, err := txscript.ExtractPkScriptAddrs(pkScript, &config.ChainParams)
		if err != nil {
			return 0, err
		}
		// get redeemScript
		addr := addrs[0]
		if massutil.IsWitnessStakingAddress(addr) {
			addr, _ = massutil.NewAddressWitnessScriptHash(addr.ScriptAddress(), w.chainParams)
		}
		acctM, err := w.ksmgr.GetAddrManager(addr.String())
		if err != nil {
			return 0, err
		}
		mAddr, err := acctM.Address(addr.String())
		if err != nil {
			return 0, err
		}
		script, err := mAddr.RedeemScript(w.chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "create witnessScriptAddress failed",
				logging.LogFormat{
					"error": err,
				})
			return 0, err
		}

		class, _, _, nrequired, err := txscript.ExtractPkScriptAddrs(script,
			&config.ChainParams)
		if err != nil || class != txscript.MultiSigTy {
			return 0, err
		}
		//a signature size is 73 at most,Sequence + PreviousOut
		signedSize = signedSize + len(script) + 73*nrequired + 8 + 32 + 4
	}

	//(Value 8 bytes +  PkScript 55 bytes at most)*N + lockTime(8 byte) + version(4 byte).
	// See standard_test.go
	return int64(signedSize + 63*TxOutLen + 12), nil
}

func (w *WalletManager) findEligibleUtxos(amount massutil.Amount, witnessAddr []string) (
	[]*txmgr.Credit, string, massutil.Amount, bool, error) {
	zeroAmount := massutil.ZeroAmount()
	if len(witnessAddr) == 0 {
		logging.CPrint(logging.ERROR, "inputParams can not be nil",
			logging.LogFormat{
				"err": ErrInvalidParameter,
			})
		return nil, "", zeroAmount, false, ErrInvalidParameter
	}
	if amount.IsZero() {
		logging.CPrint(logging.ERROR, "amount must be greater than 0",
			logging.LogFormat{
				"err": ErrInvalidParameter,
			})
		return nil, "", zeroAmount, false, ErrInvalidParameter
	}

	utxos, overfull, err := w.getUtxosExcludeBindingAndStaking(witnessAddr, amount)
	if err != nil {
		logging.CPrint(logging.ERROR, "get utxos failed", logging.LogFormat{"err": err})
		return nil, "", zeroAmount, false, err
	}

	selections, sumSelection, _, err := optOutputs(amount, utxos)
	if err != nil {
		return nil, "", zeroAmount, false, err
	}
	firstAddr := ""
	if len(selections) > 0 {
		am := w.ksmgr.CurrentKeystore()
		for _, addr := range witnessAddr {
			ma, _ := am.Address(addr)
			if bytes.Equal(ma.ScriptAddress(), selections[0].ScriptHash) {
				firstAddr = addr
				break
			}
		}
		if len(firstAddr) == 0 {
			logging.CPrint(logging.ERROR, "unexpected error: change address not found", logging.LogFormat{
				"wallet":    am.Name(),
				"first":     selections[0].ScriptHash,
				"addresses": witnessAddr,
			})
			return nil, "", zeroAmount, false, fmt.Errorf("change address not found")
		}
	}

	return selections, firstAddr, sumSelection, overfull && len(utxos) == len(selections), nil
}

func (w *WalletManager) getUtxos(addrs []string) (map[string][]*txmgr.Credit, []*txmgr.Credit, error) {
	am := w.ksmgr.CurrentKeystore()
	if am == nil {
		return nil, nil, ErrNoWalletInUse
	}
	scriptToAddrs := make(map[string][]string)
	scriptSet := make(map[string]struct{})
	for _, addr := range addrs {
		ma, err := am.Address(addr)
		if err != nil {
			return nil, nil, err
		}

		addrs, ok := scriptToAddrs[string(ma.ScriptAddress())]
		if !ok {
			addrs = make([]string, 0)
		}
		scriptToAddrs[string(ma.ScriptAddress())] = append(addrs, addr)
		scriptSet[string(ma.ScriptAddress())] = struct{}{}
	}

	ret := make(map[string][]*txmgr.Credit)
	retList := make([]*txmgr.Credit, 0)
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		syncedTo, err := w.syncStore.SyncedTo(tx)
		if err != nil {
			return err
		}
		m, err := w.utxoStore.ScriptAddressUnspents(tx, scriptSet, syncedTo.Height,
			func(item *txmgr.Credit) (bool, bool) {
				if !item.Flags.SpentByUnmined && !item.Flags.Spent {
					return false, true
				}
				return false, false
			})
		if err != nil {
			return err
		}
		for script, credits := range m {
			unspents := make([]*txmgr.Credit, 0)
			for _, credit := range credits {
				// if !credit.Flags.SpentByUnmined && !credit.Flags.Spent {
				unspents = append(unspents, credit)
				retList = append(retList, credit)
				// }
			}
			for _, addr := range scriptToAddrs[script] {
				ret[addr] = unspents
			}
			//ret[addr] = unspents
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return ret, retList, nil
}

func (w *WalletManager) getUtxosExcludeBindingAndStaking(stdAddresses []string,
	wantAmt massutil.Amount) ([]*txmgr.Credit, bool, error) {

	am := w.ksmgr.CurrentKeystore()
	if am == nil {
		return nil, false, ErrNoWalletInUse
	}

	scriptSet := make(map[string]struct{})
	for _, addr := range stdAddresses {
		ma, err := am.Address(addr)
		if err != nil {
			return nil, false, err
		}
		scriptSet[string(ma.ScriptAddress())] = struct{}{}
	}

	selector := newTopKSelector(wantAmt)
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		syncedTo, err := w.syncStore.SyncedTo(tx)
		if err != nil {
			return err
		}
		_, err = w.utxoStore.ScriptAddressUnspents(tx, scriptSet, syncedTo.Height,
			func(item *txmgr.Credit) (stopIter, selected bool) {
				if item.Confirmations >= item.Maturity && !item.Flags.SpentByUnmined && !item.Flags.Spent &&
					item.Flags.Class != txmgr.ClassBindingUtxo && item.Flags.Class != txmgr.ClassStakingUtxo &&
					!w.UTXOUsed(&item.OutPoint) && !w.server.TxMemPool().CheckPoolOutPointSpend(&item.OutPoint) {
					selector.submit(item)
				}
				return
			})
		return err
	})
	if err != nil {
		return nil, false, err
	}
	items := selector.Items()
	return items, len(items) == selector.K(), nil
}

func optOutputs(amount massutil.Amount, utxos []*txmgr.Credit) ([]*txmgr.Credit, massutil.Amount, massutil.Amount, error) {
	zeroAmount := massutil.ZeroAmount()
	//sort the utxo by amount, bigger amount in front
	var (
		optAmount    = massutil.ZeroAmount()
		sumSelection = massutil.ZeroAmount()
		sumReserve   = massutil.ZeroAmount()
		selections   = make([]*txmgr.Credit, 0)
	)
	if amount.IsZero() {
		return nil, zeroAmount, zeroAmount, nil
	}

	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Amount.Cmp(utxos[j].Amount) > 0
	})

	unSelectedUtIndex := make([]int, 0)
	selectedUtIndex := make([]int, 0)

	var err error
	for index, u11 := range utxos {
		sumReserve, err = sumReserve.Add(u11.Amount)
		if err != nil {
			return nil, zeroAmount, zeroAmount, err
		}
		optAmount, err = optAmount.Add(u11.Amount)
		if err != nil {
			return nil, zeroAmount, zeroAmount, err
		}
		if index == len(utxos)-1 {
			selectedUtIndex = append(selectedUtIndex, index)
			if optAmount.Cmp(amount) < 0 {
				if len(unSelectedUtIndex) > 0 {
					lastUnselIdx := unSelectedUtIndex[len(unSelectedUtIndex)-1]
					tmp := make([]int, 0)
					for _, idx := range selectedUtIndex {
						if idx < lastUnselIdx {
							tmp = append(tmp, idx)
						}
					}
					tmp = append(tmp, lastUnselIdx)
					selectedUtIndex = tmp
				}
			}
			break
		}

		if optAmount.Cmp(amount) > 0 {
			unSelectedUtIndex = append(unSelectedUtIndex, index)
			optAmount, _ = optAmount.Sub(u11.Amount)
			continue
		}
		selectedUtIndex = append(selectedUtIndex, index)
		if optAmount.Cmp(amount) == 0 {
			break
		}
	}

	for _, index := range selectedUtIndex {
		sumSelection, err = sumSelection.Add(utxos[index].Amount)
		if err != nil {
			return nil, zeroAmount, zeroAmount, err
		}
		selections = append(selections, utxos[index])
	}

	return selections, sumSelection, sumReserve, nil
}

func (w *WalletManager) SignHash(pub *btcec.PublicKey, hash, password []byte) (*btcec.Signature, error) {
	return w.ksmgr.SignHash(pub, hash, password)
}

func (w *WalletManager) signWitnessTx(password []byte, tx *wire.MsgTx, hashType txscript.SigHashType, params *config.Params) error {

	hashCache := txscript.NewTxSigHashes(tx)

	var err error

	getSign := txscript.SignClosure(func(pub *btcec.PublicKey, hash []byte) (*btcec.Signature, error) {
		sig, err := w.ksmgr.SignHash(pub, hash, password)
		if err != nil {
			return nil, err
		}
		return sig, nil
	})

	getScript := txscript.ScriptClosure(func(addr massutil.Address) ([]byte, error) {
		scriptHash := addr.ScriptAddress()

		address, err := massutil.NewAddressWitnessScriptHash(scriptHash, w.chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "ScriptClosure error", logging.LogFormat{"err": err})
			return nil, keystore.ErrBuildWitnessScript
		}
		addrStr := address.EncodeAddress()

		acctM := w.ksmgr.CurrentKeystore()
		mAddr, err := acctM.Address(addrStr)
		if err != nil {
			logging.CPrint(logging.ERROR, "ScriptClosure error", logging.LogFormat{"err": err})
			return nil, keystore.ErrUnexpectedPubKeyToSign
		}
		script, err := mAddr.RedeemScript(w.chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "ScriptClosure error", logging.LogFormat{"err": err})
			return nil, keystore.ErrBuildWitnessScript
		}
		return script, nil
	})

	cache := make(map[wire.Hash]*wire.MsgTx)
	for i, txIn := range tx.TxIn {
		prevTx, ok := cache[txIn.PreviousOutPoint.Hash]
		if !ok {
			prevTx, err = w.existsMsgTx(&txIn.PreviousOutPoint)
			if err == txmgr.ErrNotFound {
				prevTx, err = w.existsUnminedTx(&txIn.PreviousOutPoint.Hash)
			}

			if err != nil {
				logging.CPrint(logging.ERROR, "failed to check previous transaction", logging.LogFormat{
					"err": err,
				})
				return ErrUTXONotExists
			}
			cache[txIn.PreviousOutPoint.Hash] = prevTx
		}
		// check index
		if txIn.PreviousOutPoint.Index > uint32(len(prevTx.TxOut)-1) {
			logging.CPrint(logging.ERROR, "Ouput index number (vout) does not exist for transaction", logging.LogFormat{
				"index": txIn.PreviousOutPoint.Index,
			})
			return ErrInvalidIndex
		}

		flags, err := w.existsOutPoint(&txIn.PreviousOutPoint)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to check previous output", logging.LogFormat{
				"err": err,
			})
			return ErrUTXONotExists
		}
		if flags.Spent {
			logging.CPrint(logging.ERROR, "Ouput index for txid has been spent", logging.LogFormat{
				"index": txIn.PreviousOutPoint.Index,
				"txid":  &txIn.PreviousOutPoint.Hash,
			})
			return ErrDoubleSpend
		}

		prevTxOut := prevTx.TxOut[txIn.PreviousOutPoint.Index]

		// SigHashSingle inputs can only be signed if there's a
		// corresponding output. However this could be already signed,
		// so we always verify the output.
		if (hashType&txscript.SigHashSingle) !=
			txscript.SigHashSingle || i < len(tx.TxOut) {

			script, err := txscript.SignTxOutputWit(params, tx, i, prevTxOut.Value, prevTxOut.PkScript, hashCache, hashType, getSign, getScript)

			// Failure to sign isn't an error, it just means that
			// the tx isn't complete.
			if err != nil {
				logging.CPrint(logging.ERROR, "Err in txscript.SignTxOutputWit", logging.LogFormat{
					"err": err,
				})
				return err
			}
			txIn.Witness = script
		}

		// Either it was already signed or we just signed it.
		// Find out if it is completely satisfied or still needs more.
		vm, err := txscript.NewEngine(prevTxOut.PkScript, tx, i,
			txscript.StandardVerifyFlags, nil, hashCache, prevTxOut.Value)
		if err == nil {
			err = vm.Execute()
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
	}
	w.ksmgr.ClearPrivKey()

	return nil
}

func (w *WalletManager) EstimateManualTxFee(txins []*TxIn, txoutLen int) (massutil.Amount, error) {
	credits := make([]*txmgr.Credit, 0)
	for _, txin := range txins {
		hash, err := wire.NewHashFromStr(txin.TxId)
		if err != nil {
			return massutil.ZeroAmount(), err
		}
		input := &txmgr.Credit{
			OutPoint: wire.OutPoint{
				Hash:  *hash,
				Index: txin.Vout,
			},
		}
		credits = append(credits, input)
	}
	size, err := w.estimateSignedSize(credits, txoutLen)
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	return blockchain.CalcMinRequiredTxRelayFee(size, massutil.MinRelayTxFee())
}

func (w *WalletManager) GetTxHistory(wanted int, addr string) ([]*pb.TxHistoryDetails, error) {
	am := w.ksmgr.CurrentKeystore()
	if am == nil {
		return nil, ErrNoWalletInUse
	}

	if wanted == 0 {
		wanted = TxHistoryMax
	}

	scripts := make([][]byte, 0)
	if len(addr) > 0 {
		address, err := massutil.DecodeAddress(addr, w.chainParams)
		if err != nil {
			return nil, err
		}
		script := address.ScriptAddress()
		mAddr, err := w.ksmgr.GetManagedAddressByScriptHashInCurrent(script)
		if err != nil {
			return nil, err
		}
		scripts = append(scripts, mAddr.ScriptAddress())
	} else {
		err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) (err error) {
			list, err := w.utxoStore.GetAddresses(tx, am.Name())
			if err != nil {
				logging.CPrint(logging.ERROR, "failed to get address", logging.LogFormat{
					"err":      err,
					"walletId": am.Name(),
				})
				return err
			}
			set := make(map[string]struct{})
			for _, ad := range list {
				// logging.CPrint(logging.INFO, "list address", logging.LogFormat{"address": ad.Address})
				if ad.Used {
					addr, err := massutil.DecodeAddress(ad.Address, w.chainParams)
					if err != nil {
						return err
					}
					// logging.CPrint(logging.INFO, "have address", logging.LogFormat{"addr": addr})
					set[string(addr.ScriptAddress())] = struct{}{}
				}
			}
			for script := range set {
				scripts = append(scripts, []byte(script))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		if len(scripts) == 0 {
			return nil, nil
		}
	}

	_, bestHeight, err := w.chainFetcher.NewestSha()
	if err != nil {
		return nil, err
	}

	// calculate the batchStart height
	count := 0
	var batchStart, batchEnd uint64
	if bestHeight < BlockBatch {
		batchStart = 1
	} else {
		batchStart = bestHeight - BlockBatch + 1
	}
	batchEnd = bestHeight + 1

	rTxLimit := &ifc.HeightSortedRelatedTx{
		Descending:    false,
		SortedHeights: make([]uint64, 0),
		Data:          make(map[uint64][]*wire.TxLoc),
	}

	// calculate the start height for each fetch
	calc := func(start uint64) (uint64, uint64) {
		if start <= BlockBatch {
			return 1, start
		} else {
			return start - BlockBatch, start
		}
	}
	// for each search, fetch related txs in block [start, start+BlockBatch)

	var start, end uint64
	for start, end = batchStart, batchEnd; start > 0; start, end = calc(start) {
		logging.CPrint(logging.DEBUG, "tx history", logging.LogFormat{"start": start, "end": end})
		rTxs, err := w.chainFetcher.FetchScriptHashRelatedTx(scripts, start, end, w.chainParams)
		if err != nil {
			return nil, err
		}
		countInBatch := 0
		for _, txLocs := range rTxs.Data {
			countInBatch += len(txLocs)
		}
		if count+countInBatch <= wanted {
			rTxLimit.SortedHeights = append(rTxLimit.SortedHeights, rTxs.SortedHeights...)
			for _, height := range rTxs.SortedHeights {
				rTxLimit.Data[height] = rTxs.Data[height]
			}
			count += countInBatch
		} else {
			rest := wanted - count
			if rest == 0 {
				break
			} else {
				res := selectRelatedTx(rTxs, rest)
				rTxLimit.SortedHeights = append(rTxLimit.SortedHeights, res.SortedHeights...)
				for _, height := range res.SortedHeights {
					rTxLimit.Data[height] = res.Data[height]
				}
				break
			}
		}
		if start == 1 {
			break
		}
	}

	histories := make([]*pb.TxHistoryDetails, 0)
	for _, height := range rTxLimit.Heights() {
		txlocs := rTxLimit.Get(height)
		if len(txlocs) == 0 {
			continue
		}
		for _, txLoc := range txlocs {

			mtx, err := w.chainFetcher.FetchTxByLoc(height, txLoc)
			if err != nil {
				return nil, err
			}
			if mtx == nil {
				return nil, errors.New("transaction not found")
			}

			// inputs
			fromSet := make(map[string]struct{}, 0)
			isCoinbase := blockchain.IsCoinBaseTx(mtx)

			txPbIns := make([]*pb.TxHistoryDetails_Input, 0)

			if isCoinbase {
				fromSet["COINBASE"] = struct{}{}
			} else {
				for _, txIn := range mtx.TxIn {
					txPbIn := &pb.TxHistoryDetails_Input{
						TxId:  txIn.PreviousOutPoint.Hash.String(),
						Index: int64(txIn.PreviousOutPoint.Index),
					}
					txPbIns = append(txPbIns, txPbIn)
					prevMtx, err := w.chainFetcher.FetchLastTxUntilHeight(&txIn.PreviousOutPoint.Hash, height)
					if err != nil {
						return nil, err
					}
					if prevMtx == nil {
						return nil, fmt.Errorf("transaction %s not found", txIn.PreviousOutPoint.Hash.String())
					}
					ps, err := utils.ParsePkScript(prevMtx.TxOut[txIn.PreviousOutPoint.Index].PkScript, w.chainParams)
					if err != nil {
						return nil, err
					}
					fromSet[ps.StdEncodeAddress()] = struct{}{}
				}
			}

			// outputs
			txPbOuts := make([]*pb.TxHistoryDetails_Output, 0)
			for _, txOut := range mtx.TxOut {
				ps, err := utils.ParsePkScript(txOut.PkScript, w.chainParams)
				if err != nil {
					return nil, err
				}
				val, err := AmountToString(txOut.Value)
				if err != nil {
					return nil, err
				}
				txPbOut := &pb.TxHistoryDetails_Output{
					Amount: val,
				}
				txPbOut.Address = ps.StdEncodeAddress()
				txPbOuts = append(txPbOuts, txPbOut)
			}

			froms := make([]string, 0)
			for from := range fromSet {
				froms = append(froms, from)
			}

			txPb := &pb.TxHistoryDetails{
				TxId:          mtx.TxHash().String(),
				BlockHeight:   height,
				FromAddresses: froms,
				Inputs:        txPbIns,
				Outputs:       txPbOuts,
			}
			histories = append(histories, txPb)
		}
	}
	return histories, nil
}

func selectRelatedTx(h *ifc.HeightSortedRelatedTx, num int) *ifc.HeightSortedRelatedTx {
	count := 0
	result := &ifc.HeightSortedRelatedTx{
		Descending:    h.Descending,
		SortedHeights: make([]uint64, 0),
		Data:          make(map[uint64][]*wire.TxLoc),
	}
	// for _, height := range h.SortedHeights {
	for i := len(h.SortedHeights) - 1; i >= 0; i-- {
		height := h.SortedHeights[i]
		if count+len(h.Data[height]) <= num {
			count += len(h.Data[height])
			result.SortedHeights = append(result.SortedHeights, height)
			result.Data[height] = h.Data[height]
		} else {
			rest := num - count
			if rest == 0 {
				break
			}
			sort.Slice(h.Data[height], func(i, j int) bool {
				return h.Data[height][i].TxStart < h.Data[height][j].TxStart
			})
			result.SortedHeights = append(result.SortedHeights, height)
			result.Data[height] = h.Data[height][len(h.Data[height])-rest:]
			break
		}
	}
	return result
}
