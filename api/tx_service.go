package api

import (
	"encoding/hex"
	"sort"
	"strings"

	"github.com/massnetorg/mass-core/blockchain"
	"github.com/massnetorg/mass-core/consensus"
	"github.com/massnetorg/mass-core/consensus/forks"
	"github.com/massnetorg/mass-core/logging"
	"github.com/massnetorg/mass-core/massutil"
	"github.com/massnetorg/mass-core/poc"
	"github.com/massnetorg/mass-core/txscript"
	"github.com/massnetorg/mass-core/wire"
	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/masswallet"
	"massnet.org/mass-wallet/masswallet/utils"

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

	vouts, totalOutValue, err := createVoutList(mtx, config.ChainParams)
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
		TxId:          mtx.TxHash().String(),
		Version:       int32(mtx.Version),
		LockTime:      int64(mtx.LockTime),
		Size:          int32(mtx.PlainSize()),
		Vin:           make([]*pb.DecodeRawTransactionResponse_Vin, 0, len(mtx.TxIn)),
		Vout:          make([]*pb.DecodeRawTransactionResponse_Vout, 0, len(mtx.TxOut)),
		PayloadHex:    hex.EncodeToString(mtx.Payload),
		PayloadDecode: blockchain.DecodePayload(mtx.Payload).String(),
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
		val, err := AmountToString(txOut.Value)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to transfer amount to string", logging.LogFormat{"err": err})
			return nil, err
		}

		disbuf, err := txscript.DisasmString(txOut.PkScript)
		if err != nil {
			logging.CPrint(logging.WARN, "DisasmString error", logging.LogFormat{"err": err})
			return nil, err
		}

		scriptClass, recipient, staking, binding, _, err := extractAddressInfos(txOut.PkScript)
		if err != nil {
			return nil, err
		}

		decodedOut := &pb.DecodeRawTransactionResponse_Vout{
			Value:            val,
			N:                uint32(n),
			Type:             uint32(scriptClass),
			ScriptAsm:        disbuf,
			ScriptHex:        hex.EncodeToString(txOut.PkScript),
			RecipientAddress: recipient,
			StakingAddress:   staking,
			BindingTarget:    binding,
		}

		resp.Vout = append(resp.Vout, decodedOut)
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
func createVoutList(mtx *wire.MsgTx, chainParams *config.Params) (vouts []*pb.Vout, totalValue int64, err error) {
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
		scriptClass, recipient, staking, binding, reqSigs, err := extractAddressInfos(txout.PkScript)
		if err != nil {
			return nil, 0, err
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
				Asm:              disbuf,
				Hex:              hex.EncodeToString(txout.PkScript),
				ReqSigs:          int32(reqSigs),
				RecipientAddress: recipient,
				StakingAddress:   staking,
				BindingTarget:    binding,
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
		scriptClass, recipient, staking, binding, _, err := extractAddressInfos(prevVout.PkScript)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to decode address", logging.LogFormat{"err": err})
			return nil, 0, err
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
				TxId:           txIn.PreviousOutPoint.Hash.String(),
				Vout:           txIn.PreviousOutPoint.Index,
				Sequence:       txIn.Sequence,
				Witness:        witnessToHex(txIn.Witness),
				FromAddress:    recipient,
				StakingAddress: staking,
				BindingTarget:  binding,
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

	err = checkTxFeeLimit(s.config, fee)
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

	if !wire.IsValidStakingValue(val.IntValue()) {
		return nil, status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount]).Err()
	}

	valFee, err := checkParseAmount(in.Fee)
	if err != nil {
		return nil, status.New(ErrAPIUserTxFee, ErrCode[ErrAPIUserTxFee]).Err()
	}

	if len(in.FromAddress) > 0 {
		_, err = checkWitnessAddress(in.FromAddress, false, config.ChainParams)
		if err != nil {
			return nil, err
		}
	}

	_, err = checkWitnessAddress(in.StakingAddress, true, config.ChainParams)
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

	err = checkTxFeeLimit(s.config, fee)
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
		_, err = checkWitnessAddress(in.FromAddress, false, config.ChainParams)
		if err != nil {
			return nil, err
		}
	}

	bindings := make([]*masswallet.BindingOutput, 0)
	for _, output := range in.Outputs {
		holder, err := checkWitnessAddress(output.HolderAddress, false, config.ChainParams)
		if err != nil {
			return nil, err
		}
		target, err := parseBindingTarget(output.BindingAddress, config.ChainParams)
		if err != nil {
			return nil, err
		}

		val, err := checkParseAmount(output.Amount)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, &masswallet.BindingOutput{
			Holder:        holder,
			BindingTarget: target,
			Amount:        val,
		})
	}
	//construct binding transaction
	mtxHex, fee, err := s.massWallet.CreateBindingTransaction(in.FromAddress, txFee, bindings)
	if err != nil {
		logging.CPrint(logging.ERROR, "CreateBindingTransaction failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	err = checkTxFeeLimit(s.config, fee)
	if err != nil {
		return nil, err
	}

	logging.CPrint(logging.INFO, "api: CreateBindingTransaction completed", logging.LogFormat{})
	return &pb.CreateRawTransactionResponse{Hex: mtxHex}, nil
}

func (s *APIServer) CreatePoolPkCoinbaseTransaction(ctx context.Context, in *pb.CreatePoolPkCoinbaseTransactionRequest) (*pb.CreateRawTransactionResponse, error) {
	from, err := checkWitnessAddress(strings.TrimSpace(in.FromAddress), false, config.ChainParams)
	if err != nil {
		return nil, err
	}

	raw, err := hex.DecodeString(in.Payload)
	if err != nil {
		return nil, status.New(ErrAPIInvalidParameter, "invalid hex-encoded payload").Err()
	}
	payload := blockchain.DecodePayload(raw)
	if payload == nil || payload.Method != blockchain.BindPoolCoinbase {
		return nil, status.New(ErrAPIInvalidParameter, "unexpected payload").Err()
	}

	value, _ := massutil.NewAmountFromInt(1000000) // 0.01 MASS
	outputs := map[string]massutil.Amount{
		from.EncodeAddress(): value,
	}

	requiredCost, _ := massutil.NewAmountFromInt(int64(consensus.MASSIP0002SetPoolPkCoinbaseFee))

	mtxHex, fee, err := s.massWallet.AutoCreateRawTransaction(outputs, 0, requiredCost, from.EncodeAddress(), from.EncodeAddress(), raw)
	if err != nil {
		logging.CPrint(logging.ERROR, "CreateSetPoolPkCoinbaseTransaction failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	max, err := checkParseAmount(s.config.Wallet.Settings.MaxTxFee)
	if err != nil {
		max, _ = checkParseAmount(config.DefaultMaxTxFee)
	}
	max, _ = max.AddInt(int64(consensus.MASSIP0002SetPoolPkCoinbaseFee))
	if max.Cmp(fee) < 0 {
		logging.CPrint(logging.ERROR, "big transaction fee", logging.LogFormat{"fee": fee, "limit": max})
		st := status.New(ErrAPIBigTransactionFee, ErrCode[ErrAPIBigTransactionFee])
		return nil, st.Err()
	}
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
		_, err = checkWitnessAddress(fromAddr, false, config.ChainParams)
		if err != nil {
			return nil, err
		}
	}
	changeAddr := strings.TrimSpace(in.ChangeAddress)
	if len(changeAddr) > 0 {
		_, err = checkWitnessAddress(changeAddr, false, config.ChainParams)
		if err != nil {
			return nil, err
		}
	}

	mtxHex, fee, err := s.massWallet.AutoCreateRawTransaction(amounts, in.LockTime, txFee, fromAddr, changeAddr, nil)
	if err != nil {
		logging.CPrint(logging.ERROR, "AutoCreateRawTransaction failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	err = checkTxFeeLimit(s.config, fee)
	if err != nil {
		return nil, err
	}

	logging.CPrint(logging.INFO, "api: AutoCreateTransaction completed", logging.LogFormat{})
	return &pb.CreateRawTransactionResponse{Hex: mtxHex}, nil
}

func (s *APIServer) GetTransactionFee(ctx context.Context, in *pb.GetTransactionFeeRequest) (*pb.GetTransactionFeeResponse, error) {
	logging.CPrint(logging.INFO, "api: GetTransactionFee", logging.LogFormat{"params": in})

	err := checkNotEmpty(in.Amounts)
	if err != nil {
		return nil, err
	}

	if len(s.massWallet.CurrentWallet()) == 0 {
		return nil, convertResponseError(masswallet.ErrNoWalletInUse)
	}

	txFee := massutil.ZeroAmount()

	if len(in.Inputs) == 0 {
		if in.HasBinding {
			target := mockBindingTarget()
			bindings := make([]*masswallet.BindingOutput, 0)
			for address, value := range in.Amounts {
				holder, err := checkWitnessAddress(address, false, config.ChainParams)
				if err != nil {
					return nil, err
				}
				val, err := checkParseAmount(value)
				if err != nil {
					return nil, err
				}
				bindings = append(bindings, &masswallet.BindingOutput{
					Holder:        holder,
					BindingTarget: target,
					Amount:        val,
				})
			}
			_, txFee, err = s.massWallet.EstimateBindingTxFee(bindings, 0, txFee, "", "")
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

			_, txFee, err = s.massWallet.EstimateStakingTxFee(outputs, 0, massutil.ZeroAmount(), "", "")
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

// Returns new binding
func mockBindingTarget() massutil.Address {
	var h [ripemd160.Size + 2]byte
	h[21] = 32
	target, _ := massutil.NewAddressBindingTarget(h[:], config.ChainParams)
	return target
}

func getEstimateStakingAddress() string {
	var h wire.Hash
	esAddr, _ := massutil.NewAddressStakingScriptHash(h[:], config.ChainParams)
	return esAddr.EncodeAddress()
}

func (s *APIServer) TxHistory(ctx context.Context, in *pb.TxHistoryRequest) (*pb.TxHistoryResponse, error) {
	logging.CPrint(logging.INFO, "api: TxHistory", logging.LogFormat{
		"count":   in.Count,
		"address": in.Address,
	})

	if len(in.Address) > 0 {
		err := checkAddressLen(in.Address)
		if err != nil {
			return nil, err
		}
	}
	if in.Count > 1000 {
		return nil, status.New(ErrAPIInvalidTxHistoryCount, ErrCode[ErrAPIInvalidTxHistoryCount]).Err()
	}

	histories, err := s.massWallet.GetTxHistory(int(in.Count), in.Address)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetTxHistory failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	sort.Slice(histories, func(i, j int) bool {
		return histories[i].BlockHeight > histories[j].BlockHeight
	})
	reps := &pb.TxHistoryResponse{
		Histories: histories,
	}

	logging.CPrint(logging.INFO, "api: TxHistory completed", logging.LogFormat{"num": len(histories)})
	return reps, nil
}

func (s *APIServer) GetStakingHistory(ctx context.Context, in *pb.GetStakingHistoryRequest) (*pb.GetStakingHistoryResponse, error) {
	logging.CPrint(logging.INFO, "api: GetStakingHistory", logging.LogFormat{})
	newestHeight := s.node.Blockchain().BestBlockHeight()
	rewards, err := s.node.Blockchain().GetUnexpiredStakingRank(newestHeight)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to FetchAllLockTx from chainDB", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIGetStakingTxDetail, ErrCode[ErrAPIGetStakingTxDetail])
		return nil, st.Err()
	}
	excludeWithdrawn := true
	if in.Type == "all" {
		excludeWithdrawn = false
	}
	stakingTxs, err := s.massWallet.GetStakingHistory(excludeWithdrawn)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to GetStakingHistory from walletDB", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIGetStakingTxDetail, ErrCode[ErrAPIGetStakingTxDetail])
		return nil, st.Err()
	}

	weights := make(map[string]float64)
	txs := make([]*pb.GetStakingHistoryResponse_Tx, 0)

	for _, lTx := range stakingTxs {
		amt, err := AmountToString(lTx.Utxo.Amount.IntValue())
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to transfer amount to string", logging.LogFormat{
				"err": err,
			})
			st := status.New(ErrAPIGetStakingTxDetail, ErrCode[ErrAPIGetStakingTxDetail])
			return nil, st.Err()
		}
		tx := &pb.GetStakingHistoryResponse_Tx{
			TxId:        lTx.TxHash.String(),
			BlockHeight: lTx.BlockHeight,
			Utxo: &pb.GetStakingHistoryResponse_StakingUTXO{
				TxId:         lTx.Utxo.Hash.String(),
				Vout:         lTx.Utxo.Index,
				Address:      lTx.Utxo.Address,
				Amount:       amt,
				FrozenPeriod: lTx.Utxo.FrozenPeriod,
			},
		}
		if tx.BlockHeight == 0 {
			tx.Status = stakingStatusPending
		} else {
			confs := newestHeight - lTx.BlockHeight
			switch {
			case confs < consensus.StakingTxRewardStart:
				tx.Status = stakingStatusImmature
			case confs > uint64(lTx.Utxo.FrozenPeriod):
				switch {
				case lTx.Utxo.Spent:
					tx.Status = stakingStatusWithdrawn
				case lTx.Utxo.SpentByUnmined:
					tx.Status = stakingStatusWithdrawing
				default:
					tx.Status = stakingStatusExpired
				}
			default:
				tx.Status = stakingStatusMature
			}
		}
		txs = append(txs, tx)
		_, ok := weights[tx.Utxo.Address]
		if ok {
			continue
		}

		for _, reward := range rewards {
			scriptHash := make([]byte, 0)
			scriptHash = append(scriptHash, reward.ScriptHash[:]...)

			scriptHashStruct, err := massutil.NewAddressStakingScriptHash(scriptHash, config.ChainParams)
			if err != nil {
				logging.CPrint(logging.ERROR, "create addressLocktimeScriptHash failed",
					logging.LogFormat{
						"err": err,
					})
				st := status.New(ErrAPIGetStakingTxDetail, ErrCode[ErrAPIGetStakingTxDetail])
				return nil, st.Err()
			}
			witAddress := scriptHashStruct.EncodeAddress()
			if strings.Compare(witAddress, tx.Utxo.Address) == 0 {
				weights[witAddress] = reward.Weight.Float64()
				break
			}
		}
	}
	reply := &pb.GetStakingHistoryResponse{
		Txs:     txs,
		Weights: weights,
	}

	logging.CPrint(logging.INFO, "api: GetStakingHistory completed", logging.LogFormat{
		"num": len(stakingTxs),
	})
	return reply, nil
}

func (s *APIServer) GetBindingHistory(ctx context.Context, in *pb.GetBindingHistoryRequest) (*pb.GetBindingHistoryResponse, error) {
	logging.CPrint(logging.INFO, "api: GetBindingHistory", logging.LogFormat{})

	excludeWithdrawn := true
	if in.Type == "all" {
		excludeWithdrawn = false
	}
	details, err := s.massWallet.GetBindingHistory(excludeWithdrawn)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetBindingHistory failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	histories := make([]*pb.GetBindingHistoryResponse_History, 0)
	for _, detail := range details {
		amt, err := checkFormatAmount(detail.Utxo.Amount)
		if err != nil {
			return nil, err
		}
		froms := make([]string, 0)

		if detail.IsDeposit() {
			if detail.MsgTx == nil {
				logging.CPrint(logging.ERROR, "transaction not found", logging.LogFormat{
					"tx": detail.TxHash.String(),
				})
			} else {
				if blockchain.IsCoinBaseTx(detail.MsgTx) {
					froms = append(froms, "COINBASE")
				} else {
					fromSet := make(map[string]struct{}, 0)
					for _, txIn := range detail.MsgTx.TxIn {
						list, err := s.node.Blockchain().GetTransactionInDB(&txIn.PreviousOutPoint.Hash)
						if err != nil || len(list) == 0 {
							logging.CPrint(logging.ERROR, "transaction not found", logging.LogFormat{
								"tx":  txIn.PreviousOutPoint.Hash.String(),
								"err": err,
							})
							return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
						} else {
							prevMtx := list[len(list)-1].Tx
							ps, err := utils.ParsePkScript(prevMtx.TxOut[txIn.PreviousOutPoint.Index].PkScript, config.ChainParams)
							if err != nil {
								logging.CPrint(logging.ERROR, "ParsePkScript failed", logging.LogFormat{
									"err": err,
								})
								return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
							}
							fromSet[ps.StdEncodeAddress()] = struct{}{}
						}
					}
					for from := range fromSet {
						froms = append(froms, from)
					}
				}
			}
		}

		history := &pb.GetBindingHistoryResponse_History{
			TxId:        detail.TxHash.String(),
			BlockHeight: detail.BlockHeight,
			Utxo: &pb.GetBindingHistoryResponse_BindingUTXO{
				TxId:          detail.Utxo.Hash.String(),
				Vout:          detail.Utxo.Index,
				HolderAddress: detail.Utxo.Holder.EncodeAddress(),
				BindingTarget: detail.Utxo.BindingTarget.EncodeAddress(),
				Amount:        amt,
			},
			FromAddresses: froms,
		}
		{
			history.Utxo.TargetType = "MASS"
			if target, ok := detail.Utxo.BindingTarget.(*massutil.AddressBindingTarget); ok {
				if target.ScriptAddress()[20] == 1 {
					history.Utxo.TargetType = "Chia"
				}
				history.Utxo.TargetSize = uint32(target.ScriptAddress()[21])
			}
		}
		switch {
		case history.BlockHeight == 0:
			history.Status = bindingStatusPending
		case detail.Utxo.Spent:
			history.Status = bindingStatusWithdrawn
		case detail.Utxo.SpentByUnmined:
			history.Status = bindingStatusWithdrawing
		default:
			history.Status = bindingStatusConfirmed
		}
		histories = append(histories, history)
	}

	logging.CPrint(logging.INFO, "api: GetBindingHistory completed", logging.LogFormat{})
	return &pb.GetBindingHistoryResponse{Histories: histories}, nil
}

func (s *APIServer) SendRawTransaction(ctx context.Context, in *pb.SendRawTransactionRequest) (*pb.SendRawTransactionResponse, error) {
	// Deserialize and send off to tx relay
	logging.CPrint(logging.INFO, "api: SendRawTransaction", logging.LogFormat{})
	if len(in.Hex) == 0 {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidTxHex], logging.LogFormat{
			"err": in.Hex})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}
	serializedTx, err := decodeHexStr(in.Hex)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to decode txHex", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}
	msgtx := wire.NewMsgTx()
	err = msgtx.SetBytes(serializedTx, wire.Packet)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to deserialize transaction", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}

	if !forks.EnforceMASSIP0002WarmUp(s.node.Blockchain().BestBlockHeight()) {
		if payload := blockchain.DecodePayload(msgtx.Payload); payload != nil && payload.Method == blockchain.BindPoolCoinbase {
			return nil, status.New(ErrAPIUnacceptable, "cannot yet set coinbase for pool pk").Err()
		}
		for _, txout := range msgtx.TxOut {
			if _, target, _ := txscript.GetBindingScriptHash(txout.PkScript); len(target) == 22 {
				return nil, status.New(ErrAPIUnacceptable, "cannot yet send new binding transaction").Err()
			}
		}
	} else {
		for _, txout := range msgtx.TxOut {
			if _, target, _ := txscript.GetBindingScriptHash(txout.PkScript); len(target) == 20 {
				return nil, status.New(ErrAPIUnacceptable, "deprecated binding transaction").Err()
			}
		}
	}

	tx := massutil.NewTx(msgtx)
	_, err = s.node.Blockchain().ProcessTx(tx)
	if err != nil {
		logging.CPrint(logging.ERROR, "ProcessTx failed", logging.LogFormat{"err": err})
		s.massWallet.ClearUsedUTXOMark(msgtx)
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIRejectTx, err.Error()).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: SendRawTransaction completed", logging.LogFormat{
		"txHash": tx.Hash().String(),
	})
	return &pb.SendRawTransactionResponse{TxId: tx.Hash().String()}, nil
}

