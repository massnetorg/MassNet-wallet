package api

import (
	"bytes"
	"encoding/hex"

	pb "github.com/massnetorg/MassNet-wallet/api/proto"
	"github.com/massnetorg/MassNet-wallet/blockchain"
	cfg "github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/logging"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/txscript"
	"github.com/massnetorg/MassNet-wallet/wire"

	"golang.org/x/net/context"
	"google.golang.org/grpc/status"
)

// messageToHex serializes a message to the wire protocol encoding using the
// latest protocol version and returns a hex-encoded string of the result.
func messageToHex(msg wire.Message) (string, error) {
	var buf bytes.Buffer
	if err := msg.MassEncode(&buf, wire.Packet); err != nil {
		st := status.New(ErrAPIEncode, ErrCode[ErrAPIEncode])
		return "", st.Err()
	}

	return hex.EncodeToString(buf.Bytes()), nil
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
func createVoutList(mtx *wire.MsgTx, chainParams *cfg.Params, filterAddrMap map[string]struct{}) ([]*pb.Vout, error) {
	voutList := make([]*pb.Vout, 0, len(mtx.TxOut))
	for i, v := range mtx.TxOut {
		// reset filter flag for each.
		passesFilter := len(filterAddrMap) == 0

		disbuf, err := txscript.DisasmString(v.PkScript)
		if err != nil {
			logging.CPrint(logging.WARN, "decode pkscript to asm exists err", logging.LogFormat{"err": err})
			st := status.New(ErrAPIDisasmScript, ErrCode[ErrAPIDisasmScript])
			return nil, st.Err()
		}

		scriptClass, addrs, _, reqSigs, err := txscript.ExtractPkScriptAddrs(
			v.PkScript, chainParams)
		if err != nil {
			st := status.New(ErrAPIExtractPKScript, ErrCode[ErrAPIExtractPKScript])
			return nil, st.Err()
		}

		encodedAddrs := make([]string, len(addrs))
		for j, addr := range addrs {
			encodedAddrs[j] = addr.EncodeAddress()

			if len(filterAddrMap) > 0 {
				if _, exists := filterAddrMap[encodedAddrs[j]]; exists {
					passesFilter = true
				}
			}
		}

		if !passesFilter {
			continue
		}

		vout := &pb.Vout{
			N:     uint32(i),
			Value: massutil.Amount(v.Value).ToMASS(),
			ScriptPubKey: &pb.ScriptPubKeyResult{
				Asm:       disbuf,
				Hex:       hex.EncodeToString(v.PkScript),
				ReqSigs:   int32(reqSigs),
				Type:      scriptClass.String(),
				Addresses: encodedAddrs,
			},
		}

		voutList = append(voutList, vout)
	}

	return voutList, nil
}

// createVinList returns a slice of JSON objects for the inputs of the passed
// transaction.
func (s *Server) createVinList(mtx *wire.MsgTx) ([]*pb.Vin, []string, []*pb.InputsInTx, float64, error) {
	vinList := make([]*pb.Vin, len(mtx.TxIn))
	addrs := make([]string, 0)
	inputs := make([]*pb.InputsInTx, 0)
	totalInValue := 0.0
	if blockchain.IsCoinBaseTx(mtx) {
		txIn := mtx.TxIn[0]
		vinTemp := &pb.Vin{
			Sequence: txIn.Sequence,
			Witness:  witnessToHex(txIn.Witness),
		}
		vinList[0] = vinTemp

		for i, txIn := range mtx.TxIn[1:] {
			vinTemp := &pb.Vin{
				Txid:     txIn.PreviousOutPoint.Hash.String(),
				Vout:     txIn.PreviousOutPoint.Index,
				Sequence: txIn.Sequence,
			}
			vinList[i+1] = vinTemp
		}

		return vinList, addrs, inputs, totalInValue, nil
	}

	for i, txIn := range mtx.TxIn {
		vinEntry := &pb.Vin{
			Txid:     txIn.PreviousOutPoint.Hash.String(),
			Vout:     txIn.PreviousOutPoint.Index,
			Sequence: txIn.Sequence,
			Witness:  witnessToHex(txIn.Witness),
		}
		vinList[i] = vinEntry

		addr, inValue, err := s.getTxInAddr(&txIn.PreviousOutPoint.Hash, txIn.PreviousOutPoint.Index)
		if err != nil {
			logging.CPrint(logging.ERROR, "No information available about transaction in db", logging.LogFormat{"err": err.Error(), "txid": txIn.PreviousOutPoint.Hash.String()})
			st := status.New(ErrAPINoTxInfo, ErrCode[ErrAPINoTxInfo])
			return nil, nil, nil, totalInValue, st.Err()
		}
		totalInValue = totalInValue + inValue
		inputs = append(inputs, &pb.InputsInTx{Txid: txIn.PreviousOutPoint.Hash.String(), Index: txIn.PreviousOutPoint.Index, Address: addr, Value: inValue})
	}

	return vinList, addrs, inputs, totalInValue, nil
}

func (s *Server) getTxInAddr(txid *wire.Hash, index uint32) (string, float64, error) {
	addr := ""
	inValue := 0.0
	tx, err := s.txMemPool.FetchTransaction(txid)
	var inmtx *wire.MsgTx
	if err != nil {
		txReply, err := s.db.FetchTxBySha(txid)
		if err != nil || len(txReply) == 0 {
			logging.CPrint(logging.ERROR, "No information available about transaction in db", logging.LogFormat{"err": err, "txid": txid.String()})
			st := status.New(ErrAPINoTxInfo, ErrCode[ErrAPINoTxInfo])
			return addr, inValue, st.Err()
		}
		lastTx := txReply[len(txReply)-1]
		inmtx = lastTx.Tx
	} else {
		inmtx = tx.MsgTx()
	}

	addrWS, err := massutil.NewAddressWitnessScriptHash(inmtx.TxOut[int(index)].PkScript[2:], &cfg.ChainParams)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to encode address", logging.LogFormat{"err": err})
		st := status.New(ErrAPICreatePkScript, ErrCode[ErrAPICreatePkScript])
		return addr, inValue, st.Err()
	}
	inValue = massutil.Amount(inmtx.TxOut[int(index)].Value).ToMASS()
	return addrWS.String(), inValue, nil
}

