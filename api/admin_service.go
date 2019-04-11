package api

import (
	"bytes"
	"encoding/hex"
	"sort"

	pb "github.com/massnetorg/MassNet-wallet/api/proto"
	"github.com/massnetorg/MassNet-wallet/blockchain"
	"github.com/massnetorg/MassNet-wallet/btcec"
	cfg "github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/logging"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/txscript"
	"github.com/massnetorg/MassNet-wallet/wallet"
	"github.com/massnetorg/MassNet-wallet/wire"

	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"
	"google.golang.org/grpc/status"
)

func (s *Server) constructTx(Amounts map[string]massutil.Amount, LockTime int64) (*wire.MsgTx, error) {
	mtx := wire.NewMsgTx()
	allAmount := massutil.Amount(0)
	for encodedAddr, amount := range Amounts {
		if amount <= 0 || amount > massutil.MaxMaxwell {
			st := status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount])
			return nil, st.Err()
		}
		allAmount = allAmount + amount

		addr, err := massutil.DecodeAddress(encodedAddr,
			&cfg.ChainParams)
		if err != nil {
			st := status.New(ErrAPIFailedDecodeAddress, ErrCode[ErrAPIFailedDecodeAddress])
			return nil, st.Err()
		}

		switch addr.(type) {
		case *massutil.AddressWitnessScriptHash:
		default:
			logging.CPrint(logging.ERROR, "Invalid address or key", logging.LogFormat{"address": addr})
			st := status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress])
			return nil, st.Err()
		}
		if !addr.IsForNet(&cfg.ChainParams) {
			st := status.New(ErrAPINet, ErrCode[ErrAPINet])
			return nil, st.Err()
		}

		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to create pkScript", logging.LogFormat{})
			st := status.New(ErrAPICreatePkScript, ErrCode[ErrAPICreatePkScript])
			return nil, st.Err()
		}

		txOut := wire.NewTxOut(int64(amount), pkScript)
		mtx.AddTxOut(txOut)
	}
	utxos, selectedamount, _, err := s.utxo.FindEligibleUtxos(allAmount, s.wallet.AddressList)
	if err != nil {
		logging.CPrint(logging.ERROR, "err in finding utxo", logging.LogFormat{})
		st := status.New(ErrAPIFindingUtxo, ErrCode[ErrAPIFindingUtxo])
		return nil, st.Err()
	}

	if selectedamount < allAmount {
		logging.CPrint(logging.ERROR, "Not enough money to pay", logging.LogFormat{"realamount": selectedamount, "wantamount": allAmount})
		st := status.New(ErrAPIInsufficient, ErrCode[ErrAPIInsufficient])
		return nil, st.Err()
	}

	for _, utxo := range utxos {
		txid := utxo.Index
		txhash := utxo.TxSha
		outPoint := wire.NewOutPoint(txhash, txid)
		txIn := wire.NewTxIn(outPoint, nil)
		if LockTime != 0 {
			txIn.Sequence = wire.MaxTxInSequenceNum - 1
		}
		mtx.AddTxIn(txIn)
	}
	return mtx, nil
}

func (s *Server) GetTransactionFee(ctx context.Context, in *pb.GetTransactionFeeRequest) (*pb.GetTransactionFeeResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for transaction fee", logging.LogFormat{})
	if in.LockTime < 0 || in.LockTime > int64(wire.MaxTxInSequenceNum) {
		logging.CPrint(logging.ERROR, "Invalid lockTime", logging.LogFormat{
			"lockTime": in.LockTime,
		})
		st := status.New(ErrAPIInvalidLockTime, ErrCode[ErrAPIInvalidLockTime])
		return nil, st.Err()
	}
	if len(in.Amounts) == 0 {
		logging.CPrint(logging.ERROR, "Invalid amount", logging.LogFormat{
			"amounts": in.Amounts,
		})
		st := status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter])
		return nil, st.Err()
	}
	if len(s.wallet.AddressList) == 0 {
		logging.CPrint(logging.ERROR, "No address in wallet", logging.LogFormat{
			"addressList number": 0,
		})
		st := status.New(ErrAPINoAddressInWallet, ErrCode[ErrAPINoAddressInWallet])
		return nil, st.Err()
	}

	maxwells := make(map[string]massutil.Amount)
	for addr, amt := range in.Amounts {
		maxwell, err := massutil.NewAmount(amt)
		if err != nil {
			logging.CPrint(logging.ERROR, "invalid mass amount", logging.LogFormat{"amount": amt})
			st := status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount])
			return nil, st.Err()
		}
		maxwells[addr] = maxwell
	}

	_, txFee, _, err := s.wallet.EstimateTxFee(maxwells, in.LockTime, s.utxo, s.wallet.AddressList[0], 0, s.db)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to estimate transaction fee", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIEstimateTxFee, ErrCode[ErrAPIEstimateTxFee])
		return nil, st.Err()
	}

	logging.CPrint(logging.INFO, "Got the transaction fee", logging.LogFormat{"txFee": txFee})
	return &pb.GetTransactionFeeResponse{UserTxFee: txFee.ToMASS()}, nil
}