func (s *APIServer) GetNetworkBinding(ctx context.Context, in *pb.GetNetworkBindingRequest) (*pb.GetNetworkBindingResponse, error) {
	height := in.Height
	if height == 0 || height > s.node.Blockchain().BestBlockHeight() {
		height = s.node.Blockchain().BestBlockHeight()
	}
	networkTotal, err := s.node.Blockchain().GetNetworkBinding(height)
	if err != nil {
		return nil, status.New(ErrAPIQueryDataFailed, err.Error()).Err()
	}

	resp := &pb.GetNetworkBindingResponse{
		Height:                    height,
		TotalBinding:              networkTotal.String(),
		BindingPriceMassBitlength: make(map[uint32]string),
		BindingPriceChiaK:         make(map[uint32]string),
	}

	// mass
	for bl := 32; bl <= 40; bl += 2 {
		required, err := forks.GetRequiredBinding(height, poc.PlotSize(poc.ProofTypeDefault, bl), bl, networkTotal)
		if err != nil {
			return nil, status.New(ErrAPIQueryDataFailed, err.Error()).Err()
		}
		resp.BindingPriceMassBitlength[uint32(bl)] = required.String()
	}

	// chia
	if forks.EnforceMASSIP0002WarmUp(height) {
		for bl := 32; bl <= 40; bl++ {
			required, err := forks.GetRequiredBinding(height, poc.PlotSize(poc.ProofTypeChia, bl), bl, networkTotal)
			if err != nil {
				return nil, status.New(ErrAPIQueryDataFailed, err.Error()).Err()
			}
			resp.BindingPriceChiaK[uint32(bl)] = required.String()
		}
	}

	return resp, nil
}