func (s *Server) ValidateAddress(ctx context.Context, in *pb.ValidateAddressRequest) (*pb.ValidateAddressResponse, error) {
	logging.CPrint(logging.INFO, "a request is received to check the validity of the address", logging.LogFormat{"address": in.Address})
	addr, err := massutil.DecodeAddress(in.Address, &cfg.ChainParams)
	if err != nil {
		logging.CPrint(logging.INFO, "invalid address", logging.LogFormat{"address": in.Address})
		return &pb.ValidateAddressResponse{
			IsValid: false,
		}, nil
	}
	logging.CPrint(logging.INFO, "valid address", logging.LogFormat{"address": in.Address})
	return &pb.ValidateAddressResponse{
		IsValid: true,
		Address: addr.EncodeAddress(),
	}, nil
}

func (s *Server) GetTxStatus(ctx context.Context, in *pb.GetTxStatusRequest) (*pb.GetTxStatusResponse, error) {
	logging.CPrint(logging.INFO, "a request is received to query the status of the transaction according to the transaction hash", logging.LogFormat{"transaction hash": in.TxId})
	txHash, err := wire.NewHashFromStr(in.TxId)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decode the input string into hash", logging.LogFormat{"input string": in.TxId, "error": err})
		st := status.New(ErrAPIShaHashFromStr, ErrCode[ErrAPIShaHashFromStr])
		return nil, st.Err()
	}

	txList, err := s.db.FetchTxBySha(txHash)
	if err != nil || len(txList) == 0 {
		_, err := s.txMemPool.FetchTransaction(txHash)
		if err != nil {
			logging.CPrint(logging.INFO, "failed to query the status of the transaction since the transaction could not be found in txPool or database according to the transaction hash",
				logging.LogFormat{"transaction hash": in.TxId})
			return &pb.GetTxStatusResponse{
				Code:   "-1",
				Status: "failed",
			}, nil
		} else {
			logging.CPrint(logging.INFO, "the request to query the status of the transaction according to the transaction hash was successfully answered, "+
				"the transaction is waiting to be packaged", logging.LogFormat{"transaction hash": in.TxId})
			return &pb.GetTxStatusResponse{
				Code:   "2",
				Status: "packing",
			}, nil
		}
	}

	lastTx := txList[len(txList)-1]
	txHeight := lastTx.Height
	_, bestHeight, err := s.db.NewestSha()
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to query the best block height", logging.LogFormat{"error": err})
		st := status.New(ErrAPINewestHash, ErrCode[ErrAPINewestHash])
		return nil, st.Err()
	}
	confirmations := 1 + bestHeight - txHeight
	if confirmations < 0 {
		logging.CPrint(logging.INFO, "failed to query the status of the transaction since the transaction could not be found in txPool or database according to the transaction hash",
			logging.LogFormat{"transaction hash": in.TxId})
		return &pb.GetTxStatusResponse{
			Code:   "-1",
			Status: "failed",
		}, nil
	}
	if blockchain.IsCoinBaseTx(lastTx.Tx) {
		if confirmations >= blockchain.CoinbaseMaturity {
			logging.CPrint(logging.INFO, "the request to query the status of the transaction according to the transaction hash was successfully answered, "+
				"the transaction have been confirmed", logging.LogFormat{"transaction hash": in.TxId, "mined block hash": lastTx.BlkSha.String(), "mined block height": txHeight, "confirmations": confirmations})
			return &pb.GetTxStatusResponse{
				Code:   "4",
				Status: "succeed",
			}, nil
		} else {
			logging.CPrint(logging.INFO, "the request to query the status of the transaction according to the transaction hash was successfully answered, "+
				"the transaction have been packaged but not confirmed", logging.LogFormat{"transaction hash": in.TxId, "mined block hash": lastTx.BlkSha.String(), "mined block height": txHeight, "confirmations": confirmations})
			return &pb.GetTxStatusResponse{
				Code:   "3",
				Status: "confirming",
			}, nil
		}
	}
	if confirmations >= blockchain.TransactionMaturity {
		logging.CPrint(logging.INFO, "the request to query the status of the transaction according to the transaction hash was successfully answered, "+
			"the transaction have been confirmed", logging.LogFormat{"transaction hash": in.TxId, "mined block hash": lastTx.BlkSha.String(), "mined block height": txHeight, "confirmations": confirmations})
		return &pb.GetTxStatusResponse{
			Code:   "4",
			Status: "succeed",
		}, nil
	} else {
		logging.CPrint(logging.INFO, "the request to query the status of the transaction according to the transaction hash was successfully answered, "+
			"the transaction have been packaged but not confirmed", logging.LogFormat{"transaction hash": in.TxId, "mined block hash": lastTx.BlkSha.String(), "mined block height": txHeight, "confirmations": confirmations})
		return &pb.GetTxStatusResponse{
			Code:   "3",
			Status: "confirming",
		}, nil
	}
}

