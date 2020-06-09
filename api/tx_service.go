package api

import (
	"encoding/hex"
	"strconv"
	"strings"

	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/config"
	cfg "massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/consensus"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/masswallet"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"

	"golang.org/x/crypto/ripemd160"
	"golang.org/x/net/context"
	"google.golang.org/grpc/status"
)

// messageToHex serializes a message to the wire protocol encoding using the
// latest protocol version and returns a hex-encoded string of the result.
func messageToHex(msg wire.Message) (string, error) {
	bs, err := msg.Bytes(wire.Packet)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bs), nil
}

func (s *APIServer) createTxRawResult(mtx *wire.MsgTx, blkHeader *wire.BlockHeader, chainHeight uint64) (*pb.GetRawTransactionResponse, error) {

	isCoinbase := blockchain.IsCoinBaseTx(mtx)

	vouts, totalOutValue, err := createVoutList(mtx, &cfg.ChainParams)
	if err != nil {
		return nil, err
	}
	if mtx.Payload == nil {
		mtx.Payload = make([]byte, 0, 0)
	}

	vins, totalInValue, err := s.createVinList(mtx, isCoinbase)
	if err != nil {
		return nil, err
	}

	fee := "0"
	if !isCoinbase {
		fee, err = AmountToString(totalInValue - totalOutValue)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to transfer amount to string", logging.LogFormat{"err": err})
			return nil, err
		}
	}

	// Hex
	bs, err := mtx.Bytes(wire.Packet)
	if err != nil {
		return nil, err
	}
	mtxHex := hex.EncodeToString(bs)
	// TxId
	txHash := mtx.TxHash()
	// Status
	code, err := s.getStatus(&txHash)
	if err != nil {
		return nil, err
	}

	txReply := &pb.GetRawTransactionResponse{
		Hex:      mtxHex,
		TxId:     txHash.String(),
		Version:  int32(mtx.Version),
		LockTime: int64(mtx.LockTime),
		Vin:      vins,
		Vout:     vouts,
		Payload:  hex.EncodeToString(mtx.Payload),
		Size:     int32(mtx.PlainSize()),
		Fee:      fee,
		Status:   code,
		Coinbase: isCoinbase,
	}

	if blkHeader != nil {
		txReply.Block = &pb.BlockInfoForTx{
			Height:    blkHeader.Height,
			BlockHash: blkHeader.BlockHash().String(),
			Timestamp: blkHeader.Timestamp.Unix(),
		}
		txReply.Confirmations = 1 + chainHeight - blkHeader.Height
	}

	return txReply, nil
}

func (s *APIServer) buildDecodeRawTxResponse(mtx *wire.MsgTx) (*pb.DecodeRawTransactionResponse, error) {

	resp := &pb.DecodeRawTransactionResponse{
		TxId:     mtx.TxHash().String(),
		Version:  int32(mtx.Version),
		LockTime: int64(mtx.LockTime),
		Size:     int32(mtx.PlainSize()),
		Vin:      make([]*pb.DecodeRawTransactionResponse_Vin, 0, len(mtx.TxIn)),
		Vout:     make([]*pb.DecodeRawTransactionResponse_Vout, 0, len(mtx.TxOut)),
		Payload:  hex.EncodeToString(mtx.Payload),
	}

	for _, txIn := range mtx.TxIn {
		resp.Vin = append(resp.Vin, &pb.DecodeRawTransactionResponse_Vin{
			TxId:     txIn.PreviousOutPoint.Hash.String(),
			Vout:     txIn.PreviousOutPoint.Index,
			Sequence: txIn.Sequence,
			Witness:  witnessToHex(txIn.Witness),
		})
	}

	for n, txOut := range mtx.TxOut {
		disbuf, err := txscript.DisasmString(txOut.PkScript)
		if err != nil {
			logging.CPrint(logging.WARN, "DisasmString error", logging.LogFormat{"err": err})
			return nil, err
		}

		scriptClass, addrs, _, _, err := txscript.ExtractPkScriptAddrs(txOut.PkScript, &cfg.ChainParams)
		if err != nil {
			return nil, err
		}

		encodedAddrs := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			if scriptClass == txscript.StakingScriptHashTy {
				std, err := massutil.NewAddressWitnessScriptHash(addr.ScriptAddress(), &cfg.ChainParams)
				if err != nil {
					return nil, err
				}
				encodedAddrs = append(encodedAddrs, std.EncodeAddress())
			}
			encodedAddrs = append(encodedAddrs, addr.EncodeAddress())
		}

		vout := &pb.DecodeRawTransactionResponse_Vout{
			Value:     strconv.Itoa(int(txOut.Value)),
			N:         uint32(n),
			Type:      uint32(scriptClass),
			ScriptAsm: disbuf,
			ScriptHex: hex.EncodeToString(txOut.PkScript),
			Addresses: encodedAddrs,
		}
		resp.Vout = append(resp.Vout, vout)
	}
	return resp, nil
}

