package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/massnetorg/mass-core/blockchain"
	"github.com/massnetorg/mass-core/interfaces"
	"github.com/massnetorg/mass-core/logging"
	"github.com/massnetorg/mass-core/massutil"
	"github.com/massnetorg/mass-core/pocec"
	"github.com/massnetorg/mass-core/txscript"
	"github.com/massnetorg/mass-core/wire"
	wirepb "github.com/massnetorg/mass-core/wire/pb"
	"google.golang.org/grpc/status"
	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/config"
)

func (s *APIServer) GetBestBlock(ctx context.Context, in *empty.Empty) (*pb.GetBlockResponse, error) {
	logging.CPrint(logging.INFO, "api: GetBestBlock", logging.LogFormat{})

	height := s.node.Blockchain().BestBlockHeight()
	block, err := s.node.Blockchain().GetBlockByHeight(height)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to get best block", logging.LogFormat{"height": height, "err": err})
		st := status.New(ErrAPIBlockNotFound, ErrCode[ErrAPIBlockNotFound])
		return nil, st.Err()
	}

	return s.marshalGetBlockResponse(block)
}

func (s *APIServer) GetBlockByHeight(ctx context.Context, in *pb.GetBlockByHeightRequest) (*pb.GetBlockResponse, error) {
	logging.CPrint(logging.INFO, "api: GetBlockByHeight", logging.LogFormat{"height": in.Height})
	block, err := s.node.Blockchain().GetBlockByHeight(in.Height)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to query the block according to the block height", logging.LogFormat{"height": in.Height, "error": err})
		st := status.New(ErrAPIBlockNotFound, ErrCode[ErrAPIBlockNotFound])
		return nil, st.Err()
	}

	return s.marshalGetBlockResponse(block)
}

func (s *APIServer) marshalGetBlockResponse(blk *massutil.Block) (*pb.GetBlockResponse, error) {
	idx := blk.Height()
	maxIdx := s.node.Blockchain().BestBlockHeight()
	var shaNextStr string
	if idx < maxIdx {
		shaNext, err := s.node.Blockchain().GetBlockHashByHeight(idx + 1)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to query next block hash according to the block height", logging.LogFormat{"height": idx, "error": err})
			st := status.New(ErrAPIBlockNotFound, ErrCode[ErrAPIBlockNotFound])
			return nil, st.Err()
		}
		shaNextStr = shaNext.String()
	}
	blockHeader := &blk.MsgBlock().Header
	blockHash := blk.Hash().String()

	banList := make([]string, 0, len(blockHeader.BanList))
	for _, pk := range blockHeader.BanList {
		banList = append(banList, hex.EncodeToString(pk.SerializeCompressed()))
	}

	var proof = wirepb.ProofToProto(blockHeader.Proof)
	blockReply := &pb.GetBlockResponse{
		Hash:            blockHash,
		ChainId:         blockHeader.ChainID.String(),
		Version:         blockHeader.Version,
		Height:          idx,
		Confirmations:   maxIdx + 1 - idx,
		Time:            blockHeader.Timestamp.Unix(),
		PreviousHash:    blockHeader.Previous.String(),
		NextHash:        shaNextStr,
		TransactionRoot: blockHeader.TransactionRoot.String(),
		WitnessRoot:     blockHeader.WitnessRoot.String(),
		ProposalRoot:    blockHeader.ProposalRoot.String(),
		Target:          blockHeader.Target.Text(16),
		Quality:         blk.MsgBlock().Header.Quality().Text(16),
		Challenge:       hex.EncodeToString(blockHeader.Challenge.Bytes()),
		PublicKey:       hex.EncodeToString(blockHeader.PubKey.SerializeCompressed()),
		Proof:           &pb.GetBlockResponse_Proof{X: hex.EncodeToString(proof.X), XPrime: hex.EncodeToString(proof.XPrime), BitLength: proof.BitLength},
		BlockSignature:  createPoCSignatureResult(blockHeader.Signature),
		BanList:         banList,
		Size:            uint32(blk.Size()),
		TimeUtc:         blockHeader.Timestamp.UTC().Format(time.RFC3339),
		TxCount:         uint32(len(blk.Transactions())),
		BindingRoot:     hex.EncodeToString(blockHeader.BindingRoot[:]),
	}
	proposalArea := blk.MsgBlock().Proposals
	punishments := createFaultPubKeyResult(proposalArea.PunishmentArea)
	others := createNormalProposalResult(proposalArea.OtherArea)

	blockReply.ProposalArea = &pb.GetBlockResponse_ProposalArea{
		PunishmentArea: punishments,
		OtherArea:      others,
	}

	txns := blk.Transactions()
	rawTxns := make([]*pb.GetBlockResponse_TxRawResult, len(txns))
	for i, tx := range txns {
		rawTxn, err := s.createBlockTx(tx.MsgTx(), blockHeader, idx)

		if err != nil {
			logging.CPrint(logging.ERROR, "failed to query transactions in the block", logging.LogFormat{
				"height":   idx,
				"error":    err,
				"function": "GetBlock",
			})
			return nil, err
		}
		rawTxns[i] = rawTxn
	}
	blockReply.RawTx = rawTxns

	return blockReply, nil
}

