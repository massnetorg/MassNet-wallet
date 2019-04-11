package wallet

import (
	"encoding/hex"

	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/utxo"
	"massnet.org/mass-wallet/wire"

	"google.golang.org/grpc/status"
)

type SignatureError struct {
	InputIndex uint32
	Error      error
}

func (w *Wallet) ConstructTx(Amounts map[string]massutil.Amount, LockTime int64, utxo *utxo.Utxo) (*wire.MsgTx, error) {

	mtx := wire.NewMsgTx()
	allAmount := massutil.Amount(0)
	for encodedAddr, amount := range Amounts {
		if amount <= 0 || amount > massutil.MaxMaxwell {
			st := status.New(errors.ErrAPIInvalidAmount, errors.ErrCode[errors.ErrAPIInvalidAmount])
			return nil, st.Err()
		}
		allAmount = allAmount + amount

		// Decode the provided wallet.
		addr, err := massutil.DecodeAddress(encodedAddr,
			&config.ChainParams)
		if err != nil {
			st := status.New(errors.ErrAPIFailedDecodeAddress, errors.ErrCode[errors.ErrAPIFailedDecodeAddress])
			return nil, st.Err()
		}

		// Ensure the wallet is one of the supported types and that
		// the network encoded with the wallet matches the network the
		// server is currently on.
		switch addr.(type) {
		case *massutil.AddressWitnessScriptHash:
		default:
			logging.CPrint(logging.ERROR, "Invalid address or key", logging.LogFormat{"address": addr})
			st := status.New(errors.ErrAPIInvalidAddress, errors.ErrCode[errors.ErrAPIInvalidAddress])
			return nil, st.Err()
		}
		//todo: Always false
		if !addr.IsForNet(&config.ChainParams) {
			st := status.New(errors.ErrAPINet, errors.ErrCode[errors.ErrAPINet])
			return nil, st.Err()
		}

		// Create a new script which pays to the provided wallet.
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to create pkScript", logging.LogFormat{})
			st := status.New(errors.ErrAPICreatePkScript, errors.ErrCode[errors.ErrAPICreatePkScript])
			return nil, st.Err()
		}

		txOut := wire.NewTxOut(int64(amount), pkScript)
		mtx.AddTxOut(txOut)
	}
	utxos, selectedamount, _, err := utxo.FindEligibleUtxos(allAmount, w.AddressList)
	if err != nil {
		logging.CPrint(logging.ERROR, "err in finding utxo", logging.LogFormat{})
		st := status.New(errors.ErrAPIFindingUtxo, errors.ErrCode[errors.ErrAPIFindingUtxo])
		return nil, st.Err()
	}

	if selectedamount < allAmount {
		logging.CPrint(logging.ERROR, "Not enough money to pay", logging.LogFormat{"realamount": selectedamount, "wantamount": allAmount})
		st := status.New(errors.ErrAPIInsufficient, errors.ErrCode[errors.ErrAPIInsufficient])
		return nil, st.Err()
	}

	for _, txo := range utxos {
		txid := txo.Index
		txhash := txo.TxSha
		outPoint := wire.NewOutPoint(txhash, txid)
		txIn := wire.NewTxIn(outPoint, nil)
		if LockTime != 0 {
			txIn.Sequence = wire.MaxTxInSequenceNum - 1
		}
		mtx.AddTxIn(txIn)
	}
	return mtx, nil
}

func ConstructTxIn(Inputs []*pb.TransactionInput, mtx *wire.MsgTx, LockTime int64) (*wire.MsgTx, error) {
	for _, input := range Inputs {
		txHash, err := wire.NewHashFromStr(input.TxId)
		if err != nil {
			logging.CPrint(logging.ERROR, "exist err in constructTxin", logging.LogFormat{"err": err.Error()})
			st := status.New(errors.ErrAPIShaHashFromStr, errors.ErrCode[errors.ErrAPIShaHashFromStr])
			return nil, st.Err()
		}

		prevOut := wire.NewOutPoint(txHash, uint32(input.Vout))
		txIn := wire.NewTxIn(prevOut, nil)
		if LockTime != 0 {
			txIn.Sequence = wire.MaxTxInSequenceNum - 1
		}
		mtx.AddTxIn(txIn)
	}
	return mtx, nil
}