func witnessToHex(witness wire.TxWitness) []string {
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

// createVoutList returns a slice of JSON objects for the outputs of the passed
// transaction.
func createVoutList(mtx *wire.MsgTx, chainParams *cfg.Params) (vouts []*pb.Vout, totalValue int64, err error) {
	vouts = make([]*pb.Vout, 0, len(mtx.TxOut))
	for i, txout := range mtx.TxOut {
		// The disassembled string will contain [error] inline if the
		// script doesn't fully parse, so ignore the error here.
		disbuf, err := txscript.DisasmString(txout.PkScript)
		if err != nil {
			logging.CPrint(logging.WARN, "decode pkscript to asm exists err", logging.LogFormat{"err": err})
			return nil, 0, err
		}

		// Ignore the error here since an error means the script
		// couldn't parse and there is no additional information about
		// it anyways.
		scriptClass, addrs, _, reqSigs, err := txscript.ExtractPkScriptAddrs(txout.PkScript, chainParams)
		if err != nil {
			return nil, 0, err
		}

		encodedAddrs := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			if scriptClass == txscript.StakingScriptHashTy {
				std, err := massutil.NewAddressWitnessScriptHash(addr.ScriptAddress(), chainParams)
				if err != nil {
					return nil, 0, err
				}
				encodedAddrs = append(encodedAddrs, std.EncodeAddress())
			}
			encodedAddrs = append(encodedAddrs, addr.EncodeAddress())
		}

		val, err := AmountToString(txout.Value)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to transfer amount to string", logging.LogFormat{"err": err})
			return nil, 0, err
		}
		vout := &pb.Vout{
			N:     uint32(i),
			Value: val,
			Type:  uint32(scriptClass),
			ScriptDetail: &pb.Vout_ScriptDetail{
				Asm:       disbuf,
				Hex:       hex.EncodeToString(txout.PkScript),
				ReqSigs:   int32(reqSigs),
				Addresses: encodedAddrs,
			},
		}
		vouts = append(vouts, vout)
		totalValue += txout.Value
	}
	return
}