// Tx type codes are shown below:
//  -----------------------------------------------------
// |  Tx Type  | Staking | Binding | Ordinary | Coinbase |
// |-----------------------------------------------------|
// | Type Code |    1    |    2    |     3    |     4    |
//   ----------------------------------------------------
func (s *APIServer) getTxType(tx *wire.MsgTx) (int32, error) {
	if blockchain.IsCoinBaseTx(tx) {
		return 4, nil
	}
	for _, txOut := range tx.TxOut {
		if txscript.IsPayToStakingScriptHash(txOut.PkScript) {
			return 1, nil
		}
		if txscript.IsPayToBindingScriptHash(txOut.PkScript) {
			return 2, nil
		}
	}
	for _, txIn := range tx.TxIn {
		hash := txIn.PreviousOutPoint.Hash
		index := txIn.PreviousOutPoint.Index
		tx, err := s.node.Blockchain().GetTransaction(&hash)
		if err != nil {
			logging.CPrint(logging.ERROR, "No information available about transaction in db", logging.LogFormat{"err": err, "txid": hash.String()})
			st := status.New(ErrAPINoTxInfo, ErrCode[ErrAPINoTxInfo])
			return -1, st.Err()
		}
		if txscript.IsPayToStakingScriptHash(tx.TxOut[index].PkScript) {
			return 1, nil
		}
		if txscript.IsPayToBindingScriptHash(tx.TxOut[index].PkScript) {
			return 2, nil
		}
	}
	return 3, nil
}

func (s *APIServer) createBlockTx(mtx *wire.MsgTx, blkHeader *wire.BlockHeader, chainHeight uint64) (*pb.GetBlockResponse_TxRawResult, error) {

	txType, err := s.getTxType(mtx)
	if err != nil {
		return nil, err
	}

	vouts, totalOutValue, err := createVoutList(mtx, config.ChainParams)
	if err != nil {
		return nil, err
	}
	if mtx.Payload == nil {
		mtx.Payload = make([]byte, 0, 0)
	}

	vins, totalInValue, err := s.createVinList(mtx, txType == 4)
	if err != nil {
		return nil, err
	}

	fee := "0"
	if txType != 4 {
		fee, err = AmountToString(totalInValue - totalOutValue)
		if err != nil {
			logging.CPrint(logging.ERROR, "Failed to transfer amount to string", logging.LogFormat{"err": err})
			return nil, err
		}
	}

	// TxId
	txHash := mtx.TxHash()
	// Status
	code, err := s.getStatus(&txHash)
	if err != nil {
		return nil, err
	}

	txReply := &pb.GetBlockResponse_TxRawResult{
		Txid:          txHash.String(),
		Version:       mtx.Version,
		LockTime:      mtx.LockTime,
		Vin:           vins,
		Vout:          vouts,
		Payload:       hex.EncodeToString(mtx.Payload),
		Confirmations: 1 + chainHeight - blkHeader.Height,
		Size:          uint32(mtx.PlainSize()),
		Fee:           fee,
		Status:        code,
		Type:          txType,
	}

	return txReply, nil
}