func ConstructTxOut(amounts map[string]massutil.Amount, mtx *wire.MsgTx) (*wire.MsgTx, error) {
	// Add all transaction outputs to the transaction after performing
	// some validity checks.
	for encodedAddr, amount := range amounts {
		// Ensure amount is in the valid range for monetary amounts.
		if amount <= 0 || amount > massutil.MaxMaxwell {
			st := status.New(errors.ErrAPIInvalidAmount, errors.ErrCode[errors.ErrAPIInvalidAmount])
			return nil, st.Err()
		}

		// Decode the provided wallet.
		addr, err := massutil.DecodeAddress(encodedAddr,
			&config.ChainParams)
		if err != nil {
			st := status.New(errors.ErrAPIFailedDecodeAddress, errors.ErrCode[errors.ErrAPIFailedDecodeAddress])
			return nil, st.Err()
		}

		// Ensure the wallet is one of the supported types and that
		// the network encoded with the wallet matches the network the
		// server is currently on.
		switch addr.(type) {
		case *massutil.AddressWitnessScriptHash:
		default:
			logging.CPrint(logging.ERROR, "Invalid address or key", logging.LogFormat{"address": addr})
			st := status.New(errors.ErrAPIInvalidAddress, errors.ErrCode[errors.ErrAPIInvalidAddress])
			return nil, st.Err()
		}
		//TODO: Always false
		if !addr.IsForNet(&config.ChainParams) {
			st := status.New(errors.ErrAPINet, errors.ErrCode[errors.ErrAPINet])
			return nil, st.Err()
		}

		// Create a new script which pays to the provided wallet.
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to create pkScript", logging.LogFormat{})
			st := status.New(errors.ErrAPICreatePkScript, errors.ErrCode[errors.ErrAPICreatePkScript])
			return nil, st.Err()
		}

		txOut := wire.NewTxOut(int64(amount), pkScript)
		mtx.AddTxOut(txOut)
	}
	return mtx, nil
}

func (w *Wallet) SignWitnessTx(password string, tx *wire.MsgTx, hashType txscript.SigHashType,
	additionalPrevScripts map[wire.OutPoint][]byte,
	value map[wire.OutPoint]int64, params *config.Params) error {

	hashCache := txscript.NewTxSigHashes(tx)

	for i, txIn := range tx.TxIn {
		prevOutScript, _ := additionalPrevScripts[txIn.PreviousOutPoint]
		getSign := txscript.SignClosure(func(pub *btcec.PublicKey, hash []byte) (*btcec.Signature, error) {
			pubkey := pub
			privkey, err := w.PrivKeyFromPubKey(pubkey, password)
			if err != nil {
				// FIXME: print error
				logging.CPrint(logging.ERROR, "cannot find privkey in wallet", logging.LogFormat{})
				return nil, errors.New("cannot find privkey in wallet")
			}
			key := privkey
			signature, err := key.Sign(hash)
			if err != nil {

				return nil, errors.New("Failed to generate signature")
			}
			return signature, nil
		})

		getScript := txscript.ScriptClosure(func(addr massutil.Address) ([]byte, error) {
			script, ok := w.WitnessMap[addr.String()]
			if !ok {
				return nil, errors.New("Failed to find redeem script from address")

			}
			return script, nil
		})

		// SigHashSingle inputs can only be signed if there's a
		// corresponding output. However this could be already signed,
		// so we always verify the output.
		if (hashType&txscript.SigHashSingle) !=
			txscript.SigHashSingle || i < len(tx.TxOut) {

			script, err := txscript.SignTxOutputWit(params, tx, i, value[txIn.PreviousOutPoint], prevOutScript, hashType, getSign, getScript)

			// Failure to sign isn't an error, it just means that
			// the tx isn't complete.
			if err != nil {
				logging.CPrint(logging.ERROR, "Err in txscript.SignTxOutputWit", logging.LogFormat{
					"err": err,
				})
				st := status.New(errors.ErrAPISignTx, errors.ErrCode[errors.ErrAPISignTx])
				return st.Err()

			}
			txIn.Witness = script
		}

		// Either it was already signed or we just signed it.
		// Find out if it is completely satisfied or still needs more.
		vm, err := txscript.NewEngine(prevOutScript, tx, i,
			txscript.StandardVerifyFlags, nil, hashCache, value[txIn.PreviousOutPoint])
		if err == nil {
			err = vm.Execute()
			if err != nil {
				st := status.New(errors.ErrAPINewEngine, errors.ErrCode[errors.ErrAPINewEngine])
				return st.Err()
			}
		}
		if err != nil {
			st := status.New(errors.ErrAPIExecute, errors.ErrCode[errors.ErrAPIExecute])
			return st.Err()
		}
	}

	return nil
}