// createVinList returns a slice of JSON objects for the inputs of the passed
// transaction.
func (s *APIServer) createVinList(mtx *wire.MsgTx, isCoinbase bool) (vins []*pb.Vin, totalValue int64, err error) {

	cache := make(map[wire.Hash]*wire.MsgTx)
	for i, txIn := range mtx.TxIn {
		if isCoinbase && i == 0 {
			continue
		}
		prevTx, ok := cache[txIn.PreviousOutPoint.Hash]
		if !ok {
			tx, err := s.node.TxMemPool().FetchTransaction(&txIn.PreviousOutPoint.Hash)
			if err != nil {
				txReply, err := s.node.Blockchain().GetTransactionInDB(&txIn.PreviousOutPoint.Hash)
				if err != nil || len(txReply) == 0 {
					logging.CPrint(logging.ERROR, "No information available about transaction in db",
						logging.LogFormat{
							"err":  err,
							"txid": txIn.PreviousOutPoint.Hash.String(),
						})
					return nil, 0, err
				}
				prevTx = txReply[len(txReply)-1].Tx
			} else {
				prevTx = tx.MsgTx()
			}
			cache[txIn.PreviousOutPoint.Hash] = prevTx
		}

		prevVout := prevTx.TxOut[txIn.PreviousOutPoint.Index]

		// parse type and addrs
		scriptClass, addrs, _, _, err := txscript.ExtractPkScriptAddrs(prevVout.PkScript, &cfg.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to encode address", logging.LogFormat{"err": err})
			return nil, 0, err
		}
		encodedAddrs := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			if scriptClass == txscript.StakingScriptHashTy {
				std, err := massutil.NewAddressWitnessScriptHash(addr.ScriptAddress(), &cfg.ChainParams)
				if err != nil {
					return nil, 0, err
				}
				encodedAddrs = append(encodedAddrs, std.EncodeAddress())
			}
			encodedAddrs = append(encodedAddrs, addr.EncodeAddress())
		}

		// prev vout value
		val, err := AmountToString(prevVout.Value)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to transfer amount to string", logging.LogFormat{"err": err})
			return nil, 0, err
		}

		vin := &pb.Vin{
			Value: val,
			N:     uint32(i),
			Type:  uint32(scriptClass),
			RedeemDetail: &pb.Vin_RedeemDetail{
				TxId:      txIn.PreviousOutPoint.Hash.String(),
				Vout:      txIn.PreviousOutPoint.Index,
				Sequence:  txIn.Sequence,
				Witness:   witnessToHex(txIn.Witness),
				Addresses: encodedAddrs,
			},
		}
		vins = append(vins, vin)
		totalValue += prevVout.Value
	}
	return
}

func (s *APIServer) GetTxStatus(ctx context.Context, in *pb.GetTxStatusRequest) (*pb.GetTxStatusResponse, error) {
	logging.CPrint(logging.INFO, "api: GetTxStatus", logging.LogFormat{"txid": in.TxId})
	err := checkTransactionIdLen(in.TxId)
	if err != nil {
		return nil, err
	}

	txHash, err := wire.NewHashFromStr(in.TxId)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decode the input string into hash", logging.LogFormat{"input string": in.TxId, "error": err})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}

	code, err := s.getStatus(txHash)
	if err != nil {
		logging.CPrint(logging.ERROR, "getStatus failed", logging.LogFormat{"err": err})
		return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
	}
	resp := &pb.GetTxStatusResponse{
		Code:   code,
		Status: txStatusDesc[code],
	}
	logging.CPrint(logging.INFO, "api: GetTxStatus completed", logging.LogFormat{"response": resp})
	return resp, nil
}

func (s *APIServer) getStatus(txHash *wire.Hash) (code int32, err error) {

	_, err = s.node.TxMemPool().FetchTransaction(txHash)
	if err == nil {
		code = txStatusPacking
	} else {
		code = txStatusMissing
	}

	txList, err := s.node.Blockchain().GetTransactionInDB(txHash)
	if err != nil {
		logging.CPrint(logging.ERROR, "FetchTxBySha error",
			logging.LogFormat{
				"txid": txHash.String(),
				"err":  err,
			})
		return txStatusUndefined, err
	}
	if len(txList) == 0 {
		return code, nil
	}

	lastTx := txList[len(txList)-1]
	txHeight := lastTx.Height
	bestHeight := s.node.Blockchain().BestBlockHeight()
	confirmations := 1 + bestHeight - txHeight
	if blockchain.IsCoinBaseTx(lastTx.Tx) {
		if confirmations >= consensus.CoinbaseMaturity {
			return txStatusConfirmed, nil
		} else {
			return txStatusConfirming, nil
		}
	}
	if confirmations >= consensus.TransactionMaturity {
		return txStatusConfirmed, nil
	} else {
		return txStatusConfirming, nil
	}
}

