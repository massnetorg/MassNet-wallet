package api

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"massnet.org/mass-wallet/consensus"

	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"
	"google.golang.org/grpc/status"
	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/blockchain"
	cfg "massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"
)

// no error for return
func (s *APIServer) GetClientStatus(ctx context.Context, in *empty.Empty) (*pb.GetClientStatusResponse, error) {
	logging.CPrint(logging.INFO, "api: GetClientStatus", logging.LogFormat{})

	height, err := s.massWallet.SyncedTo()
	if err != nil {
		return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
	}

	resp := &pb.GetClientStatusResponse{
		PeerListening:    s.node.SyncManager().Switch().IsListening(),
		Syncing:          !s.node.SyncManager().IsCaughtUp(),
		LocalBestHeight:  s.node.Blockchain().BestBlockHeight(),
		ChainId:          s.node.Blockchain().ChainID().String(),
		WalletSyncHeight: height,
	}

	if bestPeer := s.node.SyncManager().BestPeer(); bestPeer != nil {
		resp.KnownBestHeight = bestPeer.Height
	}
	if resp.LocalBestHeight > resp.KnownBestHeight {
		resp.KnownBestHeight = resp.LocalBestHeight
	}

	var outCount, inCount uint32
	resp.Peers = &pb.GetClientStatusResponsePeerList{
		Outbound: make([]*pb.GetClientStatusResponsePeerInfo, 0),
		Inbound:  make([]*pb.GetClientStatusResponsePeerInfo, 0),
		Other:    make([]*pb.GetClientStatusResponsePeerInfo, 0),
	}
	for _, info := range s.node.SyncManager().GetPeerInfos() {
		peer := &pb.GetClientStatusResponsePeerInfo{
			Id:      info.ID,
			Address: info.RemoteAddr,
		}
		if info.IsOutbound {
			outCount++
			peer.Direction = "outbound"
			resp.Peers.Outbound = append(resp.Peers.Outbound, peer)
			continue
		}
		inCount++
		peer.Direction = "inbound"
		resp.Peers.Inbound = append(resp.Peers.Inbound, peer)
	}
	resp.PeerCount = &pb.GetClientStatusResponsePeerCountInfo{Total: outCount + inCount, Outbound: outCount, Inbound: inCount}

	logging.CPrint(logging.INFO, "api: GetClientStatus completed", logging.LogFormat{})
	return resp, nil
}

func (s *APIServer) QuitClient(ctx context.Context, in *empty.Empty) (*pb.QuitClientResponse, error) {
	logging.CPrint(logging.INFO, "api: QuitClient completed", logging.LogFormat{})
	defer func() {
		go s.quitClient()
	}()
	return &pb.QuitClientResponse{
		Code: 200,
		Msg:  "wait for client quitting process",
	}, nil
}

func (s *APIServer) SignRawTransaction(ctx context.Context, in *pb.SignRawTransactionRequest) (*pb.SignRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "api: SignRawTransaction", logging.LogFormat{})

	if len(in.RawTx) == 0 {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidTxHex], logging.LogFormat{
			"err": in.RawTx})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}
	serializedTx, err := decodeHexStr(in.RawTx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in decodeHexStr", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}
	var tx wire.MsgTx
	err = tx.SetBytes(serializedTx, wire.Packet)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to decode tx", logging.LogFormat{"err": err.Error()})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}

	err = checkPassLen(in.Passphrase)
	if err != nil {
		s.massWallet.ClearUsedUTXOMark(&tx)
		return nil, err
	}

	//check in.Flags
	var flag string
	if len(in.Flags) == 0 {
		flag = "ALL"
	} else {
		flag = in.Flags
	}

	logging.CPrint(logging.INFO, "get tx", logging.LogFormat{"tx input count": len(tx.TxIn), "tx output count": len(tx.TxOut)})

	bufBytes, err := s.massWallet.SignRawTx([]byte(in.Passphrase), flag, &tx)
	if err != nil {
		s.massWallet.ClearUsedUTXOMark(&tx)
		return nil, convertResponseError(err)
	}

	logging.CPrint(logging.INFO, "api: SignRawTransaction completed", logging.LogFormat{})

	return &pb.SignRawTransactionResponse{
		Hex:      hex.EncodeToString(bufBytes),
		Complete: true,
	}, nil
}