func WitnessToHex(witness wire.TxWitness) []string {
	// Ensure nil is returned when there are no entries versus an empty
	// slice so it can properly be omitted as necessary.
	if len(witness) == 0 {
		return nil
	}

	result := make([]string, 0, len(witness))
	for _, wit := range witness {
		result = append(result, hex.EncodeToString(wit))
	}

	return result
}

func (w *Wallet) AutoConstructTx(Amounts map[string]massutil.Amount, LockTime int64, utxo *utxo.Utxo, changeAddr string, userTxFee massutil.Amount, db database.Db) (*wire.MsgTx, massutil.Amount, error) {
	//*wire.MsgTx, massutil.Amount, int64
	var (
		txReturn        *wire.MsgTx
		txFeeReturn     massutil.Amount
		remainingReturn massutil.Amount
	)
	mtx, esTxFee, remainingAmount, err := w.EstimateTxFee(Amounts, LockTime, utxo, changeAddr, 0, db)
	if err != nil {
		st := status.New(errors.ErrAPIEstimateTxFee, errors.ErrCode[errors.ErrAPIEstimateTxFee])
		logging.CPrint(logging.ERROR, "estimate txFee failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, st.Err()

	}

	txReturn = mtx
	txFeeReturn = esTxFee
	remainingReturn = remainingAmount

	if userTxFee > esTxFee {
		txFeeReturn = userTxFee

		if userTxFee > remainingAmount {
			mtx, _, remainingAmount, err := w.EstimateTxFee(Amounts, LockTime, utxo, changeAddr, userTxFee, db)
			if err != nil {
				st := status.New(errors.ErrAPIEstimateTxFee, errors.ErrCode[errors.ErrAPIEstimateTxFee])
				logging.CPrint(logging.ERROR, "estimate txFee failed",
					logging.LogFormat{
						"err": err,
					})
				return nil, 0, st.Err()
			}
			txReturn = mtx
			remainingReturn = remainingAmount
		}

	}

	changeAmount := remainingReturn - txFeeReturn

	changeTxOut, err := amountToTxOut(changeAddr, changeAmount)
	if err != nil {
		logging.CPrint(logging.ERROR, "amountToTxOut failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	txReturn.AddTxOut(changeTxOut)

	return txReturn, txFeeReturn, nil
}

func (w *Wallet) EstimateTxFee(Amounts map[string]massutil.Amount, LockTime int64, utxo *utxo.Utxo, changeAddr string, userTxFee massutil.Amount, db database.Db) (*wire.MsgTx, massutil.Amount, massutil.Amount, error) {
	Amounts0 := make(map[string]massutil.Amount, 0)
	for k, amoutMass := range Amounts {
		Amounts0[k] = amoutMass
	}

	txReturn, txFee, remainingAmountReturn, err := estimateTxFee(w, Amounts0, LockTime, utxo, changeAddr, userTxFee, db)
	if err != nil {
		logging.CPrint(logging.ERROR, "estimate txFee failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, 0, err
	}

	return txReturn, txFee, remainingAmountReturn, nil

}

func estimateTxFee(w *Wallet, Amounts map[string]massutil.Amount, LockTime int64, utxo *utxo.Utxo, changeAddr string, userTxFee massutil.Amount, db database.Db) (*wire.MsgTx, massutil.Amount, massutil.Amount, error) {

	minRelayTxFee, err := massutil.NewAmount(config.MinRelayTxFee)
	if err != nil {
		logging.CPrint(logging.ERROR, "newAmount failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, 0, err
	}
	targetTxFee := minRelayTxFee

	if userTxFee != 0 {
		targetTxFee = userTxFee
	}

	for {
		mtx0 := wire.NewMsgTx()
		mtx1 := wire.NewMsgTx()
		var txFeeReturn massutil.Amount
		var outAmounts massutil.Amount

		for encodedAddr, amount := range Amounts {
			outAmounts += amount

			txOut, err := amountToTxOut(encodedAddr, amount)
			if err != nil {
				logging.CPrint(logging.ERROR, "amountToTxOut failed",
					logging.LogFormat{
						"err": err,
					})
				return nil, 0, 0, err
			}
			mtx0.AddTxOut(txOut)
			mtx1.AddTxOut(txOut)
		}

		targetAmount := targetTxFee + outAmounts
		targetMass := (massutil.Amount)(targetAmount)

		utxos, inMass, _, err := utxo.FindEligibleUtxos(targetMass, w.AddressList)
		if err != nil {
			logging.CPrint(logging.ERROR, "err in finding utxo",
				logging.LogFormat{
					"err": err,
				})
			st := status.New(errors.ErrAPIFindingUtxo, errors.ErrCode[errors.ErrAPIFindingUtxo])
			return nil, 0, 0, st.Err()
		}

		for _, utx := range utxos {
			txid := utx.Index
			txhash := utx.TxSha
			outPoint := wire.NewOutPoint(txhash, txid)
			txIn := wire.NewTxIn(outPoint, nil)
			if LockTime != 0 {
				txIn.Sequence = wire.MaxTxInSequenceNum - 1
			}
			mtx0.AddTxIn(txIn)
			mtx1.AddTxIn(txIn)
		}

		inAmounts := inMass

		if inAmounts < outAmounts {
			logging.CPrint(logging.ERROR, "Not enough money to pay", logging.LogFormat{"realamount": inAmounts, "wantamount": outAmounts})
			st := status.New(errors.ErrAPIInsufficient, errors.ErrCode[errors.ErrAPIInsufficient])
			return nil, 0, 0, st.Err()
		}

		changeAmount0 := inAmounts - outAmounts - targetTxFee

		changeOut0, err := amountToTxOut(changeAddr, changeAmount0)
		if err != nil {
			logging.CPrint(logging.ERROR, "amountToTxOut failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, 0, 0, err
		}
		mtx0.AddTxOut(changeOut0)

		signedTxSize, err := estimateSignedSize(utxos, len(mtx0.TxOut), db, w)
		if err != nil {
			logging.CPrint(logging.ERROR, "estimate signedSize failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, 0, 0, err
		}

		remainingAmount := inAmounts - outAmounts

		requiredTxFeeMaxwell := calcMinTxRelayFee(signedTxSize, minRelayTxFee)

		txFeeReturn = requiredTxFeeMaxwell

		if remainingAmount > requiredTxFeeMaxwell {

			return mtx1, txFeeReturn, remainingAmount, nil
		}
		targetTxFee = requiredTxFeeMaxwell

	}

}

func amountToTxOut(encodedAddr string, amount massutil.Amount) (*wire.TxOut, error) {
	if amount <= 0 || amount > massutil.MaxMaxwell {
		st := status.New(errors.ErrAPIInvalidAmount, errors.ErrCode[errors.ErrAPIInvalidAmount])
		logging.CPrint(logging.ERROR, "invalid input amount",
			logging.LogFormat{
				"err": st.Err(),
			})
		return nil, st.Err()
	}

	// Decode the provided wallet.
	addr, err := massutil.DecodeAddress(encodedAddr,
		&config.ChainParams)
	if err != nil {
		st := status.New(errors.ErrAPIFailedDecodeAddress, errors.ErrCode[errors.ErrAPIFailedDecodeAddress])
		logging.CPrint(logging.ERROR, "decode address failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, st.Err()
	}

	// Ensure the wallet is one of the supported types and that
	// the network encoded with the wallet matches the network the
	// server is currently on.
	switch addr.(type) {
	case *massutil.AddressWitnessScriptHash:
	default:
		logging.CPrint(logging.ERROR, "Invalid address or key", logging.LogFormat{"address": addr})
		st := status.New(errors.ErrAPIInvalidAddress, errors.ErrCode[errors.ErrAPIInvalidAddress])
		return nil, st.Err()
	}
	//todo: always false
	if !addr.IsForNet(&config.ChainParams) {
		st := status.New(errors.ErrAPINet, errors.ErrCode[errors.ErrAPINet])
		logging.CPrint(logging.ERROR, "address is not in the net",
			logging.LogFormat{
				"err": st.Err(),
			})
		return nil, st.Err()
	}

	// Create a new script which pays to the provided wallet.
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to create pkScript",
			logging.LogFormat{
				"err": err,
			})
		st := status.New(errors.ErrAPICreatePkScript, errors.ErrCode[errors.ErrAPICreatePkScript])
		return nil, st.Err()
	}

	txOut := wire.NewTxOut(int64(amount), pkScript)

	return txOut, nil
}

func calcMinTxRelayFee(serializedSize int64, minRelayTxFee massutil.Amount) massutil.Amount {
	minFee := (serializedSize * int64(minRelayTxFee)) / 1000

	if minFee == 0 && minRelayTxFee > 0 {
		minFee = int64(minRelayTxFee)
	}

	// Set the minimum fee to the maximum possible value if the calculated
	// fee is not in the valid range for monetary amounts.
	if minFee < 0 || minFee > massutil.MaxMaxwell {
		minFee = massutil.MaxMaxwell
	}

	return massutil.Amount(minFee)
}

//
func estimateSignedSize(utxos []*database.UtxoListReply, TxOutLen int, db database.Db, wallet *Wallet) (int64, error) {
	signedSize := 0
	for _, utxo := range utxos {
		txidx := utxo.Index
		txhash := utxo.TxSha
		outPoint := wire.NewOutPoint(txhash, txidx)
		txIn := wire.NewTxIn(outPoint, nil)
		txList, err := db.FetchTxBySha(&txIn.PreviousOutPoint.Hash)
		if err != nil {
			return 0, err
		}
		lastTx := txList[len(txList)-1]
		mtx := lastTx.Tx
		pkScript := mtx.TxOut[txidx].PkScript
		_, addr, _, _, err := txscript.ExtractPkScriptAddrs(pkScript, &config.ChainParams)
		if err != nil {
			return 0, err
		}
		script := wallet.WitnessMap[addr[0].String()]
		class, _, _, nrequired, err := txscript.ExtractPkScriptAddrs(script,
			&config.ChainParams)
		if err != nil || class != txscript.MultiSigTy {
			return 0, err
		}
		//a signature size is 73 at most,Sequence + PreviousOut
		signedSize = signedSize + len(script) + 73*nrequired + 4 + 32 + 4
	}
	//Value 8 bytes +  PkScript 23 bytes, which is 31 bytes
	return int64(signedSize + 31*TxOutLen), nil
}