func (s *APIServer) GetRawTransaction(ctx context.Context, in *pb.GetRawTransactionRequest) (*pb.GetRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "api: GetRawTransaction", logging.LogFormat{"txid": in.TxId})
	err := checkTransactionIdLen(in.TxId)
	if err != nil {
		return nil, err
	}

	// Convert the provided transaction hash hex to a Hash.
	txHash, err := wire.NewHashFromStr(in.TxId)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decode the input string into hash", logging.LogFormat{
			"input string": in.TxId,
			"error":        err})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}

	// Try to fetch the transaction from the memory pool and if that fails,
	// try the block database.
	var mtx *wire.MsgTx
	var chainHeight uint64
	var blkHeader *wire.BlockHeader

	tx, err := s.node.TxMemPool().FetchTransaction(txHash)
	if err != nil {
		txList, err := s.node.Blockchain().GetTransactionInDB(txHash)
		if err != nil || len(txList) == 0 {
			logging.CPrint(logging.ERROR, "failed to query the transaction information in txPool or database according to the transaction hash",
				logging.LogFormat{
					"hash":  txHash.String(),
					"error": err,
				})
			st := status.New(ErrAPINoTxInfo, ErrCode[ErrAPINoTxInfo])
			return nil, st.Err()
		}

		lastTx := txList[len(txList)-1]
		mtx = lastTx.Tx

		// query block header
		blkHeader, err = s.node.Blockchain().GetHeaderByHash(lastTx.BlkSha)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to query the block header according to the block hash",
				logging.LogFormat{
					"block": lastTx.BlkSha.String(),
					"error": err,
				})
			st := status.New(ErrAPIBlockHeaderNotFound, ErrCode[ErrAPIBlockHeaderNotFound])
			return nil, st.Err()
		}

		chainHeight = s.node.Blockchain().BestBlockHeight()
	} else {
		mtx = tx.MsgTx()
	}

	rep, err := s.createTxRawResult(mtx, blkHeader, chainHeight)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to query information of transaction according to the transaction hash",
			logging.LogFormat{
				"hash":  in.TxId,
				"error": err,
			})
		st := status.New(ErrAPIRawTx, ErrCode[ErrAPIRawTx])
		return nil, st.Err()
	}
	logging.CPrint(logging.INFO, "api: GetRawTransaction completed", logging.LogFormat{"hash": in.TxId})
	return rep, nil
}

func (s *APIServer) DecodeRawTransaction(ctx context.Context, in *pb.DecodeRawTransactionRequest) (*pb.DecodeRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "api: DecodeRawTransaction", logging.LogFormat{})

	serializedTx, err := decodeHexStr(in.Hex)
	if err != nil {
		logging.CPrint(logging.ERROR, "decodeHexStr error", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}
	var mtx wire.MsgTx
	err = mtx.SetBytes(serializedTx, wire.Packet)
	if err != nil {
		logging.CPrint(logging.ERROR, "deserialize tx error", logging.LogFormat{"err": err.Error()})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}

	resp, err := s.buildDecodeRawTxResponse(&mtx)
	if err != nil {
		logging.CPrint(logging.ERROR, "buildDecodeRawTxResponse error",
			logging.LogFormat{
				"txid": mtx.TxHash(),
				"err":  err,
			})
		st := status.New(ErrAPIRawTx, ErrCode[ErrAPIRawTx])
		return nil, st.Err()
	}
	logging.CPrint(logging.INFO, "api: DecodeRawTransaction completed", logging.LogFormat{})
	return resp, nil
}