func decodeHexStr(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "Hex string decode failed", logging.LogFormat{"err": err.Error()})
		return nil, err
	}
	return decoded, nil
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

	tx := massutil.NewTx(msgtx)
	_, err = s.node.Blockchain().ProcessTx(tx)
	if err != nil {
		logging.CPrint(logging.ERROR, "ProcessTx failed", logging.LogFormat{"err": err})
		s.massWallet.ClearUsedUTXOMark(msgtx)
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIRejectTx, ErrCode[ErrAPIRejectTx]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: SendRawTransaction completed", logging.LogFormat{
		"txHash": tx.Hash().String(),
	})
	return &pb.SendRawTransactionResponse{TxId: tx.Hash().String()}, nil

}

func (s *APIServer) GetLatestRewardList(ctx context.Context, in *empty.Empty) (*pb.GetLatestRewardListResponse, error) {

	bestHeight := s.node.Blockchain().BestBlockHeight()
	logging.CPrint(logging.INFO, "api: GetLatestRewardList", logging.LogFormat{"height": bestHeight})

	ranks, count, err := s.node.Blockchain().GetRewardStakingTx(bestHeight)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetRewardLockTx failed", logging.LogFormat{"err": err, "height": bestHeight})
		return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
	}

	scriptHashExist := make(map[[sha256.Size]byte]struct{})
	for _, rank := range ranks {
		scriptHashExist[rank.ScriptHash] = struct{}{}
	}

	block, err := s.node.Blockchain().GetBlockByHeight(bestHeight)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetBlockByHeight failed", logging.LogFormat{"err": err, "height": bestHeight})
		return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
	}
	coinbase, err := block.Tx(0)
	if err != nil {
		logging.CPrint(logging.ERROR, "coinbase error", logging.LogFormat{"err": err, "height": bestHeight})
		return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
	}

	txOuts := coinbase.MsgTx().TxOut

	coinbasePayload := blockchain.NewCoinbasePayload()
	err = coinbasePayload.SetBytes(coinbase.MsgTx().Payload)
	if err != nil {
		logging.CPrint(logging.ERROR, "coinbase payload error", logging.LogFormat{"err": err, "height": bestHeight})
		return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
	}
	numLock := coinbasePayload.NumStakingReward()
	addrToProfit := make(map[[sha256.Size]byte]int64)
	for j := 0; uint32(j) < numLock; j++ {
		class, pops := txscript.GetScriptInfo(txOuts[j].PkScript)
		_, hash, err := txscript.GetParsedOpcode(pops, class)
		if err != nil {
			logging.CPrint(logging.ERROR, "pkscript error", logging.LogFormat{"err": err, "height": bestHeight})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}

		if _, ok := scriptHashExist[hash]; !ok {
			break
		}
		addrToProfit[hash] = txOuts[j].Value
	}

	var address string
	details := make([]*pb.GetLatestRewardListResponse_RewardDetail, 0)
	for _, rank := range ranks {

		key := make([]byte, sha256.Size)
		copy(key, rank.ScriptHash[:])
		scriptHashStruct, err := massutil.NewAddressStakingScriptHash(key, &cfg.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "script hash error", logging.LogFormat{"err": err, "height": bestHeight})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		address = scriptHashStruct.EncodeAddress()
		if err != nil {
			logging.CPrint(logging.ERROR, "encode address error", logging.LogFormat{"err": err, "height": bestHeight})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}

		total := massutil.ZeroAmount()
		for _, ltxInfo := range rank.StakingTx {
			total, err = total.AddInt(int64(ltxInfo.Value))
			if err != nil {
				logging.CPrint(logging.ERROR, "total amount error", logging.LogFormat{"err": err, "height": bestHeight})
				return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
			}
		}
		amt, err := AmountToString(total.IntValue())
		if err != nil {
			logging.CPrint(logging.ERROR, "AmountToString error", logging.LogFormat{
				"err":    err,
				"height": bestHeight,
			})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		profit, err := AmountToString(addrToProfit[rank.ScriptHash])
		if err != nil {
			logging.CPrint(logging.ERROR, "AmountToString failed", logging.LogFormat{
				"err":    err,
				"height": bestHeight,
			})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		details = append(details, &pb.GetLatestRewardListResponse_RewardDetail{
			Rank:    rank.Rank,
			Address: address,
			Weight:  rank.Weight.Float64(),
			Amount:  amt,
			Profit:  profit,
		})
	}

	sort.Slice(details, func(i, j int) bool {
		return details[i].Rank < details[j].Rank
	})

	reply := &pb.GetLatestRewardListResponse{
		Details: details,
	}
	logging.CPrint(logging.INFO, "api: GetLatestRewardList completed", logging.LogFormat{
		"number": count,
	})
	return reply, nil
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

func (s *APIServer) GetStakingHistory(ctx context.Context, in *empty.Empty) (*pb.GetStakingHistoryResponse, error) {
	logging.CPrint(logging.INFO, "api: GetStakingHistory", logging.LogFormat{})
	newestHeight := s.node.Blockchain().BestBlockHeight()
	rewards, _, err := s.node.Blockchain().GetInStakingTx(newestHeight)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to FetchAllLockTx from chainDB", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIGetStakingTxDetail, ErrCode[ErrAPIGetStakingTxDetail])
		return nil, st.Err()
	}

	stakingTxs, err := s.massWallet.GetStakingHistory()
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

			scriptHashStruct, err := massutil.NewAddressStakingScriptHash(scriptHash, &cfg.ChainParams)
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

func (s *APIServer) CreateAddress(ctx context.Context, in *pb.CreateAddressRequest) (*pb.CreateAddressResponse, error) {
	logging.CPrint(logging.INFO, "api: CreateAddress", logging.LogFormat{"version": in.Version})

	addressClass := uint16(in.Version)
	if !massutil.IsValidAddressClass(addressClass) {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidVersion], logging.LogFormat{
			"version": in.Version,
		})
		st := status.New(ErrAPIInvalidVersion, ErrCode[ErrAPIInvalidVersion])
		return nil, st.Err()
	}

	ads, err := s.massWallet.GetAddresses(addressClass)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetAddresses failed", logging.LogFormat{"err": err})
		return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
	}

	count := 0
	for _, ad := range ads {
		if !ad.Used {
			count++
		}
	}
	if addressClass == massutil.AddressClassWitnessStaking &&
		count >= int(s.config.Advanced.MaxUnusedStakingAddress) {
		return nil, status.New(ErrAPIUnusedAddressLimit, ErrCode[ErrAPIUnusedAddressLimit]).Err()
	}
	if addressClass == massutil.AddressClassWitnessV0 &&
		count >= int(s.config.Advanced.AddressGapLimit-s.config.Advanced.MaxUnusedStakingAddress) {
		return nil, status.New(ErrAPIUnusedAddressLimit, ErrCode[ErrAPIUnusedAddressLimit]).Err()
	}

	address, err := s.massWallet.NewAddress(addressClass)
	if err != nil {
		logging.CPrint(logging.ERROR, "new address error", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: CreateAddress completed", logging.LogFormat{})
	return &pb.CreateAddressResponse{Address: address}, nil
}

func (s *APIServer) GetAddresses(ctx context.Context, in *pb.GetAddressesRequest) (*pb.GetAddressesResponse, error) {
	logging.CPrint(logging.INFO, "api: GetAddresses", logging.LogFormat{"version": in.Version})
	addressClass := uint16(in.Version)
	if !massutil.IsValidAddressClass(addressClass) {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidVersion], logging.LogFormat{
			"version": in.Version,
		})
		return nil, status.New(ErrAPIInvalidVersion, ErrCode[ErrAPIInvalidVersion]).Err()
	}

	ads, err := s.massWallet.GetAddresses(addressClass)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetAddresses failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}

	details := make([]*pb.GetAddressesResponse_AddressDetail, 0)
	for _, ad := range ads {
		pbAd := &pb.GetAddressesResponse_AddressDetail{
			Address:    ad.Address,
			Version:    int32(ad.AddressClass),
			Used:       ad.Used,
			StdAddress: ad.StdAddress,
		}
		details = append(details, pbAd)
	}
	reps := &pb.GetAddressesResponse{
		Details: details,
	}

	logging.CPrint(logging.INFO, "api: GetAddress completed", logging.LogFormat{
		"addressNumber": len(details),
	})
	return reps, nil
}