func (s *Server) AutoCreateTransaction(ctx context.Context, in *pb.CreateRawTransactionAutoRequest) (*pb.CreateRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for autoCreateTransaction", logging.LogFormat{})
	if in.LockTime < 0 || in.LockTime > int64(wire.MaxTxInSequenceNum) {
		logging.CPrint(logging.ERROR, "Invalid lockTime", logging.LogFormat{
			"lockTime": in.LockTime,
		})
		st := status.New(ErrAPIInvalidLockTime, ErrCode[ErrAPIInvalidLockTime])
		return nil, st.Err()
	}
	if len(in.Amounts) == 0 || in.Amounts == nil {
		logging.CPrint(logging.ERROR, "Invalid amount", logging.LogFormat{
			"amounts": in.Amounts,
		})
		st := status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter])
		return nil, st.Err()
	}
	if len(s.wallet.AddressList) == 0 {
		logging.CPrint(logging.ERROR, "No address in wallet", logging.LogFormat{
			"addressList number": 0,
		})
		st := status.New(ErrAPINoAddressInWallet, ErrCode[ErrAPINoAddressInWallet])
		return nil, st.Err()
	}

	var totalAmounts float64

	for _, v := range in.Amounts {
		totalAmounts += v
	}

	if in.UserTxFee < 0 || in.UserTxFee > 0.8*totalAmounts {
		logging.CPrint(logging.ERROR, "Unreasonable useTxFee", logging.LogFormat{
			"UserTxFee": in.UserTxFee,
		})
		st := status.New(ErrAPIUserTxFee, ErrCode[ErrAPIUserTxFee])
		return nil, st.Err()
	}

	maxwells := make(map[string]massutil.Amount)
	for addr, amt := range in.Amounts {
		maxwell, err := massutil.NewAmount(amt)
		if err != nil {
			logging.CPrint(logging.ERROR, "invalid mass amount", logging.LogFormat{"amount": amt})
			st := status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount])
			return nil, st.Err()
		}
		maxwells[addr] = maxwell
	}

	mxwTxFee, err := massutil.NewAmount(in.UserTxFee)
	if err != nil {
		logging.CPrint(logging.ERROR, "invalid mass amount for txFee", logging.LogFormat{"amount": in.UserTxFee})
		st := status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount])
		return nil, st.Err()
	}

	mtx, _, err := s.wallet.AutoConstructTx(maxwells, in.LockTime, s.utxo, s.wallet.AddressList[0], mxwTxFee, s.db)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to constructTx", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}

	mtx.LockTime = uint32(in.LockTime)
	mtxHex, err := messageToHex(mtx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in messageToHex(mtx)", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}
	logging.CPrint(logging.INFO, "AutoCreateTransaction completed", logging.LogFormat{
		"mtxHex": mtxHex,
	})
	return &pb.CreateRawTransactionResponse{Hex: mtxHex}, nil
}