func createNormalProposalResult(proposals []*wire.NormalProposal) []*pb.GetBlockResponse_ProposalArea_NormalProposal {
	result := make([]*pb.GetBlockResponse_ProposalArea_NormalProposal, 0, len(proposals))
	for _, p := range proposals {
		np := &pb.GetBlockResponse_ProposalArea_NormalProposal{
			Version:      p.Version(),
			ProposalType: uint32(p.Type()),
			Data:         hex.EncodeToString(p.Content()),
		}
		result = append(result, np)
	}
	return result
}

func createFaultPubKeyResult(proposals []*wire.FaultPubKey) []*pb.GetBlockResponse_ProposalArea_FaultPubKey {
	result := make([]*pb.GetBlockResponse_ProposalArea_FaultPubKey, 0, len(proposals))
	for _, p := range proposals {
		t := make([]*pb.GetBlockResponse_ProposalArea_FaultPubKey_Header, 0, wire.HeadersPerProposal)
		for _, h := range p.Testimony {
			ban := make([]string, 0, len(h.BanList))
			for _, pk := range h.BanList {
				ban = append(ban, hex.EncodeToString(pk.SerializeCompressed()))
			}

			var proof = wirepb.ProofToProto(h.Proof)
			th := &pb.GetBlockResponse_ProposalArea_FaultPubKey_Header{
				Hash:            h.BlockHash().String(),
				ChainId:         h.ChainID.String(),
				Version:         h.Version,
				Height:          h.Height,
				Time:            h.Timestamp.Unix(),
				PreviousHash:    h.Previous.String(),
				TransactionRoot: h.TransactionRoot.String(),
				WitnessRoot:     h.WitnessRoot.String(),
				ProposalRoot:    h.ProposalRoot.String(),
				Target:          h.Target.Text(16),
				Challenge:       hex.EncodeToString(h.Challenge.Bytes()),
				PublicKey:       hex.EncodeToString(h.PubKey.SerializeCompressed()),
				Proof:           &pb.GetBlockResponse_Proof{X: hex.EncodeToString(proof.X), XPrime: hex.EncodeToString(proof.XPrime), BitLength: proof.BitLength},
				BlockSignature:  createPoCSignatureResult(h.Signature),
				BanList:         ban,
			}
			t = append(t, th)
		}

		fpk := &pb.GetBlockResponse_ProposalArea_FaultPubKey{
			Version:      p.Version(),
			ProposalType: uint32(p.Type()),
			PublicKey:    hex.EncodeToString(p.PubKey.SerializeCompressed()),
			Testimony:    t,
		}
		result = append(result, fpk)
	}
	return result
}

// TODO: show more types
func createPoCSignatureResult(sigI interfaces.Signature) *pb.GetBlockResponse_PoCSignature {
	if sig, ok := sigI.(*pocec.Signature); ok {
		return &pb.GetBlockResponse_PoCSignature{R: hex.EncodeToString(sig.R.Bytes()), S: hex.EncodeToString(sig.S.Bytes())}
	}
	return &pb.GetBlockResponse_PoCSignature{}
}