func (s *APIServer) ValidateAddress(ctx context.Context, in *pb.ValidateAddressRequest) (*pb.ValidateAddressResponse, error) {
	logging.CPrint(logging.INFO, "api: ValidateAddress", logging.LogFormat{"address": in.Address})
	err := checkAddressLen(in.Address)
	if err != nil {
		return nil, err
	}

	witAddr, isMine, err := s.massWallet.IsAddressInCurrent(in.Address)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to check validate address", logging.LogFormat{
			"err":     err,
			"address": in.Address,
		})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress]).Err()
		}
		return nil, cvtErr
	}

	var ver uint16
	if massutil.IsWitnessV0Address(witAddr) {
		ver = massutil.AddressClassWitnessV0
	} else if massutil.IsWitnessStakingAddress(witAddr) {
		ver = massutil.AddressClassWitnessStaking
	} else {
		return &pb.ValidateAddressResponse{}, nil
	}
	logging.CPrint(logging.INFO, "api: ValidateAddress completed", logging.LogFormat{"address": in.Address})
	return &pb.ValidateAddressResponse{
		IsValid: true,
		IsMine:  isMine,
		Address: witAddr.EncodeAddress(),
		Version: int32(ver),
	}, nil
}

func (s *APIServer) GetWalletBalance(ctx context.Context, in *pb.GetWalletBalanceRequest) (*pb.GetWalletBalanceResponse, error) {
	logging.CPrint(logging.INFO, "api: GetWalletBalance", logging.LogFormat{"params": in})

	// uint32 is the set of all unsigned 32-bit integers.
	// Range: 0 through 4294967295. (in.RequiredConfirmations int32)
	// int32 is the set of all signed 32-bit integers.
	// Range: -2147483648 through 2147483647.
	if in.RequiredConfirmations < 0 {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidParameter], logging.LogFormat{
			"confs": in.RequiredConfirmations,
		})
		return nil, status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter]).Err()
	}

	bal, err := s.massWallet.WalletBalance(uint32(in.RequiredConfirmations), in.Detail)
	if err != nil {
		logging.CPrint(logging.ERROR, "WalletBalance failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}

	total, err := checkFormatAmount(bal.Total)
	if err != nil {
		return nil, err
	}
	spendable, err := checkFormatAmount(bal.Spendable)
	if err != nil {
		return nil, err
	}
	withdrawableStaking, err := checkFormatAmount(bal.WithdrawableStaking)
	if err != nil {
		return nil, err
	}
	withdrawableBinding, err := checkFormatAmount(bal.WithdrawableBinding)
	if err != nil {
		return nil, err
	}
	logging.CPrint(logging.INFO, "api: GetWalletBalance completed", logging.LogFormat{"walletId": bal.WalletID})
	return &pb.GetWalletBalanceResponse{
		WalletId: bal.WalletID,
		Total:    total,
		Detail: &pb.GetWalletBalanceResponse_Detail{
			Spendable:           spendable,
			WithdrawableStaking: withdrawableStaking,
			WithdrawableBinding: withdrawableBinding,
		},
	}, nil
}

func (s *APIServer) GetAddressBalance(ctx context.Context, in *pb.GetAddressBalanceRequest) (*pb.GetAddressBalanceResponse, error) {
	logging.CPrint(logging.INFO, "api: GetAddressBalance", logging.LogFormat{"params": in})

	if in.RequiredConfirmations < 0 {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidParameter], logging.LogFormat{
			"confs": in.RequiredConfirmations,
		})
		return nil, status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter]).Err()
	}

	for _, addr := range in.Addresses {
		err := checkAddressLen(addr)
		if err != nil {
			return nil, err
		}
	}

	bals, err := s.massWallet.AddressBalance(uint32(in.RequiredConfirmations), in.Addresses)
	if err != nil {
		logging.CPrint(logging.ERROR, "AddressBalance failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	balances := make([]*pb.AddressAndBalance, 0)
	for _, bal := range bals {
		total, err := checkFormatAmount(bal.Total)
		if err != nil {
			return nil, err
		}
		spendable, err := checkFormatAmount(bal.Spendable)
		if err != nil {
			return nil, err
		}
		withdrawableStaking, err := checkFormatAmount(bal.WithdrawableStaking)
		if err != nil {
			return nil, err
		}
		withdrawableBinding, err := checkFormatAmount(bal.WithdrawableBinding)
		if err != nil {
			return nil, err
		}
		balances = append(balances, &pb.AddressAndBalance{
			Address:             bal.Address,
			Total:               total,
			Spendable:           spendable,
			WithdrawableStaking: withdrawableStaking,
			WithdrawableBinding: withdrawableBinding,
		})
	}

	logging.CPrint(logging.INFO, "api: GetAddressBalance completed", logging.LogFormat{})
	return &pb.GetAddressBalanceResponse{
		Balances: balances,
	}, nil
}

func (s *APIServer) UseWallet(ctx context.Context, in *pb.UseWalletRequest) (*pb.UseWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: UseWallet", logging.LogFormat{"walletId": in.WalletId})

	err := checkWalletIdLen(in.WalletId)
	if err != nil {
		return nil, err
	}

	info, err := s.massWallet.UseWallet(in.WalletId)
	if err != nil {
		logging.CPrint(logging.ERROR, "UseWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}
	totalBalance, err := checkFormatAmount(info.TotalBalance)
	if err != nil {
		return nil, err
	}

	logging.CPrint(logging.INFO, "api: UseWallet completed", logging.LogFormat{})
	return &pb.UseWalletResponse{
		ChainId:          info.ChainID,
		WalletId:         info.WalletID,
		Type:             info.Type,
		TotalBalance:     totalBalance,
		ExternalKeyCount: info.ExternalKeyCount,
		InternalKeyCount: info.InternalKeyCount,
		Remarks:          info.Remarks,
	}, nil
}

func (s *APIServer) Wallets(ctx context.Context, in *empty.Empty) (*pb.WalletsResponse, error) {
	logging.CPrint(logging.INFO, "api: Wallets", logging.LogFormat{})

	wallets := make([]*pb.WalletsResponse_WalletSummary, 0)
	summaries, err := s.massWallet.Wallets()
	if err != nil {
		logging.CPrint(logging.ERROR, "Wallets failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	for _, summary := range summaries {
		wallets = append(wallets, &pb.WalletsResponse_WalletSummary{
			WalletId:     summary.WalletID,
			Type:         summary.Type,
			Remarks:      summary.Remarks,
			Ready:        summary.Ready,
			SyncedHeight: summary.SyncedHeight,
		})
	}

	logging.CPrint(logging.INFO, "api: Wallets completed", logging.LogFormat{})
	return &pb.WalletsResponse{
		Wallets: wallets,
	}, nil
}

func (s *APIServer) GetUtxo(ctx context.Context, in *pb.GetUtxoRequest) (*pb.GetUtxoResponse, error) {
	logging.CPrint(logging.INFO, "api: GetUtxo", logging.LogFormat{"addresses": in.Addresses})

	for _, addr := range in.Addresses {
		err := checkAddressLen(addr)
		if err != nil {
			return nil, err
		}
	}

	m, err := s.massWallet.GetUtxo(in.Addresses)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetUtxo failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	list := make([]*pb.AddressUTXO, 0)
	for k, v := range m {
		utxos := make([]*pb.UTXO, 0)
		for _, item := range v {
			amt, err := checkFormatAmount(item.Amount)
			if err != nil {
				return nil, err
			}
			utxos = append(utxos, &pb.UTXO{
				TxId:           item.TxId,
				Vout:           item.Vout,
				Amount:         amt,
				BlockHeight:    uint64(item.BlockHeight),
				Maturity:       item.Maturity,
				Confirmations:  item.Confirmations,
				SpentByUnmined: item.SpentByUnmined,
			})
		}
		list = append(list, &pb.AddressUTXO{
			Address: k,
			Utxos:   utxos,
		})
	}

	logging.CPrint(logging.INFO, "api: GetUtxo completed", logging.LogFormat{})
	return &pb.GetUtxoResponse{
		AddressUtxos: list,
	}, nil
}

func (s *APIServer) ImportWallet(ctx context.Context, in *pb.ImportWalletRequest) (*pb.ImportWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: ImportWallet", logging.LogFormat{})
	err := checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	ws, err := s.massWallet.ImportWallet(in.Keystore, in.Passphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "ImportWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: ImportWallet completed",
		logging.LogFormat{
			"walletId": ws.WalletID,
			"remarks":  ws.Remarks,
		})
	return &pb.ImportWalletResponse{
		Ok:       true,
		WalletId: ws.WalletID,
		Type:     ws.Type,
		Remarks:  ws.Remarks,
	}, nil
}

func (s *APIServer) ImportWalletWithMnemonic(ctx context.Context, in *pb.ImportWalletWithMnemonicRequest) (*pb.ImportWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: ImportWalletWithMnemonic", logging.LogFormat{"remarks": in.Remarks})

	err := checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	err = checkMnemonicLen(in.Mnemonic)
	if err != nil {
		return nil, err
	}

	remarks := checkRemarksLen(in.Remarks)

	ws, err := s.massWallet.ImportWalletWithMnemonic(in.Mnemonic, in.Passphrase, remarks, in.ExternalIndex, in.InternalIndex)
	if err != nil {
		logging.CPrint(logging.ERROR, "ImportWalletWithMnemonic failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: ImportWalletWithMnemonic completed",
		logging.LogFormat{
			"wallet id": ws.WalletID,
		})
	return &pb.ImportWalletResponse{
		Ok:       true,
		WalletId: ws.WalletID,
		Type:     ws.Type,
		Remarks:  ws.Remarks,
	}, nil
}

func (s *APIServer) CreateWallet(ctx context.Context, in *pb.CreateWalletRequest) (*pb.CreateWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: CreateWallet",
		logging.LogFormat{
			"bit size": in.BitSize,
			"remarks":  in.Remarks,
		})

	err := checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	remarks := checkRemarksLen(in.Remarks)

	name, mnemonic, err := s.massWallet.CreateWallet(in.Passphrase, remarks, int(in.BitSize))
	if err != nil {
		logging.CPrint(logging.ERROR, "CreateWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: CreateWallet completed", logging.LogFormat{})
	return &pb.CreateWalletResponse{
		WalletId: name,
		Mnemonic: mnemonic,
	}, nil
}

func (s *APIServer) ExportWallet(ctx context.Context, in *pb.ExportWalletRequest) (*pb.ExportWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: ExportWallet", logging.LogFormat{"walletId": in.WalletId})

	err := checkWalletIdLen(in.WalletId)
	if err != nil {
		return nil, err
	}

	err = checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	keystoreJSON, err := s.massWallet.ExportWallet(in.WalletId, in.Passphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "ExportWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: ExportWallet completed", logging.LogFormat{})
	return &pb.ExportWalletResponse{
		Keystore: keystoreJSON,
	}, nil
}

func (s *APIServer) RemoveWallet(ctx context.Context, in *pb.RemoveWalletRequest) (*pb.RemoveWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: RemoveWallet", logging.LogFormat{"walletId": in.WalletId})

	err := checkWalletIdLen(in.WalletId)
	if err != nil {
		return nil, err
	}

	err = checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	err = s.massWallet.RemoveWallet(in.WalletId, in.Passphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "RemoveWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: RemoveWallet completed", logging.LogFormat{})
	return &pb.RemoveWalletResponse{
		Ok: true,
	}, nil
}

func (s *APIServer) GetAddressBinding(ctx context.Context, in *pb.GetAddressBindingRequest) (*pb.GetAddressBindingResponse, error) {
	logging.CPrint(logging.INFO, "api: GetAddressBinding", logging.LogFormat{"addresses": in.Addresses})

	if len(in.Addresses) == 0 {
		logging.CPrint(logging.ERROR, "failed to decode address", logging.LogFormat{
			"addresses": in.Addresses,
		})
		return nil, status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter]).Err()
	}

	for _, addr := range in.Addresses {
		err := checkAddressLen(addr)
		if err != nil {
			return nil, err
		}
	}

	addrToValue := make(map[string]string)
	for _, addr := range in.Addresses {
		witnessAddr, err := checkPoCPubKeyAddress(addr, &cfg.ChainParams)
		if err != nil {
			return nil, err
		}

		gTxs, err := s.node.Blockchain().FetchScriptHashRelatedBindingTx(witnessAddr.ScriptAddress(), &cfg.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to get BindingTx", logging.LogFormat{
				"err": err,
			})
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		total := massutil.ZeroAmount()
		for _, gTx := range gTxs {
			total, err = total.AddInt(gTx.Value)
			if err != nil {
				logging.CPrint(logging.ERROR, "amount error", logging.LogFormat{"err": err})
				return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
			}
		}
		amt, err := checkFormatAmount(total)
		if err != nil {
			return nil, err
		}
		addrToValue[addr] = amt
	}

	logging.CPrint(logging.INFO, "api: GetAddressBinding completed", logging.LogFormat{})
	return &pb.GetAddressBindingResponse{
		Amounts: addrToValue,
	}, nil
}

func (s *APIServer) GetBindingHistory(ctx context.Context, in *empty.Empty) (*pb.GetBindingHistoryResponse, error) {
	logging.CPrint(logging.INFO, "api: GetBindingHistory", logging.LogFormat{})

	details, err := s.massWallet.GetBindingHistory()
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
							ps, err := utils.ParsePkScript(prevMtx.TxOut[txIn.PreviousOutPoint.Index].PkScript, &cfg.ChainParams)
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
				TxId:           detail.Utxo.Hash.String(),
				Vout:           detail.Utxo.Index,
				HolderAddress:  detail.Utxo.HolderAddress,
				BindingAddress: detail.Utxo.BindingAddress,
				Amount:         amt,
			},
			FromAddresses: froms,
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

// no error for return
func (s *APIServer) GetBestBlock(ctx context.Context, in *empty.Empty) (*pb.GetBestBlockResponse, error) {
	logging.CPrint(logging.INFO, "api: GetBestBlock", logging.LogFormat{})

	bestHeader := s.node.Blockchain().BestBlockHeader()

	logging.CPrint(logging.INFO, "api: GetBestBlock completed", logging.LogFormat{})
	return &pb.GetBestBlockResponse{
		Height: bestHeader.Height,
		Target: bestHeader.Target.Text(16),
	}, nil
}

func (s *APIServer) GetWalletMnemonic(ctw context.Context, in *pb.GetWalletMnemonicRequest) (*pb.GetWalletMnemonicResponse, error) {
	logging.CPrint(logging.INFO, "api: GetWalletMnemonic", logging.LogFormat{"walletId": in.WalletId})

	err := checkWalletIdLen(in.WalletId)
	if err != nil {
		return nil, err
	}

	err = checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	mnemonic, err := s.massWallet.GetMnemonic(in.WalletId, in.Passphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetMnemonic failed", logging.LogFormat{"err": err})
		return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
	}
	logging.CPrint(logging.INFO, "api: GetWalletMnemonic completed", logging.LogFormat{})
	return &pb.GetWalletMnemonicResponse{
		Mnemonic: mnemonic,
	}, nil
}