func (s *APIServer) CheckPoolPkCoinbase(ctx context.Context, in *pb.CheckPoolPkCoinbaseRequest) (*pb.CheckPoolPkCoinbaseResponse, error) {
	poolPks := make([][]byte, 0, len(in.PoolPubkeys))
	for _, pkStr := range in.PoolPubkeys {
		pk, err := hex.DecodeString(pkStr)
		if err != nil {
			return nil, status.New(ErrAPIInvalidParameter, "pubkey cannot be decoded").Err()
		}
		poolPks = append(poolPks, pk)
	}
	pkToCb, pkToNonce, err := s.node.Blockchain().GetPoolPkCoinbase(poolPks)
	if err != nil {
		return nil, status.New(ErrAPIQueryDataFailed, err.Error()).Err()
	}

	result := make(map[string]*pb.CheckPoolPkCoinbaseResponse_Info)
	for pk, cb := range pkToCb {
		result[pk] = &pb.CheckPoolPkCoinbaseResponse_Info{
			Coinbase: cb,
			Nonce:    pkToNonce[pk],
		}
	}
	return &pb.CheckPoolPkCoinbaseResponse{Result: result}, nil
}

func (s *APIServer) CheckTargetBinding(ctx context.Context, in *pb.CheckTargetBindingRequest) (*pb.CheckTargetBindingResponse, error) {
	infos := make(map[string]*pb.CheckTargetBindingResponse_Info)
	for _, addr := range in.Targets {
		addr = strings.TrimSpace(addr)
		if _, ok := infos[addr]; ok {
			continue
		}
		target, err := parseBindingTarget(addr, config.ChainParams)
		if err != nil {
			infos[addr] = &pb.CheckTargetBindingResponse_Info{TargetType: "Unknown"}
			logging.CPrint(logging.INFO, "failed to decode target", logging.LogFormat{"target": addr, "err": err})
			continue
		}

		info := &pb.CheckTargetBindingResponse_Info{
			TargetType: "MASS",
		}
		infos[addr] = info

		amount := massutil.ZeroAmount()
		if massutil.IsAddressPubKeyHash(target) { // old binding
			list, err := s.node.Blockchain().FetchOldBinding(target.ScriptAddress())
			if err != nil {
				logging.CPrint(logging.ERROR, "failed to get old binding", logging.LogFormat{"target": addr, "err": err})
				return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
			}
			for _, binding := range list {
				amount, err = amount.AddInt(binding.Value)
				if err != nil {
					logging.CPrint(logging.ERROR, "amount error", logging.LogFormat{"err": err})
					return nil, status.New(ErrAPIAbnormalData, err.Error()).Err()
				}
			}
		} else { // new binding
			amount, err = s.node.Blockchain().GetNewBinding(target.ScriptAddress())
			if err != nil {
				logging.CPrint(logging.ERROR, "failed to get new biniding", logging.LogFormat{"target": addr, "err": err})
				return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
			}
			if target.ScriptAddress()[20] == byte(poc.ProofTypeChia) {
				info.TargetType = "Chia"
			}
			info.TargetSize = uint32(target.ScriptAddress()[21])
		}

		info.Amount = amount.String()
	}
	return &pb.CheckTargetBindingResponse{Result: infos}, nil
}