func (s *APIServer) GetBlockStakingReward(ctx context.Context, in *pb.GetBlockStakingRewardRequest) (*pb.GetBlockStakingRewardResponse, error) {

	logging.CPrint(logging.INFO, "api: GetBlockStakingReward", logging.LogFormat{"height": in.Height})
	if in.Height > s.node.Blockchain().BestBlockHeight() {
		return nil, status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter]).Err()
	}

	height := in.Height
	if height == 0 {
		height = s.node.Blockchain().BestBlockHeight()
	}
	ranks, err := s.node.Blockchain().GetBlockStakingRewardRankOnList(height)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetBlockStakingRewardList failed", logging.LogFormat{"err": err, "height": height})
		return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
	}

	block, err := s.node.Blockchain().GetBlockByHeight(height)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetBlockByHeight failed", logging.LogFormat{"err": err, "height": height})
		return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
	}
	coinbase, err := block.Tx(0)
	if err != nil {
		logging.CPrint(logging.ERROR, "coinbase error", logging.LogFormat{"err": err, "height": height})
		return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
	}

	coinbasePayload := blockchain.NewCoinbasePayload()
	err = coinbasePayload.SetBytes(coinbase.MsgTx().Payload)
	if err != nil {
		logging.CPrint(logging.ERROR, "coinbase payload error", logging.LogFormat{"err": err, "height": height})
		return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
	}

	txOuts := coinbase.MsgTx().TxOut
	rankToProfit := make(map[[sha256.Size]byte]int64)
	for _, rank := range ranks {
		rankToProfit[rank.ScriptHash] = 0
	}

	for j := 0; uint32(j) < coinbasePayload.NumStakingReward(); j++ {
		class, pops := txscript.GetScriptInfo(txOuts[j].PkScript)
		_, hash, err := txscript.GetParsedOpcode(pops, class)
		if err != nil {
			logging.CPrint(logging.ERROR, "pkscript error", logging.LogFormat{"err": err, "height": height})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}

		if profit, ok := rankToProfit[hash]; !ok || profit != 0 {
			logging.CPrint(logging.ERROR, "unexpected staking reward",
				logging.LogFormat{
					"exist":  ok,
					"height": height,
					"profit": profit,
				})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		rankToProfit[hash] = txOuts[j].Value
	}

	list := make([]*pb.GetBlockStakingRewardResponse_RewardDetail, 0)
	for idx, rank := range ranks {
		if rankToProfit[rank.ScriptHash] == 0 {
			break
		}

		scriptHashStruct, err := massutil.NewAddressStakingScriptHash(rank.ScriptHash[:], config.ChainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "script hash error", logging.LogFormat{
				"err":        err,
				"height":     height,
				"rank_index": idx,
			})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		address := scriptHashStruct.EncodeAddress()
		if err != nil {
			logging.CPrint(logging.ERROR, "encode address error", logging.LogFormat{
				"err":        err,
				"height":     height,
				"rank_index": idx,
			})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}

		total := massutil.ZeroAmount()
		for _, stk := range rank.StakingTx {
			total, err = total.AddInt(int64(stk.Value))
			if err != nil {
				logging.CPrint(logging.ERROR, "total amount error", logging.LogFormat{
					"err":        err,
					"height":     height,
					"rank_index": idx,
				})
				return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
			}
		}
		amt, err := AmountToString(total.IntValue())
		if err != nil {
			logging.CPrint(logging.ERROR, "AmountToString error", logging.LogFormat{
				"err":        err,
				"height":     height,
				"rank_index": idx,
			})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}

		profit, err := AmountToString(rankToProfit[rank.ScriptHash])
		if err != nil {
			logging.CPrint(logging.ERROR, "AmountToString failed", logging.LogFormat{
				"err":        err,
				"height":     height,
				"rank_index": idx,
			})
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}

		list = append(list, &pb.GetBlockStakingRewardResponse_RewardDetail{
			Rank:    rank.Rank,
			Address: address,
			Weight:  rank.Weight.Float64(),
			Amount:  amt,
			Profit:  profit,
		})
	}

	if int(coinbasePayload.NumStakingReward()) != len(list) {
		logging.CPrint(logging.ERROR, "unexpected error",
			logging.LogFormat{
				"reward":      coinbasePayload.NumStakingReward(),
				"list_length": len(list),
				"height":      height,
			})
		return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
	}

	reply := &pb.GetBlockStakingRewardResponse{
		Details: list,
		Height:  height,
	}
	logging.CPrint(logging.INFO, "api: GetBlockStakingReward completed", logging.LogFormat{
		"number_of_rank": len(ranks),
		"number_of_list": len(list),
	})
	return reply, nil
}