func (s *Server) CreateRawTransaction(ctx context.Context, in *pb.CreateRawTransactionRequest) (*pb.CreateRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for createRawTransaction", logging.LogFormat{})
	if in.LockTime < 0 || in.LockTime > int64(wire.MaxTxInSequenceNum) {
		logging.CPrint(logging.ERROR, "Invalid lockTime", logging.LogFormat{
			"lockTime": in.LockTime,
		})
		st := status.New(ErrAPIInvalidLockTime, ErrCode[ErrAPIInvalidLockTime])
		return nil, st.Err()
	}

	if len(in.Inputs) == 0 || in.Inputs == nil {
		logging.CPrint(logging.ERROR, "Invalid inputs", logging.LogFormat{
			"input": in.Inputs,
		})
		st := status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter])
		return nil, st.Err()
	}

	if len(in.Amounts) == 0 || in.Amounts == nil {
		logging.CPrint(logging.ERROR, "Invalid amount", logging.LogFormat{
			"amounts": in.Amounts,
		})
		st := status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter])
		return nil, st.Err()
	}

	mtx := wire.NewMsgTx()
	mtx, err := wallet.ConstructTxIn(in.Inputs, mtx, in.LockTime)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to constructTxIn", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}

	maxwells := make(map[string]massutil.Amount)
	for addr, amt := range in.Amounts {
		maxwell, err := massutil.NewAmount(amt)
		if err != nil {
			logging.CPrint(logging.ERROR, "invalid mass amount", logging.LogFormat{"amount": amt})
			st := status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount])
			return nil, st.Err()
		}
		maxwells[addr] = maxwell
	}

	mtx, err = wallet.ConstructTxOut(maxwells, mtx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to constructTxOut", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}

	mtx.LockTime = uint32(in.LockTime)
	mtxHex, err := messageToHex(mtx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in messageToHex(mtx)", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}

	logging.CPrint(logging.INFO, "CreateRawTransaction completed", logging.LogFormat{
		"mtxHex": mtxHex,
	})
	return &pb.CreateRawTransactionResponse{Hex: mtxHex}, nil
}

func decodeHexStr(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "Hex string decode failed", logging.LogFormat{"err": err.Error()})
		st := status.New(ErrAPIDecodeHexString, ErrCode[ErrAPIDecodeHexString])
		return nil, st.Err()
	}
	return decoded, nil
}