func (s *Server) getStatus(txHash *wire.Hash) (code int32, err error) {
	txList, err := s.db.FetchTxBySha(txHash)
	if err != nil || len(txList) == 0 {
		_, err := s.txMemPool.FetchTransaction(txHash)
		if err != nil {
			code = -1
			return code, nil
		} else {
			code = 2
			return code, nil
		}
	}

	lastTx := txList[len(txList)-1]
	txHeight := lastTx.Height
	_, bestHeight, err := s.db.NewestSha()
	if err != nil {
		st := status.New(ErrAPINewestHash, ErrCode[ErrAPINewestHash])
		return 0, st.Err()
	}
	confirmations := 1 + bestHeight - txHeight
	if confirmations < 0 {
		code = -1
		return code, nil
	}
	if blockchain.IsCoinBaseTx(lastTx.Tx) {
		if confirmations >= blockchain.CoinbaseMaturity {
			code = 4
			return code, nil
		} else {
			code = 3
			return code, nil
		}
	}
	if confirmations >= blockchain.TransactionMaturity {
		code = 4
		return code, nil
	} else {
		code = 3
		return code, nil
	}
}

// GetRawTransaction responds to the getRawTransaction request.
func (s *Server) GetRawTransaction(ctx context.Context, in *pb.GetRawTransactionRequest) (*pb.GetRawTransactionResponse, error) {
	logging.CPrint(logging.INFO,
		"a request is received to query information of transaction according to the transaction hash",
		logging.LogFormat{"hash": in.TxId})
	txHash, err := wire.NewHashFromStr(in.TxId)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decode the input string into hash", logging.LogFormat{"input string": in.TxId, "error": err})
		st := status.New(ErrAPIShaHashFromStr, ErrCode[ErrAPIShaHashFromStr])
		return nil, st.Err()
	}

	var mtx *wire.MsgTx
	var blkHash *wire.Hash
	var blkHeight int32
	tx, err := s.txMemPool.FetchTransaction(txHash)
	if err != nil {
		txList, err := s.db.FetchTxBySha(txHash)
		if err != nil || len(txList) == 0 {
			logging.CPrint(logging.ERROR, "failed to query the transaction information in txPool or database according to the transaction hash", logging.LogFormat{"hash": txHash, "error": err})
			st := status.New(ErrAPINoTxInfo, ErrCode[ErrAPINoTxInfo])
			return nil, st.Err()
		}

		lastTx := txList[len(txList)-1]
		mtx = lastTx.Tx
		blkHash = lastTx.BlkSha
		blkHeight = lastTx.Height
	} else {
		mtx = tx.MsgTx()
	}

	var blkHeader *wire.BlockHeader
	var blkHashStr string
	var chainHeight int32
	if blkHash != nil {
		blkHeader, err = s.db.FetchBlockHeaderBySha(blkHash)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to query the block header according to the block hash", logging.LogFormat{"block hash": blkHash.String(), "error": err})
			st := status.New(ErrAPIBlockHeaderNotFound, ErrCode[ErrAPIBlockHeaderNotFound])
			return nil, st.Err()
		}

		_, chainHeight, err = s.db.NewestSha()
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to query the best block height", logging.LogFormat{"error": err})
			st := status.New(ErrAPINewestHash, ErrCode[ErrAPINewestHash])
			return nil, st.Err()
		}

		blkHashStr = blkHash.String()
	}

	rawTxn, err := s.createTxRawResult(&cfg.ChainParams, mtx,
		txHash.String(), blkHeader, blkHashStr, blkHeight, chainHeight)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to query information of transaction according to the transaction hash", logging.LogFormat{"hash": in.TxId, "error": err})
		st := status.New(ErrAPIRawTx, ErrCode[ErrAPIRawTx])
		return nil, st.Err()
	}
	logging.CPrint(logging.INFO, "the request to query information of transaction according to the transaction hash was successfully answered", logging.LogFormat{"hash": in.TxId})
	return &pb.GetRawTransactionResponse{RawTxn: rawTxn}, nil
}