func (s *APIServer) CreateRawTransaction(ctx context.Context, in *pb.CreateRawTransactionRequest) (*pb.CreateRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "api: CreateRawTransaction", logging.LogFormat{"params": in})

	err := checkLocktime(in.LockTime)
	if err != nil {
		return nil, err
	}

	err = checkNotEmpty(in.Inputs)
	if err != nil {
		logging.CPrint(logging.ERROR, "transaction inputs illegal", logging.LogFormat{})
		return nil, err
	}

	err = checkNotEmpty(in.Amounts)
	if err != nil {
		logging.CPrint(logging.ERROR, "transaction outputs illegal", logging.LogFormat{})
		return nil, err
	}

	// inputs
	inputs := make([]*masswallet.TxIn, 0)
	for _, txInput := range in.Inputs {
		txid := strings.TrimSpace(txInput.TxId)
		err := checkTransactionIdLen(txid)
		if err != nil {
			return nil, err
		}
		input := &masswallet.TxIn{
			TxId: txid,
			Vout: txInput.Vout,
		}
		inputs = append(inputs, input)
	}

	// outputs
	amounts := make(map[string]massutil.Amount)
	for addr, valStr := range in.Amounts {
		addr = strings.TrimSpace(addr)
		err := checkAddressLen(addr)
		if err != nil {
			return nil, err
		}
		valStr = strings.TrimSpace(valStr)
		val, err := checkParseAmount(valStr)
		if err != nil {
			return nil, err
		}
		amounts[addr] = val
	}

	// change
	changeAddr := strings.TrimSpace(in.ChangeAddress)

	// subtractfeefrom
	subtractfeefrom := make(map[string]struct{})
	for _, subfrom := range in.Subtractfeefrom {
		subfrom = strings.TrimSpace(subfrom)
		if len(subfrom) == 0 {
			continue
		}
		subtractfeefrom[subfrom] = struct{}{}
	}

	mtxHex, fee, err := s.massWallet.CreateRawTransaction(inputs, amounts, in.LockTime, changeAddr, subtractfeefrom)
	if err != nil {
		logging.CPrint(logging.ERROR, "CreateRawTransaction failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	err = checkTxFeeLimit(s.config.Config, fee)
	if err != nil {
		return nil, err
	}

	logging.CPrint(logging.INFO, "api: CreateRawTransaction completed", logging.LogFormat{})
	return &pb.CreateRawTransactionResponse{Hex: mtxHex}, nil
}

func (s *APIServer) CreateStakingTransaction(ctx context.Context, in *pb.CreateStakingTransactionRequest) (*pb.CreateRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "api: CreateStakingTransaction", logging.LogFormat{"params": in})

	val, err := checkParseAmount(in.Amount)
	if err != nil {
		return nil, err
	}

	valFee, err := checkParseAmount(in.Fee)
	if err != nil {
		return nil, status.New(ErrAPIUserTxFee, ErrCode[ErrAPIUserTxFee]).Err()
	}

	if len(in.FromAddress) > 0 {
		_, err = checkWitnessAddress(in.FromAddress, false, &cfg.ChainParams)
		if err != nil {
			return nil, err
		}
	}

	_, err = checkWitnessAddress(in.StakingAddress, true, &cfg.ChainParams)
	if err != nil {
		return nil, err
	}

	outputs := make([]*masswallet.StakingTxOut, 0)
	output := &masswallet.StakingTxOut{
		Address:      in.StakingAddress,
		Amount:       val,
		FrozenPeriod: in.FrozenPeriod,
	}
	outputs = append(outputs, output)

	mtxHex, fee, err := s.massWallet.CreateStakingTransaction(in.FromAddress, outputs, uint64(0), valFee)
	if err != nil {
		logging.CPrint(logging.ERROR, "AutoCreateStakingTransaction failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	err = checkTxFeeLimit(s.config.Config, fee)
	if err != nil {
		return nil, err
	}

	logging.CPrint(logging.INFO, "api: CreateStakingTransaction completed", logging.LogFormat{})
	return &pb.CreateRawTransactionResponse{Hex: mtxHex}, nil
}

// Creates a binding transaction with utxos that randomly selected.
func (s *APIServer) CreateBindingTransaction(ctx context.Context, in *pb.CreateBindingTransactionRequest) (*pb.CreateRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "api: CreateBindingTransaction", logging.LogFormat{"params": in})

	err := checkNotEmpty(in.Outputs)
	if err != nil {
		return nil, err
	}

	totalOutValue := massutil.ZeroAmount()
	for _, m := range in.Outputs {
		val, err := checkParseAmount(m.Amount)
		if err != nil {
			return nil, err
		}
		totalOutValue, err = totalOutValue.Add(val)
		if err != nil {
			logging.CPrint(logging.ERROR, "total amount error", logging.LogFormat{"err": err})
			return nil, status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount]).Err()
		}
	}

	txFee, err := checkParseAmount(in.Fee)
	if err != nil {
		return nil, status.New(ErrAPIUserTxFee, ErrCode[ErrAPIUserTxFee]).Err()
	}

	if len(in.FromAddress) > 0 {
		_, err = checkWitnessAddress(in.FromAddress, false, &cfg.ChainParams)
		if err != nil {
			return nil, err
		}
	}

	//load output
	outPut := make([]*masswallet.BindingOutput, 0)
	for _, num := range in.Outputs {
		_, err = checkWitnessAddress(num.HolderAddress, false, &cfg.ChainParams)
		if err != nil {
			return nil, err
		}
		_, err = checkPoCPubKeyAddress(num.BindingAddress, &cfg.ChainParams)
		if err != nil {
			return nil, err
		}
		val, err := checkParseAmount(num.Amount)
		if err != nil {
			return nil, err
		}
		tempBindingOutput := &masswallet.BindingOutput{
			HolderAddress:  num.HolderAddress,
			BindingAddress: num.BindingAddress,
			Amount:         val,
		}
		outPut = append(outPut, tempBindingOutput)
	}
	//construct binding transaction
	mtxHex, fee, err := s.massWallet.CreateBindingTransaction(in.FromAddress, txFee, outPut)
	if err != nil {
		logging.CPrint(logging.ERROR, "CreateBindingTransaction failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	err = checkTxFeeLimit(s.config.Config, fee)
	if err != nil {
		return nil, err
	}

	logging.CPrint(logging.INFO, "api: CreateBindingTransaction completed", logging.LogFormat{})
	return &pb.CreateRawTransactionResponse{Hex: mtxHex}, nil
}

func (s *APIServer) AutoCreateTransaction(ctx context.Context, in *pb.AutoCreateTransactionRequest) (*pb.CreateRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "api: AutoCreateTransaction", logging.LogFormat{"params": in})

	if err := checkLocktime(in.LockTime); err != nil {
		return nil, err
	}
	if err := checkNotEmpty(in.Amounts); err != nil {
		return nil, err
	}

	amounts := make(map[string]massutil.Amount)
	for addr, valStr := range in.Amounts {
		addr = strings.TrimSpace(addr)
		err := checkAddressLen(addr)
		if err != nil {
			return nil, err
		}
		valStr = strings.TrimSpace(valStr)
		val, err := checkParseAmount(valStr)
		if err != nil {
			return nil, err
		}
		amounts[addr] = val
	}

	txFee, err := checkParseAmount(in.Fee)
	if err != nil {
		return nil, status.New(ErrAPIUserTxFee, ErrCode[ErrAPIUserTxFee]).Err()
	}

	fromAddr := strings.TrimSpace(in.FromAddress)
	if len(fromAddr) > 0 {
		_, err = checkWitnessAddress(fromAddr, false, &cfg.ChainParams)
		if err != nil {
			return nil, err
		}
	}
	changeAddr := strings.TrimSpace(in.ChangeAddress)
	if len(changeAddr) > 0 {
		_, err = checkWitnessAddress(changeAddr, false, &cfg.ChainParams)
		if err != nil {
			return nil, err
		}
	}

	mtxHex, fee, err := s.massWallet.AutoCreateRawTransaction(amounts, in.LockTime, txFee, fromAddr, changeAddr)
	if err != nil {
		logging.CPrint(logging.ERROR, "AutoCreateRawTransaction failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	err = checkTxFeeLimit(s.config.Config, fee)
	if err != nil {
		return nil, err
	}

	logging.CPrint(logging.INFO, "api: AutoCreateTransaction completed", logging.LogFormat{})
	return &pb.CreateRawTransactionResponse{Hex: mtxHex}, nil
}

func (s *APIServer) GetTransactionFee(ctx context.Context, in *pb.GetTransactionFeeRequest) (*pb.GetTransactionFeeResponse, error) {
	logging.CPrint(logging.INFO, "api: GetTransactionFee", logging.LogFormat{"params": in})

	err := checkLocktime(in.LockTime)
	if err != nil {
		return nil, err
	}

	err = checkNotEmpty(in.Amounts)
	if err != nil {
		return nil, err
	}

	if len(s.massWallet.CurrentWallet()) == 0 {
		return nil, convertResponseError(masswallet.ErrNoWalletInUse)
	}

	txFee := massutil.ZeroAmount()

	if len(in.Inputs) == 0 {
		if in.HasBinding {
			gAddr := getEstimateBindingAddress()
			gOutputs := make([]*masswallet.BindingOutput, 0)
			for address, value := range in.Amounts {
				val, err := checkParseAmount(value)
				if err != nil {
					return nil, err
				}
				tempBindingOutput := &masswallet.BindingOutput{
					HolderAddress:  address,
					BindingAddress: gAddr,
					Amount:         val,
				}
				gOutputs = append(gOutputs, tempBindingOutput)
			}
			_, txFee, err = s.massWallet.EstimateBindingTxFee(gOutputs, 0, txFee, "", "")
			if err != nil {
				logging.CPrint(logging.ERROR, "EstimateBindingTxFee failed", logging.LogFormat{"err": err})
				cvtErr := convertResponseError(err)
				if cvtErr == apiUnknownError {
					return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
				}
				return nil, cvtErr
			}
		} else {
			// Normal transaction outputs cost the same fee as staking.
			outputs := make([]*masswallet.StakingTxOut, 0)
			lAddr := getEstimateStakingAddress()
			for _, amt := range in.Amounts {
				val, err := checkParseAmount(amt)
				if err != nil {
					return nil, err
				}
				output := &masswallet.StakingTxOut{
					Address:      lAddr,
					Amount:       val,
					FrozenPeriod: uint32(consensus.MinFrozenPeriod),
				}
				outputs = append(outputs, output)
			}

			_, txFee, err = s.massWallet.EstimateStakingTxFee(outputs, uint64(in.LockTime), massutil.ZeroAmount(), "", "")
			if err != nil {
				logging.CPrint(logging.ERROR, "EstimateStakingTxFee failed", logging.LogFormat{"err": err})
				cvtErr := convertResponseError(err)
				if cvtErr == apiUnknownError {
					return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
				}
				return nil, cvtErr
			}
		}
	} else {
		inputs := make([]*masswallet.TxIn, 0)
		for _, txInput := range in.Inputs {
			err := checkTransactionIdLen(txInput.TxId)
			if err != nil {
				return nil, err
			}
			input := &masswallet.TxIn{
				TxId: txInput.TxId,
				Vout: txInput.Vout,
			}
			inputs = append(inputs, input)
		}

		txFee, err = s.massWallet.EstimateManualTxFee(inputs, len(in.Amounts))
		if err != nil {
			logging.CPrint(logging.ERROR, "EstimateManualTxFee failed", logging.LogFormat{"err": err})
			cvtErr := convertResponseError(err)
			if cvtErr == apiUnknownError {
				return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
			}
			return nil, cvtErr
		}
	}
	fee, err := AmountToString(txFee.IntValue())
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to estimate manual transaction fee", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIUnknownErr, ErrCode[ErrAPIUnknownErr])
		return nil, st.Err()
	}
	logging.CPrint(logging.INFO, "api: GetTransactionFee completed", logging.LogFormat{"txFee": txFee})
	return &pb.GetTransactionFeeResponse{Fee: fee}, nil
}

func getEstimateBindingAddress() string {
	var h [ripemd160.Size]byte
	esAddr, _ := massutil.NewAddressPubKeyHash(h[:], &config.ChainParams)
	return esAddr.EncodeAddress()
}

func getEstimateStakingAddress() string {
	var h wire.Hash
	esAddr, _ := massutil.NewAddressStakingScriptHash(h[:], &config.ChainParams)
	return esAddr.EncodeAddress()
}