func (s *Server) SignTransaction(password string, flag string, tx wire.MsgTx) (bytes.Buffer, error) {
	var buf bytes.Buffer
	var hashType txscript.SigHashType
	switch flag {
	case "ALL":
		hashType = txscript.SigHashAll
	case "NONE":
		hashType = txscript.SigHashNone
	case "SINGLE":
		hashType = txscript.SigHashSingle
	case "ALL|ANYONECANPAY":
		hashType = txscript.SigHashAll | txscript.SigHashAnyOneCanPay
	case "NONE|ANYONECANPAY":
		hashType = txscript.SigHashNone | txscript.SigHashAnyOneCanPay
	case "SINGLE|ANYONECANPAY":
		hashType = txscript.SigHashSingle | txscript.SigHashAnyOneCanPay
	default:
		logging.CPrint(logging.ERROR, "Invalid sighash parameter", logging.LogFormat{"flag": flag})
		st := status.New(ErrAPIInvalidFlag, ErrCode[ErrAPIInvalidFlag])
		return buf, st.Err()
	}

	inputs := make(map[wire.OutPoint][]byte)
	value := make(map[wire.OutPoint]int64)
	params := &cfg.ChainParams
	var script []byte
	var mtx *wire.MsgTx
	var dbSpentInfo []bool

	for _, txIn := range tx.TxIn {
		if s.txMemPool.HaveTransaction(&txIn.PreviousOutPoint.Hash) {
			tx, err := s.txMemPool.FetchTransaction(&txIn.PreviousOutPoint.Hash)
			if err != nil {
				logging.CPrint(logging.ERROR, "No information available about transaction in mempool", logging.LogFormat{"err": err, "txid": &txIn.PreviousOutPoint.Hash})
				st := status.New(ErrAPINoTxInfo, ErrCode[ErrAPINoTxInfo])
				return buf, st.Err()
			}
			mtx = tx.MsgTx()
		} else {
			txList, err := s.db.FetchTxBySha(&txIn.PreviousOutPoint.Hash)
			if err != nil || len(txList) == 0 {
				logging.CPrint(logging.ERROR, "No information available about transaction in database", logging.LogFormat{"err": err, "txid": &txIn.PreviousOutPoint.Hash})
				st := status.New(ErrAPINoTxInfo, ErrCode[ErrAPINoTxInfo])
				return buf, st.Err()
			}

			lastTx := txList[len(txList)-1]
			mtx = lastTx.Tx
			dbSpentInfo = lastTx.TxSpent
		}

		if txIn.PreviousOutPoint.Index > uint32(len(mtx.TxOut)-1) {
			logging.CPrint(logging.ERROR, "Ouput index number (vout) does not exist for transaction", logging.LogFormat{"index": txIn.PreviousOutPoint.Index})
			st := status.New(ErrAPIInvalidIndex, ErrCode[ErrAPIInvalidIndex])
			return buf, st.Err()
		}

		txOut := mtx.TxOut[txIn.PreviousOutPoint.Index]
		if txOut == nil {
			logging.CPrint(logging.ERROR, "Ouput index for txid does not exist", logging.LogFormat{"index": txIn.PreviousOutPoint.Index, "txid": &txIn.PreviousOutPoint.Hash})
			st := status.New(ErrAPINoTxOut, ErrCode[ErrAPINoTxOut])
			return buf, st.Err()
		}

		if dbSpentInfo != nil && dbSpentInfo[txIn.PreviousOutPoint.Index] {
			logging.CPrint(logging.ERROR, "Ouput index for txid has been spent", logging.LogFormat{"index": txIn.PreviousOutPoint.Index, "txid": &txIn.PreviousOutPoint.Hash})
			st := status.New(ErrAPIDuplicateTx, ErrCode[ErrAPIDuplicateTx])
			return buf, st.Err()
		}

		script = txOut.PkScript
		inputs[txIn.PreviousOutPoint] = script
		value[txIn.PreviousOutPoint] = txOut.Value
	}

	err := s.wallet.SignWitnessTx(password, &tx, hashType, inputs, value, params)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to sign the transaction", logging.LogFormat{"err": err.Error()})
		return buf, err
	}

	buf.Grow(tx.SerializeSize())

	if err = tx.Serialize(&buf, wire.Packet); err != nil {
		logging.CPrint(logging.FATAL, "err in api tx serialize", logging.LogFormat{"err": err, "buf": buf.String()})
		st := status.New(ErrAPIEncode, ErrCode[ErrAPIEncode])
		return buf, st.Err()
	}

	return buf, nil
}

func (s *Server) SignRawTransaction(ctx context.Context, in *pb.SignRawTransactionRequest) (*pb.SignRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for SignRawTransaction", logging.LogFormat{
		"rawTx": in.RawTx,
	})
	serializedTx, err := decodeHexStr(in.RawTx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in decodeHexStr", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}
	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewBuffer(serializedTx), wire.Packet)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to decode tx", logging.LogFormat{"err": err.Error()})
		st := status.New(ErrAPIDeserialization, ErrCode[ErrAPIDeserialization])
		return nil, st.Err()
	}

	buf, err := s.SignTransaction(wallet.DefaultPassword, in.Flags, tx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to sign tx", logging.LogFormat{"err": err.Error()})
		return nil, err
	}

	logging.CPrint(logging.INFO, "SignRawTransaction completed", logging.LogFormat{
		"txHex": hex.EncodeToString(buf.Bytes()),
	})

	return &pb.SignRawTransactionResponse{
		Hex:      hex.EncodeToString(buf.Bytes()),
		Complete: true,
	}, nil
}