// createTxRawResult converts the passed transaction and associated parameters
// to a raw transaction JSON object.
func (s *Server) createTxRawResult(chainParams *cfg.Params, mtx *wire.MsgTx,
	txHash string, blkHeader *wire.BlockHeader, blkHash string,
	blkHeight int32, chainHeight int32) (*pb.TxRawResult, error) {

	mtxHex, err := messageToHex(mtx)
	if err != nil {
		return nil, err
	}

	voutList, err := createVoutList(mtx, chainParams, nil)
	if err != nil {
		return nil, err
	}
	totalOutValue := 0.0
	to := make([]*pb.ToAddressForTx, 0)
	for _, txout := range mtx.TxOut {
		if len(txout.PkScript) != 22 {
			continue
		}
		addr, err := massutil.NewAddressWitnessScriptHash(txout.PkScript[2:], &cfg.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to encode address", logging.LogFormat{"err": err})
			st := status.New(ErrAPICreatePkScript, ErrCode[ErrAPICreatePkScript])
			return nil, st.Err()
		}
		value := massutil.Amount(txout.Value).ToMASS()
		totalOutValue = totalOutValue + value
		to = append(to, &pb.ToAddressForTx{Address: addr.String(), Value: massutil.Amount(txout.Value).ToMASS()})
	}
	if blockchain.IsCoinBaseTx(mtx) {
		totalOutValue = 0.0
	}

	if mtx.Payload == nil {
		mtx.Payload = make([]byte, 0, 0)
	}

	vins, fromAddrs, inputs, totalInValue, err := s.createVinList(mtx)
	if err != nil {
		return nil, err
	}

	txid, err := wire.NewHashFromStr(txHash)
	if err != nil {
		st := status.New(ErrAPIShaHashFromStr, ErrCode[ErrAPIShaHashFromStr])
		return nil, st.Err()
	}
	code, err := s.getStatus(txid)
	if err != nil {
		return nil, err
	}

	txReply := &pb.TxRawResult{
		Hex:         mtxHex,
		Txid:        txHash,
		Version:     mtx.Version,
		LockTime:    mtx.LockTime,
		Vin:         vins,
		Vout:        voutList,
		FromAddress: fromAddrs,
		To:          to,
		Inputs:      inputs,
		Payload:     hex.EncodeToString(mtx.Payload),
		Size:        int32(mtx.SerializeSize()),
		Fee:         totalInValue - totalOutValue,
		Status:      code,
	}

	if blkHeader != nil {
		txReply.Block = &pb.BlockInfoForTx{Height: uint64(blkHeight), BlockHash: blkHash, Timestamp: blkHeader.Timestamp.Unix()}
		txReply.Confirmations = uint64(1 + chainHeight - blkHeight)
	}

	return txReply, nil
}