func (s *Server) SendRawTransaction(ctx context.Context, in *pb.SendRawTransactionRequest) (*pb.SendRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for SendRawTransaction", logging.LogFormat{
		"txHex": in.HexTx,
	})
	hexStr := in.HexTx
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	serializedTx, err := hex.DecodeString(hexStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to decode txHex", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIFailedDecodeAddress, ErrCode[ErrAPIFailedDecodeAddress])
		return nil, st.Err()
	}
	msgtx := wire.NewMsgTx()
	err = msgtx.Deserialize(bytes.NewReader(serializedTx), wire.Packet)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to deserialize transaction", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIDeserialization, ErrCode[ErrAPIDeserialization])
		return nil, st.Err()
	}

	tx := massutil.NewTx(msgtx)
	_, err = s.txMemPool.ProcessTransaction(tx, true, false)
	if err != nil {
		if _, ok := err.(blockchain.RuleError); ok {
			logging.CPrint(logging.DEBUG, "Rejected transaction", logging.LogFormat{"txid": tx.Hash(), "err": err})
		} else {
			logging.CPrint(logging.ERROR, "Failed to process transaction", logging.LogFormat{"txid": tx.Hash(), "err": err})
		}
		st := status.New(ErrAPIRejectTx, ErrCode[ErrAPIRejectTx])
		return nil, st.Err()
	}

	logging.CPrint(logging.INFO, "SendRawTransaction completed", logging.LogFormat{
		"txHash": tx.Hash().String(),
	})
	return &pb.SendRawTransactionResponse{Hex: tx.Hash().String()}, nil

}

func generatePubkey(s *Server, n int) error {
	seeds := s.wallet.RootPkExStrList
	if len(seeds) == 0 {
		st := status.New(ErrAPINoSeedsInWallet, ErrCode[ErrAPINoSeedsInWallet])
		logging.CPrint(logging.ERROR, "no seeds", logging.LogFormat{
			"err": st.Err(),
		})
		return st.Err()
	}

	_, _, err := s.wallet.GenerateChildKeysForSpace(s.wallet.RootPkExStrList[0], n)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to generate pubkey from seed[0]", logging.LogFormat{})
		st := status.New(ErrAPICreatePubKey, ErrCode[ErrAPICreatePubKey])
		return st.Err()
	}
	return nil
}

func (s *Server) CreateAddress(ctx context.Context, in *pb.CreateAddressRequest) (*pb.CreateAddressResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for CreateAddress", logging.LogFormat{
		"signNumber":   in.SignRequire,
		"pubKeyNumber": in.PubKeyNumber,
	})
	if in.SignRequire <= 0 || in.PubKeyNumber <= 0 || in.SignRequire > in.PubKeyNumber || in.PubKeyNumber > txscript.MaxPubKeysPerMultiSig {
		logging.CPrint(logging.ERROR, "the parameter is not standard", logging.LogFormat{
			"numPubKey":  in.PubKeyNumber,
			"numRequire": in.SignRequire,
		})
		st := status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter])
		return nil, st.Err()
	}

	var version int
	if in.Version == 0 {
		version = 0
	} else if in.Version == 10 {
		version = 10
	} else {
		logging.CPrint(logging.ERROR, "the parameter is not standard", logging.LogFormat{"numPubKey": in.PubKeyNumber, "numRequire": in.SignRequire})
		st := status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter])
		return nil, st.Err()
	}

	for _, childPkList := range s.wallet.RootPkExStrToChildPkMap {
		if len(childPkList) == 0 {
			st := status.New(ErrAPIWalletInternal, ErrCode[ErrAPIWalletInternal])
			logging.CPrint(logging.ERROR, "No pubkey  in wallet", logging.LogFormat{
				"err": st.Err(),
			})
			return nil, st.Err()
		}
	}

	err := generatePubkey(s, int(in.PubKeyNumber))
	if err != nil {
		logging.CPrint(logging.ERROR, "Cannot create pubKey", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPICreatePubKey, ErrCode[ErrAPICreatePubKey])
		return nil, st.Err()
	}

	var pubkeys []*btcec.PublicKey
	m := len(s.wallet.RootPkExStrToChildPkMap[s.wallet.RootPkExStrList[0]])
	for i := 0; i < int(in.PubKeyNumber); i++ {
		m--
		pubkeys = append(pubkeys, s.wallet.RootPkExStrToChildPkMap[s.wallet.RootPkExStrList[0]][m])
	}

	address, err := s.wallet.NewWitnessScriptAddress(pubkeys, int(in.SignRequire), version)
	if err != nil {
		logging.CPrint(logging.ERROR, "Cannot create redeem script", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPICreateRedeemScript, ErrCode[ErrAPICreateRedeemScript])
		return nil, st.Err()
	}
	if version == 10 {
		_, err := s.wallet.NewWitnessScriptAddress(pubkeys, int(in.SignRequire), 0)
		if err != nil {
			logging.CPrint(logging.ERROR, "Cannot create redeem script", logging.LogFormat{
				"err": err,
			})
			st := status.New(ErrAPICreateRedeemScript, ErrCode[ErrAPICreateRedeemScript])
			return nil, st.Err()
		}
	}
	logging.CPrint(logging.INFO, "CreateAddress completed", logging.LogFormat{
		"address": address,
	})
	return &pb.CreateAddressResponse{Address: address}, nil
}

func (s *Server) GetAddresses(ctx context.Context, msg *empty.Empty) (*pb.GetAddressesResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for GetAddress", logging.LogFormat{})
	_, addrToBalanceMap, err := s.utxo.GetBalance(s.wallet.AddressList)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to get balance", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIFindingBalance, ErrCode[ErrAPIFindingBalance])
		return nil, st.Err()
	}

	var addressAndBalance addrAndBalanceList
	for addr, balance := range addrToBalanceMap {
		addressAndBalance = append(addressAndBalance, &pb.AddressAndBalance{Address: addr, Balance: balance.ToMASS()})
	}
	sort.Sort(sort.Reverse(addressAndBalance))
	reps := &pb.GetAddressesResponse{
		AddressList: addressAndBalance,
	}

	logging.CPrint(logging.INFO, "GetAddress completed", logging.LogFormat{
		"addressNumber": len(addressAndBalance),
	})
	return reps, nil
}

func (s *Server) GetBalance(ctx context.Context, in *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for GetBalance", logging.LogFormat{
		"address": in.Addresses,
	})
	_, addrToBalanceMap, err := s.utxo.GetBalance(in.Addresses)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to get balance", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIFindingBalance, ErrCode[ErrAPIFindingBalance])
		return nil, st.Err()
	}

	addressAndBalance := make([]*pb.AddressAndBalance, 0)
	for addr, balance := range addrToBalanceMap {
		addressAndBalance = append(addressAndBalance, &pb.AddressAndBalance{Address: addr, Balance: balance.ToMASS()})
	}
	logging.CPrint(logging.INFO, "GetBalance completed", logging.LogFormat{})
	return &pb.GetBalanceResponse{
		Balance: addressAndBalance,
	}, nil
}

func (s *Server) GetAllBalance(ctx context.Context, msg *empty.Empty) (*pb.GetAllBalanceResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for GetAllBalance", logging.LogFormat{})
	totalAmount, addrToBalanceMap, err := s.utxo.GetBalance(s.wallet.AddressList)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to get balance", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIFindingBalance, ErrCode[ErrAPIFindingBalance])
		return nil, st.Err()
	}

	addressAndBalance := make([]*pb.AddressAndBalance, 0)
	for addr, balance := range addrToBalanceMap {
		addressAndBalance = append(addressAndBalance, &pb.AddressAndBalance{Address: addr, Balance: balance.ToMASS()})
	}
	logging.CPrint(logging.INFO, "GetAllBalance completed", logging.LogFormat{
		"balance": totalAmount,
	})
	return &pb.GetAllBalanceResponse{
		Balance: totalAmount.ToMASS(),
	}, nil
}

func (s *Server) GetUtxo(ctx context.Context, in *pb.GetUtxoRequest) (*pb.GetUtxoResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for GetUtxo", logging.LogFormat{
		"address": in.Addresses,
	})
	_, addrToUnspentMap, err := s.utxo.GetUtxos(in.Addresses)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to get utxo", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIFindingUtxo, ErrCode[ErrAPIFindingUtxo])
		return nil, st.Err()
	}

	addressToUnspentList := make([]*pb.AddressToUnspent, 0)
	for addr, utxos := range addrToUnspentMap {
		unspentList := make([]*pb.Unspent, 0)
		for _, u := range utxos {
			unspentList = append(unspentList, &pb.Unspent{TxId: u.TxSha.String(), Vout: u.Index, Amount: u.Value.ToMASS()})
		}
		addressToUnspentList = append(addressToUnspentList, &pb.AddressToUnspent{Address: addr, Unspents: unspentList})
	}
	logging.CPrint(logging.INFO, "GetUtxo completed", logging.LogFormat{})
	return &pb.GetUtxoResponse{
		AddressToUtxo: addressToUnspentList,
	}, nil
}

func (s *Server) GetAllUtxo(ctx context.Context, msg *empty.Empty) (*pb.GetUtxoResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for GetAllUtxo", logging.LogFormat{})
	_, addrToUnspentMap, err := s.utxo.GetUtxos(s.wallet.AddressList)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to get utxos", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIFindingUtxo, ErrCode[ErrAPIFindingUtxo])
		return nil, st.Err()
	}

	addressToUnspentList := make([]*pb.AddressToUnspent, 0)
	for addr, unspents := range addrToUnspentMap {
		unspentList := make([]*pb.Unspent, 0)
		for _, unspent := range unspents {
			unspentList = append(unspentList, &pb.Unspent{TxId: unspent.TxSha.String(), Vout: unspent.Index, Amount: unspent.Value.ToMASS()})
		}
		addressToUnspentList = append(addressToUnspentList, &pb.AddressToUnspent{Address: addr, Unspents: unspentList})
	}
	logging.CPrint(logging.INFO, "GetAllUtxo completed", logging.LogFormat{})
	return &pb.GetUtxoResponse{
		AddressToUtxo: addressToUnspentList,
	}, nil
}

func (s *Server) GetUtxoByAmount(ctx context.Context, in *pb.GetUtxoByAmountRequest) (*pb.GetUtxoByAmountResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for GetUtxoByAmount", logging.LogFormat{
		"amount": in.Amount,
	})
	unspents, _, err := s.utxo.GetUtxos(s.wallet.AddressList)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to get utxos", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIFindingUtxo, ErrCode[ErrAPIFindingUtxo])
		return nil, st.Err()
	}
	unspentList := make([]*pb.Unspent, 0)

	mxwAmount, err := massutil.NewAmount(in.Amount)
	if err != nil {
		logging.CPrint(logging.ERROR, "invalid mass amount", logging.LogFormat{"amount": in.Amount})
		st := status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount])
		return nil, st.Err()
	}

	for _, unspent := range unspents {
		if unspent.Value >= mxwAmount {
			unspentList = append(unspentList, &pb.Unspent{TxId: unspent.TxSha.String(), Vout: unspent.Index, Amount: unspent.Value.ToMASS()})
		}
	}
	logging.CPrint(logging.INFO, "GetUtxoByAmount completed", logging.LogFormat{})
	return &pb.GetUtxoByAmountResponse{
		Utxo: unspentList,
	}, nil
}

func (s *Server) DumpWallet(ctx context.Context, in *pb.DumpWalletRequest) (*pb.DumpWalletResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for DumpWallet", logging.LogFormat{
		"path": in.DumpDirPath,
	})
	err := s.wallet.DumpWallet(in.DumpDirPath)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to dump wallet", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIDumpWallet, ErrCode[ErrAPIDumpWallet])
		return nil, st.Err()
	}
	logging.CPrint(logging.INFO, "DumpWallet completed", logging.LogFormat{})
	return &pb.DumpWalletResponse{}, nil
}

func (s *Server) ImportWallet(ctx context.Context, in *pb.ImportWalletRequest) (*pb.ImportWalletResponse, error) {
	logging.CPrint(logging.INFO, "Received a request for DumpWallet", logging.LogFormat{
		"path": in.ImportDirPath,
	})
	err := s.wallet.ImportWallet(in.ImportDirPath, wallet.DefaultPassword, wallet.DefaultPassword)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to import wallet", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIImportWallet, ErrCode[ErrAPIImportWallet])
		return nil, st.Err()
	}
	logging.CPrint(logging.INFO, "ImportWallet completed", logging.LogFormat{})
	return &pb.ImportWalletResponse{}, nil
}
